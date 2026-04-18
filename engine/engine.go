package engine

import (
	"sync"
	"time"

	"github.com/hyperloom/hyperloom/core"
	"github.com/hyperloom/hyperloom/trie"
)

// Transaction tracks all touched nodes for a specific atomic sequence of agent diffs.
type Transaction struct {
	ID        string
	AgentID   string
	StartTime time.Time
	// Nodes touched by this transaction's shadow edits.
	Nodes []*trie.Node
}

// Sub Engine orchestrates atomic commits and rollbacks by cleaning up shadow state.
type ProcessEngine struct {
	mu        sync.RWMutex
	ActiveTxs map[string]*Transaction
	Memory    *trie.Trie
	Timeout   time.Duration

	// PubSub trigger function to fan-out commits
	OnCommit func(path string, node *trie.Node)
}

func NewProcessEngine(memory *trie.Trie, timeout time.Duration) *ProcessEngine {
	return &ProcessEngine{
		ActiveTxs: make(map[string]*Transaction),
		Memory:    memory,
		Timeout:   timeout,
	}
}

// Stage orchestrates passing the diff to the Memory tree and recording the node track for rollback.
func (e *ProcessEngine) Stage(diff core.HyperDiff) {
	e.mu.Lock()
	tx, exists := e.ActiveTxs[diff.TxID]
	if !exists {
		tx = &Transaction{
			ID:        diff.TxID,
			AgentID:   diff.AgentID,
			StartTime: time.Now(),
			Nodes:     make([]*trie.Node, 0),
		}
		e.ActiveTxs[diff.TxID] = tx
	}
	e.mu.Unlock()

	// 1. Stage in the Trie
	node := e.Memory.StageDiff(diff)

	// 2. Track node for this Tx
	e.mu.Lock()
	tx.Nodes = append(tx.Nodes, node)
	e.mu.Unlock()
}

// Commit finalizes the transaction. Fast, parallel safe.
func (e *ProcessEngine) Commit(txID string) {
	e.mu.Lock()
	tx, exists := e.ActiveTxs[txID]
	if exists {
		delete(e.ActiveTxs, txID)
	}
	e.mu.Unlock()

	if !exists {
		return
	}

	for _, node := range tx.Nodes {
		node.Commit(txID)
		
		// Inform Pub/Sub system, if bound.
		if e.OnCommit != nil {
			e.OnCommit(node.Key, node)
		}
	}
}

// Revert instantly drops the shadow branches, abandoning the changes.
func (e *ProcessEngine) Revert(txID string) {
	e.mu.Lock()
	tx, exists := e.ActiveTxs[txID]
	if exists {
		delete(e.ActiveTxs, txID)
	}
	e.mu.Unlock()

	if !exists {
		return
	}

	for _, node := range tx.Nodes {
		node.Revert(txID)
	}
}

// GarbageCollect scans for timed out active transactions and ruthlessly reverts them.
func (e *ProcessEngine) GarbageCollect() {
	e.mu.RLock()
	now := time.Now()
	var toRevert []string
	for id, tx := range e.ActiveTxs {
		if now.Sub(tx.StartTime) > e.Timeout {
			toRevert = append(toRevert, id)
		}
	}
	e.mu.RUnlock()

	for _, id := range toRevert {
		e.Revert(id)
	}
}
