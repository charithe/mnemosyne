// +build integration

package memcache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

var c *Client

func TestMain(m *testing.M) {
	var err error
	c, err = NewSimpleClient("localhost:11211")
	if err != nil {
		panic(err)
	}

	retVal := m.Run()
	c.Close()
	os.Exit(retVal)
}

func TestSetGetAndMultiGet(t *testing.T) {
	type testCase struct {
		key   []byte
		value []byte
	}

	numKV := 25
	testCases := make([]testCase, numKV)
	keys := make([][]byte, numKV)

	for i := 0; i < numKV; i++ {
		testCases[i] = testCase{
			key:   []byte(fmt.Sprintf("key%d", i)),
			value: []byte(fmt.Sprintf("value%d", i)),
		}

		keys[i] = testCases[i].key
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s", tc.key), func(t *testing.T) {
			ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFunc()
			r, err := c.Set(ctx, tc.key, tc.value, WithExpiry(1*time.Hour))
			assert.NoError(t, err)
			assert.NoError(t, r.Err())

			r, err = c.Get(ctx, tc.key)
			assert.NoError(t, err)
			assert.NoError(t, r.Err())
			assert.Equal(t, tc.key, r.Key())
			assert.Equal(t, tc.value, r.Value())
		})
	}

	t.Run("MultiGet", func(t *testing.T) {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		results, err := c.MultiGet(ctx, keys...)
		assert.NoError(t, err)
		assert.Equal(t, numKV, len(results))
	})
}

func TestCAS(t *testing.T) {
	key := []byte(fmt.Sprintf("keyXXX%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r, err := c.Add(ctx, key, []byte("valueXXX"))
	assert.NoError(t, err)
	assert.NoError(t, r.Err())

	cas := r.CAS()

	// Should succeed because the CAS value matches
	r, err = c.Set(ctx, key, []byte("valueXXY"), WithCASValue(cas))
	assert.NoError(t, err)
	assert.NoError(t, r.Err())

	// Should fail because the key already exists
	r, err = c.Add(ctx, key, []byte("valueXXX"), WithCASValue(cas))
	assert.NoError(t, err)
	assert.Error(t, r.Err())
	assert.Equal(t, ErrKeyExists, r.Err())

	// Should fail because the CAS value doesn't match
	r, err = c.Set(ctx, key, []byte("valueYYY"), WithCASValue(cas+uint64(1000)))
	assert.NoError(t, err)
	assert.Error(t, r.Err())
	assert.Equal(t, ErrKeyExists, r.Err())

	r, err = c.Get(ctx, key)
	assert.NoError(t, err)
	assert.NoError(t, r.Err())
	assert.Equal(t, []byte("valueXXY"), r.Value())
}
