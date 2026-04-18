package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type BenchmarkProfile struct {
	Name        string
	Method      string // "GET", "POST", or "POST+REVERT"
	URLPath     string
	PayloadFunc func(workerID int, reqIdx int) []byte
}

func runProfile(p BenchmarkProfile, workers, reqPerWorker int, tr *http.Transport) {
	var successes int64
	var totalTime int64
	var wg sync.WaitGroup

	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}
	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < reqPerWorker; j++ {
				reqStart := time.Now()

				var resp *http.Response
				var err error

				if p.Method == "POST" {
					payload := p.PayloadFunc(workerID, j)
					resp, err = client.Post("http://localhost:8080"+p.URLPath, "application/json", bytes.NewReader(payload))
				} else if p.Method == "GET" {
					resp, err = client.Get("http://localhost:8080" + p.URLPath)
				} else if p.Method == "POST+REVERT" {
					tx := fmt.Sprintf("hx_%d_%d", workerID, j)
					payload := []byte(fmt.Sprintf(`{"tx_id":"%s", "agent_id":"bench", "path":"/conflict", "op":"APPEND", "value": "1"}`, tx))
					resp, err = client.Post("http://localhost:8080/write", "application/json", bytes.NewReader(payload))
					if err == nil && resp.StatusCode == 200 {
						resp.Body.Close()
						// Trigger the immediate Rollback PRUNE
						revResp, rErr := client.Get("http://localhost:8080/revert?tx_id=" + tx)
						if rErr == nil {
							revResp.Body.Close()
						}
					}
				}

				if err == nil && resp != nil && resp.StatusCode == 200 {
					if p.Method != "POST+REVERT" {
						resp.Body.Close()
					}
					atomic.AddInt64(&successes, 1)
				}
				atomic.AddInt64(&totalTime, int64(time.Since(reqStart)))
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	succ := atomic.LoadInt64(&successes)
	if succ == 0 {
		log.Printf("[%s] Fatal: Failed to connect to local broker.\n", p.Name)
		return
	}

	reqPerSec := float64(succ) / duration.Seconds()
	avgLatency := time.Duration(atomic.LoadInt64(&totalTime) / succ)

	log.Printf("---- Profile: %s ----\n", p.Name)
	log.Printf("Throughput: %.2f req/sec\n", reqPerSec)
	log.Printf("Avg Latency: %v\n\n", avgLatency)

	fmt.Printf("| %s | %.2f req/s | %v |\n", p.Name, reqPerSec, avgLatency)
}

func main() {
	log.Println("Starting Advanced Tri-Matrix Benchmark...")

	tr := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
	}

	workers := 500
	requests := 50

	fmt.Println("\n## Real-World Multi-Agent Profile Run")
	fmt.Println("| Benchmark Profile | Sustained Throughput | Avg Latency |")
	fmt.Println("|---|---|---|")

	// P1: Read-Heavy Swarm
	runProfile(BenchmarkProfile{
		Name:   "Read-Heavy Swarm (90% Read)",
		Method: "GET",
		URLPath: fmt.Sprintf("/read?path=/global/mem/1"),
	}, workers, requests, tr)

	// P2: Write-Heavy Conflict 
	runProfile(BenchmarkProfile{
		Name:    "Write-Heavy Conflict (100% Writes)",
		Method:  "POST",
		URLPath: "/write",
		PayloadFunc: func(w, r int) []byte {
			return []byte(`{"agent_id":"wbench", "path":"/hot/target", "op":"APPEND", "value":"[1]"}`)
		},
	}, workers, requests, tr)

	// P3: Hallucination Rollback
	runProfile(BenchmarkProfile{
		Name:   "Hallucination Trim (Ghost Branch Pruning)",
		Method: "POST+REVERT",
	}, workers, requests, tr)
}
