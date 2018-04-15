package memcache

import (
	"context"
	"errors"
	"fmt"
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
)

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

// SetWithExpiry SETs a keyvalue pair with expiry
func (c *Client) SetWithExpiry(ctx context.Context, key []byte, value []byte, expiry time.Duration) error {
	fmt.Println("HERE")
	extras := c.bufPool.GetBuf()
	extras.WriteUint32(0)
	extras.WriteUint32(uint32(expiry / time.Second))
	_, err := c.makeRequest(ctx, internal.OpSet, key, value, extras.Bytes())
	c.bufPool.PutBuf(extras)
	return err
}

// Get GETs a single keyvalue pair
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	resp, err := c.makeRequest(ctx, internal.OpGet, key, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

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
	fmt.Printf("RESP CHAN: %+v\n", respChan)
	if err := node.Send(ctx, respChan, &internal.Request{OpCode: opCode, Key: key, Value: value, Extras: extras}); err != nil {
		close(respChan)
		return nil, err
	}

	// wait for response
	select {
	case <-ctx.Done():
		fmt.Println("Context expiry")
		return nil, ctx.Err()
	case resp, ok := <-respChan:
		fmt.Println("GOT RESPONSE")
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
