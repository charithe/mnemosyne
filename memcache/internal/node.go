package internal

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/fatih/pool"
)

var (
	ErrNodeShutdown = errors.New("Node shutdown")
)

// RequestBatch represents a set of requests that belong to the same group (eg. MultiGet)
type requestBatch struct {
	ctx          context.Context
	requests     []*Request
	responseChan chan<- *Response
}

// Node represents a memcached node with an associated request queue
type Node struct {
	connPool     pool.Pool
	bufPool      *BufPool
	requests     chan *requestBatch
	mu           sync.Mutex
	shutdownChan chan struct{}
}

func NewNode(bufPool *BufPool, connectFunc func() (net.Conn, error), queueSize int, initialConns int, maxConns int) (*Node, error) {
	connPool, err := pool.NewChannelPool(initialConns, maxConns, connectFunc)
	if err != nil {
		return nil, err
	}

	node := &Node{
		connPool:     connPool,
		bufPool:      bufPool,
		requests:     make(chan *requestBatch, queueSize),
		shutdownChan: make(chan struct{}),
	}

	go node.requestLoop()
	return node, nil
}

func (n *Node) Send(ctx context.Context, responseChan chan<- *Response, requests ...*Request) error {
	select {
	case <-n.shutdownChan:
		return ErrNodeShutdown
	case <-ctx.Done():
		return ctx.Err()
	case n.requests <- &requestBatch{ctx: ctx, requests: requests, responseChan: responseChan}:
		return nil
	}
}

func (n *Node) requestLoop() {
	for {
		select {
		case <-n.shutdownChan:
			return
		case reqBatch, ok := <-n.requests:
			if !ok {
				return
			}

			// send the new request(s)
			if len(reqBatch.requests) == 1 {
				n.processSingleRequest(reqBatch.ctx, reqBatch.responseChan, reqBatch.requests[0])
			} else {
				n.processBatchRequest(reqBatch.ctx, reqBatch.responseChan, reqBatch.requests)
			}
		}
	}
}

func (n *Node) processSingleRequest(ctx context.Context, responseChan chan<- *Response, req *Request) {
	defer close(responseChan)

	conn, err := n.connPool.Get()
	if err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		return
	}

	cw := NewConnWrapper(conn, n.bufPool)
	cw.FlushReadBuffer()

	if err := cw.WritePacket(ctx, req); err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		cw.Close()
		return
	}

	if err := cw.FlushWriteBuffer(); err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		cw.Close()
		return
	}

	resp, err := cw.ReadPacket(ctx)
	if err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		cw.Close()
		return
	}

	responseChan <- resp
	cw.Close()
}

func (n *Node) processBatchRequest(ctx context.Context, responseChan chan<- *Response, requests []*Request) {
	conn, err := n.connPool.Get()
	if err != nil {
		responseChan <- &Response{Err: err}
		return
	}

	cw := NewConnWrapper(conn, n.bufPool)
	cw.FlushReadBuffer()

	pendingReq := make(chan *Request, len(requests))
	sentinel := uint32(len(requests) - 1)

	// spawn a goroutine to gather the responses as they arrive
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for pr := range pendingReq {
			// make sure the context hasn't expired
			if err := ctx.Err(); err != nil {
				responseChan <- &Response{Key: pr.Key, Err: err}
				return
			}

			resp, err := cw.ReadPacket(ctx)
			if err != nil {
				responseChan <- &Response{Key: pr.Key, Err: err}
				continue
			}

			responseChan <- resp

			// if the response is the last in the batch, break out of the loop
			if resp.Opaque == sentinel {
				return
			}
		}
	}()

	// pipeline the requests
	for i, req := range requests {
		// make sure the context hasn't expired
		if err := ctx.Err(); err != nil {
			responseChan <- &Response{Key: req.Key, Err: err}
			break
		}

		req.Opaque = uint32(i)
		if err := cw.WritePacket(ctx, req); err != nil {
			responseChan <- &Response{Key: req.Key, Err: err}
			continue
		}

		pendingReq <- req
	}

	close(pendingReq)
	cw.FlushWriteBuffer()
	wg.Wait()
	close(responseChan)
	cw.Close()
}

func (n *Node) Shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()
	select {
	case <-n.shutdownChan:
	default:
		close(n.shutdownChan)
		n.connPool.Close()
	}
}
