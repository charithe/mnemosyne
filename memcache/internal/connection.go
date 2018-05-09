package internal

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/charithe/mnemosyne/ocmemcache"
	"go.opencensus.io/stats"
)

const (
	headerSize            = 24
	connRetryWaitDuration = 200 * time.Millisecond
	defaultReadDeadline   = 200 * time.Millisecond
	defaultWriteDeadline  = 200 * time.Millisecond
	numConnectRetries     = 3
	tempBufferSize        = 256
)

var (
	ErrConnectionShutdown = errors.New("Connection shutdown")
	ErrAlreadyShutdown    = errors.New("Already shutdown")
	ErrUnexpectedResponse = errors.New("Unexpected response")
	ErrUnexpectedRequest  = errors.New("Unexpected request")
)

type ConnectFunc func() (net.Conn, error)

// ConnWrapper wraps a connection and provides convenient packet read/write functions
type ConnWrapper struct {
	net.Conn
	connectFunc ConnectFunc
	bufReader   *bufio.Reader
	bufWriter   *bufio.Writer
	bufPool     *BufPool
	tempBuffer  []byte
	healthy     bool
}

func NewConnWrapper(connectFunc ConnectFunc, bufPool *BufPool) (*ConnWrapper, error) {
	conn, err := connectFunc()
	if err != nil {
		return nil, err
	}

	return &ConnWrapper{
		Conn:        conn,
		connectFunc: connectFunc,
		bufReader:   bufio.NewReader(conn),
		bufWriter:   bufio.NewWriter(conn),
		bufPool:     bufPool,
		tempBuffer:  make([]byte, tempBufferSize),
		healthy:     true,
	}, nil
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
	if pkt != nil {
		stats.Record(ctx, ocmemcache.ResponseBytesMetric.M(float64(len(pkt.Key)+len(pkt.Value)+len(pkt.Extras))))
	}
	cw.checkError(ctx, err)
	return
}

func (cw *ConnWrapper) WritePacket(ctx context.Context, pkt *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	b := cw.bufPool.GetBuf()
	pkt.AssembleBytes(b)
	stats.Record(ctx, ocmemcache.RequestBytesMetric.M(float64(len(pkt.Key)+len(pkt.Value)+len(pkt.Extras))))

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(defaultWriteDeadline)
	}

	cw.SetWriteDeadline(dl)
	_, err := cw.bufWriter.Write(b.Bytes())
	cw.bufPool.PutBuf(b)

	cw.checkError(ctx, err)
	return err
}

func (cw *ConnWrapper) FlushWriteBuffer() error {
	err := cw.bufWriter.Flush()
	cw.checkError(context.Background(), err)
	return err
}

func (cw *ConnWrapper) FlushReadBuffer() error {
	cw.SetReadDeadline(time.Now())
	if _, err := cw.bufReader.Read(cw.tempBuffer); err != nil {
		cw.checkError(context.Background(), err)
		return err
	}
	cw.bufReader.Reset(cw)
	return nil
}

func (cw *ConnWrapper) checkError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	if err == io.EOF {
		stats.Record(ctx, ocmemcache.ConnectionErrorsMetric.M(1))
		cw.healthy = false
		return
	}

	if neterr, ok := err.(net.Error); ok {
		if !neterr.Temporary() {
			stats.Record(ctx, ocmemcache.ConnectionErrorsMetric.M(1))
			cw.healthy = false
			return
		}
	}
}

func (cw *ConnWrapper) ReconnectIfUnhealthy() error {
	if cw.healthy {
		return nil
	}

	cw.Conn.Close()
	conn, err := cw.connectFunc()
	if err != nil {
		return err
	}

	cw.Conn = conn
	cw.healthy = true
	cw.bufReader.Reset(cw.Conn)
	cw.bufWriter.Reset(cw.Conn)
	return nil
}
