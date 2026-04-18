package broker

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/hyperloom/hyperloom/core"
	"github.com/hyperloom/hyperloom/engine"
	"github.com/hyperloom/hyperloom/stream"
	"github.com/hyperloom/hyperloom/trie"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// PubSubServer spins up the lightweight WebSocket broker.
type PubSubServer struct {
	Stream *stream.StreamLog
	Engine *engine.ProcessEngine

	mu      sync.RWMutex
	Clients map[*websocket.Conn]string // Conn -> Subscribed Path (e.g. "/agent1")
}

func NewPubSubServer(s *stream.StreamLog, e *engine.ProcessEngine) *PubSubServer {
	return &PubSubServer{
		Stream:  s,
		Engine:  e,
		Clients: make(map[*websocket.Conn]string),
	}
}

// HandleIngest is the WebSocket firehose where agents stream HyperDiff operations.
func (ps *PubSubServer) HandleIngest(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ingest Upgrade failed:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// Soft disconnect, transactions time out inherently in the engine.
			break
		}

		var diff core.HyperDiff
		if err := json.Unmarshal(msg, &diff); err != nil {
			log.Println("Invalid HyperDiff payload:", err)
			continue
		}

		// Instantly pipe to in-memory log
		ps.Stream.Push(diff)
	}
}

// HandleSubscribe lets agents attach to a specific sub-tree path and listen for commits.
func (ps *PubSubServer) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	ps.mu.Lock()
	ps.Clients[conn] = path
	ps.mu.Unlock()

	defer func() {
		ps.mu.Lock()
		delete(ps.Clients, conn)
		ps.mu.Unlock()
		conn.Close()
	}()

	// Block to keep conn alive.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Broadcast fans-out pushed State changes exclusively to interested listeners.
func (ps *PubSubServer) Broadcast(path string, node *trie.Node) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	nodeVal := node.GetCommittedValue()

	for conn, subPath := range ps.Clients {
		if strings.HasPrefix(path, subPath) {
			payload := map[string]interface{}{
				"path":  path,
				"value": json.RawMessage(nodeVal),
				"hash":  node.Hash,
			}
			data, _ := json.Marshal(payload)
			_ = conn.WriteMessage(websocket.TextMessage, data)
		}
	}
}

// StartHTTP hooks the endpoints and begins serving.
func (ps *PubSubServer) StartHTTP(addr string) error {
	http.HandleFunc("/ingest", ps.HandleIngest)
	http.HandleFunc("/subscribe", ps.HandleSubscribe)

	// In a complete system, commits could be via TCP packet or REST. We use REST.
	http.HandleFunc("/commit", func(w http.ResponseWriter, r *http.Request) {
		txID := r.URL.Query().Get("tx_id")
		if txID != "" {
			ps.Engine.Commit(txID)
			w.WriteHeader(http.StatusOK)
		}
	})

	// Explcit Revert trigger for agents throwing 400 errors.
	http.HandleFunc("/revert", func(w http.ResponseWriter, r *http.Request) {
		txID := r.URL.Query().Get("tx_id")
		if txID != "" {
			ps.Engine.Revert(txID)
			w.WriteHeader(http.StatusOK)
		}
	})

	log.Printf("Hyperloom Broker active on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}
