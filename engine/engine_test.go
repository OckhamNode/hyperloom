package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperloom/hyperloom/core"
	"github.com/hyperloom/hyperloom/trie"
)

func TestEngine_TransactionCommit(t *testing.T) {
	mem := trie.NewTrie()
	eng := NewProcessEngine(mem, 5*time.Second)

	diff := core.HyperDiff{
		AgentID:   "agent1",
		TxID:      "tx1",
		Path:      "/memory/test",
		Operation: core.OpSet,
		Value:     json.RawMessage(`"hello"`),
	}

	// 1. Stage the diff
	eng.Stage(diff)

	// Verify it's not publicly committed
	node := mem.Navigate("/memory/test")
	if len(node.GetCommittedValue()) != 0 {
		t.Errorf("Should not be committed yet")
	}

	// 2. Commit transaction
	eng.Commit("tx1")

	// Verify publicly committed
	if string(node.GetCommittedValue()) != `"hello"` {
		t.Errorf("Value not committed")
	}

	eng.mu.RLock()
	_, active := eng.ActiveTxs["tx1"]
	eng.mu.RUnlock()
	if active {
		t.Errorf("Tx should be removed from active pool")
	}
}

func TestEngine_GarbageCollectionRevert(t *testing.T) {
	mem := trie.NewTrie()
	eng := NewProcessEngine(mem, 1*time.Millisecond)

	diff := core.HyperDiff{
		TxID:      "tx2",
		Path:      "/memory/gc",
		Operation: core.OpSet,
		Value:     json.RawMessage(`"fail"`),
	}
	eng.Stage(diff)

	time.Sleep(5 * time.Millisecond)

	// Simulate GC tick after timeout
	eng.GarbageCollect()

	node := mem.Navigate("/memory/gc")
	if len(node.GetCommittedValue()) != 0 {
		t.Errorf("Should not be committed")
	}

	// Shadow shouldn't exist anymore
	if node.HasShadow("tx2") {
		t.Errorf("Ghost branch was not pruned by GC")
	}
}
