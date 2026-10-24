# GoBalance: High-Throughput Layer 4 TCP Reverse Proxy

A concurrent, low-latency Layer 4 TCP load balancer built from scratch in Go, designed to open-source systems engineering standards.

## 🚀 Performance & Benchmark Highlights
* **Throughput:** 22,353.18 requests/second on a single laptop.
* **Concurrency:** 10,000 requests sustained across 200 concurrent workers with 0 errors (100% success rate).
* **Heap Footprint:** ~18.8 MB flat heap allocation under load via zero-allocation buffer pooling (`sync.Pool`).
* **Concurrency Safety:** Certified with ThreadSanitizer (`go test -race`).

BenchmarkStandardCopy-6          4306077       270.2 ns/op      1120 B/op     3 allocs/op
BenchmarkPooledCopy-6           47481747        26.02 ns/op         0 B/op     0 allocs/op

## 🛠 Architectural Features

### 1. Zero-Allocation Socket Streaming (Layer 4)
* Full-duplex bidirectional byte streaming bridging incoming client sockets to backend servers using pre-allocated, recycled 32KB buffers (`sync.Pool` + `io.CopyBuffer`).
* Proper TCP Half-Close state handling using `*net.TCPConn.CloseWrite()` to avoid premature socket termination on `EOF`.

### 2. Lock-Free Atomic Routing Engine
* **Least-Connections Load Balancing:** Routes traffic dynamically to the healthiest node with the lowest active workload.
* **Hardware-Level Atomics:** Uses `sync/atomic` (`AddInt64`, `LoadInt64`, `AddUint64`) for connection counters and modulo-based Round-Robin, eliminating lock contention on hot paths.
* **Thread-Safe Registry:** Uses `sync.RWMutex` for dynamic registration and hot-reloading of backend pools.

### 3. Fault Tolerance & Passive Failure Detection
* Wrapped connections in `context.WithTimeout` and `net.Dialer.DialContext` (1s timeout).
* Passive detection of failed nodes with automatic retry failover across healthy nodes without severing client sockets.

### 4. Active Health Probing Daemon
* Background health daemon using `time.NewTicker` and parallel fan-out goroutines to periodically test dead nodes and revive them upon recovery.

### 5. Graceful Drain & Signal Trapping
* Traps OS signals (`SIGINT`, `SIGTERM`).
* Closes listener to stop new ingress, signals health daemons, and drains active in-flight client streams via `sync.WaitGroup` with a 15-second timeout fallback.

### 6. Observability & Telemetry
* **Prometheus Metrics (`:9090/metrics`):** Exposes real-time plain-text metrics (`gobalance_requests_total`, `gobalance_failed_dials_total`, `gobalance_active_connections`, `gobalance_backend_alive`).
* **Live Profiling (`:6060/debug/pprof`):** Exposes internal runtime CPU flamegraphs and heap snapshots.