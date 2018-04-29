package internal

import (
	"bytes"
	"context"
	"log"
	"net"
	"reflect"
	"time"

	"github.com/akutz/memconn"
)

// Scenario defines the optional delay and response to send back when the request matches the expected request
type Scenario struct {
	Expect  *Request
	Respond *Response
	Delay   time.Duration
}

type session struct {
	scenarios []*Scenario
}

// MockServer is a controllable server useful for unit testing the end-to-end behaviour of the client
type MockServer struct {
	listener net.Listener
	sessions chan *session
}

func NewMockServer(connName string) (*MockServer, error) {
	listener, err := memconn.Listen("memu", connName)
	if err != nil {
		return nil, err
	}

	ms := &MockServer{
		listener: listener,
		sessions: make(chan *session, 16),
	}

	go ms.serve()
	return ms, nil
}

func (ms *MockServer) serve() {
	conn, err := ms.listener.Accept()
	if err != nil {
		log.Printf("Failed to accept connection: %#v", err)
		return
	}

	defer conn.Close()
	for sess := range ms.sessions {
		for _, scn := range sess.scenarios {
			conn.SetReadDeadline(time.Now().Add(defaultReadDeadline))
			b := NewBuf()
			req, err := ParseRequest(conn, b)
			if err != nil {
				log.Printf("Failed to parse request: %#v", err)
				return
			}

			var resp *Response
			if requestsEqual(req, scn.Expect) {
				time.Sleep(scn.Delay)
				resp = scn.Respond
			} else {
				log.Printf("Unexpected request: Expected=%+v Actual=%+v", scn.Expect, req)
				resp = &Response{OpCode: req.OpCode, StatusCode: StatusUnknownCommand, Key: req.Key}
			}

			if resp == nil {
				continue
			}

			b.Reset()
			resp.AssembleBytes(b)
			conn.SetWriteDeadline(time.Now().Add(defaultWriteDeadline))
			_, err = conn.Write(b.Bytes())
			if err != nil {
				log.Printf("Failed to write response: %#v", err)
				return
			}
		}
	}
}

func (ms *MockServer) AddSession(scenarios ...*Scenario) {
	ms.sessions <- &session{scenarios: scenarios}
}

func (ms *MockServer) Shutdown() {
	close(ms.sessions)
	ms.listener.Close()
}

func requestsEqual(r1, r2 *Request) bool {
	return reflect.DeepEqual(r1.Key, r2.Key) &&
		reflect.DeepEqual(r1.Value, r2.Value) &&
		reflect.DeepEqual(r1.Extras, r2.Extras) &&
		r1.OpCode == r2.OpCode &&
		r1.Opaque == r2.Opaque &&
		r1.CAS == r2.CAS
}

// MockConnector implements the Connector interface for memconn connections
type MockConnector struct {
}

func (mc *MockConnector) Connect(ctx context.Context, nodeID string) (net.Conn, error) {
	return memconn.DialContext(ctx, "memu", nodeID)
}

// MockNodePicker implements the NodePicker interface
// It assumes that the key starts with nodeID followed by an underscore
type MockNodePicker struct {
}

func (mnp *MockNodePicker) Pick(key []byte) string {
	p := bytes.Split(key, []byte("_"))
	return string(p[0])
}
