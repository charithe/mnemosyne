package memcache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"golang.org/x/sync/errgroup"
)

const (
	defaultQueueSize = 16
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

// Set inserts or overwrites the value pointed to by the key
func (c *Client) Set(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) Result {
	return c.doMutation(ctx, internal.OpSet, key, value, mutationOpts...)
}

// Add inserts the value iff the key doesn't already exist
func (c *Client) Add(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) Result {
	return c.doMutation(ctx, internal.OpAdd, key, value, mutationOpts...)
}

// Replace overwrites the value iff the key already exists
func (c *Client) Replace(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) Result {
	return c.doMutation(ctx, internal.OpReplace, key, value, mutationOpts...)
}

// helper for mutations (SET, ADD, REPLACE)
func (c *Client) doMutation(ctx context.Context, opCode uint8, key, value []byte, mutationOpts ...MutationOpt) Result {
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
func (c *Client) Get(ctx context.Context, key []byte) Result {
	req := &internal.Request{
		OpCode: internal.OpGetK,
		Key:    key,
	}

	return c.makeRequest(ctx, req)
}

// MultiGet performs a lookup for a set of keys and returns the found values
func (c *Client) MultiGet(ctx context.Context, keys ...[]byte) []Result {
	// we can short-circuit a lot of code if there's only one key to lookup
	if len(keys) == 1 {
		res := c.Get(ctx, keys[0])
		return []Result{res}
	}

	groups := c.groupRequests(internal.OpGetKQ, internal.OpGetK, keys)
	return c.distributeRequests(ctx, groups)
}

// Delete performs a DELETE for the given key
func (c *Client) Delete(ctx context.Context, key []byte) Result {
	req := &internal.Request{
		OpCode: internal.OpDelete,
		Key:    key,
	}

	return c.makeRequest(ctx, req)
}

// MultiDelete performs a batch delete of the keys
func (c *Client) MultiDelete(ctx context.Context, keys ...[]byte) []Result {
	// we can short-circuit a lot of code if there's only one key to delete
	if len(keys) == 1 {
		res := c.Delete(ctx, keys[0])
		return []Result{res}
	}

	groups := c.groupRequests(internal.OpDeleteQ, internal.OpDelete, keys)
	return c.distributeRequests(ctx, groups)
}

// Increment performs an INCR operation
// If the key already exists, the value will be incremented by delta. If the key does not exist, it will be set to initial.
func (c *Client) Increment(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) NumericResult {
	return c.doNumericOp(ctx, internal.OpIncrement, key, initial, delta, mutationOpts...)
}

// Decrement performs a DECR operation
// If the key already exists, the value will be decremented by delta. If the key does not exist, it will be set to initial.
func (c *Client) Decrement(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) NumericResult {
	return c.doNumericOp(ctx, internal.OpDecrement, key, initial, delta, mutationOpts...)
}

func (c *Client) doNumericOp(ctx context.Context, opCode uint8, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) NumericResult {
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

	return c.makeRequest(ctx, req).(NumericResult)
}

// Append appends the value to the existing value
func (c *Client) Append(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) Result {
	return c.doUpdateOp(ctx, internal.OpAppend, key, value, mutationOpts...)
}

// Prepend  prepends the value to the existing value
func (c *Client) Prepend(ctx context.Context, key, value []byte, mutationOpts ...MutationOpt) Result {
	return c.doUpdateOp(ctx, internal.OpPrepend, key, value, mutationOpts...)
}

func (c *Client) doUpdateOp(ctx context.Context, opCode uint8, key, value []byte, mutationOpts ...MutationOpt) Result {
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
func (c *Client) Touch(ctx context.Context, expiry time.Duration, key []byte) Result {
	return c.doTouchOp(ctx, internal.OpTouch, expiry, key)
}

// GetAndTouch gets the key and sets a new expiry time
func (c *Client) GetAndTouch(ctx context.Context, expiry time.Duration, key []byte) Result {
	return c.doTouchOp(ctx, internal.OpGAT, expiry, key)
}

func (c *Client) doTouchOp(ctx context.Context, opCode uint8, expiry time.Duration, key []byte) Result {
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
			r := c.sendToNode(newCtx, nid, req)
			return r[0].Err()
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
			node := v.(*internal.Node)
			node.Shutdown()
			return true
		})
		return nil
	}
}

func (c *Client) makeRequest(ctx context.Context, request *internal.Request) Result {
	// ensure that the client is not shutdown
	select {
	case <-c.shutdownChan:
		return newErrResult(ErrClientShutdown)
	default:
	}

	// ensure that the context hasn't expired
	if err := ctx.Err(); err != nil {
		return newErrResult(err)
	}

	nodeID := c.nodePicker.Pick(request.Key)
	results := c.sendToNode(ctx, nodeID, request)
	return results[0]
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
func (c *Client) distributeRequests(ctx context.Context, requests map[string][]*internal.Request) []Result {
	g, newCtx := errgroup.WithContext(ctx)
	resultChan := make(chan []Result, len(requests))
	for nodeID, batch := range requests {
		n := nodeID
		b := batch
		g.Go(func() error {
			resultChan <- c.sendToNode(newCtx, n, b...)
			return nil
		})
	}

	err := g.Wait()
	close(resultChan)

	if err != nil {
		return []Result{newErrResult(err)}
	}

	var results []Result
	for res := range resultChan {
		results = append(results, res...)
	}
	return results
}

// send a batch of requests to a single node
func (c *Client) sendToNode(ctx context.Context, nodeID string, requests ...*internal.Request) []Result {
	if err := ctx.Err(); err != nil {
		return []Result{newErrResult(err)}
	}

	node, err := c.getNode(ctx, nodeID)
	if err != nil {
		return []Result{newErrResult(err)}
	}

	respChan := make(chan *internal.Response, len(requests))
	if err := node.Send(ctx, respChan, requests...); err != nil {
		close(respChan)
		return []Result{newErrResult(err)}
	}

	var results []Result
	for resp := range respChan {
		results = append(results, newResult(resp))
	}
	return results
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
