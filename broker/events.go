package broker

import (
	"encoding/json"
	"time"
)

// EventType classifies the lifecycle stage of a Trie mutation.
type EventType string

const (
	EventStaged    EventType = "staged"
	EventCommitted EventType = "committed"
	EventReverted  EventType = "reverted"
)

// DebugEvent is the rich payload streamed to the Time-Travel Debugger UI.
type DebugEvent struct {
	Type      EventType       `json:"type"`
	AgentID   string          `json:"agent_id"`
	TxID      string          `json:"tx_id"`
	Path      string          `json:"path"`
	Op        string          `json:"op"`
	Value     json.RawMessage `json:"value,omitempty"`
	Hash      string          `json:"hash,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

func NewDebugEvent(eventType EventType, agentID, txID, path, op string, value json.RawMessage, hash string) DebugEvent {
	return DebugEvent{
		Type:      eventType,
		AgentID:   agentID,
		TxID:      txID,
		Path:      path,
		Op:        op,
		Value:     value,
		Hash:      hash,
		Timestamp: time.Now().UnixMilli(),
	}
}
