package internal

import (
	"bytes"
	"fmt"
	"sync"
)

//DecodeErr is an error that could spring up during read operations
type DecodeErr struct {
	err error
}

func (d *DecodeErr) Error() string {
	return fmt.Sprintf("Decode error: %+v", d.err)
}

//Buf is a wrapper around bytes.Buffer to provide additional network-order read/write operations
type Buf struct {
	*bytes.Buffer
}

func NewBuf() *Buf {
	return &Buf{new(bytes.Buffer)}
}

func NewBufFrom(b []byte) *Buf {
	return &Buf{bytes.NewBuffer(b)}
}

func (b *Buf) WriteUint8(v uint8) {
	b.WriteByte(byte(v))
}

func (b *Buf) ReadUint8() uint8 {
	v, err := b.ReadByte()
	if err != nil {
		panic(&DecodeErr{err: err})
	}
	return uint8(v)
}

func (b *Buf) WriteUint16(v uint16) {
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}

func (b *Buf) ReadUint16() uint16 {
	var v [2]byte
	if _, err := b.Read(v[:]); err != nil {
		panic(&DecodeErr{err: err})
	}

	return uint16(v[1]) | uint16(v[0])<<8
}

func (b *Buf) WriteUint32(v uint32) {
	b.WriteByte(byte(v >> 24))
	b.WriteByte(byte(v >> 16))
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}

func (b *Buf) ReadUint32() uint32 {
	var v [4]byte
	if _, err := b.Read(v[:]); err != nil {
		panic(&DecodeErr{err: err})
	}

	return uint32(v[3]) | uint32(v[2])<<8 | uint32(v[1])<<16 | uint32(v[0])<<24
}

func (b *Buf) WriteUint64(v uint64) {
	b.WriteByte(byte(v >> 56))
	b.WriteByte(byte(v >> 48))
	b.WriteByte(byte(v >> 40))
	b.WriteByte(byte(v >> 32))
	b.WriteByte(byte(v >> 24))
	b.WriteByte(byte(v >> 16))
	b.WriteByte(byte(v >> 8))
	b.WriteByte(byte(v))
}

func (b *Buf) ReadUint64() uint64 {
	var v [8]byte
	if _, err := b.Read(v[:]); err != nil {
		panic(&DecodeErr{err: err})
	}

	return uint64(v[7]) | uint64(v[6])<<8 | uint64(v[5])<<16 | uint64(v[4])<<24 | uint64(v[3])<<32 | uint64(v[2])<<40 | uint64(v[1])<<48 | uint64(v[0])<<56
}

func (b *Buf) SkipBytes(n int) {
	for i := 0; i < n; i++ {
		if _, err := b.ReadByte(); err != nil {
			panic(&DecodeErr{err: err})
		}
	}
}

func (b *Buf) ReadBytes(n int) []byte {
	v := make([]byte, n)
	_, err := b.Read(v)
	if err != nil {
		panic(&DecodeErr{err: err})
	}

	return v
}

//BufPool is a pool of Buf objects
type BufPool struct {
	*sync.Pool
}

func NewBufPool() *BufPool {
	return &BufPool{&sync.Pool{}}
}

func (bp *BufPool) GetBuf() *Buf {
	if b := bp.Get(); b != nil {
		return b.(*Buf)
	}

	return NewBuf()
}

func (bp *BufPool) PutBuf(b *Buf) {
	b.Reset()
	bp.Put(b)
}
