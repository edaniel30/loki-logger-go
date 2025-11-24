# Distributed Tracing with Loki Logger Go

## Overview

The Gin middleware automatically extracts and indexes `trace_id` from HTTP requests, enabling distributed tracing across your microservices.

## How It Works

### Automatic Trace ID Extraction

The middleware automatically looks for trace IDs in these headers (in order):

1. `X-Trace-Id` - Common custom header
2. `X-Request-Id` - Alternative custom header
3. `traceparent` - W3C Trace Context standard
4. Gin context (`trace_id` key) - Set by other middleware

### Automatic Indexing

When a trace ID is found:
- ✅ **Indexed as a label** in Loki (fast queries)
- ✅ **Added to log fields** (visible in log content)
- ✅ **Applied automatically** to all logs in that request

## Usage

### Basic Setup

```go
package main

import (
    "loki-logger-go"
    "loki-logger-go/middleware"
    "loki-logger-go/models"
    "github.com/gin-gonic/gin"
)

func main() {
    logger, _ := loki.New(
        models.DefaultConfig(),
        models.WithAppName("my-api"),
    )
    defer logger.Close()

    r := gin.New()

    // Just add the middleware - trace_id will be auto-indexed!
    r.Use(middleware.GinLogger(logger))

    r.GET("/users/:id", func(c *gin.Context) {
        // All logs in this request will have the same trace_id
        logger.Info("Processing user request")

        c.JSON(200, gin.H{"id": c.Param("id")})
    })

    r.Run(":8080")
}
```

### With Trace ID Generation

If you want to generate trace IDs for requests that don't have them:

```go
package main

import (
    "loki-logger-go/middleware"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func TraceIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Check existing headers
        traceID := c.GetHeader("X-Trace-Id")

        // Generate if not present
        if traceID == "" {
            traceID = uuid.New().String()
        }

        // Store in context
        c.Set("trace_id", traceID)

        // Add to response
        c.Header("X-Trace-Id", traceID)

        c.Next()
    }
}

func main() {
    logger, _ := loki.New(models.DefaultConfig())
    defer logger.Close()

    r := gin.New()

    // Add trace ID middleware BEFORE Loki middleware
    r.Use(TraceIDMiddleware())
    r.Use(middleware.GinLogger(logger))

    r.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "trace_id": c.GetString("trace_id"),
        })
    })

    r.Run(":8080")
}
```

## Supported Headers

The middleware supports these standard trace ID headers:

| Header | Standard | Example |
|--------|----------|---------|
| `X-Trace-Id` | Custom (common) | `550e8400-e29b-41d4-a716-446655440000` |
| `X-Request-Id` | Custom (common) | `req_abc123xyz` |
| `traceparent` | W3C Trace Context | `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01` |


## Best Practices

### 1. Always Generate at Entry Point

Generate trace IDs at your API gateway or first service:

```go
// ✅ GOOD - Generate at entry point
r.Use(TraceIDMiddleware())
r.Use(middleware.GinLogger(logger))
```

### 2. Propagate Through All Services

Always pass trace ID to downstream services:

```go
// ✅ GOOD - Propagate trace ID
req.Header.Set("X-Trace-Id", c.GetString("trace_id"))

// ❌ BAD - Don't generate new ones in downstream services
// traceID := uuid.New().String() // NO!
```

### 3. Include in Responses (Optional)

Help clients correlate requests:

```go
c.Header("X-Trace-Id", c.GetString("trace_id"))
c.JSON(200, gin.H{
    "data": data,
    "trace_id": c.GetString("trace_id"),
})
```

### 4. Log Important Events

Log key events in your trace:

```go
logger.Info("Request received")
logger.Info("Database query executed", models.Fields{"duration_ms": 45})
logger.Info("External API called", models.Fields{"service": "payment-api"})
logger.Info("Response sent")
```

### 5. Don't Overuse Trace IDs as Labels

⚠️ **Warning**: Trace IDs have **very high cardinality** (millions of unique values).

The middleware only indexes trace_id when it exists in the request. If you're generating millions of unique trace IDs:

- ✅ Use for debugging specific traces
- ✅ Short retention (e.g., 7 days)
- ❌ Don't use for long-term storage
- ❌ Don't use for general queries

## Performance Considerations

### Label Cardinality

Trace IDs create **one stream per unique trace ID**. This is fine for:
- ✅ Short retention periods (7-30 days)
- ✅ Moderate traffic (< 100k RPM)
- ✅ Debugging and troubleshooting

May cause issues with:
- ❌ Very high traffic (> 1M RPM)
- ❌ Long retention (> 90 days)
- ❌ Very large number of concurrent traces

### Optimization Tips

1. **Use retention policies**:
   ```yaml
   # Loki config
   limits_config:
     retention_period: 7d  # Short retention for traces
   ```

2. **Query optimization**:
   ```logql
   # ✅ GOOD - Specific trace
   {trace_id="abc123"}

   # ❌ BAD - Scanning all traces
   {app="my-api"} | json | trace_id =~ ".*"
   ```

3. **Conditional indexing** (future):
   Only index trace IDs for errors or slow requests


### High Cardinality Warning in Loki

If you see "cardinality limit exceeded":

1. Reduce trace ID retention
2. Use sampling (only trace 10% of requests)
3. Consider not indexing trace_id as label for all requests

## See Also

- [LABELS.md](./LABELS.md) - Understanding Loki labels
- [W3C Trace Context](https://www.w3.org/TR/trace-context/) - Standard specification
- [README.md](./README.md) - Complete usage examples
