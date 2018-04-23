package memcache

import (
	"context"
	"errors"
	"fmt"
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

// Result represents a result from a single operation
type Result interface {
	fmt.Stringer
	Err() error
	Key() []byte
	Value() []byte
	CAS() uint64
}

// result implements the Result interface
type result struct {
	resp *internal.Response
}

func (r *result) Key() []byte {
	return r.resp.Key
}

func (r *result) Value() []byte {
	return r.resp.Value
}

func (r *result) CAS() uint64 {
	return r.resp.CAS
}

func (r *result) Err() error {
	if r.resp.Err != nil {
		return r.resp.Err
	}

	return statusCodeToErr[r.resp.StatusCode]
}

func (r *result) String() string {
	return fmt.Sprintf("%+v", r.resp)
}

// Helper functions for creating Result objects
func newResult(resp *internal.Response) Result {
	return &result{resp: resp}
}

func newErrResult(err error) Result {
	return &result{resp: &internal.Response{Err: err}}
}

// MutationOpt is an optional modification to apply to the request
type MutationOpt func(req *internal.Request)

// WithExpiry sets an expiry time for the key being mutated
func WithExpiry(expiry time.Duration) MutationOpt {
	return func(req *internal.Request) {
		extras := internal.NewBuf()
		switch req.OpCode {
		case internal.OpIncrement:
			fallthrough
		case internal.OpDecrement:
			fallthrough
		case internal.OpIncrementQ:
			fallthrough
		case internal.OpDecrementQ:
			fallthrough
		case internal.OpFlush:
			fallthrough
		case internal.OpFlushQ:
			extras.WriteUint32(uint32(expiry.Seconds()))
		default:
			extras.WriteUint32(0)
			extras.WriteUint32(uint32(expiry.Seconds()))
		}
		req.Extras = extras.Bytes()
	}
}

func WithCASValue(cas uint64) MutationOpt {
	return func(req *internal.Request) {
		req.CAS = cas
	}
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
func (c *Client) Increment(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) Result {
	return c.doNumericOp(ctx, internal.OpIncrement, key, initial, delta, mutationOpts...)
}

// Decrement performs a DECR operation
// If the key already exists, the value will be decremented by delta. If the key does not exist, it will be set to initial.
func (c *Client) Decrement(ctx context.Context, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) Result {
	return c.doNumericOp(ctx, internal.OpDecrement, key, initial, delta, mutationOpts...)
}

func (c *Client) doNumericOp(ctx context.Context, opCode uint8, key []byte, initial, delta uint64, mutationOpts ...MutationOpt) Result {
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
		// normally, extras would contain flags (4 bytes) followed by the expiry (4 bytes)
		extras.Write(req.Extras)
	} else {
		extras.WriteUint32(0)
	}
	req.Extras = extras.Bytes()

	return c.makeRequest(ctx, req)
}

// Flush flushes all the keys
func (c *Client) Flush(ctx context.Context, nodeIDs []string, mutationOpts ...MutationOpt) Result {
	req := &internal.Request{
		OpCode: internal.OpFlush,
	}

	for _, mo := range mutationOpts {
		mo(req)
	}

	if len(nodeIDs) == 1 {
		r := c.sendToNode(ctx, nodeIDs[0], req)
		return r[0]
	}

	g, newCtx := errgroup.WithContext(ctx)
	for _, n := range nodeIDs {
		nid := n
		g.Go(func() error {
			r := c.sendToNode(newCtx, nid, req)
			return r[0].Err()
		})
	}

	err := g.Wait()
	return newErrResult(err)
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
func (c *Client) groupRequests(quietOpCode, normalOpCode uint8, keys [][]byte) map[string][]*internal.Request {
	groups := make(map[string][]*internal.Request)
	for _, k := range keys {
		nodeID := c.nodePicker.Pick(k)

		// If there is more than one request to the node, we can make the previous command  a quiet one to save on network roundtrips
		grp := groups[nodeID]
		grpSize := len(grp)
		if grpSize > 0 {
			grp[grpSize-1].OpCode = quietOpCode
		}

		groups[nodeID] = append(grp, &internal.Request{OpCode: normalOpCode, Key: k})
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
