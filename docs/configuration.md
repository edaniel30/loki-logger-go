# Configuration

Complete configuration options for loki-logger-go.

## Quick Start

### Minimal Setup

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
```

### Console-Only (Development)

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithOnlyConsole(true),  // No Loki required
)
```

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AppName` | string | `"app"` | Application name (used as `app` label) |
| `AppVersion` | string | `"1.0.0"` | Application version (used as `version` label) |
| `AppEnv` | string | `"local"` | Application environment (used as `environment` label) |
| `LokiHost` | string | `"http://localhost:3100"` | Loki server URL |
| `LokiUsername` | string | `""` | Basic auth username |
| `LokiPassword` | string | `""` | Basic auth password |
| `LogLevel` | Level | `LevelInfo` | Minimum log level to process |
| `Labels` | Labels | `{}` | Additional custom labels for all logs |
| `OnlyConsole` | bool | `false` | Skip Loki, only console output |
| `BatchSize` | int | `100` | Max logs per batch |
| `FlushInterval` | Duration | `5s` | Auto-flush interval |
| `MaxRetries` | int | `3` | HTTP retry attempts |
| `Timeout` | Duration | `10s` | Operation timeout |
| `TraceIDExtractor` | func | `nil` | Function to extract trace ID from context |

\* `LokiHost` not required if `OnlyConsole = true`

## Automatic Labels

The logger automatically adds the following labels to every log entry:

| Label | Source | Description |
|-------|--------|-------------|
| `app` | `AppName` config | Application identifier |
| `level` | log level | Log severity level |
| `version` | `AppVersion` config | Application version |
| `environment` | `AppEnv` config | Deployment environment |

These labels are **reserved** — user-provided labels with the same names will be overridden by the system.

## Automatic Fields

The logger automatically injects these fields into every log entry unless the caller provides them explicitly:

| Field | Description |
|-------|-------------|
| `file` | Basename of the source file that called the logger |
| `line` | Line number of the log call |
| `trace_id` | Trace ID extracted from context (requires `WithTraceIDExtractor`) |

## Functional Options

### Application Identity

```go
loki.WithAppName("my-service")        // Sets 'app' label
loki.WithAppVersion("2.1.0")          // Sets 'version' label
loki.WithAppEnv("production")         // Sets 'environment' label
```

### Server Connection

```go
// Basic setup
loki.WithLokiHost("http://localhost:3100")

// With authentication
loki.WithLokiBasicAuth("admin", "password")

// Kubernetes
loki.WithLokiHost("http://loki-gateway.namespace:80")
```

### Log Levels

```go
loki.WithLogLevel(types.LevelInfo)  // Info, Warn, Error, Fatal only
```

| Level | Value | Usage |
|-------|-------|-------|
| `LevelDebug` | 0 | Development, verbose logging |
| `LevelInfo` | 1 | General information |
| `LevelWarn` | 2 | Warnings, potential issues |
| `LevelError` | 3 | Errors, requires attention |
| `LevelFatal` | 4 | Critical failures |

### Custom Labels

```go
loki.WithLabels(types.Labels{
    "region": "us-east-1",
    "team":   "backend",
})
```

> **Note:** The keys `app`, `level`, `version`, and `environment` are reserved and will be overridden by system labels even if set here.

See [Labels Documentation](labels.md) for best practices.

### Stack Traces

Stack traces are **always enabled** for `Error` and `Fatal` log levels. They are automatically appended to the log message — no configuration required.

### Trace ID Extraction

Use `WithTraceIDExtractor` to automatically propagate a trace ID from your `context.Context` into every log entry as the `trace_id` field. This is useful when integrating with distributed tracing systems.

```go
// Example with a custom context key
loki.WithTraceIDExtractor(func(ctx context.Context) string {
    if id, ok := ctx.Value(myTraceKey{}).(string); ok {
        return id
    }
    return ""
})
```

The `trace_id` field is only added when:
1. A `TraceIDExtractor` is configured.
2. The extractor returns a non-empty string.
3. The caller has **not** already set `"trace_id"` in the fields map.

### Performance Tuning

```go
// Batch settings
loki.WithBatchSize(200)                   // Larger batches = better throughput
loki.WithFlushInterval(10 * time.Second)  // Longer interval = more batching
```

| Scenario | BatchSize | FlushInterval | Notes |
|----------|-----------|---------------|-------|
| **High throughput** | 200-500 | 10-30s | Better batching, higher latency |
| **Low latency** | 10-50 | 1-2s | Faster delivery, more requests |
| **Balanced** | 100-200 | 5-10s | Recommended for most cases |

## Common Configurations

### Development

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithAppVersion("0.1.0"),
    loki.WithAppEnv("local"),
    loki.WithOnlyConsole(true),           // No Loki needed
    loki.WithLogLevel(types.LevelDebug),  // Verbose
)
```

### Staging

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-api"),
    loki.WithAppVersion("1.2.0"),
    loki.WithAppEnv("staging"),
    loki.WithLokiHost("http://loki-staging:3100"),
    loki.WithLogLevel(types.LevelDebug),
)
```

### Production

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-api"),
    loki.WithAppVersion(os.Getenv("APP_VERSION")),
    loki.WithAppEnv("production"),
    loki.WithLokiHost(os.Getenv("LOKI_URL")),
    loki.WithLokiBasicAuth(
        os.Getenv("LOKI_USER"),
        os.Getenv("LOKI_PASSWORD"),
    ),
    loki.WithLogLevel(types.LevelInfo),
    loki.WithBatchSize(200),
    loki.WithFlushInterval(10 * time.Second),
    loki.WithLabels(types.Labels{
        "region": os.Getenv("AWS_REGION"),
        "team":   "backend",
    }),
    loki.WithTraceIDExtractor(func(ctx context.Context) string {
        id, _ := ctx.Value(traceKey{}).(string)
        return id
    }),
)
```

## Best Practices

### ✅ Do

```go
// Always close logger on shutdown
defer logger.Close()

// Set app identity explicitly
loki.WithAppName("my-service")
loki.WithAppVersion(buildVersion)
loki.WithAppEnv(os.Getenv("ENV"))

// Use TraceIDExtractor for distributed tracing correlation
loki.WithTraceIDExtractor(myTracer.ExtractTraceID)

// Use OnlyConsole for tests
loki.WithOnlyConsole(true)
```

### ❌ Don't

```go
// Don't override reserved labels — they will be ignored
loki.WithLabels(types.Labels{
    "app":         "my-service", // Overridden by AppName
    "environment": "production", // Overridden by AppEnv
    "version":     "1.0.0",      // Overridden by AppVersion
    "level":       "info",       // Overridden by log level
})

// Don't use Debug level in production
loki.WithLogLevel(types.LevelDebug)  // Too verbose

// Don't use tiny batch sizes
loki.WithBatchSize(1)  // Too many HTTP requests

// Don't hardcode credentials
loki.WithLokiBasicAuth("admin", "password123")  // Use env vars

// Don't forget to close
// Missing: defer logger.Close()
```
