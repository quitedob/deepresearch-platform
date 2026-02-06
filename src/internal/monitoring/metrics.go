package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application
type Metrics struct {
	// HTTP metrics
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestTotal     *prometheus.CounterVec
	HTTPRequestsInFlight prometheus.Gauge

	// LLM provider metrics
	LLMRequestTotal    *prometheus.CounterVec
	LLMRequestDuration *prometheus.HistogramVec
	LLMRequestErrors   *prometheus.CounterVec
	LLMProviderStatus  *prometheus.GaugeVec

	// Research session metrics
	ResearchSessionsActive  prometheus.Gauge
	ResearchSessionsTotal   *prometheus.CounterVec
	ResearchSessionDuration *prometheus.HistogramVec
	ResearchTasksTotal      *prometheus.CounterVec
	ResearchTaskDuration    *prometheus.HistogramVec

	// Database metrics
	DBConnectionsActive prometheus.Gauge
	DBQueryDuration     *prometheus.HistogramVec
	DBQueryTotal        *prometheus.CounterVec

	// Cache metrics
	CacheHits   *prometheus.CounterVec
	CacheMisses *prometheus.CounterVec
	CacheSize   *prometheus.GaugeVec
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// HTTP metrics
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestsInFlight: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),

		// LLM provider metrics
		LLMRequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_requests_total",
				Help: "Total number of LLM requests",
			},
			[]string{"provider", "model", "status"},
		),
		LLMRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "llm_request_duration_seconds",
				Help:    "LLM request duration in seconds",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"provider", "model"},
		),
		LLMRequestErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "llm_request_errors_total",
				Help: "Total number of LLM request errors",
			},
			[]string{"provider", "model", "error_type"},
		),
		LLMProviderStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "llm_provider_status",
				Help: "LLM provider status (1=available, 0=unavailable)",
			},
			[]string{"provider"},
		),

		// Research session metrics
		ResearchSessionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "research_sessions_active",
				Help: "Number of active research sessions",
			},
		),
		ResearchSessionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "research_sessions_total",
				Help: "Total number of research sessions",
			},
			[]string{"status"},
		),
		ResearchSessionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "research_session_duration_seconds",
				Help:    "Research session duration in seconds",
				Buckets: []float64{10, 30, 60, 120, 300, 600, 1800, 3600},
			},
			[]string{"status"},
		),
		ResearchTasksTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "research_tasks_total",
				Help: "Total number of research tasks",
			},
			[]string{"task_type", "status"},
		),
		ResearchTaskDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "research_task_duration_seconds",
				Help:    "Research task duration in seconds",
				Buckets: []float64{0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"task_type"},
		),

		// Database metrics
		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
			[]string{"operation"},
		),
		DBQueryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"operation", "status"},
		),

		// Cache metrics
		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_tier"},
		),
		CacheMisses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_tier"},
		),
		CacheSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cache_size_bytes",
				Help: "Current cache size in bytes",
			},
			[]string{"cache_tier"},
		),
	}
}

// RecordHTTPRequest records metrics for an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration) {
	m.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(duration.Seconds())
	m.HTTPRequestTotal.WithLabelValues(method, path, status).Inc()
}

// RecordLLMRequest records metrics for an LLM request
func (m *Metrics) RecordLLMRequest(provider, model, status string, duration time.Duration) {
	m.LLMRequestTotal.WithLabelValues(provider, model, status).Inc()
	m.LLMRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

// RecordLLMError records an LLM error
func (m *Metrics) RecordLLMError(provider, model, errorType string) {
	m.LLMRequestErrors.WithLabelValues(provider, model, errorType).Inc()
}

// SetLLMProviderStatus sets the status of an LLM provider
func (m *Metrics) SetLLMProviderStatus(provider string, available bool) {
	status := 0.0
	if available {
		status = 1.0
	}
	m.LLMProviderStatus.WithLabelValues(provider).Set(status)
}

// RecordResearchSession records metrics for a research session
func (m *Metrics) RecordResearchSession(status string, duration time.Duration) {
	m.ResearchSessionsTotal.WithLabelValues(status).Inc()
	m.ResearchSessionDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordResearchTask records metrics for a research task
func (m *Metrics) RecordResearchTask(taskType, status string, duration time.Duration) {
	m.ResearchTasksTotal.WithLabelValues(taskType, status).Inc()
	m.ResearchTaskDuration.WithLabelValues(taskType).Observe(duration.Seconds())
}

// RecordDBQuery records metrics for a database query
func (m *Metrics) RecordDBQuery(operation, status string, duration time.Duration) {
	m.DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
	m.DBQueryTotal.WithLabelValues(operation, status).Inc()
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(tier string) {
	m.CacheHits.WithLabelValues(tier).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(tier string) {
	m.CacheMisses.WithLabelValues(tier).Inc()
}

// SetCacheSize sets the current cache size
func (m *Metrics) SetCacheSize(tier string, sizeBytes float64) {
	m.CacheSize.WithLabelValues(tier).Set(sizeBytes)
}
