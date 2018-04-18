package memcache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"golang.org/x/sync/errgroup"
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

type KV struct {
	Key   []byte
	Value []byte
	Err   error
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
	req := &internal.Request{
		OpCode: internal.OpSet,
		Key:    key,
		Value:  value,
		Extras: extras.Bytes(),
	}

	resp, err := c.makeRequests(ctx, req)
	c.bufPool.PutBuf(extras)

	if err != nil {
		return err
	}
	return statusCodeToErr[resp[0].StatusCode]
}

// Get performs a GET for the given key
func (c *Client) Get(ctx context.Context, key []byte) ([]byte, error) {
	req := &internal.Request{
		OpCode: internal.OpGet,
		Key:    key,
	}

	resp, err := c.makeRequests(ctx, req)
	if err != nil || resp == nil {
		return nil, err
	}
	return resp[0].Value, nil
}

// MultiGet performs a lookup for a set of keys and returns the found values
func (c *Client) MultiGet(ctx context.Context, keys ...[]byte) ([]*KV, error) {
	// we can short-circuit a lot of code if there's only one key to lookup
	if len(keys) == 1 {
		val, err := c.Get(ctx, keys[0])
		if err != nil {
			return nil, err
		}
		return []*KV{&KV{Key: keys[0], Value: val}}, nil
	}

	groups := make(map[string][]*internal.Request)
	for _, k := range keys {
		nodeID := c.nodePicker.Pick(k)

		// If there is more than one request to the node, we can make the previous GET a quiet one to save on network roundtrips
		grp := groups[nodeID]
		grpSize := len(grp)
		if grpSize > 0 {
			grp[grpSize-1].OpCode = internal.OpGetKQ
		}

		groups[nodeID] = append(grp, &internal.Request{OpCode: internal.OpGetK, Key: k})
	}

	responses, err := c.distributeRequests(ctx, groups)
	if err != nil {
		return nil, err
	}

	results := make([]*KV, len(responses))
	for i, r := range responses {
		if r.Err == nil {
			results[i] = &KV{Key: r.Key, Value: r.Value}
		} else {
			results[i] = &KV{Key: r.Key, Err: r.Err}
		}
	}

	return results, nil
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

// build the extras for the request
func (c *Client) buildExtras(flags uint32, expiry time.Duration) *internal.Buf {
	extras := c.bufPool.GetBuf()
	extras.WriteUint32(flags)
	extras.WriteUint32(uint32(expiry / time.Second))
	return extras
}

// decide on how the requests should be made
func (c *Client) makeRequests(ctx context.Context, requests ...*internal.Request) ([]*internal.Response, error) {
	// ensure that the client is not shutdown
	select {
	case <-c.shutdownChan:
		return nil, ErrClientShutdown
	default:
	}

	// ensure that the context hasn't expired
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// don't bother with goroutines if there's only a single request
	if len(requests) == 1 {
		nodeID := c.nodePicker.Pick(requests[0].Key)
		return c.sendToNode(ctx, nodeID, requests[0])
	}

	groups := c.groupRequests(requests...)
	return c.distributeRequests(ctx, groups)
}

// group requests by destination node
func (c *Client) groupRequests(requests ...*internal.Request) map[string][]*internal.Request {
	batches := make(map[string][]*internal.Request)
	for _, req := range requests {
		nodeID := c.nodePicker.Pick(req.Key)
		batches[nodeID] = append(batches[nodeID], req)
	}
	return batches
}

// distribute the requests to nodes in parallel
func (c *Client) distributeRequests(ctx context.Context, requests map[string][]*internal.Request) ([]*internal.Response, error) {
	g, newCtx := errgroup.WithContext(ctx)
	respChan := make(chan []*internal.Response, len(requests))
	for nodeID, batch := range requests {
		n := nodeID
		b := batch
		g.Go(func() error {
			r, err := c.sendToNode(newCtx, n, b...)
			if err != nil {
				return err
			}
			respChan <- r
			return nil
		})
	}

	err := g.Wait()
	close(respChan)
	if err != nil {
		return nil, err
	}

	// join up all the responses
	var finalResponses []*internal.Response
	for resp := range respChan {
		finalResponses = append(finalResponses, resp...)
	}
	return finalResponses, nil
}

// send a batch of requests to a single node
func (c *Client) sendToNode(ctx context.Context, nodeID string, requests ...*internal.Request) ([]*internal.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	node, err := c.getNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	respChan := make(chan *internal.Response, len(requests))
	if err := node.Send(ctx, respChan, requests...); err != nil {
		close(respChan)
		return nil, err
	}

	var responses []*internal.Response
	for resp := range respChan {
		responses = append(responses, resp)
	}
	return responses, nil
}

func (c *Client) getNode(ctx context.Context, nodeID string) (*internal.Node, error) {
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
