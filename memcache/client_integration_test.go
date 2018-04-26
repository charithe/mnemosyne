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
	//c, err = NewSimpleClient("localhost:11211", "localhost:11212", "localhost:11213")
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
			r := c.Set(ctx, tc.key, tc.value, WithExpiry(1*time.Hour))
			assert.NoError(t, r.Err())

			r = c.Get(ctx, tc.key)
			assert.NoError(t, r.Err())
			assert.Equal(t, tc.key, r.Key())
			assert.Equal(t, tc.value, r.Value())
		})
	}

	t.Run("MultiGet", func(t *testing.T) {
		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		results := c.MultiGet(ctx, keys...)
		assert.Equal(t, numKV, len(results))
	})
}

func TestDelete(t *testing.T) {
	key := []byte("keydel")
	value := []byte("valdel")

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r := c.Set(ctx, key, value, WithExpiry(1*time.Hour))
	assert.NoError(t, r.Err())

	r = c.Get(ctx, key)
	assert.NoError(t, r.Err())
	assert.Equal(t, key, r.Key())
	assert.Equal(t, value, r.Value())

	r = c.Delete(ctx, key)
	assert.NoError(t, r.Err())

	r = c.Get(ctx, key)
	assert.Error(t, r.Err())
}

func TestMultiDelete(t *testing.T) {
	numKeys := 25
	keys := make([][]byte, numKeys)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	for i := 0; i < numKeys; i++ {
		keys[i] = []byte(fmt.Sprintf("delkey%d", i))
		r := c.Set(ctx, keys[i], []byte(fmt.Sprintf("value%d", i)))
		assert.NoError(t, r.Err())
	}

	rr := c.MultiGet(ctx, keys...)
	assert.Len(t, rr, numKeys)
	for _, r := range rr {
		assert.NoError(t, r.Err())
	}

	rr = c.MultiDelete(ctx, keys...)
	assert.Len(t, rr, 1)
	assert.NoError(t, rr[0].Err())

	rr = c.MultiGet(ctx, keys...)
	assert.Len(t, rr, 1)
	assert.Error(t, rr[0].Err())
}

func TestCAS(t *testing.T) {
	key := []byte(fmt.Sprintf("keyXXX%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r := c.Add(ctx, key, []byte("valueXXX"))
	assert.NoError(t, r.Err())

	cas := r.CAS()

	// Should succeed because the CAS value matches
	r = c.Set(ctx, key, []byte("valueXXY"), WithCASValue(cas))
	assert.NoError(t, r.Err())

	// Should fail because the key already exists
	r = c.Add(ctx, key, []byte("valueXXX"), WithCASValue(cas))
	assert.Error(t, r.Err())
	assert.Equal(t, ErrKeyExists, r.Err())

	// Should fail because the CAS value doesn't match
	r = c.Set(ctx, key, []byte("valueYYY"), WithCASValue(cas+uint64(1000)))
	assert.Error(t, r.Err())
	assert.Equal(t, ErrKeyExists, r.Err())

	r = c.Get(ctx, key)
	assert.NoError(t, r.Err())
	assert.Equal(t, []byte("valueXXY"), r.Value())

	//clean up
	c.Delete(ctx, key)
}

func TestIncrDecr(t *testing.T) {
	key := []byte(fmt.Sprintf("keyINC%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r := c.Increment(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, r.Err())
	assert.Equal(t, uint64(1000), r.NumericValue())

	r = c.Increment(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, r.Err())
	assert.Equal(t, uint64(1100), r.NumericValue())

	r = c.Decrement(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, r.Err())
	assert.Equal(t, uint64(1000), r.NumericValue())

	//clean up
	c.Delete(ctx, key)
}

func TestFlush(t *testing.T) {
	numKeys := 25
	keys := make([][]byte, numKeys)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	for i := 0; i < numKeys; i++ {
		keys[i] = []byte(fmt.Sprintf("flushkey%d", i))
		r := c.Set(ctx, keys[i], []byte(fmt.Sprintf("value%d", i)))
		assert.NoError(t, r.Err())
	}

	rr := c.MultiGet(ctx, keys...)
	assert.Len(t, rr, numKeys)
	for _, r := range rr {
		assert.NoError(t, r.Err())
	}

	err := c.Flush(ctx)
	assert.NoError(t, err)

	rr = c.MultiGet(ctx, keys...)
	assert.Len(t, rr, 1)
	assert.Error(t, rr[0].Err())
}

func TestAppendAndPrepend(t *testing.T) {
	key := []byte(fmt.Sprintf("keyAPP%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r := c.Set(ctx, key, []byte("value"))
	assert.NoError(t, r.Err())

	r = c.Append(ctx, key, []byte("YYY"), WithCASValue(r.CAS()))
	assert.NoError(t, r.Err())

	r = c.Prepend(ctx, key, []byte("XXX"), WithCASValue(r.CAS()))
	assert.NoError(t, r.Err())

	r = c.Get(ctx, key)
	assert.NoError(t, r.Err())
	assert.Equal(t, []byte("XXXvalueYYY"), r.Value())

	//clean up
	c.Delete(ctx, key)
}

func TestTouchAndGAT(t *testing.T) {
	key := []byte(fmt.Sprintf("keyTouch%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r := c.Set(ctx, key, []byte("value"))
	assert.NoError(t, r.Err())

	r = c.Touch(ctx, 1*time.Hour, key)
	assert.NoError(t, r.Err())

	r = c.GetAndTouch(ctx, 1*time.Hour, key)
	assert.NoError(t, r.Err())
	assert.Equal(t, []byte("value"), r.Value())

	//clean up
	c.Delete(ctx, key)
}
