package main

import (
	"context"
	"log"
	"time"

	"github.com/hyperloom/hyperloom/broker"
	"github.com/hyperloom/hyperloom/core"
	"github.com/hyperloom/hyperloom/engine"
	"github.com/hyperloom/hyperloom/stream"
	"github.com/hyperloom/hyperloom/trie"
)

func main() {
	log.Println("Booting Hyperloom Event Broker...")

	// 1. Initialize the global State Graph
	memoryTree := trie.NewTrie()

	// 2. Initialize the Rollback/Commit Engine (Timeout for abandoned tx set to 2 mins)
	rollbackEngine := engine.NewProcessEngine(memoryTree, 2*time.Minute)

	// 3. Initialize the ultra-fast intake pipe with 10k event buffer capacity
	eventStream := stream.NewStreamLog(10000)

	// 4. Initialize Network Broker for Pub/Sub and Socket streaming
	pubsub := broker.NewPubSubServer(eventStream, rollbackEngine)

	// Wire the commit trigger to the Broadcast Fan-Out system
	rollbackEngine.OnCommit = pubsub.Broadcast

	// Wire debug event emitters for the Time-Travel Debugger UI
	rollbackEngine.OnStage = func(diff core.HyperDiff) {
		pubsub.BroadcastDebugEvent(broker.NewDebugEvent(
			broker.EventStaged, diff.AgentID, diff.TxID, diff.Path, string(diff.Operation), diff.Value, "",
		))
	}
	rollbackEngine.OnRevert = func(txID string, agentID string) {
		pubsub.BroadcastDebugEvent(broker.NewDebugEvent(
			broker.EventReverted, agentID, txID, "", "", nil, "",
		))
	}

	// Start Stream Consumer pointing linearly to the Engine stager
	ctx := context.Background()
	eventStream.Start(ctx, rollbackEngine.Stage)

	// Start the background Garbage Collector for dead AI agent branches (Timeout revert)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			rollbackEngine.GarbageCollect()
		}
	}()

	// Bind to massive concurrency connection port
	log.Println("Initialization absolute. Systems massive concurrency readied.")
	if err := pubsub.StartHTTP(":8080"); err != nil {
		log.Fatalf("Fatal Broker socket failure: %v", err)
	}
}
