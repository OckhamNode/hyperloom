package trie

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

// Node is the primitive building block of the Hyperloom memory Trie.
// Its design allows for pinpoint locking logic, preventing massive lock contention.
type Node struct {
	mu sync.RWMutex

	Key      string
	Value    json.RawMessage
	Children map[string]*Node

	// Hash represents the deterministic state of this node.
	Hash string

	// Shadows isolate uncommitted transaction state. Active transactions 
	// can write safely here without affecting the global committed read path.
	ShadowValues map[string]json.RawMessage
}

func NewNode(key string) *Node {
	return &Node{
		Key:          key,
		Children:     make(map[string]*Node),
		ShadowValues: make(map[string]json.RawMessage),
	}
}

// ComputeHash regenerates the node's hash based on its committed Value.
// Assumes lock is held.
func (n *Node) ComputeHash() {
	if len(n.Value) == 0 {
		n.Hash = ""
		return
	}
	hash := sha256.Sum256(n.Value)
	n.Hash = hex.EncodeToString(hash[:])
}

// ApplyShadow stages a value under a transaction ID locally.
func (n *Node) ApplyShadow(txID string, val json.RawMessage) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ShadowValues[txID] = val
}

// Commit elevates a transaction's shadow value to the global state.
func (n *Node) Commit(txID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if val, ok := n.ShadowValues[txID]; ok {
		n.Value = val
		n.ComputeHash()
		delete(n.ShadowValues, txID)
	}
}

// Revert instantly discards a transaction's shadow state. The pointer 
// remains at the last valid hash automatically.
func (n *Node) Revert(txID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.ShadowValues, txID)
}

// GetOrCreateChild ensures a safe concurrent traversal/creation.
func (n *Node) GetOrCreateChild(childKey string) *Node {
	n.mu.RLock()
	child, exists := n.Children[childKey]
	n.mu.RUnlock()

	if exists {
		return child
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	// Double-check after acquiring write lock
	if child, exists = n.Children[childKey]; exists {
		return child
	}

	newChild := NewNode(childKey)
	n.Children[childKey] = newChild
	return newChild
}

// GetCommittedValue is a highly concurrent fast-path read.
func (n *Node) GetCommittedValue() json.RawMessage {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Value
}
