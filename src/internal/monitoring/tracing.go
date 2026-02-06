package monitoring

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// TracingConfig contains configuration for distributed tracing
type TracingConfig struct {
	Enabled      bool
	ServiceName  string
	Environment  string
	SamplingRate float64
	Endpoint     string
}

// TracingManager manages distributed tracing
type TracingManager struct {
	config TracingConfig
	logger *zap.Logger
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config TracingConfig, logger *zap.Logger) *TracingManager {
	return &TracingManager{
		config: config,
		logger: logger,
	}
}

// Initialize initializes the tracing system with OpenTelemetry
func (tm *TracingManager) Initialize(ctx context.Context) error {
	if !tm.config.Enabled {
		tm.logger.Info("Distributed tracing is disabled")
		return nil
	}

	tm.logger.Info("Distributed tracing initialized",
		zap.String("service", tm.config.ServiceName),
		zap.String("environment", tm.config.Environment),
		zap.Float64("sampling_rate", tm.config.SamplingRate),
	)

	// OpenTelemetry initialization is handled by the infrastructure layer
	// The TracingManager provides a simplified interface for application code
	// For full OpenTelemetry integration, configure OTEL_EXPORTER_OTLP_ENDPOINT
	// and use the official OpenTelemetry Go SDK

	return nil
}

// Shutdown gracefully shuts down the tracing system
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if !tm.config.Enabled {
		return nil
	}

	tm.logger.Info("Shutting down distributed tracing")

	// Flush any pending spans - in production this would call
	// tracerProvider.Shutdown(ctx) to ensure all spans are exported

	return nil
}

// StartSpan starts a trace span and returns a context with the span and an end function
func (tm *TracingManager) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if !tm.config.Enabled {
		return ctx, func() {}
	}

	startTime := time.Now()
	spanID := fmt.Sprintf("span-%d", time.Now().UnixNano())
	
	// Store span info in context
	spanCtx := context.WithValue(ctx, spanContextKey, &spanInfo{
		name:      name,
		spanID:    spanID,
		startTime: startTime,
	})

	tm.logger.Debug("Trace span started", 
		zap.String("name", name),
		zap.String("span_id", spanID),
	)

	return spanCtx, func() {
		duration := time.Since(startTime)
		tm.logger.Debug("Trace span ended", 
			zap.String("name", name),
			zap.String("span_id", spanID),
			zap.Duration("duration", duration),
		)
	}
}

// spanContextKey is the key for span info in context
type spanContextKeyType struct{}
var spanContextKey = spanContextKeyType{}

// spanInfo holds span information
type spanInfo struct {
	name      string
	spanID    string
	startTime time.Time
	attrs     map[string]interface{}
}

// AddSpanAttribute adds an attribute to the current span
func (tm *TracingManager) AddSpanAttribute(ctx context.Context, key string, value interface{}) {
	if !tm.config.Enabled {
		return
	}

	if info, ok := ctx.Value(spanContextKey).(*spanInfo); ok {
		if info.attrs == nil {
			info.attrs = make(map[string]interface{})
		}
		info.attrs[key] = value
	}

	tm.logger.Debug("Span attribute added",
		zap.String("key", key),
		zap.Any("value", value),
	)
}

// RecordError records an error in the current span
func (tm *TracingManager) RecordError(ctx context.Context, err error) {
	if !tm.config.Enabled || err == nil {
		return
	}

	if info, ok := ctx.Value(spanContextKey).(*spanInfo); ok {
		tm.logger.Error("Span error recorded",
			zap.String("span_name", info.name),
			zap.String("span_id", info.spanID),
			zap.Error(err),
		)
	} else {
		tm.logger.Error("Span error recorded", zap.Error(err))
	}
}

// InjectTraceContext injects trace context into HTTP headers or other carriers
func (tm *TracingManager) InjectTraceContext(ctx context.Context, carrier interface{}) error {
	if !tm.config.Enabled {
		return nil
	}

	// Extract span info from context and inject into carrier
	if info, ok := ctx.Value(spanContextKey).(*spanInfo); ok {
		if headers, ok := carrier.(map[string]string); ok {
			headers["X-Trace-ID"] = tm.GetTraceID(ctx)
			headers["X-Span-ID"] = info.spanID
		}
	}

	return nil
}

// ExtractTraceContext extracts trace context from HTTP headers or other carriers
func (tm *TracingManager) ExtractTraceContext(ctx context.Context, carrier interface{}) (context.Context, error) {
	if !tm.config.Enabled {
		return ctx, nil
	}

	// Extract trace info from carrier and add to context
	if headers, ok := carrier.(map[string]string); ok {
		if traceID, exists := headers["X-Trace-ID"]; exists {
			ctx = context.WithValue(ctx, traceIDContextKey, traceID)
		}
		if spanID, exists := headers["X-Span-ID"]; exists {
			ctx = context.WithValue(ctx, parentSpanIDContextKey, spanID)
		}
	}

	return ctx, nil
}

// Context keys for trace propagation
type traceIDContextKeyType struct{}
type parentSpanIDContextKeyType struct{}

var traceIDContextKey = traceIDContextKeyType{}
var parentSpanIDContextKey = parentSpanIDContextKeyType{}

// GetTraceID returns the trace ID from the context
func (tm *TracingManager) GetTraceID(ctx context.Context) string {
	if !tm.config.Enabled {
		return ""
	}

	// Check if trace ID is already in context
	if traceID, ok := ctx.Value(traceIDContextKey).(string); ok {
		return traceID
	}

	// Generate a new trace ID based on service name and timestamp
	return fmt.Sprintf("%s-trace-%d", tm.config.ServiceName, time.Now().UnixNano())
}

// GetSpanID returns the span ID from the context
func (tm *TracingManager) GetSpanID(ctx context.Context) string {
	if !tm.config.Enabled {
		return ""
	}

	// Get span ID from current span info
	if info, ok := ctx.Value(spanContextKey).(*spanInfo); ok {
		return info.spanID
	}

	return fmt.Sprintf("span-%d", time.Now().UnixNano())
}
