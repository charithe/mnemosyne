package memcache

import (
	"context"
	"testing"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	mockServer, err := internal.NewMockServer("n1")
	assert.NoError(t, err)
	defer mockServer.Shutdown()

	c, err := NewClient(WithConnector(&internal.MockConnector{}), WithNodePicker(&internal.MockNodePicker{}))
	assert.NoError(t, err)
	defer c.Close()

	testCases := []struct {
		name          string
		serverSession []*internal.Scenario
		clientSession func(t *testing.T, client *Client, expectedResults []Result)
	}{
		{
			name: "MakeRequest success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key")},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n1_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Get(context.Background(), []byte("n1_key"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
		{
			name: "MakeRequest not found",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key")},
					Respond: &internal.Response{StatusCode: internal.StatusKeyNotFound, OpCode: internal.OpGetK, Key: []byte("n1_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Get(context.Background(), []byte("n1_key"))
				assert.Error(t, err)
				assert.NotNil(t, r)
				assert.Equal(t, ErrKeyNotFound, r.Err())
			},
		},
		{
			name: "MakeRequest timeout",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key")},
					Delay:  10 * time.Millisecond,
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Millisecond)
				defer cancelFunc()

				_, err := c.Get(ctx, []byte("n1_key"))
				assert.Error(t, err)
			},
		},
		{
			name: "Mutation with CAS and expiry",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpSet, Key: []byte("n1_key"), Value: []byte("n1_value"), Extras: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0E, 0x10}, CAS: 1234},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpSet, Key: []byte("n1_key"), CAS: 1234},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Set(context.Background(), []byte("n1_key"), []byte("n1_value"), WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
		{
			name: "NumericOp with CAS and expiry",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpIncrement, Key: []byte("n1_key"), Extras: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0A, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0E, 0x10}, CAS: 1234},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpIncrement, Key: []byte("n1_key"), Value: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0A}, CAS: 1234},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Increment(context.Background(), []byte("n1_key"), 0, 10, WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(10), r.NumericValue())
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
		{
			name: "Update with CAS and expiry",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpAppend, Key: []byte("n1_key"), Value: []byte("n1_value"), CAS: 1234},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpAppend, Key: []byte("n1_key"), CAS: 1234},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Append(context.Background(), []byte("n1_key"), []byte("n1_value"), WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
		{
			name: "Touch command",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpTouch, Key: []byte("n1_key"), Extras: []byte{0x0, 0x0, 0x0E, 0x10}},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpTouch, Key: []byte("n1_key")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.Touch(context.Background(), 1*time.Hour, []byte("n1_key"))
				assert.NoError(t, err)
			},
		},
		{
			name: "Flush command with expiry",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpFlush, Extras: []byte{0x0, 0x0, 0x0E, 0x10}},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpFlush},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.Flush(context.Background(), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
			},
		},
		{
			name: "DistributeRequests with missing keys",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Opaque: 1},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key3"), Opaque: 2},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n1_key3"), Value: []byte("value3"), Opaque: 2},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				rr, err := c.MultiGet(context.Background(), []byte("n1_key1"), []byte("n1_key2"), []byte("n1_key3"))
				assert.NoError(t, err)
				assert.Len(t, rr, 2)
				assert.ElementsMatch(t, expectedResults, rr)
			},
		},
		{
			name: "DistributeRequests with no response on sentinel key",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Value: []byte("value2"), Opaque: 1},
				},
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key3"), Opaque: 2},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancelFunc()

				rr, err := c.MultiGet(ctx, []byte("n1_key1"), []byte("n1_key2"), []byte("n1_key3"))
				assert.Error(t, err)
				assert.Len(t, rr, 3)
				for _, r := range rr {
					k := string(r.Key())
					switch k {
					case "n1_key1":
						assert.Equal(t, []byte("value1"), r.Value())
					case "n1_key2":
						assert.Equal(t, []byte("value2"), r.Value())
					default:
						assert.Equal(t, []byte("n1_key3"), r.Key())
						assert.Error(t, r.Err())
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer.AddSession(tc.serverSession...)
			expectedResults := toExpectedResults(tc.serverSession)
			tc.clientSession(t, c, expectedResults)
		})
	}
}

func TestDistribution(t *testing.T) {
	mockServer1, err := internal.NewMockServer("n2")
	assert.NoError(t, err)
	defer mockServer1.Shutdown()

	mockServer2, err := internal.NewMockServer("n3")
	assert.NoError(t, err)
	defer mockServer2.Shutdown()

	c, err := NewClient(WithConnector(&internal.MockConnector{}), WithNodePicker(&internal.MockNodePicker{}))
	assert.NoError(t, err)
	defer c.Close()

	testCases := []struct {
		name           string
		server1Session []*internal.Scenario
		server2Session []*internal.Scenario
		clientSession  func(t *testing.T, client *Client, expectedResults []Result)
	}{
		{
			name: "MultiGet fan out",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Value: []byte("value2"), Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Opaque: 0},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Value: []byte("value3"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n3_key4"), Opaque: 1},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("n3_key4"), Value: []byte("value4"), Opaque: 1},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				rr, err := c.MultiGet(context.Background(), []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"), []byte("n3_key4"))
				assert.NoError(t, err)
				assert.Len(t, rr, 4)
				assert.ElementsMatch(t, expectedResults, rr)
			},
		},
		{
			name: "MultiGet node timeout",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Value: []byte("value2"), Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n3_key3"), Opaque: 0},
					Delay:  10 * time.Millisecond,
				},
			},

			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Millisecond)
				defer cancelFunc()

				rr, err := c.MultiGet(ctx, []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"))
				assert.Error(t, err)
				assert.Len(t, rr, 3)
				for _, r := range rr {
					k := string(r.Key())
					switch k {
					case "n2_key1":
						assert.Equal(t, []byte("value1"), r.Value())
					case "n2_key2":
						assert.Equal(t, []byte("value2"), r.Value())
					default:
						assert.Equal(t, []byte("n3_key3"), r.Key())
						assert.Error(t, r.Err())
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expectedResults := toExpectedResults(tc.server1Session)
			expectedResults = append(expectedResults, toExpectedResults(tc.server2Session)...)

			mockServer1.AddSession(tc.server1Session...)
			mockServer2.AddSession(tc.server2Session...)

			tc.clientSession(t, c, expectedResults)
		})
	}
}

func toExpectedResults(scenarios []*internal.Scenario) []Result {
	var expectedResults []Result
	for _, s := range scenarios {
		if s.Respond == nil {
			continue
		}
		expectedResults = append(expectedResults, newResult(s.Respond))
	}
	return expectedResults
}
