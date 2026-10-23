package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gobalance_requests_total",
			Help: "Total number of client requests received by GoBalance.",
		},
	)

	metricDialFailuresTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "gobalance_backend_dial_failures_total",
			Help: "Total number of dial failures when connecting to backends.",
		},
	)

	metricBytesTransferred = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gobalance_bytes_transferred_total",
			Help: "Total bytes proxied through GoBalance.",
		},
		[]string{"backend", "direction"},
	)

	metricActiveConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gobalance_active_connections",
			Help: "Current number of active connections per backend.",
		},
		[]string{"backend"},
	)

	metricBackendAlive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gobalance_backend_alive",
			Help: "Health status of backends (1 = healthy, 0 = dead).",
		},
		[]string{"backend"},
	)
)
