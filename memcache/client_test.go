package memcache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	c, err := NewSimpleClient("localhost:11211")
	assert.NoError(t, err)
	defer c.Close()

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	assert.NoError(t, c.SetWithExpiry(ctx, []byte("key1"), []byte("value1"), 1*time.Hour))
	v, err := c.Get(ctx, []byte("key1"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), v)
}
