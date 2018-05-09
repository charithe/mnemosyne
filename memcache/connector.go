package memcache

import (
	"context"
	"net"
	"time"

	"github.com/eapache/go-resiliency/retrier"
)

// Connector defines the mechanism for connecting to a node
type Connector interface {
	Connect(nodeID string) (net.Conn, error)
}

func NewSimpleTCPConnector() Connector {
	return NewRetryingTCPConnector(100*time.Millisecond, 3, 200*time.Millisecond)
}

// NewRetryingTCPConnector returns a TCP connector that retries failed connections
func NewRetryingTCPConnector(connectTimeout time.Duration, numRetries int, retryWait time.Duration) Connector {
	return &RetryingTCPConnector{
		connectTimeout: connectTimeout,
		numRetries:     numRetries,
		retryWait:      retryWait,
	}
}

// RetryingTCPConnector implements a Connector with configurable connection retry parameters
type RetryingTCPConnector struct {
	connectTimeout time.Duration
	numRetries     int
	retryWait      time.Duration
}

func (rtc *RetryingTCPConnector) Connect(nodeID string) (net.Conn, error) {
	retry := retrier.New(retrier.ExponentialBackoff(rtc.numRetries, rtc.retryWait), nil)
	dialer := &net.Dialer{}
	var conn net.Conn
	err := retry.Run(func() error {
		var err error
		ctx, cancelFunc := context.WithTimeout(context.Background(), rtc.connectTimeout)
		defer cancelFunc()
		conn, err = dialer.DialContext(ctx, "tcp", nodeID)
		return err
	})

	return conn, err
}
