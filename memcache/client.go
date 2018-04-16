package memcache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
)

const (
	defaultQueueSize = 16
)

var (
	ErrInvalidClientConfig = errors.New("Invalid client configuration")
	ErrNoResult            = errors.New("No result")
	ErrClientShutdown      = errors.New("Client shutdown")

	ErrKeyNotFound                   = errors.New("Key not found")
	ErrKeyExists                     = errors.New("Key exists")
	ErrValueTooLarge                 = errors.New("Value too large")
	ErrInvalidArguments              = errors.New("Invalid arguments")
	ErrItemNotStored                 = errors.New("Item not stored")
	ErrIncrOnNonNumericValue         = errors.New("Incr on non-numeric value")
	ErrVBucketBelongsToAnotherServer = errors.New("VBucket belongs to another server")
	ErrAuthenticationError           = errors.New("Authentication error")
	ErrUnknownCommand                = errors.New("Unknown command")
	ErrOutOfMemory                   = errors.New("Out of memory")
	ErrNotSupported                  = errors.New("Not supported")
	ErrInternalError                 = errors.New("Internal error")
	ErrBusy                          = errors.New("Busy")
	ErrTemporaryFailure              = errors.New("Temporary failure")
)

var statusCodeToErr = map[uint16]error{
	internal.StatusNoError:                       nil,
	internal.StatusKeyNotFound:                   ErrKeyNotFound,
	internal.StatusKeyExists:                     ErrKeyExists,
	internal.StatusValueTooLarge:                 ErrValueTooLarge,
	internal.StatusInvalidArguments:              ErrInvalidArguments,
	internal.StatusItemNotStored:                 ErrItemNotStored,
	internal.StatusIncrOnNonNumericValue:         ErrIncrOnNonNumericValue,
	internal.StatusVBucketBelongsToAnotherServer: ErrVBucketBelongsToAnotherServer,
	internal.StatusAuthenticationError:           ErrAuthenticationError,
	internal.StatusUnknownCommand:                ErrUnknownCommand,
	internal.StatusOutOfMemory:                   ErrOutOfMemory,
	internal.StatusNotSupported:                  ErrNotSupported,
	internal.StatusInternalError:                 ErrInternalError,
	internal.StatusBusy:                          ErrBusy,
	internal.StatusTemporaryFailure:              ErrTemporaryFailure,
}

// Client implements a memcache client that uses the binary protocol
type Client struct {
	connector    Connector
	nodePicker   NodePicker
	queueSize    int
	nodes        sync.Map
	bufPool      *internal.BufPool
	mu           sync.Mutex
	shutdownChan chan struct{}
}

// ClientOpt defines an option that can be set on a Client object
type ClientOpt func(c *Client)

// WithNodePicker sets the NodePicker to use in determining the mapping between a key and a node
func WithNodePicker(np NodePicker) ClientOpt {
	return func(c *Client) {
		c.nodePicker = np
	}
}

// WithConnector sets the Connector that can be used to obtain a connection to a node given its ID
func WithConnector(connector Connector) ClientOpt {
	return func(c *Client) {
		c.connector = connector
	}
}

// WithQueueSize sets the request queue size for each memcache node
func WithQueueSize(queueSize int) ClientOpt {
	return func(c *Client) {
		c.queueSize = queueSize
	}
}

// NewSimpleClient creates a client that uses the SimpleNodePicker to interact with the given set of memcache nodes
func NewSimpleClient(nodeAddr ...string) (*Client, error) {
	return NewClient(WithNodePicker(NewSimpleNodePicker(nodeAddr...)))
}

// NewClient creates a client with the provided options
func NewClient(clientOpts ...ClientOpt) (*Client, error) {
	c := &Client{
		connector:    &SimpleTCPConnector{},
		queueSize:    defaultQueueSize,
		bufPool:      internal.NewBufPool(),
		shutdownChan: make(chan struct{}),
	}

	for _, co := range clientOpts {
		co(c)
	}

	if c.connector == nil || c.nodePicker == nil {
		return nil, ErrInvalidClientConfig
	}

	return c, nil
}

// Set performs a SET operation
func (c *Client) Set(ctx context.Context, key, value []byte) error {
	return c.SetWithExpiry(ctx, key, value, 0*time.Second)
}

// SetWithExpiry performs a SET with the specified expiry
func (c *Client) SetWithExpiry(ctx context.Context, key []byte, value []byte, expiry time.Duration) error {
	extras := c.buildExtras(0, expiry)
	resp, err := c.makeRequest(ctx, internal.OpSet, key, value, extras.Bytes())
	c.bufPool.PutBuf(extras)

	if err != nil {
		return err
	}
	return statusCodeToErr[resp.StatusCode]
}

// Get performas a GET for the given key
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	resp, err := c.makeRequest(ctx, internal.OpGet, key, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// Close closes all connections to the remote nodes and cancels any pending requests
// The client is not usable after this operation
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.shutdownChan:
		return ErrClientShutdown
	default:
		close(c.shutdownChan)
		c.nodes.Range(func(k, v interface{}) bool {
			node := v.(*internal.Node)
			node.Shutdown()
			return true
		})
		return nil
	}
}

func (c *Client) buildExtras(flags uint32, expiry time.Duration) *internal.Buf {
	extras := c.bufPool.GetBuf()
	extras.WriteUint32(flags)
	extras.WriteUint32(uint32(expiry / time.Second))
	return extras
}

func (c *Client) makeRequest(ctx context.Context, opCode uint8, key, value, extras []byte) (*internal.Response, error) {
	// ensure that the client is not shutdown
	select {
	case <-c.shutdownChan:
		return nil, ErrClientShutdown
	default:
	}

	// esnure that the context is not cancelled
	if err := ctx.Err(); err != nil {
		return nil, nil
	}

	// determine the node
	node, err := c.getNode(ctx, key)
	if err != nil {
		return nil, err
	}

	// send the request
	respChan := make(chan *internal.Response, 1)
	if err := node.Send(ctx, respChan, &internal.Request{OpCode: opCode, Key: key, Value: value, Extras: extras}); err != nil {
		close(respChan)
		return nil, err
	}

	// wait for response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		if !ok {
			return nil, ErrNoResult
		}

		if resp.Err != nil {
			return nil, resp.Err
		}

		return resp, nil
	}
}

func (c *Client) getNode(ctx context.Context, key []byte) (*internal.Node, error) {
	nodeID := c.nodePicker.Pick(key)
	node, ok := c.nodes.Load(nodeID)
	if ok {
		return node.(*internal.Node), nil
	}

	conn, err := c.connector.Connect(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	newNode := internal.NewNode(internal.NewConnWrapper(conn, c.bufPool), c.queueSize)
	if err != nil {
		conn.Close()
		return nil, err
	}

	node, alreadyExists := c.nodes.LoadOrStore(nodeID, newNode)
	if alreadyExists {
		newNode.Shutdown()
	}

	return node.(*internal.Node), nil
}
