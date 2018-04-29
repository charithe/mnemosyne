package internal

import (
	"bytes"
	"net"
	"time"
)

type mockConn struct {
	*bytes.Buffer
	readDeadline  time.Time
	writeDeadline time.Time
}

func (mc *mockConn) Close() error {
	return nil
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	mc.readDeadline = t
	return nil
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	mc.writeDeadline = t
	return nil
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	mc.readDeadline = t
	mc.writeDeadline = t
	return nil
}

func (mc *mockConn) LocalAddr() net.Addr {
	ip := net.ParseIP("127.0.0.1")
	return &net.IPAddr{IP: ip, Zone: ""}
}

func (mc *mockConn) RemoteAddr() net.Addr {
	ip := net.ParseIP("127.0.0.1")
	return &net.IPAddr{IP: ip, Zone: ""}
}

func (mc *mockConn) setPayload(p []byte) error {
	mc.Reset()
	if _, err := mc.Write(p); err != nil {
		return err
	}
	return nil
}

func (mc *mockConn) getPayload() []byte {
	return mc.Bytes()
}

func newMockConnWrapper() (*ConnWrapper, *mockConn) {
	mock := &mockConn{
		Buffer: new(bytes.Buffer),
	}
	bp := NewBufPool()
	cw := NewConnWrapper(mock, bp)
	return cw, mock
}
