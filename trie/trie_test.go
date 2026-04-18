package trie

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/hyperloom/hyperloom/core"
)

func TestSmartAppend_Array(t *testing.T) {
	node := NewNode("test")
	node.Value = json.RawMessage(`[1, 2]`)
	
	node.ApplySmartAppend("tx1", json.RawMessage(`3`))
	if string(node.ShadowValues["tx1"]) != `[1,2,3]` && string(node.ShadowValues["tx1"]) != `[1, 2, 3]` {
		t.Errorf("Expected array merge with 3, got %s", node.ShadowValues["tx1"])
	}

	node.ApplySmartAppend("tx2", json.RawMessage(`[4, 5]`))
	if string(node.ShadowValues["tx2"]) != `[1,2,4,5]` && string(node.ShadowValues["tx2"]) != `[1, 2, 4, 5]` {
		t.Errorf("Expected array merge with [4, 5], got %s", node.ShadowValues["tx2"])
	}
}

func TestSmartAppend_Object(t *testing.T) {
	node := NewNode("test")
	node.Value = json.RawMessage(`{"a": 1}`)
	
	node.ApplySmartAppend("tx1", json.RawMessage(`{"b": 2}`))
	
    val := string(node.ShadowValues["tx1"])
	if val != `{"a":1,"b":2}` && val != `{"b":2,"a":1}` {
		t.Errorf("Expected merged object, got %s", val)
	}
}

func TestConcurrentStressTrie(t *testing.T) {
	tree := NewTrie()
	var wg sync.WaitGroup

	// Fire 100,000 parallel traversals and mutations to ensure thread-safety
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			diff := core.HyperDiff{
                AgentID:   "tester",
                TxID:      fmt.Sprintf("tx_%d", id%500),
                Path:      fmt.Sprintf("/deep/tree/branch%d/node", id%100),
                Operation: core.OpSet,
                Value:     json.RawMessage(`"data"`),
            }

            // This hits GetOrCreateChild heavily in parallel
			target := tree.StageDiff(diff)
            
            // Mix writes and reads
			_ = target.GetCommittedValue()

		}(i)
	}

	wg.Wait()
}
