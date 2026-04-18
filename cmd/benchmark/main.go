package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	log.Println("Hyperloom Synchronous REST Benchmarker Initiated...")

	// Default massive load test
	workers := 1000      // 1000 aggressively parallel agents
	reqPerWorker := 100  // Each agent fires 100 payloads as fast as possible

	payload := []byte(`{"agent_id":"bench_agent", "path":"/benchmark/throughput", "op":"APPEND", "value": "{\"tick\":1}"}`)

	var successes int64
	var totalTime int64

	var wg sync.WaitGroup
	start := time.Now()

	// High concurrency transport to avoid TIME_WAIT local TCP exhaustion
	tr := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 10000,
	}
	client := &http.Client{Transport: tr, Timeout: 10 * time.Second}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < reqPerWorker; j++ {
				reqStart := time.Now()
				resp, err := client.Post("http://localhost:8080/write", "application/json", bytes.NewReader(payload))
				if err == nil && resp.StatusCode == 200 {
					resp.Body.Close()
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
		log.Println("Fatal: Benchmark failed to connect to http://localhost:8080. Is the broker loop active?")
		os.Exit(1)
	}

	reqPerSec := float64(succ) / duration.Seconds()
	avgLatency := time.Duration(atomic.LoadInt64(&totalTime) / succ)

	log.Println("\n--- HYPERLOOM BENCHMARK RESULTS ---")
	log.Printf("Total Requests Sent: %d\n", workers*reqPerWorker)
	log.Printf("Successful Commits: %d\n", succ)
	log.Printf("Concurrency Level: %d Parallel Agents\n", workers)
	log.Printf("Time Taken: %v\n", duration)
	log.Printf("Throughput: %.2f requests/second\n", reqPerSec)
	log.Printf("Avg Latency (Merge/Lock/Hash Commit): %v\n", avgLatency)

	fmt.Printf("\nMarkdown export:\n")
	fmt.Printf("- **Concurrency**: %d simultaneous agents\n", workers)
	fmt.Printf("- **Throughput**: %.2f writes/second\n", reqPerSec)
	fmt.Printf("- **Average Latency**: %v per commit\n", avgLatency)
}
