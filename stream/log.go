package stream

import (
	"context"
	"sync"

	"github.com/hyperloom/hyperloom/core"
)

// StreamLog represents an ultra-fast in-memory append-only log.
// It acts as the intake pipe for all HyperDiffs before they hit the Trie.
// In a tier-1 system, this would be backed by an mmap'ed file or RocksDB.
type StreamLog struct {
	mu     sync.RWMutex
	events []core.HyperDiff
	ingest chan core.HyperDiff
}

// NewStreamLog initializes the log with a buffered channel to prevent blocking on bursts.
func NewStreamLog(bufferSize int) *StreamLog {
	return &StreamLog{
		events: make([]core.HyperDiff, 0),
		ingest: make(chan core.HyperDiff, bufferSize),
	}
}

// Push appends an event to the stream log non-blockingly (up to bufferSize).
func (s *StreamLog) Push(diff core.HyperDiff) {
	s.ingest <- diff
}

// Start opens the ingestion valve. It reads from the channel and persists
// to our append-only event array, then pushes to the dispatcher (Trie).
func (s *StreamLog) Start(ctx context.Context, dispatcher func(core.HyperDiff)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case diff := <-s.ingest:
				s.mu.Lock()
				s.events = append(s.events, diff)
				s.mu.Unlock()

				// Dispatch to the Trie engine for structural update
				if dispatcher != nil {
					dispatcher(diff)
				}
			}
		}
	}()
}
