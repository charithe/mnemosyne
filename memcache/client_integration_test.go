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
	c, err = NewSimpleClient("localhost:11211", "localhost:11212", "localhost:11213")
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
			_, err := c.Set(ctx, tc.key, tc.value, WithExpiry(1*time.Hour))
			assert.NoError(t, err)

			r, err := c.Get(ctx, tc.key)
			assert.NoError(t, err)
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

func TestDelete(t *testing.T) {
	key := []byte("keydel")
	value := []byte("valdel")

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	_, err := c.Set(ctx, key, value, WithExpiry(1*time.Hour))
	assert.NoError(t, err)

	r, err := c.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, key, r.Key())
	assert.Equal(t, value, r.Value())

	assert.NoError(t, c.Delete(ctx, key))

	_, err = c.Get(ctx, key)
	assert.Error(t, err)
}

func TestMultiDelete(t *testing.T) {
	numKeys := 25
	keys := make([][]byte, numKeys)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	for i := 0; i < numKeys; i++ {
		keys[i] = []byte(fmt.Sprintf("delkey%d", i))
		_, err := c.Set(ctx, keys[i], []byte(fmt.Sprintf("value%d", i)))
		assert.NoError(t, err)
	}

	rr, err := c.MultiGet(ctx, keys...)
	assert.NoError(t, err)
	assert.Len(t, rr, numKeys)

	assert.NoError(t, c.MultiDelete(ctx, keys...))

	_, err = c.MultiGet(ctx, keys...)
	assert.Error(t, err)
}

func TestCAS(t *testing.T) {
	key := []byte(fmt.Sprintf("keyXXX%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r, err := c.Add(ctx, key, []byte("valueXXX"))
	assert.NoError(t, err)

	cas := r.CAS()

	// Should succeed because the CAS value matches
	_, err = c.Set(ctx, key, []byte("valueXXY"), WithCASValue(cas))
	assert.NoError(t, err)

	// Should fail because the key already exists
	_, err = c.Add(ctx, key, []byte("valueXXX"), WithCASValue(cas))
	assert.Error(t, err)

	// Should fail because the CAS value doesn't match
	_, err = c.Set(ctx, key, []byte("valueYYY"), WithCASValue(cas+uint64(1000)))
	assert.Error(t, err)

	r, err = c.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("valueXXY"), r.Value())

	//clean up
	c.Delete(ctx, key)
}

func TestIncrDecr(t *testing.T) {
	key := []byte(fmt.Sprintf("keyINC%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r, err := c.Increment(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, uint64(1000), r.NumericValue())

	r, err = c.Increment(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, uint64(1100), r.NumericValue())

	r, err = c.Decrement(ctx, key, 1000, 100, WithExpiry(2*time.Hour))
	assert.NoError(t, err)
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
		_, err := c.Set(ctx, keys[i], []byte(fmt.Sprintf("value%d", i)))
		assert.NoError(t, err)
	}

	rr, err := c.MultiGet(ctx, keys...)
	assert.NoError(t, err)
	assert.Len(t, rr, numKeys)

	assert.NoError(t, c.Flush(ctx))

	rr, err = c.MultiGet(ctx, keys...)
	assert.Error(t, err)
}

func TestAppendAndPrepend(t *testing.T) {
	key := []byte(fmt.Sprintf("keyAPP%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	r, err := c.Set(ctx, key, []byte("value"))
	assert.NoError(t, err)

	r, err = c.Append(ctx, key, []byte("YYY"), WithCASValue(r.CAS()))
	assert.NoError(t, err)

	r, err = c.Prepend(ctx, key, []byte("XXX"), WithCASValue(r.CAS()))
	assert.NoError(t, err)

	r, err = c.Get(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("XXXvalueYYY"), r.Value())

	//clean up
	c.Delete(ctx, key)
}

func TestTouchAndGAT(t *testing.T) {
	key := []byte(fmt.Sprintf("keyTouch%d", rand.Int()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	_, err := c.Set(ctx, key, []byte("value"))
	assert.NoError(t, err)

	err = c.Touch(ctx, 1*time.Hour, key)
	assert.NoError(t, err)

	r, err := c.GetAndTouch(ctx, 1*time.Hour, key)
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), r.Value())

	//clean up
	c.Delete(ctx, key)
}

/*
func TestResiliency(t *testing.T) {
	toxClient := toxiproxy.NewClient("localhost:8474")
	proxy, err := toxClient.CreateProxy("mc", "localhost:21211", "memcached:11211")
	if err != nil {
		t.Fatalf("Can't start toxiproxy: %+v", err)
	}
	defer proxy.Delete()

	rc, err := NewSimpleClient("localhost:21211")
	if err != nil {
		t.Fatalf("Failed to start client for resiliency test: %+v", err)
	}
	defer rc.Close()

	t.Run("reconnect on connection drop", func(t *testing.T) {
		_, err := rc.Set(context.Background(), []byte("rk1"), []byte("rv1"))
		assert.NoError(t, err)

		proxy.AddToxic("timeout", "timeout", "", 1.0, toxiproxy.Attributes{
			"timeout": 300,
		})

		doWithContext(200*time.Millisecond, func(ctx context.Context) {
			_, err = rc.Get(ctx, []byte("rk1"))
			assert.Error(t, err)
		})

		doWithContext(200*time.Millisecond, func(ctx context.Context) {
			_, err = rc.Get(ctx, []byte("rk1"))
			assert.Error(t, err)
		})

		proxy.RemoveToxic("timeout")

		doWithContext(200*time.Millisecond, func(ctx context.Context) {
			res, err := rc.Get(ctx, []byte("rk1"))
			assert.NoError(t, err)
			assert.Equal(t, []byte("rv1"), res.Value())
		})
	})
}

func doWithContext(timeout time.Duration, f func(ctx context.Context)) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelFunc()
	f(ctx)
}
*/
