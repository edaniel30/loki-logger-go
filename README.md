# Loki Logger Go

A powerful, flexible, and easy-to-use logging library for Go, designed to integrate seamlessly with Grafana Loki.

## Features

- 🚀 **High Performance**: Uses buffer pooling and batching to minimize allocations and network calls
- 📊 **Structured Logging**: Full support for structured fields and labels
- 🎯 **Multiple Transports**: Send logs to Loki, console, or custom destinations
- 🔄 **Auto-Batching**: Automatically batches logs for efficient network usage
- ⚡ **Async Processing**: Background flushing with configurable intervals
- 🛡️ **Thread-Safe**: Safe for concurrent use across goroutines
- 🎨 **Colored Output**: Beautiful console output with color-coded log levels
- 🔌 **Middleware Support**: Built-in middleware for Gin and other frameworks
- ⚙️ **Highly Configurable**: Flexible configuration with functional options
- 🔁 **Auto-Retry**: Automatic retry with exponential backoff for failed requests

## Installation

```bash
go get github.com/edaniel30/loki-logger-go
```

## Quick Start

### Basic Usage

```go
package main

import (
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

    // Log messages
    logger.Info("Application started", loki.Fields{
        "version": "1.0.0",
        "port":    8080,
    })

    logger.Error("Failed to connect to database", loki.Fields{
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
    AppName       string              // Application name (required)
    LokiHost      string              // Loki server URL
    LokiUsername  string              // Loki username for basic auth (optional)
    LokiPassword  string              // Loki password for basic auth (optional)
    LogLevel      Level               // Minimum log level
    Labels        map[string]string   // Default labels for all logs
    BatchSize     int                 // Number of logs to batch
    FlushInterval time.Duration       // How often to flush logs
    MaxRetries    int                 // Max retry attempts
    Timeout       time.Duration       // HTTP request timeout
    EnableConsole bool                // Enable console output
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
logger.Info("User logged in", loki.Fields{
    "user_id":  123,
    "username": "john_doe",
    "ip":       "192.168.1.1",
    "duration": 150,
})
```

## Context Logging

Create a logger with additional context:

```go
userLogger := logger.WithFields(loki.Fields{
    "component": "user_handler",
    "version":   "v2",
})

userLogger.Info("Processing request")
```

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

### Label Cardinality Warning

⚠️ **Important**: Keep label cardinality low. Don't use high-cardinality values as labels:

```go
// ❌ BAD - Don't do this
labels["user_id"] = "12345"      // Too many unique values
labels["timestamp"] = time.Now() // Changes constantly
labels["ip"] = clientIP          // Too many unique values

// ✅ GOOD - Use these as fields instead
fields := loki.Fields{
    "user_id":   12345,
    "timestamp": time.Now(),
    "ip":        clientIP,
}
```

## Best Practices

1. **Always close the logger** - Use `defer logger.Close()` to ensure logs are flushed
2. **Use appropriate log levels** - Debug for development, Info for production
3. **Add structured fields** - Better than string formatting for searchable data
4. **Use labels wisely** - Keep label cardinality low (< 10-20 unique values per label)
5. **Batch configuration** - Tune BatchSize and FlushInterval for your needs
6. **Query by level** - Use `{app="my-app", level="error"}` in Grafana for fast filtering

## Architecture

```
loki-logger-go/
├── logger.go              # Main logger API
├── levels.go              # Log level definitions
├── entry.go               # Log entry structure
├── config.go              # Configuration
├── errors.go              # Error types
├── formatter/             # Log formatters
│   ├── formatter.go       # Formatter interface
│   ├── json.go           # JSON formatter
│   └── text.go           # Text formatter
├── transport/             # Transport layer
│   ├── transport.go      # Transport interface
│   ├── console.go        # Console transport
│   ├── loki.go          # Loki transport
│   └── multi.go         # Multiple transports
├── internal/              # Internal packages
│   ├── pool/             # Buffer pooling
│   └── client/           # HTTP client
└── middleware/            # Framework middlewares
    └── gin.go            # Gin middleware
```

## Performance

The library is designed for high performance:

- **Buffer pooling** reduces memory allocations
- **Batching** minimizes network calls
- **Async flushing** doesn't block your application
- **Efficient JSON encoding** with minimal overhead

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
