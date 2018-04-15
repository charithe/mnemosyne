package memcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleNodePicker(t *testing.T) {
	testCases := []struct {
		nodes        []string
		key          []byte
		expectedNode string
	}{
		{
			nodes:        []string{"n1"},
			key:          []byte("key1"),
			expectedNode: "n1",
		},
		{
			nodes:        []string{"n1", "n2", "n3"},
			key:          []byte("key60"),
			expectedNode: "n2",
		},
	}

	for _, tc := range testCases {
		np := NewSimpleNodePicker(tc.nodes...)
		node := np.Pick(tc.key)
		assert.Equal(t, tc.expectedNode, node)
	}
}
