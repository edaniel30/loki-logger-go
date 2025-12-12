package middleware

import (
	"github.com/edaniel30/loki-logger-go"
	"github.com/edaniel30/loki-logger-go/models"
	"time"

	"github.com/gin-gonic/gin"
)

// getTraceID extracts trace ID from common headers.
// Tries multiple header formats in order of preference.
func getTraceID(c *gin.Context) string {
	// Try X-Trace-Id (common custom header)
	if traceID := c.GetHeader("X-Trace-Id"); traceID != "" {
		return traceID
	}

	// Try X-Request-Id (common alternative)
	if traceID := c.GetHeader("X-Request-Id"); traceID != "" {
		return traceID
	}

	// Try traceparent (W3C Trace Context)
	if traceparent := c.GetHeader("traceparent"); traceparent != "" {
		return traceparent
	}

	// Try from Gin context (if set by another middleware)
	if traceID := c.GetString("trace_id"); traceID != "" {
		return traceID
	}

	return ""
}

// GinLogger returns a Gin middleware that logs HTTP requests.
// Automatically indexes trace_id as a label in Loki if present in the request.
func GinLogger(logger *loki.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Extract trace ID from headers
		traceID := getTraceID(c)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Create logger with trace_id label if present
		currentLogger := logger
		if traceID != "" {
			currentLogger = logger.WithFields(models.Fields{
				"trace_id": traceID, // This will be added as a label for indexing
			})
		}

		// Build fields
		fields := models.Fields{
			"method":            c.Request.Method,
			"path":              path,
			"query":             raw,
			"status":            statusCode,
			"latency_ms":        latency.Milliseconds(),
			"client_ip":         c.ClientIP(),
			"user_agent":        c.Request.UserAgent(),
			"_skip_stack_trace": true, // Skip stack trace for HTTP errors
		}

		// Add trace_id as field too (for log content)
		if traceID != "" {
			fields["trace_id"] = traceID
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Get request context
		ctx := c.Request.Context()

		// Determine log level based on status code and log
		if statusCode >= 500 {
			currentLogger.Error(ctx, "HTTP Request", fields)
		} else if statusCode >= 400 {
			currentLogger.Warn(ctx, "HTTP Request", fields)
		} else {
			currentLogger.Info(ctx, "HTTP Request", fields)
		}
	}
}

// GinRecovery returns a Gin middleware that recovers from panics and logs them.
// Automatically indexes trace_id as a label in Loki if present in the request.
func GinRecovery(logger *loki.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Extract trace ID
				traceID := getTraceID(c)

				// Create logger with trace_id label if present
				currentLogger := logger
				if traceID != "" {
					currentLogger = logger.WithFields(models.Fields{
						"trace_id": traceID,
					})
				}

				// Build error fields
				fields := models.Fields{
					"error":             err,
					"path":              c.Request.URL.Path,
					"method":            c.Request.Method,
					"client_ip":         c.ClientIP(),
					"_skip_stack_trace": true, // Skip stack trace for HTTP panics
				}

				// Add trace_id as field too
				if traceID != "" {
					fields["trace_id"] = traceID
				}

				// Get request context
				ctx := c.Request.Context()
				currentLogger.Error(ctx, "Panic recovered", fields)

				// Abort with internal server error
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}

// GinLoggerConfig defines configuration for the Gin logger middleware.
type GinLoggerConfig struct {
	// Logger is the loki logger instance
	Logger *loki.Logger

	// SkipPaths is a list of paths to skip logging
	SkipPaths []string

	// SkipPathsWithPrefix is a list of path prefixes to skip logging
	SkipPathsWithPrefix []string
}

// GinLoggerWithConfig returns a Gin middleware with custom configuration.
func GinLoggerWithConfig(config GinLoggerConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Skip logging for specified paths
		if skipPaths[path] {
			c.Next()
			return
		}

		// Skip logging for paths with specified prefixes
		for _, prefix := range config.SkipPathsWithPrefix {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				c.Next()
				return
			}
		}

		start := time.Now()
		raw := c.Request.URL.RawQuery

		// Extract trace ID from headers
		traceID := getTraceID(c)

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Create logger with trace_id label if present
		currentLogger := config.Logger
		if traceID != "" {
			currentLogger = config.Logger.WithFields(models.Fields{
				"trace_id": traceID,
			})
		}

		// Build fields
		fields := models.Fields{
			"method":            c.Request.Method,
			"path":              path,
			"query":             raw,
			"status":            statusCode,
			"latency_ms":        latency.Milliseconds(),
			"client_ip":         c.ClientIP(),
			"user_agent":        c.Request.UserAgent(),
			"_skip_stack_trace": true, // Skip stack trace for HTTP errors
		}

		// Add trace_id as field too (for log content)
		if traceID != "" {
			fields["trace_id"] = traceID
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Get request context
		ctx := c.Request.Context()

		// Determine log level based on status code and log
		if statusCode >= 500 {
			currentLogger.Error(ctx, "HTTP Request", fields)
		} else if statusCode >= 400 {
			currentLogger.Warn(ctx, "HTTP Request", fields)
		} else {
			currentLogger.Info(ctx, "HTTP Request", fields)
		}
	}
}
