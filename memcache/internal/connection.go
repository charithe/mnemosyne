package internal

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"time"
)

const (
	headerSize           = 24
	defaultReadDeadline  = 200 * time.Millisecond
	defaultWriteDeadline = 200 * time.Millisecond
	tempBufferSize       = 256
)

var (
	ErrConnectionShutdown = errors.New("Connection shutdown")
	ErrAlreadyShutdown    = errors.New("Already shutdown")
	ErrUnexpectedResponse = errors.New("Unexpected response")
	ErrUnexpectedRequest  = errors.New("Unexpected request")
)

// Connecction is a physical connection to a memcache server
type Connection interface {
	io.ReadWriteCloser
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// ConnWrapper wraps a connection and provides convenient packet read/write functions
type ConnWrapper struct {
	net.Conn
	reqID      uint32
	conn       Connection
	bufReader  *bufio.Reader
	bufWriter  *bufio.Writer
	bufPool    *BufPool
	tempBuffer []byte
}

func NewConnWrapper(conn net.Conn, bufPool *BufPool) *ConnWrapper {
	return &ConnWrapper{
		Conn:       conn,
		bufReader:  bufio.NewReader(conn),
		bufWriter:  bufio.NewWriter(conn),
		bufPool:    bufPool,
		tempBuffer: make([]byte, tempBufferSize),
	}
}

func (cw *ConnWrapper) Check(ctx context.Context) error {
	req := &Request{
		OpCode: OpNoOp,
	}

	return cw.WritePacket(ctx, req)
}

func (cw *ConnWrapper) ReadPacket(ctx context.Context) (pkt *Response, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(defaultReadDeadline)
	}

	cw.SetReadDeadline(dl)
	b := NewBuf()
	pkt, err = ParseResponse(cw.bufReader, b)
	return
}

func (cw *ConnWrapper) WritePacket(ctx context.Context, pkt *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b := cw.bufPool.GetBuf()
	pkt.AssembleBytes(b)

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(defaultWriteDeadline)
	}

	cw.SetWriteDeadline(dl)
	_, err := cw.bufWriter.Write(b.Bytes())
	cw.bufPool.PutBuf(b)

	return err
}

func (cw *ConnWrapper) FlushWriteBuffer() error {
	return cw.bufWriter.Flush()
}

func (cw *ConnWrapper) FlushReadBuffer() error {
	cw.SetReadDeadline(time.Now())
	if _, err := cw.bufReader.Read(cw.tempBuffer); err != nil {
		return err
	}
	cw.bufReader.Reset(cw)
	return nil
}
