package monitoring

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware creates a middleware that records HTTP metrics
func MetricsMiddleware(metrics *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Increment in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method

		metrics.RecordHTTPRequest(method, path, status, duration)
	}
}
