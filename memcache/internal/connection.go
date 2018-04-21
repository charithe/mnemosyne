package internal

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"
)

const (
	headerSize           = 24
	defaultReadDeadline  = 100 * time.Millisecond
	defaultWriteDeadline = 100 * time.Millisecond
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

// Response represents a parsed response from memcache
type Response struct {
	OpCode     uint8
	StatusCode uint16
	Opaque     uint32
	CAS        uint64
	Extras     []byte
	Key        []byte
	Value      []byte
	Err        error
}

// ParseResponse parses a response from the provided ReadWriter
func ParseResponse(conn io.Reader, b *Buf) (pkt *Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*DecodeErr); ok {
				err = e
			}
		}
	}()

	if _, err := io.CopyN(b, conn, headerSize); err != nil {
		return nil, err
	}

	if magic := b.ReadUint8(); magic != responseMagic {
		return nil, ErrUnexpectedResponse
	}

	opCode := b.ReadUint8()
	keyLen := int(b.ReadUint16())
	extrasLen := int(b.ReadUint8())
	b.SkipBytes(1) // data type
	status := b.ReadUint16()
	bodyLen := int(b.ReadUint32())
	opaque := b.ReadUint32()
	cas := b.ReadUint64()
	valueLen := bodyLen - (keyLen + extrasLen)

	if bodyLen > 0 {
		if _, err := io.CopyN(b, conn, int64(bodyLen)); err != nil {
			return nil, err
		}
	}

	pkt = &Response{
		OpCode:     opCode,
		StatusCode: status,
		Opaque:     opaque,
		CAS:        cas,
	}

	if extrasLen > 0 {
		pkt.Extras = b.ReadBytes(extrasLen)
	}

	if keyLen > 0 {
		pkt.Key = b.ReadBytes(keyLen)
	}

	if valueLen > 0 {
		pkt.Value = b.ReadBytes(valueLen)
	}

	return pkt, nil
}

// AssembleBytes assembles the byte representation of the response over the wire
func (pkt *Response) AssembleBytes(b *Buf) {
	keyLen := len(pkt.Key)
	extrasLen := len(pkt.Extras)
	valueLen := len(pkt.Value)

	b.WriteUint8(responseMagic)                          //magic
	b.WriteUint8(pkt.OpCode)                             //opcode
	b.WriteUint16(uint16(keyLen))                        //key length
	b.WriteUint8(uint8(extrasLen))                       //extras
	b.WriteUint8(0)                                      //data type
	b.WriteUint16(pkt.StatusCode)                        //status code
	b.WriteUint32(uint32(keyLen + extrasLen + valueLen)) //total body length
	b.WriteUint32(pkt.Opaque)                            //opaque
	b.WriteUint64(pkt.CAS)                               //CAS

	for _, c := range pkt.Extras {
		b.WriteByte(c)
	}

	for _, c := range pkt.Key {
		b.WriteByte(c)
	}

	for _, c := range pkt.Value {
		b.WriteByte(c)
	}
}

// Request represents a memcache request
type Request struct {
	OpCode    uint8
	VBucketID uint16
	Opaque    uint32
	CAS       uint64
	Extras    []byte
	Key       []byte
	Value     []byte
}

// ParseRequest parses a request from the provided ReadWriter
func ParseRequest(conn io.Reader, b *Buf) (pkt *Request, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*DecodeErr); ok {
				err = e
			}
		}
	}()

	if _, err := io.CopyN(b, conn, headerSize); err != nil {
		return nil, err
	}

	if magic := b.ReadUint8(); magic != requestMagic {
		return nil, ErrUnexpectedRequest
	}

	opcode := b.ReadUint8()
	keyLen := int(b.ReadUint16())
	extrasLen := int(b.ReadUint8())
	b.SkipBytes(1) // data type
	vbucketID := b.ReadUint16()
	bodyLen := int(b.ReadUint32())
	opaque := b.ReadUint32()
	cas := b.ReadUint64()
	valueLen := bodyLen - (keyLen + extrasLen)

	if bodyLen > 0 {
		if _, err := io.CopyN(b, conn, int64(bodyLen)); err != nil {
			return nil, err
		}
	}

	pkt = &Request{
		OpCode:    opcode,
		Opaque:    opaque,
		CAS:       cas,
		VBucketID: vbucketID,
	}

	if extrasLen > 0 {
		pkt.Extras = b.ReadBytes(extrasLen)
	}

	if keyLen > 0 {
		pkt.Key = b.ReadBytes(keyLen)
	}

	if valueLen > 0 {
		pkt.Value = b.ReadBytes(valueLen)
	}

	return pkt, nil
}

// AssembleBytes generates the byte representation of the request to send over the wire
func (pkt *Request) AssembleBytes(b *Buf) {
	keyLen := len(pkt.Key)
	extrasLen := len(pkt.Extras)
	valueLen := len(pkt.Value)

	b.WriteUint8(requestMagic)                           //magic
	b.WriteUint8(pkt.OpCode)                             //opcode
	b.WriteUint16(uint16(keyLen))                        //key length
	b.WriteUint8(uint8(extrasLen))                       //extras
	b.WriteUint8(0)                                      //data type
	b.WriteUint16(pkt.VBucketID)                         //vbucket
	b.WriteUint32(uint32(keyLen + extrasLen + valueLen)) //total body length
	b.WriteUint32(pkt.Opaque)                            //opaque
	b.WriteUint64(pkt.CAS)                               //CAS

	for _, c := range pkt.Extras {
		b.WriteByte(c)
	}

	for _, c := range pkt.Key {
		b.WriteByte(c)
	}

	for _, c := range pkt.Value {
		b.WriteByte(c)
	}
}

// ConnWrapper wraps a connection and provides convenient packet read/write functions
type ConnWrapper struct {
	reqID     uint32
	conn      Connection
	bufReader *bufio.Reader
	bufWriter *bufio.Writer
	bufPool   *BufPool
}

// for testing purposes
func NewConnWrapper(conn Connection, bufPool *BufPool) *ConnWrapper {
	return &ConnWrapper{
		conn:      conn,
		bufReader: bufio.NewReader(conn),
		bufWriter: bufio.NewWriter(conn),
		bufPool:   bufPool,
	}
}

func (cw *ConnWrapper) ReadPacket(ctx context.Context, opCode uint8, reqID uint32) (pkt *Response, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(*DecodeErr); ok {
				err = e
			}
		}
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(defaultReadDeadline)
	}

	cw.conn.SetReadDeadline(dl)
	b := NewBuf()
	pkt, err = ParseResponse(cw.bufReader, b)
	return
}

func (cw *ConnWrapper) WritePacket(ctx context.Context, pkt *Request) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	dl, ok := ctx.Deadline()
	if !ok {
		dl = time.Now().Add(defaultWriteDeadline)
	}

	cw.conn.SetWriteDeadline(dl)
	b := cw.bufPool.GetBuf()
	pkt.AssembleBytes(b)

	_, err := cw.bufWriter.Write(b.Bytes())
	cw.bufPool.PutBuf(b)

	return err
}

func (cw *ConnWrapper) Flush() error {
	return cw.bufWriter.Flush()
}

func (cw *ConnWrapper) Close() error {
	return cw.conn.Close()
}
