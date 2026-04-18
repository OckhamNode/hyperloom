package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hyperloom/hyperloom/core"
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func sendResponse(id interface{}, result interface{}) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	b, _ := json.Marshal(resp)
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
}

func main() {
	// Must log strictly to Stderr since Claude intercepts Stdout for JSON-RPC
	log.SetOutput(os.Stderr)
	log.Println("[Hyperloom MCP] Bridge started. Awaiting JSON-RPC payloads...")

	scanner := bufio.NewScanner(os.Stdin)
	// Some payloads can be large, increase capacity
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Println("Invalid JSON input:", err)
			continue
		}

		switch req.Method {
		case "initialize":
			sendResponse(req.ID, map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "hyperloom-bridge",
					"version": "1.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
			})
		case "notifications/initialized":
			log.Println("[Hyperloom MCP] Claude initialization acknowledged.")
		case "tools/list":
			sendResponse(req.ID, map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "read_global_memory",
						"description": "Reads complex JSON structural state instantly from the global Hyperloom broker across any path.",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path": map[string]interface{}{
									"type":        "string",
									"description": "The UNIX-like path mapping into the state tree (e.g., /global/context_1)",
								},
							},
							"required": []string{"path"},
						},
					},
					{
						"name":        "write_global_memory",
						"description": "Safely deep merges new JSON objects or appends logic perfectly into existing tree data.",
						"inputSchema": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path": map[string]interface{}{
									"type":        "string",
									"description": "The target path in the state graph.",
								},
								"value": map[string]interface{}{
									"type":        "string",
									"description": "A valid stringified JSON object or Array. E.g. '{\"a\": 1}' or '[1, 2]'",
								},
							},
							"required": []string{"path", "value"},
						},
					},
				},
			})
		case "tools/call":
			var params struct {
				Name      string            `json:"name"`
				Arguments map[string]string `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)

			log.Printf("[Hyperloom MCP] Executing Native Tool: %s -> %s\n", params.Name, params.Arguments["path"])

			if params.Name == "read_global_memory" {
				resp, err := http.Get("http://localhost:8080/read?path=" + params.Arguments["path"])
				if err != nil {
					sendResponse(req.ID, map[string]interface{}{"isError": true, "content": []map[string]string{{"type": "text", "text": "Broker unreachable: " + err.Error()}}})
					continue
				}
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				sendResponse(req.ID, map[string]interface{}{
					"content": []map[string]string{{"type": "text", "text": string(body)}},
				})
			} else if params.Name == "write_global_memory" {
				diff := core.HyperDiff{
					AgentID:   "mcp-claude-agent",
					Path:      params.Arguments["path"],
					Operation: core.OpAppend,
					Value:     json.RawMessage(params.Arguments["value"]),
				}
				b, _ := json.Marshal(diff)

				resp, err := http.Post("http://localhost:8080/write", "application/json", bytes.NewReader(b))
				if err != nil {
					sendResponse(req.ID, map[string]interface{}{"isError": true, "content": []map[string]string{{"type": "text", "text": "Broker Write Exception: " + err.Error()}}})
					continue
				}
				defer resp.Body.Close()
				sendResponse(req.ID, map[string]interface{}{
					"content": []map[string]string{{"type": "text", "text": "Transaction OK - Memory Mutated Globally"}},
				})
			} else {
				sendResponse(req.ID, map[string]interface{}{"isError": true, "content": []map[string]string{{"type": "text", "text": "Unknown tool"}}})
			}
		}
	}
}
