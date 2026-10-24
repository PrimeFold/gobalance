package main

import (
	"context"
	"log"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL        *url.URL
	Alive      bool
	ActiveConn int64
	mux        sync.RWMutex
}

type ServerPool struct {
	backend       []*Backend
	current       uint64
	mux           sync.RWMutex
	totalRequests int64
	failedDials   int64
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()

	b.Alive = alive
	if alive {
		metricBackendAlive.WithLabelValues(b.URL.Host).Set(1) 
	} else {
		metricBackendAlive.WithLabelValues(b.URL.Host).Set(0)
	}
}

func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()

	return b.Alive
}

func (s *ServerPool) IncrementTotalRequests() {
	atomic.AddInt64(&s.totalRequests, 1)
	metricRequestsTotal.Inc()
}

func (s *ServerPool) IncrementFailedDials() {
	atomic.AddInt64(&s.failedDials, 1)
	metricDialFailuresTotal.Inc()
}

func (b *Backend) GetActiveConnections() int64 {
	return atomic.LoadInt64(&b.ActiveConn)
}

func (b *Backend) IncrementConnectionCount() {
	atomic.AddInt64(&b.ActiveConn, 1)
	metricActiveConnections.WithLabelValues(b.URL.Host).Inc()
}

func (b *Backend) DecrementConnectionCount() {
	atomic.AddInt64(&b.ActiveConn, -1)
	metricActiveConnections.WithLabelValues(b.URL.Host).Dec()
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.backend = append(s.backend, b)
	if b.Alive {
		metricBackendAlive.WithLabelValues(b.URL.Host).Set(1)
	} else {
		metricBackendAlive.WithLabelValues(b.URL.Host).Set(0)
	}
	metricActiveConnections.WithLabelValues(b.URL.Host).Set(0)
}

func (s *ServerPool) GetLeastConnectedBackend() *Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	var minConnectionBackend *Backend
	var minConn int64

	for _, b := range s.backend {
		if !b.IsAlive() {
			continue
		}
		activeConns := b.GetActiveConnections()

		if minConnectionBackend == nil || activeConns < minConn {
			minConnectionBackend = b
			minConn = activeConns
		}
	}

	return minConnectionBackend
}

func (s *ServerPool) HealthCheck() {
	s.mux.RLock()
	backends := make([]*Backend, len(s.backend))
	copy(backends, s.backend)
	s.mux.RUnlock()

	var wg sync.WaitGroup

	for _, b := range backends {
		wg.Add(1)

		go func(target *Backend) {
			defer wg.Done()

			conn, err := net.DialTimeout("tcp", target.URL.Host, 500*time.Millisecond)
			if err != nil {
				if target.IsAlive() {
					log.Printf("[HEALTH CHECK] Server %s died!\n", target.URL.Host)
					target.SetAlive(false)
				}
				return
			}

			conn.Close()
			if !target.IsAlive() {
				log.Printf("[HEALTH CHECK] Server %s is BACK ALIVE! 🎉\n", target.URL.Host)
				target.SetAlive(true)
			}
		}(b)
	}

	wg.Wait()
}

func (s *ServerPool) StartHealthCheckDaemon(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping health check...")
			return
		case <-ticker.C:
			s.HealthCheck()
		}
	}
}
