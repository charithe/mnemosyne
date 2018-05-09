package memcache

import (
	"context"
	"testing"
	"time"

	"github.com/charithe/mnemosyne/memcache/internal"
	"github.com/stretchr/testify/assert"
)

func TestSimpleCommands(t *testing.T) {
	mockServer, err := internal.NewMockServer("n0")
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
			name: "Get success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n0_key")},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n0_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Get(context.Background(), []byte("n0_key"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
		{
			name: "Get not found",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n0_key")},
					Respond: &internal.Response{StatusCode: internal.StatusKeyNotFound, OpCode: internal.OpGetK, Key: []byte("n0_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.Get(context.Background(), []byte("n0_key"))
				assert.Error(t, err)
				assert.NotNil(t, r)
				assert.Equal(t, ErrKeyNotFound, r.Err())
			},
		},
		{
			name: "Get timeout",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n0_key")},
					Delay:  10 * time.Millisecond,
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Millisecond)
				defer cancelFunc()

				_, err := c.Get(ctx, []byte("n0_key"))
				assert.Error(t, err)
			},
		},
		{
			name: "Delete success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDelete, Key: []byte("n0_key")},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpDelete},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.Delete(context.Background(), []byte("n0_key"))
				assert.NoError(t, err)
			},
		},
		{
			name: "Touch success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpTouch, Key: []byte("n0_key"), Extras: []byte{0x0, 0x0, 0x0E, 0x10}},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpTouch, Key: []byte("n0_key")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.Touch(context.Background(), 1*time.Hour, []byte("n0_key"))
				assert.NoError(t, err)
			},
		},
		{
			name: "GetAndTouch success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGAT, Key: []byte("n0_key"), Extras: []byte{0x0, 0x0, 0x0E, 0x10}},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGAT, Key: []byte("n0_key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				r, err := c.GetAndTouch(context.Background(), 1*time.Hour, []byte("n0_key"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
		{
			name: "Flush success",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpFlush, Extras: []byte{0x0, 0x0, 0x0E, 0x10}},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpFlush},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.FlushAll(context.Background(), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
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

func TestCommandGroups(t *testing.T) {
	type opDesc struct {
		name string
		code uint8
	}

	type testCase struct {
		name          string
		opsInGroup    []*opDesc
		serverSession func(op *opDesc) []*internal.Scenario
		clientSession func(t *testing.T, client *Client, op *opDesc, expectedResults []Result)
	}

	testCases := []*testCase{
		// simple operations (GET)
		{
			name: "read",
			opsInGroup: []*opDesc{
				&opDesc{name: "Get", code: internal.OpGetK},
			},
			serverSession: func(op *opDesc) []*internal.Scenario {
				return []*internal.Scenario{
					&internal.Scenario{
						Expect: &internal.Request{OpCode: op.code, Key: []byte("n1_key")},
						Respond: &internal.Response{
							StatusCode: internal.StatusNoError,
							OpCode:     op.code,
							Key:        []byte("n1_key"),
							Value:      []byte("value"),
						},
					},
				}
			},
			clientSession: func(t *testing.T, c *Client, op *opDesc, expectedResults []Result) {
				var f func(context.Context, []byte) (Result, error)
				switch op.code {
				case internal.OpGetK:
					f = c.Get
				default:
					t.Errorf("Unknown op: %+v", op)
					t.Fail()
				}

				r, err := f(context.Background(), []byte("n1_key"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
		// mutation operations (SET, ADD, REPLACE)
		{
			name: "mutation",
			opsInGroup: []*opDesc{
				&opDesc{name: "Set", code: internal.OpSet},
				&opDesc{name: "Add", code: internal.OpAdd},
				&opDesc{name: "Replace", code: internal.OpReplace},
			},
			serverSession: func(op *opDesc) []*internal.Scenario {
				return []*internal.Scenario{
					&internal.Scenario{
						Expect: &internal.Request{
							OpCode: op.code,
							Key:    []byte("n1_key"),
							Value:  []byte("n1_value"),
							Extras: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0E, 0x10},
							CAS:    1234,
						},
						Respond: &internal.Response{
							StatusCode: internal.StatusNoError,
							OpCode:     op.code,
							Key:        []byte("n1_key"),
							CAS:        1234},
					},
				}
			},
			clientSession: func(t *testing.T, c *Client, op *opDesc, expectedResults []Result) {
				var f func(context.Context, []byte, []byte, ...MutationOpt) (Result, error)
				switch op.code {
				case internal.OpSet:
					f = c.Set
				case internal.OpAdd:
					f = c.Add
				case internal.OpReplace:
					f = c.Replace
				default:
					t.Errorf("Unknown op: %+v", op)
					t.Fail()
				}

				r, err := f(context.Background(), []byte("n1_key"), []byte("n1_value"), WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
		// numeric operations (INCR, DECR)
		{
			name: "numeric",
			opsInGroup: []*opDesc{
				&opDesc{name: "Increment", code: internal.OpIncrement},
				&opDesc{name: "Decrement", code: internal.OpDecrement},
			},
			serverSession: func(op *opDesc) []*internal.Scenario {
				return []*internal.Scenario{
					&internal.Scenario{
						Expect: &internal.Request{
							OpCode: op.code,
							Key:    []byte("n1_key"),
							Extras: []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0A, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0A, 0x0, 0x0, 0x0E, 0x10},
							CAS:    1234,
						},
						Respond: &internal.Response{
							StatusCode: internal.StatusNoError,
							OpCode:     op.code,
							Key:        []byte("n1_key"),
							Value:      []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0A},
							CAS:        1234,
						},
					},
				}
			},
			clientSession: func(t *testing.T, c *Client, op *opDesc, expectedResults []Result) {
				var f func(context.Context, []byte, uint64, uint64, ...MutationOpt) (NumericResult, error)
				switch op.code {
				case internal.OpIncrement:
					f = c.Increment
				case internal.OpDecrement:
					f = c.Decrement
				default:
					t.Errorf("Unknown op: %+v", op)
					t.Fail()
				}

				r, err := f(context.Background(), []byte("n1_key"), 10, 10, WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(10), r.NumericValue())
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
		// modify operations (APPEND, PREPEND)
		{
			name: "modify",
			opsInGroup: []*opDesc{
				&opDesc{name: "Append", code: internal.OpAppend},
				&opDesc{name: "Prepend", code: internal.OpPrepend},
			},
			serverSession: func(op *opDesc) []*internal.Scenario {
				return []*internal.Scenario{
					&internal.Scenario{
						Expect: &internal.Request{
							OpCode: op.code,
							Key:    []byte("n1_key"),
							Value:  []byte("value"),
							CAS:    1234,
						},
						Respond: &internal.Response{
							StatusCode: internal.StatusNoError,
							OpCode:     op.code,
							CAS:        1234,
						},
					},
				}
			},
			clientSession: func(t *testing.T, c *Client, op *opDesc, expectedResults []Result) {
				var f func(context.Context, []byte, []byte, ...MutationOpt) (Result, error)
				switch op.code {
				case internal.OpAppend:
					f = c.Append
				case internal.OpPrepend:
					f = c.Prepend
				default:
					t.Errorf("Unknown op: %+v", op)
					t.Fail()
				}

				r, err := f(context.Background(), []byte("n1_key"), []byte("value"), WithCASValue(1234), WithExpiry(1*time.Hour))
				assert.NoError(t, err)
				assert.Equal(t, uint64(1234), r.CAS())
			},
		},
	}

	mockServer, err := internal.NewMockServer("n1")
	assert.NoError(t, err)
	defer mockServer.Shutdown()

	c, err := NewClient(WithConnector(&internal.MockConnector{}), WithNodePicker(&internal.MockNodePicker{}))
	assert.NoError(t, err)
	defer c.Close()

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			for _, opDesc := range tc.opsInGroup {
				op := opDesc
				t.Run(op.name, func(t *testing.T) {
					sess := tc.serverSession(op)
					mockServer.AddSession(sess...)
					expectedResults := toExpectedResults(sess)
					tc.clientSession(t, c, op, expectedResults)
				})
			}
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
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n2_key2"), Value: []byte("value2"), Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Value: []byte("value3"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n3_key4"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n3_key4"), Value: []byte("value4"), Opaque: 1},
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
			name: "MultiGet key not found",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusKeyNotFound, OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n2_key2"), Value: []byte("value2"), Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n3_key3"), Value: []byte("value3"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n3_key4"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n3_key4"), Value: []byte("value4"), Opaque: 1},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				rr, err := c.MultiGet(context.Background(), []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"), []byte("n3_key4"))
				assert.Error(t, err)
				assert.Len(t, rr, 4)
				assert.ElementsMatch(t, expectedResults, rr)
			},
		},
		{
			name: "MultiGet node timeout",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n2_key2"), Value: []byte("value2"), Opaque: 1},
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
		{
			name: "MultiGet no response on sentinel key",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetKQ, Key: []byte("n2_key1"), Value: []byte("value1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpGetK, Key: []byte("n2_key2"), Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("n3_key3"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpGetK, Key: []byte("n3_key3"), Value: []byte("value3"), Opaque: 0},
				},
			},

			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Millisecond)
				defer cancelFunc()

				rr, err := c.MultiGet(ctx, []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"))
				assert.Error(t, err)
				assert.Len(t, rr, 3)
				for _, r := range rr {
					k := string(r.Key())
					switch k {
					case "n2_key1":
						assert.Equal(t, []byte("value1"), r.Value())
					case "n3_key3":
						assert.Equal(t, []byte("value3"), r.Value())
					default:
						assert.Equal(t, []byte("n2_key2"), r.Key())
						assert.Error(t, r.Err())
					}
				}
			},
		},
		{
			name: "MultiDelete fan out",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpDeleteQ, Key: []byte("n2_key1"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDelete, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpDelete, Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpDeleteQ, Key: []byte("n3_key3"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDelete, Key: []byte("n3_key4"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpDelete, Opaque: 1},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.MultiDelete(context.Background(), []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"), []byte("n3_key4"))
				assert.NoError(t, err)
			},
		},
		{
			name: "MultiDelete key not found",
			server1Session: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDeleteQ, Key: []byte("n2_key1"), Opaque: 0},
					Respond: &internal.Response{StatusCode: internal.StatusKeyNotFound, OpCode: internal.OpDelete, Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDelete, Key: []byte("n2_key2"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpDelete, Opaque: 1},
				},
			},
			server2Session: []*internal.Scenario{
				&internal.Scenario{
					Expect: &internal.Request{OpCode: internal.OpDeleteQ, Key: []byte("n3_key3"), Opaque: 0},
				},
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpDelete, Key: []byte("n3_key4"), Opaque: 1},
					Respond: &internal.Response{StatusCode: internal.StatusNoError, OpCode: internal.OpDelete, Opaque: 1},
				},
			},
			clientSession: func(t *testing.T, c *Client, expectedResults []Result) {
				err := c.MultiDelete(context.Background(), []byte("n2_key1"), []byte("n2_key2"), []byte("n3_key3"), []byte("n3_key4"))
				assert.Error(t, err)
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
