package internal

import (
	"context"
	"errors"
	"sync"
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
	connectFunc  func() (*ConnWrapper, error)
	numConns     int
	requests     chan *requestBatch
	mu           sync.Mutex
	shutdownChan chan struct{}
}

func NewNode(connectFunc func() (*ConnWrapper, error), queueSize int, numConns int) *Node {
	node := &Node{
		connectFunc:  connectFunc,
		numConns:     numConns,
		requests:     make(chan *requestBatch, queueSize),
		shutdownChan: make(chan struct{}),
	}

	return node
}

func (n *Node) Start() error {
	for i := 0; i < n.numConns; i++ {
		conn, err := n.connectFunc()
		if err != nil {
			n.Shutdown()
			return err
		}
		go n.requestLoop(conn)
	}

	return nil
}

func (n *Node) Send(ctx context.Context, responseChan chan<- *Response, requests ...*Request) error {
	select {
	case <-n.shutdownChan:
		return ErrNodeShutdown
	case n.requests <- &requestBatch{ctx: ctx, requests: requests, responseChan: responseChan}:
		return nil
	}
}

func (n *Node) requestLoop(conn *ConnWrapper) {
	defer conn.Close()
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
				n.processSingleRequest(reqBatch.ctx, conn, reqBatch.responseChan, reqBatch.requests[0])
			} else {
				n.processBatchRequest(reqBatch.ctx, conn, reqBatch.responseChan, reqBatch.requests)
			}
		}
	}
}

func (n *Node) processSingleRequest(ctx context.Context, cw *ConnWrapper, responseChan chan<- *Response, req *Request) {
	defer close(responseChan)
	if err := ctx.Err(); err != nil {
		responseChan <- &Response{Err: err}
		return
	}

	cw.FlushReadBuffer()

	if err := cw.WritePacket(ctx, req); err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		return
	}

	if err := cw.FlushWriteBuffer(); err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		return
	}

	resp, err := cw.ReadPacket(ctx)
	if err != nil {
		responseChan <- &Response{Key: req.Key, Err: err}
		return
	}

	responseChan <- resp
}

func (n *Node) processBatchRequest(ctx context.Context, cw *ConnWrapper, responseChan chan<- *Response, requests []*Request) {
	if err := ctx.Err(); err != nil {
		responseChan <- &Response{Err: err}
		close(responseChan)
		return
	}

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
}

func (n *Node) Shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()
	select {
	case <-n.shutdownChan:
	default:
		close(n.shutdownChan)
	}
}
