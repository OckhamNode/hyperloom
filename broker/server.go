package broker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

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

	debugMu      sync.RWMutex
	DebugClients map[*websocket.Conn]bool
}

func NewPubSubServer(s *stream.StreamLog, e *engine.ProcessEngine) *PubSubServer {
	return &PubSubServer{
		Stream:       s,
		Engine:       e,
		Clients:      make(map[*websocket.Conn]string),
		DebugClients: make(map[*websocket.Conn]bool),
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

// HandleDebugEvents connects the Time-Travel Debugger UI via WebSocket.
func (ps *PubSubServer) HandleDebugEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Debug WS upgrade failed:", err)
		return
	}

	ps.debugMu.Lock()
	ps.DebugClients[conn] = true
	ps.debugMu.Unlock()

	defer func() {
		ps.debugMu.Lock()
		delete(ps.DebugClients, conn)
		ps.debugMu.Unlock()
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// BroadcastDebugEvent fans out a lifecycle event to all debugger UI clients.
func (ps *PubSubServer) BroadcastDebugEvent(evt DebugEvent) {
	ps.debugMu.RLock()
	defer ps.debugMu.RUnlock()

	data, _ := json.Marshal(evt)
	for conn := range ps.DebugClients {
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}
}

// StartHTTP hooks the endpoints and begins serving.
func (ps *PubSubServer) StartHTTP(addr string) error {
	http.HandleFunc("/ingest", ps.HandleIngest)
	http.HandleFunc("/subscribe", ps.HandleSubscribe)
	http.HandleFunc("/events", ps.HandleDebugEvents)

	// In a complete system, commits could be via TCP packet or REST. We use REST.
	http.HandleFunc("/commit", func(w http.ResponseWriter, r *http.Request) {
		txID := r.URL.Query().Get("tx_id")
		if txID != "" {
			ps.Engine.Commit(txID)
			w.WriteHeader(http.StatusOK)
		}
	})

	// Add REST endpoint for MCP direct reads
	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			path = "/"
		}
		node := ps.Engine.Memory.Navigate(path)
		w.Header().Set("Content-Type", "application/json")
		val := node.GetCommittedValue()
		if len(val) == 0 {
			val = []byte(`null`)
		}
		w.Write(val)
	})

	// Add REST endpoint for MCP synchronous writes
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		var diff core.HyperDiff
		if err := json.NewDecoder(r.Body).Decode(&diff); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if diff.TxID == "" {
			diff.TxID = fmt.Sprintf("mcp_tx_%d", time.Now().UnixNano())
		}

		// Synchronously apply rather than queueing in Stream
		// because MCP tool calls expect immediate return state.
		ps.Engine.Stage(diff)
		ps.Engine.Commit(diff.TxID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"committed"}`))
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
