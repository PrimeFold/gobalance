package main

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	totalRequests := 10000
	concurrency := 200 // 200 concurrent workers blasting requests!

	proxyAddr := "localhost:8080"

	log.Printf("Starting flood: %d requests across %d concurrent workers -> %s\n",
		totalRequests, concurrency, proxyAddr)

	var successfulRequests int64
	var failedRequests int64

	start := time.Now()

	// Channel feeding the jobs to our workers
	jobs := make(chan int, totalRequests)

	for i := 0; i < totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup

	// Launch worker pool
	for w := 0; w < concurrency; w++ {
		
		wg.Add(1)

		go func(workerID int) {

			defer wg.Done()

			for range jobs {
				conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
				if err != nil {
					atomic.AddInt64(&failedRequests, 1)
					continue
				}

				// Send payload
				msg := "Benchmarking GoBalance Engine"
				_, err = conn.Write([]byte(msg))
				if err != nil {
					conn.Close()
					atomic.AddInt64(&failedRequests, 1)
					continue
				}

				// Read response
				buf := make([]byte, len(msg))
				_, err = conn.Read(buf)
				conn.Close()

				if err != nil {
					atomic.AddInt64(&failedRequests, 1)
				} else {
					atomic.AddInt64(&successfulRequests, 1)
				}
			}
		}(w)
	}

	wg.Wait()
	duration := time.Since(start)

	log.Println("\n================= BENCHMARK REPORT =================")
	log.Printf("Total Time:          %v\n", duration)
	log.Printf("Successful Requests: %d / %d\n", successfulRequests, totalRequests)
	log.Printf("Failed Requests:     %d\n", failedRequests)
	log.Printf("Throughput:          %.2f requests/sec\n", float64(successfulRequests)/duration.Seconds())
	log.Println("====================================================")
}
