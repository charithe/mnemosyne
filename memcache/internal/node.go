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
	conn         *ConnWrapper
	requests     chan *requestBatch
	mu           sync.Mutex
	shutdownChan chan struct{}
}

func NewNode(conn *ConnWrapper, queueSize int) *Node {
	node := &Node{
		conn:         conn,
		requests:     make(chan *requestBatch, queueSize),
		shutdownChan: make(chan struct{}),
	}

	go node.requestLoop()
	return node
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
				// request channel has been closed.
				return
			}

			n.processRequestBatch(reqBatch)
		}
	}
}

func (n *Node) processRequestBatch(reqBatch *requestBatch) {
	for _, req := range reqBatch.requests {
		if err := n.conn.WritePacket(reqBatch.ctx, req); err != nil {
			reqBatch.responseChan <- &Response{Err: err}
			continue
		}

		// TODO This is not correct
		if IsQuietCommand(req.OpCode) {
			continue
		}

		resp, err := n.conn.ReadPacket(reqBatch.ctx, req.OpCode, req.Opaque)
		if err != nil {
			reqBatch.responseChan <- &Response{Err: err}
			continue
		}

		// TODO add publish timeout
		reqBatch.responseChan <- resp
	}
	close(reqBatch.responseChan)
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
