# Monitoring Package

This package provides comprehensive monitoring and observability for the AI Research Platform using Prometheus metrics and OpenTelemetry tracing preparation.

## Features

### Prometheus Metrics

The monitoring package exposes the following metrics:

#### HTTP Metrics
- `http_request_duration_seconds` - Histogram of HTTP request durations
- `http_requests_total` - Counter of total HTTP requests
- `http_requests_in_flight` - Gauge of currently processing requests

#### LLM Provider Metrics
- `llm_requests_total` - Counter of LLM requests by provider, model, and status
- `llm_request_duration_seconds` - Histogram of LLM request durations
- `llm_request_errors_total` - Counter of LLM errors by type
- `llm_provider_status` - Gauge of provider availability (1=available, 0=unavailable)

#### Research Session Metrics
- `research_sessions_active` - Gauge of active research sessions
- `research_sessions_total` - Counter of total research sessions by status
- `research_session_duration_seconds` - Histogram of research session durations
- `research_tasks_total` - Counter of research tasks by type and status
- `research_task_duration_seconds` - Histogram of research task durations

#### Database Metrics
- `db_connections_active` - Gauge of active database connections
- `db_query_duration_seconds` - Histogram of database query durations
- `db_queries_total` - Counter of database queries by operation and status

#### Cache Metrics
- `cache_hits_total` - Counter of cache hits by tier (L1/L2)
- `cache_misses_total` - Counter of cache misses by tier
- `cache_size_bytes` - Gauge of cache size in bytes

### Distributed Tracing

The package includes preparation hooks for OpenTelemetry distributed tracing:

- `TracingManager` - Manages tracing lifecycle
- `StartSpan` - Creates trace spans (placeholder for OpenTelemetry)
- `AddSpanAttribute` - Adds attributes to spans
- `RecordError` - Records errors in spans
- `InjectTraceContext` / `ExtractTraceContext` - Context propagation

## Usage

### Initialize Metrics

```go
import "github.com/ai-research-platform/internal/monitoring"

// Create metrics instance
metrics := monitoring.NewMetrics()
```

### Add Metrics Middleware

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/ai-research-platform/internal/monitoring"
)

router := gin.New()
router.Use(monitoring.MetricsMiddleware(metrics))
```

### Expose Metrics Endpoint

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/gin-gonic/gin"
)

router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### Record Custom Metrics

```go
// Record HTTP request
metrics.RecordHTTPRequest("GET", "/api/users", "200", duration)

// Record LLM request
metrics.RecordLLMRequest("deepseek", "deepseek-chat", "success", duration)

// Record research session
metrics.RecordResearchSession("completed", duration)

// Update active sessions
metrics.ResearchSessionsActive.Inc()
defer metrics.ResearchSessionsActive.Dec()

// Record database query
metrics.RecordDBQuery("select", "success", duration)

// Record cache operations
metrics.RecordCacheHit("L1")
metrics.RecordCacheMiss("L2")
```

### Initialize Tracing

```go
import (
    "github.com/ai-research-platform/internal/monitoring"
    "go.uber.org/zap"
)

logger, _ := zap.NewProduction()

tracingConfig := monitoring.TracingConfig{
    Enabled:      true,
    ServiceName:  "ai-research-platform",
    Environment:  "production",
    SamplingRate: 0.1,
    Endpoint:     "http://otel-collector:4318",
}

tm := monitoring.NewTracingManager(tracingConfig, logger)
err := tm.Initialize(ctx)
if err != nil {
    log.Fatal(err)
}
defer tm.Shutdown(ctx)
```

### Use Tracing

```go
// Start a span
ctx, endSpan := tm.StartSpan(ctx, "operation-name")
defer endSpan()

// Add attributes
tm.AddSpanAttribute(ctx, "user_id", userID)
tm.AddSpanAttribute(ctx, "request_size", size)

// Record errors
if err != nil {
    tm.RecordError(ctx, err)
}

// Get trace information
traceID := tm.GetTraceID(ctx)
spanID := tm.GetSpanID(ctx)
```

## Integration with Router

Update your router setup to include monitoring:

```go
func SetupRouter(
    db *gorm.DB,
    chatService *service.ChatService,
    researchService *service.ResearchService,
    llmScheduler *eino.LLMScheduler,
    cacheManager cache.Cache,
    config RouterConfig,
    logger *zap.Logger,
    metrics *monitoring.Metrics,  // Add metrics parameter
) *gin.Engine {
    router := gin.New()
    
    // Add metrics middleware
    if metrics != nil {
        router.Use(monitoring.MetricsMiddleware(metrics))
    }
    
    // Expose metrics endpoint
    router.GET("/metrics", gin.WrapH(promhttp.Handler()))
    
    // ... rest of router setup
    
    return router
}
```

## Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'ai-research-platform'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## Kubernetes Annotations

For Kubernetes deployments, add these annotations to your pod spec:

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
  prometheus.io/path: "/metrics"
```

## Testing

Run the monitoring tests:

```bash
# Run all monitoring tests
go test ./internal/monitoring/...

# Run specific test
go test ./internal/monitoring/... -run TestMetricsEndpointExposed

# Run with coverage
go test ./internal/monitoring/... -cover
```

## Future Enhancements

### OpenTelemetry Integration

The tracing system is currently a preparation layer. To fully integrate OpenTelemetry:

1. Add OpenTelemetry SDK dependencies:
   ```bash
   go get go.opentelemetry.io/otel
   go get go.opentelemetry.io/otel/exporters/otlp/otlptrace
   go get go.opentelemetry.io/otel/sdk/trace
   ```

2. Update `TracingManager.Initialize()` to create actual tracer provider
3. Implement real span creation in `StartSpan()`
4. Add context propagation for distributed tracing

### Additional Metrics

Consider adding:
- Memory usage metrics
- Goroutine count metrics
- Custom business metrics
- SLO/SLI metrics

### Grafana Dashboards

Create Grafana dashboards for:
- HTTP request rates and latencies
- LLM provider performance
- Research session analytics
- System resource utilization

## Dependencies

- `github.com/prometheus/client_golang` - Prometheus client library
- `go.uber.org/zap` - Structured logging
- `github.com/gin-gonic/gin` - HTTP framework

## Requirements Validation

This implementation validates:
- **Requirement 4.4**: Performance monitoring with metrics exposure
- **Requirement 4.5**: Resource usage tracking and observability
