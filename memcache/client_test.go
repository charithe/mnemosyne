package memcache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGet(t *testing.T) {
	c, err := NewSimpleClient("localhost:11211")
	assert.NoError(t, err)
	defer c.Close()

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	type testCase struct {
		key   []byte
		value []byte
	}

	testCases := make([]testCase, 25)
	keys := make([][]byte, 25)
	for i := 0; i < 25; i++ {
		testCases[i] = testCase{
			key:   []byte(fmt.Sprintf("key%d", i)),
			value: []byte(fmt.Sprintf("value%d", i)),
		}

		keys[i] = testCases[i].key
	}

	for _, tc := range testCases {
		assert.NoError(t, c.Set(ctx, tc.key, tc.value))
		v, err := c.Get(ctx, tc.key)
		assert.NoError(t, err)
		assert.Equal(t, tc.value, v)
	}

	vals, err := c.MultiGet(ctx, keys...)
	assert.NoError(t, err)
	assert.Equal(t, 25, len(vals))
}
