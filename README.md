# Loki Logger Go

A powerful, flexible, and easy-to-use logging library for Go, designed to integrate seamlessly with Grafana Loki.

## Installation

```bash
go get github.com/edaniel30/loki-logger-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "loki-logger-go"
)

func main() {
    // Create a new logger
    logger, err := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("my-app"),
        loki.WithLokiHost("http://localhost:3100"),
        loki.WithLogLevel(loki.LevelInfo),
    )

    if err != nil {
        panic(err)
    }
    defer logger.Close()

    ctx := context.Background()

    // Log messages
    logger.Info(ctx, "Application started", loki.Fields{
        "version": "1.0.0",
        "port":    8080,
    })

    logger.Error(ctx, "Failed to connect to database", loki.Fields{
        "error": "connection timeout",
        "host":  "localhost:5432",
    })
}
```

### Using Functional Options

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithLokiHost("http://localhost:3100"),
    loki.WithLogLevel(loki.LevelDebug),
    loki.WithLabels(map[string]string{
        "environment": "production",
        "region":      "us-east-1",
    }),
    loki.WithBatchSize(50),
    loki.WithFlushInterval(3 * time.Second),
)
```

### Loki Authentication

If your Loki instance requires authentication, you can provide credentials:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithLokiHost("http://loki.example.com:3100"),
    loki.WithLokiUsername("your_username"),
    loki.WithLokiPassword("your_password"),
)
```

### Using Environment Variables

For better security, load credentials from environment variables:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName(os.Getenv("APP_NAME")),
    loki.WithLokiHost(os.Getenv("LOKI_HOST")),
    loki.WithLokiUsername(os.Getenv("LOKI_USERNAME")),
    loki.WithLokiPassword(os.Getenv("LOKI_PASSWORD")),
)
```

## Gin Middleware

Automatically log all HTTP requests with built-in Gin middleware:

```go
package main

import (
    "loki-logger-go"
    "loki-logger-go/middleware"
    "github.com/gin-gonic/gin"
)

func main() {
    logger, _ := loki.New(
        loki.DefaultConfig(),
        loki.WithAppName("my-api"),
        loki.WithLokiHost("http://localhost:3100"),
    )
    defer logger.Close()

    r := gin.New()

    // Add Loki middlewares
    r.Use(middleware.GinRecovery(logger))
    r.Use(middleware.GinLogger(logger))

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "pong"})
    })

    r.Run(":8080")
}
```

### Advanced Middleware Configuration

```go
r.Use(middleware.GinLoggerWithConfig(middleware.GinLoggerConfig{
    Logger:              logger,
    SkipPaths:           []string{"/health", "/metrics"},
    SkipPathsWithPrefix: []string{"/internal", "/debug"},
}))
```

## Distributed Tracing

The Gin middleware **automatically extracts and indexes trace IDs** from HTTP requests for distributed tracing:

### Automatic Trace ID Indexing

The middleware looks for trace IDs in these headers:
- `X-Trace-Id`
- `X-Request-Id`
- `traceparent` (W3C Trace Context)

When found, the `trace_id` is:
- ✅ **Indexed as a label** in Loki (for fast queries)
- ✅ **Added to all logs** in that request automatically
- ✅ **No configuration needed** - works out of the box!

### Query Logs by Trace ID

```logql
# Find all logs for a specific trace
{app="my-api", trace_id="550e8400-e29b-41d4-a716-446655440000"}

# Find errors in a specific trace
{app="my-api", trace_id="abc-123", level="error"}
```

### Example Usage

```go
// Client sends request with trace ID
curl -H "X-Trace-Id: my-trace-123" http://localhost:8080/api/users

// All logs in that request will have trace_id="my-trace-123" indexed!
```

For complete distributed tracing setup across microservices, see [TRACING.md](./TRACING.md).

## Configuration Options

### Config Structure

```go
type Config struct {
    AppName           string              // Application name (required)
    LokiHost          string              // Loki server URL
    LokiUsername      string              // Loki username for basic auth (optional)
    LokiPassword      string              // Loki password for basic auth (optional)
    LogLevel          Level               // Minimum log level
    Labels            map[string]string   // Default labels for all logs
    BatchSize         int                 // Number of logs to batch
    FlushInterval     time.Duration       // How often to flush logs
    MaxRetries        int                 // Max retry attempts
    Timeout           time.Duration       // HTTP request timeout
    WriteTimeout      time.Duration       // Individual log write timeout (default: 5s)
    FlushTimeout      time.Duration       // Flush operation timeout (default: 10s)
    ShutdownTimeout   time.Duration       // Graceful shutdown timeout (default: 10s)
    OnlyConsole       bool                // Only log to console (disable Loki)
    IncludeStackTrace bool                // Auto stack traces for errors (default: true)
}
```

### Available Options

- `WithAppName(name string)` - Set application name
- `WithLokiHost(host string)` - Set Loki server URL
- `WithLokiUsername(username string)` - Set Loki username for basic auth
- `WithLokiPassword(password string)` - Set Loki password for basic auth
- `WithLogLevel(level Level)` - Set minimum log level
- `WithLabels(labels map[string]string)` - Set default labels
- `WithBatchSize(size int)` - Set batch size
- `WithFlushInterval(interval time.Duration)` - Set flush interval
- `WithMaxRetries(retries int)` - Set max retry attempts
- `WithTimeout(timeout time.Duration)` - Set HTTP timeout
- `WithConsole(enabled bool)` - Enable/disable console output
- `WithWriteTimeout(timeout time.Duration)` - Set timeout for individual log write operations (default: 5s)
- `WithFlushTimeout(timeout time.Duration)` - Set timeout for flush operations (default: 10s)
- `WithShutdownTimeout(timeout time.Duration)` - Set timeout for graceful shutdown (default: 10s)

### Timeout Configuration

The logger provides granular control over various timeout operations:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    // Individual log write timeout
    loki.WithWriteTimeout(5*time.Second),
    // Flush operation timeout
    loki.WithFlushTimeout(10*time.Second),
    // Graceful shutdown timeout
    loki.WithShutdownTimeout(15*time.Second),
    // HTTP request timeout to Loki
    loki.WithTimeout(10*time.Second),
)
```

**Timeout Types:**

- **WriteTimeout** (default: 5s) - Maximum time to wait when writing individual log entries. If the context doesn't already have a deadline, this timeout is applied automatically.

- **FlushTimeout** (default: 10s) - Maximum time to wait when flushing buffered logs to Loki. Used by:
  - Background flusher periodic flushes
  - `Flush()` method (can be overridden with `FlushContext()`)
  - Final flush during shutdown

- **ShutdownTimeout** (default: 10s) - Maximum time to wait for graceful shutdown when calling `Close()`. Includes time for:
  - Stopping the background flusher
  - Final flush to send remaining logs
  - Closing transports

- **Timeout** (default: 10s) - HTTP client timeout for requests to Loki server

**Example with context override:**

```go
// Use default timeouts from config
logger.Info(ctx, "Regular log")

// Override timeout using context
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
logger.Info(ctx, "Log with custom timeout")  // Uses 2s instead of default WriteTimeout

// Override flush timeout
flushCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
logger.FlushContext(flushCtx)  // Uses 30s instead of default FlushTimeout
```

## Log Levels

The library supports the following log levels (from lowest to highest):

- `LevelDebug` - Detailed information for diagnosing problems
- `LevelInfo` - General informational messages
- `LevelWarn` - Warning messages
- `LevelError` - Error messages
- `LevelFatal` - Critical errors

## Structured Logging

Add structured fields to your logs:

```go
ctx := context.Background()

logger.Info(ctx, "User logged in", loki.Fields{
    "user_id":  123,
    "username": "john_doe",
    "ip":       "192.168.1.1",
    "duration": 150,
})
```

## Logger with Fields

Create a logger with additional context fields that are added as labels:

```go
// Create a child logger with additional labels
userLogger := logger.WithFields(loki.Fields{
    "component": "user_handler",  // Added as label
    "version":   "v2",             // Added as label
})

ctx := context.Background()
userLogger.Info(ctx, "Processing request")
// All logs from userLogger will have component="user_handler" and version="v2" labels
```

### Important: String Values Only

`WithFields` only accepts **string values** for labels. Non-string values trigger a warning:

```go
// ✅ CORRECT - String values
logger.WithFields(loki.Fields{
    "environment": "production",  // OK
    "region":      "us-east-1",   // OK
})

// ❌ INCORRECT - Non-string values
logger.WithFields(loki.Fields{
    "user_id": 12345,           // Warning! Ignored
    "active":  true,            // Warning! Ignored
    "count":   100,             // Warning! Ignored
})
// Output: [loki-logger] transport error [logger]: WithFields: field 'user_id' has non-string value...

// ✅ SOLUTION - Use Fields parameter in log methods for non-string values
logger.Info(ctx, "User action", loki.Fields{
    "user_id": 12345,     // OK - fields can be any type
    "active":  true,      // OK
    "count":   100,       // OK
})
```

### Labels vs Fields

- **WithFields**: Creates labels (low cardinality, indexed in Loki)
- **Fields parameter**: Creates fields (high cardinality, searchable but not indexed)

```go
// Labels for filtering (use WithFields)
serviceLogger := logger.WithFields(loki.Fields{
    "service":     "api",
    "environment": "prod",
})

// Fields for data (use Fields parameter in log methods)
serviceLogger.Info(ctx, "Request processed", loki.Fields{
    "user_id":      12345,
    "request_id":   "abc-123",
    "duration_ms":  150,
    "status_code":  200,
})
```

## Context Support

The logger requires Go's `context.Context` as the first parameter for all logging methods, enabling proper cancellation, deadline handling, and distributed tracing support:

### Context-Aware Logging

All logging methods require a `context.Context` as the first parameter:

```go
ctx := context.Background()

// All logging methods require context
logger.Info(ctx, "Simple log")
logger.Error(ctx, "Error occurred", loki.Fields{
    "error": err.Error(),
})
```

Available logging methods:
- `Debug(ctx, message, fields...)`
- `Info(ctx, message, fields...)`
- `Warn(ctx, message, fields...)`
- `Error(ctx, message, fields...)`
- `Fatal(ctx, message, fields...)`
- `Log(ctx, level, message, fields...)`

### Context with Timeout

The logger respects context deadlines and timeouts:

```go
// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

// Log will respect the timeout
logger.Info(ctx, "Operation completed", loki.Fields{
    "duration": time.Since(start),
})
```

### Context with Cancellation

```go
ctx, cancel := context.WithCancel(context.Background())

// Cancel context when needed
go func() {
    time.Sleep(1 * time.Second)
    cancel()
}()

// Log respects cancellation
logger.Info(ctx, "Processing...", loki.Fields{
    "status": "in_progress",
})
```

### Flush and Close with Context

Control shutdown behavior with context:

```go
// Flush with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := logger.FlushContext(ctx); err != nil {
    log.Printf("Failed to flush: %v", err)
}

// Close with custom timeout
if err := logger.CloseContext(ctx); err != nil {
    log.Printf("Failed to close: %v", err)
}
```

### Graceful Shutdown

The logger implements graceful shutdown with automatic timeout:

```go
logger, _ := loki.New(loki.DefaultConfig(), loki.WithAppName("my-app"))

// Application code...
logger.Info(ctx, "Processing...")

// On shutdown (e.g., signal handler)
if err := logger.Close(); err != nil {
    // Close waits up to 10 seconds for:
    // 1. Background flusher to stop
    // 2. Final flush to Loki to complete
    log.Printf("Warning: shutdown timeout: %v", err)
}
```

**Shutdown behavior:**
- Close() signals the background flusher to stop
- Background flusher performs a final flush (5s timeout)
- Close() waits up to 10 seconds total for clean shutdown
- If timeout exceeded, returns error but doesn't leak goroutines
- All buffered logs are sent before shutdown (unless timeout exceeded)

## Error Handling

The logger provides configurable error handling for transport failures (e.g., network errors when sending to Loki):

### Default Error Handler

By default, transport errors are written to `stderr`:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
)
// Transport errors automatically written to stderr:
// [loki-logger] transport error [loki]: connection timeout
```

### Custom Error Handler

You can provide a custom error handler to integrate with your monitoring system:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithErrorHandler(func(transportName string, err error) {
        // Send to your monitoring system
        metrics.IncrementCounter("logger_transport_errors", map[string]string{
            "transport": transportName,
        })

        // Or log to another logger
        log.Printf("Transport %s failed: %v", transportName, err)
    }),
)
```

### Silent Error Handler

To suppress transport errors completely (not recommended for production):

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithErrorHandler(func(transportName string, err error) {
        // Do nothing - silently ignore errors
    }),
)
```

### Important Notes

- Transport errors are **non-blocking** - logging continues even if a transport fails
- Errors are reported for both `Write()` operations and `Flush()`/`Close()` operations
- Console transport errors are rare (typically only I/O errors)
- Loki transport errors are more common (network issues, authentication failures, etc.)

## Labels and Indexing in Loki

### What are Labels?

In Loki, **labels** are indexed metadata that allow you to efficiently query and filter logs. Unlike fields (which are part of the log content), labels are used by Loki to organize log streams.

### Default Labels

The library automatically adds these labels to all logs:

- `app` - Your application name
- `level` - The log level (debug, info, warn, error, fatal)

### Querying by Level in Grafana

Since `level` is now a label, you can efficiently filter logs by level in Grafana:

```logql
# Get all error logs
{app="my-app", level="error"}

# Get all logs except debug
{app="my-app", level!="debug"}

# Get all warn and error logs
{app="my-app", level=~"warn|error"}

# Combine with other filters
{app="my-app", environment="production", level="error"}
```

### Adding Custom Labels

You can add custom labels through configuration:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithLabels(map[string]string{
        "environment": "production",
        "region":      "us-east-1",
        "cluster":     "k8s-prod-01",
    }),
)
```

These custom labels will also be indexed and queryable:

```logql
{app="my-app", environment="production", region="us-east-1"}
```

### Labels vs Fields

**Labels** (indexed):
- Small cardinality (app, level, environment)
- Used for filtering and organizing streams
- Impact Loki performance and storage

**Fields** (not indexed):
- High cardinality data (user IDs, IPs, timestamps)
- Searchable but slower
- Part of the log content

```go
// Labels (indexed)
logger.WithLabels(map[string]string{
    "environment": "prod",  // ✓ Good label
})

// Fields (not indexed)
logger.Info("User action", loki.Fields{
    "user_id": 12345,       // ✓ Good field
    "ip":      "1.2.3.4",   // ✓ Good field
    "action":  "login",     // ✓ Good field
})
```

### Label Cardinality Validation

⚠️ **Important**: Keep label cardinality low. The logger automatically tracks and warns about high-cardinality labels.

#### Automatic Cardinality Tracking

By default, the logger tracks unique values per label and warns when a label exceeds 10 unique values:

```go
logger, _ := loki.New(
    loki.DefaultConfig(),  // MaxLabelCardinality = 10
    loki.WithAppName("my-app"),
)

// This will trigger a warning after 10 unique user IDs
for i := 0; i < 15; i++ {
    userLogger := logger.WithFields(loki.Fields{
        "user_id": fmt.Sprintf("user-%d", i),  // High cardinality!
    })
    userLogger.Info(ctx, "User action")
}
// Output: [loki-logger] WARNING: label 'user_id' has 11 unique values (threshold: 10)
```

#### Configure Cardinality Threshold

Adjust the threshold based on your needs:

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithMaxLabelCardinality(20),  // Allow up to 20 unique values
)

// Disable cardinality checking completely
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithMaxLabelCardinality(0),  // Disabled
)
```

#### Custom Cardinality Warning Handler

Integrate cardinality warnings with your monitoring system:

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithCardinalityWarningHandler(func(labelKey string, uniqueValues, threshold int) {
        // Send to metrics
        metrics.Gauge("logger.label.cardinality", uniqueValues,
            "label:"+labelKey)

        // Alert if severely exceeded
        if uniqueValues > threshold*2 {
            alerting.SendAlert("High label cardinality detected")
        }
    }),
)
```

#### Best Practices

```go
// ❌ BAD - Don't do this
labels["user_id"] = "12345"      // Too many unique values
labels["timestamp"] = time.Now() // Changes constantly
labels["ip"] = clientIP          // Too many unique values
labels["request_id"] = uuid      // Infinite cardinality

// ✅ GOOD - Use these as fields instead
fields := loki.Fields{
    "user_id":    12345,
    "timestamp":  time.Now(),
    "ip":         clientIP,
    "request_id": uuid,
}

// ✅ GOOD - Labels with low cardinality
labels := map[string]string{
    "environment": "production",  // 3-4 values (dev, staging, prod)
    "region":      "us-east-1",   // 5-10 values (AWS regions)
    "service":     "api",          // 5-20 values (microservices)
}
```

## Rate Limiting

Protect your Loki instance from log bombing and excessive load with built-in rate limiting.

### How It Works

The rate limiter uses a **token bucket algorithm** with intelligent sampling:

1. **Under the limit**: All logs pass through normally
2. **Over the limit**: Logs are sampled based on `SamplingRatio`
3. **Critical logs**: Error and Fatal logs always pass (configurable)

### Basic Configuration

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    // Allow max 1000 logs/second, sample 10% when exceeded
    loki.WithRateLimit(1000, 0.1),
)
```

### Advanced Configuration

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),

    // Set rate limit
    loki.WithMaxLogsPerSecond(1000),

    // Set sampling ratio (0.1 = keep 1 in 10 logs when over limit)
    loki.WithSamplingRatio(0.1),

    // Configure which levels are never rate-limited
    loki.WithAlwaysLogLevels([]loki.Level{
        loki.LevelError,
        loki.LevelFatal,
        // loki.LevelWarn, // Optionally never rate-limit warnings
    }),

    // Monitor rate limiting statistics
    loki.WithRateLimitHandler(func(dropped, sampled int) {
        fmt.Printf("Rate limit stats: %d dropped, %d sampled\n", dropped, sampled)

        // Send to your metrics system
        metrics.IncrementCounter("logs_dropped", dropped)
        metrics.IncrementCounter("logs_sampled", sampled)
    }),

    // Report stats every 5 seconds (default: 10s)
    loki.WithRateLimitStatsInterval(5 * time.Second),
)
```

### Behavior Example

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithRateLimit(100, 0.1), // 100 logs/sec, 10% sampling
)

ctx := context.Background()

// Scenario: Loop generates 1000 logs/second
for i := 0; i < 1000; i++ {
    if i < 100 {
        logger.Info(ctx, "Request processed")  // ✓ Allowed (under limit)
    } else if i%10 == 0 {
        logger.Info(ctx, "Request processed")  // ✓ Sampled (1 in 10)
    } else {
        logger.Info(ctx, "Request processed")  // ✗ Dropped
    }

    // Error logs ALWAYS pass through
    logger.Error(ctx, "Critical error")        // ✓ Always allowed
}
// Result: ~190 logs sent (100 normal + ~90 sampled + all errors)
```

### Tuning Guidelines

| Scenario | Recommended Config |
|----------|-------------------|
| Low-traffic app | `WithRateLimit(500, 0.1)` |
| Medium-traffic app | `WithRateLimit(1000, 0.1)` |
| High-traffic app | `WithRateLimit(5000, 0.05)` |
| Development | `WithMaxLogsPerSecond(0)` (disabled) |
| Critical services | Lower sampling ratio (0.01-0.05) |

### Monitoring

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithRateLimit(1000, 0.1),
    loki.WithRateLimitHandler(func(dropped, sampled int) {
        if dropped > 1000 {
            // Alert: Too many logs being dropped
            alerting.SendAlert("High log drop rate")
        }

        // Track in Prometheus/Datadog/etc
        metrics.Gauge("logger.dropped", float64(dropped))
        metrics.Gauge("logger.sampled", float64(sampled))
    }),
)
```

### Disabling Rate Limiting

Rate limiting is **disabled by default**. To explicitly disable:

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-app"),
    loki.WithMaxLogsPerSecond(0), // 0 = disabled
)
```

## Best Practices

1. **Always close the logger** - Use `defer logger.Close()` to ensure logs are flushed
2. **Pass context** - Always pass `context.Context` to enable proper cancellation and tracing
3. **Use appropriate log levels** - Debug for development, Info for production
4. **Add structured fields** - Better than string formatting for searchable data
5. **Keep label cardinality low** - Use the built-in cardinality validator (default threshold: 10 unique values per label)
6. **Labels vs Fields** - Use labels for low-cardinality metadata (environment, region) and fields for high-cardinality data (user IDs, timestamps)
7. **Batch configuration** - Tune BatchSize and FlushInterval for your needs
8. **Monitor transport errors** - Use `WithErrorHandler` to track logging failures
9. **Monitor label cardinality** - Use `WithCardinalityWarningHandler` to detect and fix high-cardinality labels
10. **Query by level** - Use `{app="my-app", level="error"}` in Grafana for fast filtering
11. **Enable rate limiting in production** - Protect Loki from log bombing (start with `WithRateLimit(1000, 0.1)`)
12. **Monitor rate limit stats** - Use `WithRateLimitHandler` to track dropped/sampled logs and tune your configuration

## Architecture

```
loki-logger-go/
├── logger.go              # Main logger API
├── models/                # Core models
│   ├── config.go         # Configuration
│   ├── entry.go          # Log entry structure
│   └── levels.go         # Log level definitions
├── errors/                # Error types
├── transport/             # Transport layer
│   ├── transport.go      # Transport interface
│   ├── console.go        # Console transport with formatting
│   └── loki.go           # Loki transport
├── internal/              # Internal packages
│   ├── pool/             # Buffer pooling
│   ├── client/           # HTTP client
│   └── ratelimit/        # Rate limiting with token bucket
└── middleware/            # Framework middlewares
    └── gin.go            # Gin middleware
```

## Performance

The library is designed for high performance:

- **Buffer pooling** reduces memory allocations (up to 256KB buffers for stack traces)
- **Batching** minimizes network calls (configurable batch size)
- **Async flushing** doesn't block your application
- **Efficient JSON encoding** with minimal overhead
- **Rate limiting** prevents excessive load on Loki during traffic spikes
- **Token bucket algorithm** provides fair and efficient rate limiting

## Quick Start Guide

### 1. Start Loki (Optional for Development)

```bash
# Using Docker
docker run -d --name=loki -p 3100:3100 grafana/loki
```

### 2. Test Your Integration

All examples are documented above in the Quick Start, Gin Middleware, and Distributed Tracing sections. Simply copy the code snippets and adapt them to your needs.

For testing with trace IDs:
```bash
curl -H "X-Trace-Id: test-123" http://localhost:8080/your-endpoint
```

## Contributing

We welcome contributions! To get started:

1. Read our [Contributing Guide](CONTRIBUTING.md)
2. Fork the repository
3. Create a feature branch
4. Make your changes
5. Submit a Pull Request

Please ensure:
- All tests pass
- Code follows our style guidelines
- Documentation is updated
- Commit messages are clear

For bugs, feature requests, or questions, please [open an issue](https://github.com/yourusername/loki-logger-go/issues).

## License

MIT License - see LICENSE file for details

## Similar Projects

This library was inspired by:
- [loki-logger](https://github.com/edaniel30/loki-logger) - TypeScript version
- [logrus](https://github.com/sirupsen/logrus)
- [zap](https://github.com/uber-go/zap)

## Support

For issues and questions, please open an issue on GitHub.
