package memcache

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"golang.org/x/sync/errgroup"
)

const (
	defaultQueueSize    = 16
	defaultConnsPerNode = 1
)

// Result represents a result from a single operation
// Users must always check the return value of Err() before accessing other functions
type Result interface {
	fmt.Stringer
	Err() error
	Key() []byte
	Value() []byte
	CAS() uint64
}

// NumericResult is returns the numeric results from Increment and Decrement operations
type NumericResult interface {
	Result
	NumericValue() uint64
}

// Client implements a memcache client that uses the binary protocol
type Client struct {
	connector      Connector
	nodePicker     NodePicker
	queueSize      int
	connectTimeout time.Duration
	connsPerNode   int
	nodes          sync.Map
	bufPool        *internal.BufPool
	mu             sync.Mutex
	shutdownChan   chan struct{}
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

// WithConnectionsPerNode sets the number of parallel connections to open per node
func WithConnectionsPerNode(n int) ClientOpt {
	return func(c *Client) {
		c.connsPerNode = n
	}
}

// NewSimpleClient creates a client that uses the SimpleNodePicker to interact with the given set of memcache nodes
func NewSimpleClient(nodeAddr ...string) (*Client, error) {
	return NewClient(WithNodePicker(NewSimpleNodePicker(nodeAddr...)))
}

// NewClient creates a client with the provided options
func NewClient(clientOpts ...ClientOpt) (*Client, error) {
	c := &Client{
		connector:    NewSimpleTCPConnector(),
		queueSize:    defaultQueueSize,
		connsPerNode: defaultConnsPerNode,
		bufPool:      internal.NewBufPool(),
		shutdownChan: make(chan struct{}),
	}

	for _, co := range clientOpts {
		co(c)
	}

	if c.connsPerNode <= 0 {
		c.connsPerNode = defaultConnsPerNode
	}

	if c.connector == nil || c.nodePicker == nil {
		return nil, ErrInvalidClientConfig
	}

	return c, nil
}

// Set inserts or overwrites the value pointed to by the key
func (c *Client) Set(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	return c.doMutation(ctx, internal.OpSet, key, value, mutationOpts...)
}

// Add inserts the value iff the key doesn't already exist
func (c *Client) Add(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	return c.doMutation(ctx, internal.OpAdd, key, value, mutationOpts...)
}

// Replace overwrites the value iff the key already exists
func (c *Client) Replace(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	return c.doMutation(ctx, internal.OpReplace, key, value, mutationOpts...)
}

// helper for mutations (SET, ADD, REPLACE)
func (c *Client) doMutation(ctx context.Context, opCode uint8, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	req := &internal.Request{OpCode: opCode, Key: key, Value: value}
	for _, mo := range mutationOpts {
		mo(req)
	}

	// extras are required
	if req.Extras == nil {
		WithExpiry(0 * time.Hour)(req)
	}

	return c.makeRequest(ctx, req)
}

// Get performs a GET for the given key
func (c *Client) Get(ctx context.Context, key []byte) (Result, error) {
	req := &internal.Request{
		OpCode: internal.OpGetK,
		Key:    key,
	}

	return c.makeRequest(ctx, req)
}

// MultiGet performs a lookup for a set of keys and returns the found values
// if error is an instance of *memcache.Error, partial results may be available in the result array
func (c *Client) MultiGet(ctx context.Context, keys ...[]byte) ([]Result, error) {
	// we can short-circuit a lot of code if there's only one key to lookup
	if len(keys) == 1 {
		res, err := c.Get(ctx, keys[0])
		if res != nil {
			return []Result{res}, err
		}
		return nil, err
	}

	groups := c.groupRequests(internal.OpGetKQ, internal.OpGetK, keys)
	return c.distributeRequests(ctx, groups)
}

// Delete performs a DELETE for the given key
func (c *Client) Delete(ctx context.Context, key []byte) error {
	req := &internal.Request{
		OpCode: internal.OpDelete,
		Key:    key,
	}

	_, err := c.makeRequest(ctx, req)
	return err
}

// MultiDelete performs a batch delete of the keys
func (c *Client) MultiDelete(ctx context.Context, keys ...[]byte) error {
	// we can short-circuit a lot of code if there's only one key to delete
	if len(keys) == 1 {
		return c.Delete(ctx, keys[0])
	}

	groups := c.groupRequests(internal.OpDeleteQ, internal.OpDelete, keys)
	_, err := c.distributeRequests(ctx, groups)
	return err
}

// Increment performs an INCR operation
// If the key already exists, the value will be incremented by delta. If the key does not exist, it will be set to initial.
func (c *Client) Increment(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) (NumericResult, error) {
	return c.doNumericOp(ctx, internal.OpIncrement, key, initial, delta, mutationOpts...)
}

// Decrement performs a DECR operation
// If the key already exists, the value will be decremented by delta. If the key does not exist, it will be set to initial.
func (c *Client) Decrement(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) (NumericResult, error) {
	return c.doNumericOp(ctx, internal.OpDecrement, key, initial, delta, mutationOpts...)
}

func (c *Client) doNumericOp(ctx context.Context, opCode uint8, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) (NumericResult, error) {
	req := &internal.Request{
		OpCode: opCode,
		Key:    key,
	}

	for _, mo := range mutationOpts {
		mo(req)
	}

	extras := internal.NewBuf()
	extras.WriteUint64(delta)
	extras.WriteUint64(initial)
	if req.Extras != nil {
		extras.Write(req.Extras)
	} else {
		extras.WriteUint32(0)
	}
	req.Extras = extras.Bytes()

	r, err := c.makeRequest(ctx, req)
	if r != nil {
		return r.(NumericResult), err
	}
	return nil, err
}

// Append appends the value to the existing value
func (c *Client) Append(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	return c.doModifyOp(ctx, internal.OpAppend, key, value, mutationOpts...)
}

// Prepend  prepends the value to the existing value
func (c *Client) Prepend(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	return c.doModifyOp(ctx, internal.OpPrepend, key, value, mutationOpts...)
}

func (c *Client) doModifyOp(ctx context.Context, opCode uint8, key, value []byte, mutationOpts ...MutationOpt) (Result, error) {
	req := &internal.Request{
		OpCode: opCode,
		Key:    key,
		Value:  value,
	}

	for _, mo := range mutationOpts {
		mo(req)
	}
	return c.makeRequest(ctx, req)
}

// Touch sets a new expiry time for the key
func (c *Client) Touch(ctx context.Context, expiry time.Duration, key []byte) error {
	_, err := c.doTouchOp(ctx, internal.OpTouch, expiry, key)
	return err
}

// GetAndTouch gets the key and sets a new expiry time
func (c *Client) GetAndTouch(ctx context.Context, expiry time.Duration, key []byte) (Result, error) {
	return c.doTouchOp(ctx, internal.OpGAT, expiry, key)
}

func (c *Client) doTouchOp(ctx context.Context, opCode uint8, expiry time.Duration, key []byte) (Result, error) {
	req := &internal.Request{
		OpCode: opCode,
		Key:    key,
	}

	WithExpiry(expiry)(req)
	return c.makeRequest(ctx, req)
}

// Flush flushes all the keys
func (c *Client) Flush(ctx context.Context, mutationOpts ...MutationOpt) error {
	req := &internal.Request{
		OpCode: internal.OpFlush,
	}

	for _, mo := range mutationOpts {
		mo(req)
	}

	g, newCtx := errgroup.WithContext(ctx)
	c.nodes.Range(func(k, v interface{}) bool {
		nid := k.(string)
		g.Go(func() error {
			_, err := c.sendToNode(newCtx, nid, req)
			return err
		})
		return true
	})

	return g.Wait()
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
			nodeID := k.(string)
			c.sendToNode(context.Background(), nodeID, &internal.Request{OpCode: internal.OpQuit})
			node := v.(*internal.Node)
			node.Shutdown()
			return true
		})
		return nil
	}
}

func (c *Client) makeRequest(ctx context.Context, request *internal.Request) (Result, error) {
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

	nodeID := c.nodePicker.Pick(request.Key)
	results, err := c.sendToNode(ctx, nodeID, request)
	if results != nil {
		return results[0], err
	}
	return nil, err
}

// groups the keys by node and constructs the batch of requests to send
func (c *Client) groupRequests(quietOpCode, normalOpCode uint8, keys [][]byte, mutationOpts ...MutationOpt) map[string][]*internal.Request {
	groups := make(map[string][]*internal.Request)
	for _, k := range keys {
		nodeID := c.nodePicker.Pick(k)

		// If there is more than one request to the node, we can make the previous command  a quiet one to save on network roundtrips
		grp := groups[nodeID]
		grpSize := len(grp)
		if grpSize > 0 {
			grp[grpSize-1].OpCode = quietOpCode
		}

		req := &internal.Request{OpCode: normalOpCode, Key: k}
		for _, mo := range mutationOpts {
			mo(req)
		}
		groups[nodeID] = append(grp, req)
	}

	return groups
}

// distribute the requests to nodes in parallel
func (c *Client) distributeRequests(ctx context.Context, requests map[string][]*internal.Request) ([]Result, error) {
	g, newCtx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	var results []Result
	var errors []error

	for nodeID, batch := range requests {
		n := nodeID
		b := batch
		g.Go(func() error {
			r, err := c.sendToNode(newCtx, n, b...)
			mu.Lock()
			if r != nil {
				results = append(results, r...)
			}

			if err != nil {
				errors = append(errors, err)
			}
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return results, err
	}

	if errors != nil {
		return results, newErrorFromErrList(errors...)
	}

	return results, nil
}

// send a batch of requests to a single node
func (c *Client) sendToNode(ctx context.Context, nodeID string, requests ...*internal.Request) ([]Result, error) {
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

	var results []Result
	var errors []error
	for resp := range respChan {
		r := newResult(resp)
		if r.Err() != nil {
			errors = append(errors, &KeyError{Key: r.Key(), Err: r.Err()})
		}
		results = append(results, r)
	}

	if errors != nil {
		return results, newErrorFromErrList(errors...)
	}
	return results, nil
}

func (c *Client) getNode(ctx context.Context, nodeID string) (*internal.Node, error) {
	if node, ok := c.nodes.Load(nodeID); ok {
		n := node.(*internal.Node)
		if !n.IsShutdown() {
			return n, nil
		}

		c.nodes.Delete(nodeID)
	}

	connectFunc := func() (net.Conn, error) {
		return c.connector.Connect(nodeID)
	}

	newNode := internal.NewNode(connectFunc, c.queueSize, c.connsPerNode, c.bufPool)
	node, alreadyExists := c.nodes.LoadOrStore(nodeID, newNode)
	if alreadyExists {
		newNode.Shutdown()
	} else {
		if err := newNode.Start(); err != nil {
			return nil, err
		}
	}

	return node.(*internal.Node), nil
}
