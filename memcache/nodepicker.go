package memcache

import (
	"github.com/dchest/siphash"
	jump "github.com/dgryski/go-jump"
)

const (
	k0 = 0x0706050403020100
	k1 = 0x0f0e0d0c0b0a0908
)

// NodePicker allows to define different strategies for picking a node for a key
type NodePicker interface {
	Pick(key []byte) string
}

// SimpleNodePicker implements a simplistic node picker algorithm.
// It assumes that the list of nodes is static and that the order of the nodes does not change between invocations
type SimpleNodePicker struct {
	nodeIDs []string
	n       int
}

func NewSimpleNodePicker(nodeIDs ...string) *SimpleNodePicker {
	return &SimpleNodePicker{
		nodeIDs: nodeIDs,
		n:       len(nodeIDs),
	}
}

func (snp *SimpleNodePicker) Pick(key []byte) string {
	h := siphash.Hash(k0, k1, key)
	i := jump.Hash(h, snp.n)
	return snp.nodeIDs[i]
}
