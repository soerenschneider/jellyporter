package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	namespace      = "jellyporter"
	subsystemMedia = "media"
)

var (
	Version = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "version",
		Help:      "Version information",
	}, []string{"version", "go_version"})

	Heartbeat = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "heartbeat_timestamp",
		Help:      "Heartbeat",
	})

	TotalItems = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemMedia,
		Name:      "items_total",
		Help:      "Total number of items",
	}, []string{"server", "type"})

	TotalItemsTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemMedia,
		Name:      "items_fetched_timestamp_seconds",
		Help:      "Timestamp when fetched number of items",
	}, []string{"server", "type"})

	ItemsUpdatedUserData = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystemMedia,
		Name:      "items_updated_userdata_total",
		Help:      "Total number of movies with updated UserData found",
	}, []string{"server", "type"})

	RequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "total",
		Help:      "Total amount of requests",
	})

	EventSourceRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "events",
		Name:      "requests_total",
		Help:      "Total amount of requests",
	}, []string{"source"})

	EventSourceCooldownPhases = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "events",
		Name:      "cooldown_phases_total",
		Help:      "Total amount of cooldown phases because of too frequent requests from event sources",
	})

	EventSourceErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "events",
		Name:      "request_errors_total",
		Help:      "Errors while sending requests to jellyfin",
	}, []string{"source"})

	RequestErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "errors_total",
		Help:      "Errors while sending requests to jellyfin",
	}, []string{"error", "path"})

	RequestTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "requests",
		Name:      "time_total",
		Buckets:   []float64{0.75, 0.9, 0.95, 0.99},
	}, []string{"path", "code"})

	DbQueriesTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "database",
		Name:      "queries_time_total",
		Buckets:   []float64{0.75, 0.9, 0.95, 0.99},
	}, []string{"query"})

	DbQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "database",
		Name:      "query_errors_total",
	}, []string{"query"})
)
