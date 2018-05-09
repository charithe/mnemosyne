// +build integration

package memcache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/client"
	"github.com/charithe/mnemosyne/ocmemcache"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/stretchr/testify/assert"
	"go.opencensus.io/examples/exporter"
	"go.opencensus.io/stats/view"
	"golang.org/x/exp/rand"
)

var c *Client

func TestMain(m *testing.M) {
	var err error
	c, err = NewClient(WithNodePicker(NewSimpleNodePicker("localhost:11211", "localhost:11212", "localhost:11213")), WithConnectionsPerNode(3))
	if err != nil {
		panic(err)
	}

	view.Register(ocmemcache.DefaultViews...)
	view.RegisterExporter(&exporter.PrintExporter{})
	view.SetReportingPeriod(200 * time.Millisecond)

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

func TestFlushAll(t *testing.T) {
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

	assert.NoError(t, c.FlushAll(ctx))

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

func TestStats(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	// Getting all the stats
	stats, err := c.Stats(ctx, "localhost:11211", "")
	assert.NoError(t, err)
	assert.True(t, len(stats) > 0)
	assert.Contains(t, stats, "max_connections")
	assert.Contains(t, stats, "total_connections")

	// Getting a group
	stats, err = c.Stats(ctx, "localhost:11211", "items")
	assert.NoError(t, err)
	assert.True(t, len(stats) > 0)
}

func TestVersion(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	version, err := c.Version(ctx, "localhost:11211")
	assert.NoError(t, err)
	assert.NotZero(t, version)
}

func TestResiliency(t *testing.T) {
	toxClient := toxiproxy.NewClient("localhost:8474")
	proxy, err := toxClient.CreateProxy("mc", ":21211", "resiliency:11211")
	if err != nil {
		t.Fatalf("Can't start toxiproxy: %+v", err)
	}
	defer proxy.Delete()

	rc, err := NewSimpleClient("localhost:21211")
	//rc, err := NewClient(WithNodePicker(NewSimpleNodePicker("localhost:21211")), WithConnectionsPerNode(3))
	if err != nil {
		t.Fatalf("Failed to start client for resiliency test: %+v", err)
	}
	defer rc.Close()

	t.Run("recover from initial connection failure", func(t *testing.T) {
		ctxTimeout := 100 * time.Millisecond

		proxy.Disable()

		doWithContext(ctxTimeout, func(ctx context.Context) {
			res, _ := rc.Get(ctx, []byte("rk0"))
			assert.NotEqual(t, ErrKeyNotFound, res.Err())
		})

		proxy.Enable()

		// toxiproxy operations don't take effect immediately so we have to try a couple of times
		retry := retrier.New(retrier.ExponentialBackoff(3, 100*time.Millisecond), nil)
		var res Result
		err := retry.Run(func() error {
			var err error
			doWithContext(ctxTimeout, func(ctx context.Context) {
				res, err = rc.Get(ctx, []byte("rk0"))
			})
			return err
		})

		assert.Error(t, err)
		assert.Equal(t, ErrKeyNotFound, res.Err())
	})

	t.Run("reconnect on connection drop", func(t *testing.T) {
		ctxTimeout := 100 * time.Millisecond

		doWithContext(ctxTimeout, func(ctx context.Context) {
			_, err := rc.Set(ctx, []byte("rk1"), []byte("rv1"))
			assert.NoError(t, err)
		})

		doWithContext(ctxTimeout, func(ctx context.Context) {
			res, err := rc.Get(ctx, []byte("rk1"))
			assert.NoError(t, err)
			assert.Equal(t, []byte("rv1"), res.Value())
		})

		proxy.Disable()

		doWithContext(ctxTimeout, func(ctx context.Context) {
			_, err = rc.Get(ctx, []byte("rk1"))
			assert.Error(t, err)
		})

		proxy.Enable()

		// reestablishing the connection is not instantaneous so we have to try a couple of times
		retry := retrier.New(retrier.ExponentialBackoff(3, 100*time.Millisecond), nil)
		var res Result
		err := retry.Run(func() error {
			var err error
			doWithContext(ctxTimeout, func(ctx context.Context) {
				res, err = rc.Get(ctx, []byte("rk1"))
			})
			return err
		})

		assert.NoError(t, err)
		assert.Equal(t, []byte("rv1"), res.Value())
	})

	t.Run("reconnect on connection timeout", func(t *testing.T) {
		ctxTimeout := 100 * time.Millisecond
		connTimeout := 100

		doWithContext(ctxTimeout, func(ctx context.Context) {
			_, err := rc.Set(ctx, []byte("rk2"), []byte("rv2"))
			assert.NoError(t, err)
		})

		doWithContext(ctxTimeout, func(ctx context.Context) {
			res, err := rc.Get(ctx, []byte("rk2"))
			assert.NoError(t, err)
			assert.Equal(t, []byte("rv2"), res.Value())
		})

		proxy.AddToxic("test_timeout", "timeout", "upstream", 1.0, toxiproxy.Attributes{
			"timeout": connTimeout,
		})

		doWithContext(ctxTimeout, func(ctx context.Context) {
			_, err = rc.Get(ctx, []byte("rk2"))
			assert.Error(t, err)
		})

		doWithContext(ctxTimeout, func(ctx context.Context) {
			_, err = rc.Get(ctx, []byte("rk2"))
			assert.Error(t, err)
		})

		assert.NoError(t, proxy.RemoveToxic("test_timeout"))

		// reestablishing the connection is not instantaneous so we have to try a couple of times
		retry := retrier.New(retrier.ExponentialBackoff(3, 100*time.Millisecond), nil)
		var res Result
		err := retry.Run(func() error {
			var err error
			doWithContext(ctxTimeout, func(ctx context.Context) {
				res, err = rc.Get(ctx, []byte("rk2"))
			})
			return err
		})

		assert.NoError(t, err)
		assert.Equal(t, []byte("rv2"), res.Value())
	})
}

func doWithContext(timeout time.Duration, f func(ctx context.Context)) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	f(ctx)
}
