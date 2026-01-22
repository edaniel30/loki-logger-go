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
| \`AppName\` | string | **required** | Application name (used as 'app' label) |
| \`LokiHost\` | string | **required*** | Loki server URL (e.g., \`http://localhost:3100\`) |
| \`LokiUsername\` | string | \`""\` | Basic auth username |
| \`LokiPassword\` | string | \`""\` | Basic auth password |
| \`LogLevel\` | Level | \`LevelDebug\` | Minimum log level to process |
| \`Labels\` | Labels | \`{}\` | Default labels for all logs |
| \`IncludeStackTrace\` | bool | \`false\` | Auto-include stack traces on Error/Fatal |
| \`OnlyConsole\` | bool | \`false\` | Skip Loki, only console output |
| \`BatchSize\` | int | \`100\` | Max logs per batch |
| \`FlushInterval\` | Duration | \`5s\` | Auto-flush interval |
| \`MaxRetries\` | int | \`3\` | HTTP retry attempts |
| \`Timeout\` | Duration | \`10s\` | Operation timeout |
| \`ErrorHandler\` | func | \`nil\` | Called on transport errors |

\* Not required if \`OnlyConsole = true\`

## Functional Options

### Server Connection

```go
// Basic setup
loki.WithAppName("my-service")
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
| \`LevelDebug\` | 0 | Development, verbose logging |
| \`LevelInfo\` | 1 | General information |
| \`LevelWarn\` | 2 | Warnings, potential issues |
| \`LevelError\` | 3 | Errors, requires attention |
| \`LevelFatal\` | 4 | Critical failures |

### Labels

```go
loki.WithLabels(types.Labels{
    "environment": "production",
    "region":      "us-east-1",
    "team":        "backend",
})
```

See [Labels Documentation](LABELS.md) for best practices.

### Stack Traces

```go
// Enable stack traces for Error and Fatal logs
loki.WithIncludeStackTrace(true)

// Disable for specific log
logger.Error(ctx, "Error occurred", types.Fields{
    "_skip_stack_trace": true,  // Skip this one
})
```

### Performance Tuning

```go
// Batch settings
loki.WithBatchSize(200)                   // Larger batches = better throughput
loki.WithFlushInterval(10 * time.Second)  // Longer interval = more batching

// Retries and timeout
loki.WithMaxRetries(5)
loki.WithTimeout(30 * time.Second)
```

| Scenario | BatchSize | FlushInterval | Notes |
|----------|-----------|---------------|-------|
| **High throughput** | 200-500 | 10-30s | Better batching, higher latency |
| **Low latency** | 10-50 | 1-2s | Faster delivery, more requests |
| **Balanced** | 100-200 | 5-10s | Recommended for most cases |

### Error Handling

```go
loki.WithErrorHandler(func(transport string, err error) {
    // Log to stderr
    log.Printf("Logger error in %s: %v", transport, err)

    // Send to monitoring
    metrics.IncCounter("logger_errors", map[string]string{
        "transport": transport,
    })
})
```

**Parameters**:
- \`transport\`: \`"console"\` or \`"loki"\`
- \`err\`: The error that occurred

**Common Errors**:
- Network errors (connection refused, timeout)
- HTTP errors (4xx, 5xx from Loki)
- Serialization errors

## Common Configurations

### Development

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("dev-service"),
    loki.WithOnlyConsole(true),           // No Loki needed
    loki.WithLogLevel(types.LevelDebug),  // Verbose
)
```

### Staging

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("staging-api"),
    loki.WithLokiHost("http://loki-staging:3100"),
    loki.WithLogLevel(types.LevelDebug),
    loki.WithLabels(types.Labels{
        "environment": "staging",
    }),
)
```

### Production

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("production-api"),
    loki.WithLokiHost(os.Getenv("LOKI_URL")),
    loki.WithLokiBasicAuth(
        os.Getenv("LOKI_USER"),
        os.Getenv("LOKI_PASSWORD"),
    ),
    loki.WithLogLevel(types.LevelInfo),      // Less verbose
    loki.WithIncludeStackTrace(true),        // Debug errors
    loki.WithBatchSize(200),                 // Better throughput
    loki.WithFlushInterval(10 * time.Second),
    loki.WithMaxRetries(5),
    loki.WithTimeout(30 * time.Second),
    loki.WithLabels(types.Labels{
        "environment": "production",
        "region":      os.Getenv("AWS_REGION"),
        "version":     "v2.1.0",
    }),
    loki.WithErrorHandler(func(transport string, err error) {
        // Send to monitoring/alerting
        metrics.IncLoggerError(transport)
        fmt.Fprintf(os.Stderr, "LOGGER ERROR [%s]: %v\n", transport, err)
    }),
)
```

## Best Practices

### ✅ Do

```go
// Always close logger on shutdown
defer logger.Close()

// Use environment-specific config
env := os.Getenv("ENV")
if env == "production" {
    loki.WithLogLevel(types.LevelInfo)
}

// Set appropriate timeouts
loki.WithTimeout(30 * time.Second)  // Production
loki.WithTimeout(5 * time.Second)   // Development

// Always set ErrorHandler in production
loki.WithErrorHandler(myErrorHandler)

// Use OnlyConsole for tests
loki.WithOnlyConsole(true)
```

### ❌ Don't

```go
// Don't use Debug level in production
loki.WithLogLevel(types.LevelDebug)  // Too verbose

// Don't use tiny batch sizes
loki.WithBatchSize(1)  // Too many HTTP requests

// Don't ignore errors in ErrorHandler
loki.WithErrorHandler(func(t string, e error) {
    // Empty handler - errors lost
})

// Don't hardcode credentials
loki.WithLokiBasicAuth("admin", "password123")  // Use env vars

// Don't forget to close
// Missing: defer logger.Close()
```
