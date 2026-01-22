# loki-logger-go

A powerful, flexible, and easy-to-use logging library for Go, designed to integrate seamlessly with Grafana Loki.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Installation

```bash
go get github.com/edaniel30/loki-logger-go
```

## Quick Start

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLokiHost("http://localhost:3100"),
)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

logger.Info(context.Background(), "Application started", nil)
```

For more examples (console-only mode, authentication, web applications, error handling, etc.), see **[usage examples](./docs/examples.md)**.

## Documentation

### 📖 [Configuration Guide](./docs/CONFIGURATION.md)
Learn about all configuration options, functional options, and common configuration patterns for development and production.

### 📚 [Usage Examples](./examples/)
Practical examples including web applications, HTTP middleware, error handling, structured logging, and best practices.

### 🏷️ [Labels Guide](./docs/LABELS.md)
Understanding labels, cardinality, best practices, and labels vs fields.

## Log Levels

The library supports five log levels (from lowest to highest):

- `LevelDebug` - Detailed diagnostic information
- `LevelInfo` - General informational messages
- `LevelWarn` - Warning messages
- `LevelError` - Error messages
- `LevelFatal` - Critical errors

```go
logger.Debug(ctx, "Debug message", nil)
logger.Info(ctx, "Info message", nil)
logger.Warn(ctx, "Warning message", nil)
logger.Error(ctx, "Error occurred", nil)
logger.Fatal(ctx, "Fatal error", nil)
```

Set minimum log level with `LogLevel` config option.

## Structured Logging

Add structured fields to your logs:

```go
logger.Info(ctx, "User logged in", loki.Fields{
    "user_id":  123,
    "username": "john_doe",
    "ip":       "192.168.1.1",
})
```

## Child Loggers with Labels

Create child loggers with additional context labels:

```go
// Parent logger
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLabels(types.Labels{
        "environment": "production",
    }),
)

// Child logger with component-specific labels
apiLogger := logger.WithLabels(types.Labels{
    "component": "api",
})

apiLogger.Info(ctx, "Request processed")
```

See [Labels Guide](./docs/LABELS.md) for best practices on labels vs fields and cardinality.

## Querying Logs in Grafana

Since `level` is automatically added as a label, you can efficiently filter logs:

```logql
# Get all error logs
{app="my-service", level="error"}

# Get production errors from specific component
{app="my-service", environment="production", component="api", level="error"}
```

## Performance

- **Buffer pooling** reduces memory allocations (up to 256KB buffers)
- **Batching** minimizes network calls (configurable batch size)
- **Async flushing** doesn't block your application
- **Efficient JSON encoding** with minimal overhead
- **Retry with exponential backoff** handles transient failures gracefully

## Best Practices

1. **Always close the logger** - Use `defer logger.Close()` to ensure logs are flushed
2. **Pass context** - Always pass `context.Context` for proper cancellation and timeout handling
3. **Keep label cardinality low** - Avoid high-cardinality values (user IDs, timestamps) as labels
4. **Labels vs Fields** - Use labels for low-cardinality metadata, fields for high-cardinality data
5. **Tune batch configuration** - Adjust BatchSize and FlushInterval based on your needs
6. **Monitor transport errors** - Use `WithErrorHandler` to track logging failures

## Contributing

We welcome contributions! Please ensure:
- All tests pass (`make test`)
- Code coverage meets 90% threshold (`make test-coverage`)
- Code follows Go best practices
- Documentation is updated

## License

MIT License - see [LICENSE](./LICENSE) file for details

## Similar Projects

This library was inspired by:
- [loki-logger](https://github.com/edaniel30/loki-logger) - TypeScript version
- [logrus](https://github.com/sirupsen/logrus)
- [zap](https://github.com/uber-go/zap)
