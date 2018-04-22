package memcache

import (
	"context"
	"testing"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"github.com/stretchr/testify/assert"
)

func TestSingleNode(t *testing.T) {
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
			name: "Single Command",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key")},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("n1_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r := c.Get(context.Background(), []byte("n1_key"))
				assert.NoError(t, r.Err())
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
		{
			name: "Response timeout",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key")},
					Delay:  10 * time.Millisecond,
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Millisecond)
				defer cancelFunc()

				r := c.Get(ctx, []byte("n1_key"))
				assert.Error(t, r.Err())
			},
		},
		{
			name: "MultiGet with missing keys",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Opaque: 0},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Opaque: 1},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key3"), Opaque: 2},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("n1_key3"), Value: []byte("value3"), Opaque: 2},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				rr := c.MultiGet(context.Background(), []byte("n1_key1"), []byte("n1_key2"), []byte("n1_key3"))
				assert.Len(t, rr, 2)
				assert.ElementsMatch(t, expectedResults, rr)
			},
		},
		{
			name: "MultiGet with no response on sentinel key",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Opaque: 0},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n1_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Opaque: 1},
					Respond: &internal.Response{OpCode: internal.OpGetKQ, Key: []byte("n1_key2"), Value: []byte("value2"), Opaque: 1},
				},
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n1_key3"), Opaque: 2},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancelFunc()

				rr := c.MultiGet(ctx, []byte("n1_key1"), []byte("n1_key2"), []byte("n1_key3"))
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

func TestMultipleNodes(t *testing.T) {
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
				rr := c.MultiGet(context.Background(), []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"), []byte("n3_key4"))
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

				rr := c.MultiGet(ctx, []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"))
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
