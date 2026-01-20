# Configuration Guide

This guide covers all configuration options available in loki-logger-go.

## Basic Configuration

### Minimal Configuration

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

### Console-Only Mode

For development or when Loki is not available:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithOnlyConsole(true),  // No Loki connection required
)
```

## Configuration Options

### Config Struct

```go
type Config struct {
    // Required Fields
    AppName  string // Application name (used as 'app' label)

    // Loki Connection
    LokiHost     string // Loki server URL (e.g., "http://localhost:3100")
    LokiUsername string // Basic auth username (optional)
    LokiPassword string // Basic auth password (optional)

    // Behavior
    LogLevel          Level             // Minimum log level (default: Debug)
    Labels            types.Labels // Default labels for all logs
    IncludeStackTrace bool              // Auto-include stack traces on Error/Fatal
    OnlyConsole       bool              // Skip Loki, only console output

    // Performance
    BatchSize     int           // Max logs per batch (default: 100)
    FlushInterval time.Duration // Auto-flush interval (default: 5s)
    MaxRetries    int           // HTTP retry attempts (default: 3)
    Timeout       time.Duration // Operation timeout (default: 10s)

    // Error Handling
    ErrorHandler ErrorHandler // Called when transport errors occur
}
```

### Field Details

#### AppName (Required)

The application name used as the `app` label in Loki.

```go
Config{
    AppName: "my-api-service",
}
```

**Requirements**:
- Must not be empty
- Should be consistent across all instances of the same service

#### LokiHost (Required unless OnlyConsole = true)

The base URL of your Loki server.

```go
Config{
    LokiHost: "http://localhost:3100",           // Local
    LokiHost: "https://loki.example.com",        // Remote
    LokiHost: "http://loki-gateway.namespace:80", // Kubernetes
}
```

#### LokiUsername & LokiPassword (Optional)

Basic authentication credentials for Loki.

```go
Config{
    LokiHost:     "https://loki.example.com",
    LokiUsername: "admin",
    LokiPassword: "secret123",
}
```

Or use the functional option:

```go
logger, err := loki.New(config,
    loki.WithLokiBasicAuth("admin", "secret123"),
)
```

#### LogLevel (Optional)

Minimum log level to process. Logs below this level are ignored.

```go
Config{
    LogLevel: types.LevelInfo,  // Only Info, Warn, Error, Fatal
}
```

**Available Levels**:
- `LevelDebug` (0): Most verbose
- `LevelInfo` (1): General information
- `LevelWarn` (2): Warnings
- `LevelError` (3): Errors
- `LevelFatal` (4): Critical failures

**Default**: `LevelDebug` (log everything)

#### Labels (Optional)

Global labels attached to all log entries.

```go
Config{
    Labels: types.Labels{
        "environment": "production",
        "region":      "us-east-1",
        "team":        "backend",
    },
}
```

**Best Practices**:
- Use low-cardinality values (< 50 unique values per label)
- See [Labels Guide](./labels.md) for detailed guidance

#### IncludeStackTrace (Optional)

Automatically include stack traces for Error and Fatal level logs.

```go
Config{
    IncludeStackTrace: true,  // Stack traces on Error/Fatal
}
```

**Default**: `false`

**Behavior**:
- When `true`: Error and Fatal logs include full stack trace
- Can be disabled per-log using `_skip_stack_trace` field

```go
// Disable stack trace for specific log
logger.Error(ctx, "Error occurred", types.Fields{
    "_skip_stack_trace": true,
})
```

#### OnlyConsole (Optional)

Skip Loki transport and only log to console (stdout).

```go
Config{
    OnlyConsole: true,  // Development mode
}
```

**Use Cases**:
- Local development
- Testing
- When Loki is temporarily unavailable

**Default**: `false`

#### BatchSize (Optional)

Maximum number of log entries per batch sent to Loki.

```go
Config{
    BatchSize: 100,  // Send up to 100 logs per batch
}
```

**Considerations**:
- **Larger batches** (100-500): Better throughput, higher latency
- **Smaller batches** (10-50): Lower latency, more HTTP requests
- **Very large batches** (>1000): May hit Loki size limits

**Default**: `100`
**Recommended**: `100-200` for most use cases

#### FlushInterval (Optional)

How often to automatically flush buffered logs to Loki.

```go
Config{
    FlushInterval: 5 * time.Second,  // Flush every 5 seconds
}
```

**Considerations**:
- **Shorter intervals** (1-2s): Lower latency, more HTTP requests
- **Longer intervals** (10-30s): Better batching, higher latency
- Logs are also flushed when buffer reaches `BatchSize`

**Default**: `5 * time.Second`
**Recommended**: `5-10s` for most use cases

#### MaxRetries (Optional)

Number of retry attempts for failed HTTP requests to Loki.

```go
Config{
    MaxRetries: 3,  // Retry up to 3 times
}
```

**Behavior**:
- Uses exponential backoff: 100ms, 200ms, 400ms, ...
- After max retries, error is reported via ErrorHandler

**Default**: `3`
**Recommended**: `3-5` for production

#### Timeout (Optional)

Timeout for all operations (write, flush, shutdown).

```go
Config{
    Timeout: 10 * time.Second,
}
```

**Applies to**:
- Individual log write operations (if context has no deadline)
- Flush operations
- Shutdown/close operations
- HTTP requests to Loki

**Default**: `10 * time.Second`
**Recommended**: `5-30s` depending on your latency requirements

#### ErrorHandler (Optional)

Callback function for handling transport errors.

```go
Config{
    ErrorHandler: func(transport string, err error) {
        // Log to stderr, metrics, alerting, etc.
        log.Printf("ERROR in %s transport: %v", transport, err)
    },
}
```

**Parameters**:
- `transport`: Name of the transport ("console" or "loki")
- `err`: The error that occurred

**Common Errors**:
- Network errors (connection refused, timeout)
- HTTP errors (4xx, 5xx from Loki)
- Serialization errors

**Best Practices**:
- Always set an ErrorHandler in production
- Log to a separate logging system
- Consider metrics/alerting for critical errors
- Don't panic or exit in the handler

## Functional Options

Instead of setting fields directly, you can use functional options:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLokiHost("http://localhost:3100"),
    loki.WithLokiBasicAuth("admin", "password"),
    loki.WithLogLevel(types.LevelInfo),
    loki.WithLabels(types.Labels{
        "environment": "production",
    }),
    loki.WithIncludeStackTrace(true),
    loki.WithBatchSize(200),
    loki.WithFlushInterval(10*time.Second),
    loki.WithMaxRetries(5),
    loki.WithTimeout(30*time.Second),
    loki.WithErrorHandler(myErrorHandler),
)
```

## Common Configurations

### Production Configuration

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("production-api"),
    loki.WithLokiHost("https://loki.company.com"),
    loki.WithLokiBasicAuth(os.Getenv("LOKI_USER"), os.Getenv("LOKI_PASSWORD")),
    loki.WithLogLevel(types.LevelInfo),
    loki.WithIncludeStackTrace(true),
    loki.WithBatchSize(200),
    loki.WithFlushInterval(10 * time.Second),
    loki.WithMaxRetries(5),
    loki.WithTimeout(30 * time.Second),
    loki.WithLabels(types.Labels{
        "environment": "production",
        "region":      os.Getenv("AWS_REGION"),
        "version":     "v2.1.0",
    }),
    loki.WithErrorHandler(func(transport string, err error) {
        // Send to metrics/alerting
        metrics.IncLoggerError(transport)

        // Log to stderr (separate from app logs)
        fmt.Fprintf(os.Stderr, "LOGGER ERROR [%s]: %v\n", transport, err)
    }),
)
```