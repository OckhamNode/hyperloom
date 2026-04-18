package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hyperloom/hyperloom/core"
)

// Agent represents an LLM simulation client connecting to Hyperloom.
type Agent struct {
	ID   string
	Conn *websocket.Conn
}

func ConnectToHyperloom(agentID string) *Agent {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ingest"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Agent %s failed to connect: %v", agentID, err)
	}

	return &Agent{ID: agentID, Conn: c}
}

func (a *Agent) SendDiff(txID, path, value string) {
	diff := core.HyperDiff{
		AgentID:   a.ID,
		TxID:      txID,
		Path:      path,
		Operation: core.OpSet,
		Value:     json.RawMessage(`"` + value + `"`),
	}

	if err := a.Conn.WriteJSON(diff); err != nil {
		log.Println("Send diff error:", err)
	}
	log.Printf("[Agent %s] Sent Uncommitted Target Change to %s -> %s\n", a.ID, path, value)
	time.Sleep(200 * time.Millisecond) // Simulate LLM token gen delay
}

func (a *Agent) Commit(txID string) {
	resp, _ := http.Get("http://localhost:8080/commit?tx_id=" + txID)
	resp.Body.Close()
	log.Printf("[Agent %s] >> COMMITTED transaction %s. Pushed to global graph.\n", a.ID, txID)
}

func (a *Agent) Revert(txID string) {
	resp, _ := http.Get("http://localhost:8080/revert?tx_id=" + txID)
	resp.Body.Close()
	log.Printf("[Agent %s] XX REVERTED transaction %s (Simulated error 400). Ghost-branch pruned.\n", a.ID, txID)
}

func startListener() {
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/subscribe", RawQuery: "path=/"}
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	go func() {
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}
			var result map[string]interface{}
			_ = json.NewDecoder(bytes.NewReader(msg)).Decode(&result)
			log.Printf("[GLOBAL LISTENER] Broadcast Received: Path=%v Value=%v Hash=%v\n", result["path"], result["value"], result["hash"])
		}
	}()
}

func main() {
	log.Println("Starting Multi-Agent Simulation...")
	startListener()
	time.Sleep(500 * time.Millisecond)

	agentA := ConnectToHyperloom("Claude")
	agentB := ConnectToHyperloom("GPT-4")

	go func() {
		// Claude completes successfully
		tx := "tx_claude_1"
		agentA.SendDiff(tx, "/memory/claude/session_1/feeling", "creative")
		agentA.SendDiff(tx, "/memory/claude/session_1/summary", "brainstormed hyperloom")
		time.Sleep(1 * time.Second)
		agentA.Commit(tx)
	}()

	go func() {
		// GPT-4 hallucinates, generates bad token, triggers rollback
		tx := "tx_gpt_1"
		agentB.SendDiff(tx, "/memory/gpt4/session_8/status", "processing")
		agentB.SendDiff(tx, "/memory/gpt4/session_8/danger", "critical failure")
		time.Sleep(500 * time.Millisecond)
		agentB.Revert(tx) // Tree branch vanishes instantly!
	}()

	time.Sleep(3 * time.Second)
	log.Println("Simulation End.")
}
