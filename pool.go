package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct{
	URL *url.URL
	Alive bool
	ActiveConn int64
	mux sync.RWMutex
}

type ServerPool struct{
	backend []*Backend
	current uint64
	mux sync.RWMutex
	totalRequests int64 //Counter (only increments )
	failedDials int64 //Counter ( only increments )
}



func (b *Backend) SetAlive(alive bool){
	b.mux.Lock()
	defer b.mux.Unlock()

	b.Alive = alive
}


func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()

	return b.Alive

}

func (s *ServerPool) IncrementTotalRequests(){
	atomic.AddInt64(&s.totalRequests,1)
}

func(s *ServerPool) IncrementFailedDials(){
	atomic.AddInt64(&s.failedDials,1)
}

func (b *Backend) GetActiveConnections() int64{
	b.mux.RLock()
	defer b.mux.RUnlock()

	activeConn:= atomic.LoadInt64(&b.ActiveConn)
	return activeConn

}
func (b *Backend) IncrementConnectionCount(){
	atomic.AddInt64(&b.ActiveConn,1)
}

func (b *Backend) DecrementConnectionCount(){
	atomic.AddInt64(&b.ActiveConn,-1)
}


func (s *ServerPool) AddBackend(b *Backend){
	s.mux.Lock()
	defer s.mux.Unlock()

	s.backend = append(s.backend, b)
}



func (s *ServerPool) GetLeastConnectedBackend() *Backend{
	s.mux.RLock()
	defer s.mux.RUnlock()

	var minConnectionBackend *Backend
	var minConn int64

	for _,b:= range s.backend{
		if !b.IsAlive(){
			log.Printf("Dead Server found : %s",b.URL.Host)
			continue
		}
		activeConns:= b.GetActiveConnections()
		
		if minConnectionBackend == nil || activeConns < minConn{
			minConnectionBackend = b
			minConn = activeConns
		}
	}

	return minConnectionBackend
}

func (s *ServerPool) HealthCheck(){
	s.mux.RLock()
	backends:= make([]*Backend,len(s.backend))

	copy(backends,s.backend)

	s.mux.RUnlock()

	var wg sync.WaitGroup

	for _,b:= range backends{
		wg.Add(1)

		go func (target *Backend)  {
			defer wg.Done()

			conn,err:= net.DialTimeout("tcp",target.URL.Host,500*time.Millisecond)
			if err != nil{
				// Failed: Mark dead
				if target.IsAlive() {
					log.Printf("[HEALTH CHECK] Server %s died!\n", target.URL.Host)
					target.SetAlive(false)
				}
				return
			} 

			conn.Close()
			//Mark it Alive !..
			if !target.IsAlive(){
				log.Printf("[HEALTH CHECK] Server %s is BACK ALIVE! 🎉\n", target.URL.Host)
				target.SetAlive(true)
			}

		}(b)
	}

	wg.Wait()

}


func (s *ServerPool) StartHealthCheckDaemon(ctx context.Context){
	ticker:= time.NewTicker(5 * time.Second)

	defer ticker.Stop()

	for{
		select{
		case <- ctx.Done():
			log.Println("Stopping health check...")
			return
		case <- ticker.C:
			s.HealthCheck()
		}
	}

}



func (s *ServerPool) ServeMetrics(w http.ResponseWriter, r *http.Request) {
	var aliveVal int64
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// 1. Total Requests Counter
	fmt.Fprintf(w, "# TYPE gobalance_requests_total counter\n")
	fmt.Fprintf(w, "gobalance_requests_total %d\n\n", atomic.LoadInt64(&s.totalRequests))

	// 2. Failed Dials Counter
	fmt.Fprintf(w, "# TYPE gobalance_failed_dials_total counter\n")
	fmt.Fprintf(w, "gobalance_failed_dials_total %d\n\n", atomic.LoadInt64(&s.failedDials))

	s.mux.RLock()
	defer s.mux.RUnlock()

	// 3. Active Connections Gauge per backend
	fmt.Fprintf(w, "# TYPE gobalance_active_connections gauge\n")
	for _, b := range s.backend {
		fmt.Fprintf(w, "gobalance_active_connections{backend=\"%s\"} %d\n", b.URL.Host, b.GetActiveConnections())
	}

	// 4. Backend Health Gauge (1 = Alive, 0 = Dead)
	fmt.Fprintf(w, "\n# TYPE gobalance_backend_alive gauge\n")
	for _, b := range s.backend {
		if b.IsAlive() {
			aliveVal = 1
		}
		fmt.Fprintf(w, "gobalance_backend_alive{backend=\"%s\"} %d\n", b.URL.Host, aliveVal)
	}
}
