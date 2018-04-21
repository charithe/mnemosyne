package memcache

import (
	"context"
	"testing"

	"github.com/charithe/mnemosyne/memcache/internal"
	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	mockServer, err := internal.NewMockServer()
	assert.NoError(t, err)
	defer mockServer.Shutdown()

	c, err := NewClient(WithConnector(&internal.MockConnector{}), WithNodePicker(NewSimpleNodePicker("node")))
	assert.NoError(t, err)
	defer c.Close()

	testCases := []struct {
		name          string
		serverSession []*internal.Scenario
		clientSession func(t *testing.T, client *Client)
	}{
		{
			name: "Single Command",
			serverSession: []*internal.Scenario{
				&internal.Scenario{
					Expect:  &internal.Request{OpCode: internal.OpGetK, Key: []byte("key")},
					Respond: &internal.Response{OpCode: internal.OpGetK, Key: []byte("key"), Value: []byte("value")},
				},
			},
			clientSession: func(t *testing.T, c *Client) {
				r, err := c.Get(context.Background(), []byte("key"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("value"), r.Value())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockServer.AddSession(tc.serverSession...)
			tc.clientSession(t, c)
		})
	}
}
