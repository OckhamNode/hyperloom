package trie

import (
	"bytes"
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

// ApplySmartAppend intelligently merges objects or appends to arrays
func (n *Node) ApplySmartAppend(txID string, val json.RawMessage) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Base state is mapped from shadow if exists, otherwise committed value
	baseState := n.Value
	if shadow, ok := n.ShadowValues[txID]; ok {
		baseState = shadow
	}

	if len(baseState) == 0 || string(baseState) == "null" {
		n.ShadowValues[txID] = val
		return
	}

	baseStr := bytes.TrimSpace(baseState)
	valStr := bytes.TrimSpace(val)

	// JSON Object Merge
	if len(baseStr) > 0 && baseStr[0] == '{' && len(valStr) > 0 && valStr[0] == '{' {
		var baseMap map[string]interface{}
		var valMap map[string]interface{}
		if err := json.Unmarshal(baseState, &baseMap); err == nil {
			if err := json.Unmarshal(val, &valMap); err == nil {
				for k, v := range valMap {
					baseMap[k] = v // Overwrite/Merge keys
				}
				if merged, err := json.Marshal(baseMap); err == nil {
					n.ShadowValues[txID] = merged
					return
				}
			}
		}
	}

	// JSON Array Append
	if len(baseStr) > 0 && baseStr[0] == '[' {
		var baseArr []interface{}
		if err := json.Unmarshal(baseState, &baseArr); err == nil {
			if len(valStr) > 0 && valStr[0] == '[' {
				var valArr []interface{}
				if err := json.Unmarshal(val, &valArr); err == nil {
					baseArr = append(baseArr, valArr...)
				}
			} else {
				var singleVal interface{}
				if err := json.Unmarshal(val, &singleVal); err == nil {
					baseArr = append(baseArr, singleVal)
				}
			}
			if merged, err := json.Marshal(baseArr); err == nil {
				n.ShadowValues[txID] = merged
				return
			}
		}
	}

	// Fallback: Overwrite
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

// HasShadow safely checks if a transaction is still actively staged on this node.
func (n *Node) HasShadow(txID string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, exists := n.ShadowValues[txID]
	return exists
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
