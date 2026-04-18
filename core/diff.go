package core

import "encoding/json"

// OperationType defines the supported structural operations on the Trie.
type OperationType string

const (
	OpSet    OperationType = "SET"    // Create or explicitly overwrite
	OpDel    OperationType = "DEL"    // Delete node and sub-branch
	OpAppend OperationType = "APPEND" // Append to slice (if array) or string text
)

// HyperDiff is our custom, hyper-efficient patch format.
// It allows decoupled AI agents to send isolated context updates.
type HyperDiff struct {
	// AgentID identifies the agent producing the context patch.
	AgentID string `json:"agent_id"`

	// TxID bounds a set of patches to a singular atomic transaction which can be rolled back.
	TxID string `json:"tx_id"`

	// Path is a UNIX-like slash-separated pointer into the global contextual memory.
	// e.g., "/session_84/memory/working/entity_extraction"
	Path string `json:"path"`

	// Operation designates how the Trie node should merge this value.
	Operation OperationType `json:"op"`

	// Value contains the opaque raw byte state to be merged.
	Value json.RawMessage `json:"value,omitempty"`
}
