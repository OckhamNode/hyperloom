package trie

import (
	"strings"

	"github.com/hyperloom/hyperloom/core"
)

// Trie orchestrates the root memory graph for Hyperloom.
type Trie struct {
	Root *Node
}

func NewTrie() *Trie {
	return &Trie{
		Root: NewNode("/"),
	}
}

// Navigate walks a UNIX-like path (e.g., "agentA/memory/state") and returns the target Node.
// Because each GetOrCreateChild step uses fine-grained locks sequentially downwards, 
// agents can traverse different branches completely in parallel safely.
func (t *Trie) Navigate(path string) *Node {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	curr := t.Root

	for _, seg := range segments {
		if seg == "" {
			continue
		}
		curr = curr.GetOrCreateChild(seg)
	}

	return curr
}

// StageDiff finds the node and applies the diff to the transaction's shadow state.
// We return the target Node pointer to the Engine so it knows what to commit.
func (t *Trie) StageDiff(diff core.HyperDiff) *Node {
	target := t.Navigate(diff.Path)

	// For OpSet we simply overwrite the shadow value.
	// For OpDel or OpAppend, we'd add complex merge logic, but keeping it lean:
	if diff.Operation == core.OpSet {
		target.ApplyShadow(diff.TxID, diff.Value)
	} else if diff.Operation == core.OpAppend {
		target.ApplySmartAppend(diff.TxID, diff.Value)
	} else if diff.Operation == core.OpDel {
		// A nil/empty raw message designates a tombstone in shadow.
		target.ApplyShadow(diff.TxID, nil)
	}

	return target
}
