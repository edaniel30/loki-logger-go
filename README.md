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
    loki.WithAppVersion("1.0.0"),
    loki.WithAppEnv("production"),
    loki.WithLokiHost("http://localhost:3100"),
)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

logger.Info(context.Background(), "Application started", nil)
```

## Documentation

### 📖 [Configuration Guide](./docs/configuration.md)
Learn about all configuration options, functional options, and common configuration patterns for development and production.

### 🏷️ [Labels Guide](./docs/labels.md)
Understanding labels, cardinality, best practices, and labels vs fields.

## Log Levels

The library supports five log levels (from lowest to highest):

- `LevelDebug` - Detailed diagnostic information
- `LevelInfo` - General informational messages
- `LevelWarn` - Warning messages
- `LevelError` - Error messages (stack trace automatically included)
- `LevelFatal` - Critical errors (stack trace automatically included)

```go
logger.Debug(ctx, "Debug message", nil)
logger.Info(ctx, "Info message", nil)
logger.Warn(ctx, "Warning message", nil)
logger.Error(ctx, "Error occurred", nil)
logger.Fatal(ctx, "Fatal error", nil)
```

Set minimum log level with `WithLogLevel`.

## Structured Logging

Add structured fields to your logs:

```go
logger.Info(ctx, "User logged in", loki.Fields{
    "user_id":  123,
    "username": "john_doe",
    "ip":       "192.168.1.1",
})
```

The `file` and `line` fields are automatically injected into every entry, pointing to the exact location in your code where the log was called.

## Automatic Labels

Every log entry automatically includes the following Loki labels, sourced from the logger configuration:

| Label | Option | Default |
|-------|--------|---------|
| `app` | `WithAppName` | `"app"` |
| `level` | _(log level)_ | — |
| `version` | `WithAppVersion` | `"1.0.0"` |
| `environment` | `WithAppEnv` | `"local"` |

These labels are **reserved** and cannot be overridden via `WithLabels`.

## Distributed Tracing

Automatically propagate trace IDs from `context.Context` into every log entry:

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithTraceIDExtractor(func(ctx context.Context) string {
        id, _ := ctx.Value(myTraceKey{}).(string)
        return id
    }),
)

// trace_id is added automatically to every log
logger.Info(ctx, "Processing request", nil)
```

The `trace_id` field is only added when the extractor returns a non-empty string and the caller hasn't already set it in the fields map.

## Child Loggers with Labels

Create child loggers with additional context labels:

```go
// Parent logger
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithAppEnv("production"),
)

// Child logger with component-specific labels
apiLogger := logger.WithLabels(types.Labels{
    "component": "api",
})

apiLogger.Info(ctx, "Request processed", nil)
```

See [Labels Guide](./docs/labels.md) for best practices on labels vs fields and cardinality.

## Querying Logs in Grafana

Since `app`, `level`, `version`, and `environment` are automatically added as labels, you can efficiently filter logs:

```logql
# Get all error logs from a service
{app="my-service", level="error"}

# Get production errors from specific version
{app="my-service", environment="production", version="2.1.0", level="error"}

# Filter by component label
{app="my-service", environment="production", component="api"}
```

## Performance

- **Buffer pooling** reduces memory allocations (up to 256KB buffers)
- **Batching** minimizes network calls (configurable batch size)
- **Async flushing** doesn't block your application
- **Efficient JSON encoding** with minimal overhead
- **Retry with exponential backoff** handles transient failures gracefully

## Best Practices

1. **Always close the logger** — Use `defer logger.Close()` to ensure logs are flushed
2. **Pass context** — Always pass `context.Context` for proper cancellation and timeout handling
3. **Set app identity** — Use `WithAppName`, `WithAppVersion`, and `WithAppEnv` for every service
4. **Keep label cardinality low** — Avoid high-cardinality values (user IDs, timestamps) as labels
5. **Labels vs Fields** — Use labels for low-cardinality metadata, fields for high-cardinality data
6. **Use TraceIDExtractor** — Propagate trace IDs for easier distributed tracing correlation
7. **Tune batch configuration** — Adjust `BatchSize` and `FlushInterval` based on your needs

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
