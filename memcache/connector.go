package memcache

import (
	"context"
	"net"
)

// Connector defines the mechanism for connecting to a node
type Connector interface {
	Connect(ctx context.Context, nodeID string) (net.Conn, error)
}

// SimpleTCPConnector implements a simple TCP connection factory
type SimpleTCPConnector struct{}

func (s *SimpleTCPConnector) Connect(ctx context.Context, nodeID string) (net.Conn, error) {
	dialer := &net.Dialer{}
	return dialer.DialContext(ctx, "tcp", nodeID)
}
