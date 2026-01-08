# Distributed Tracing with Loki Logger Go

## Overview

The Gin middleware automatically extracts `trace_id` from HTTP requests and includes it in log content, enabling distributed tracing across your microservices.

## How It Works

### Automatic Trace ID Extraction

The middleware automatically looks for trace IDs in these headers (in order):

1. `X-Trace-Id` - Common custom header
2. `X-Request-Id` - Alternative custom header
3. `traceparent` - W3C Trace Context standard
4. Gin context (`trace_id` key) - Set by other middleware

### Automatic Inclusion in Logs

When a trace ID is found:
- ✅ **Added to log content** (searchable with LogQL `|=` operator)
- ✅ **Included in log fields** (visible in log output)
- ✅ **Applied automatically** to all logs in that request
- ✅ **NOT indexed as a label** (avoids high cardinality issues)

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

    // Just add the middleware - trace_id will be automatically included!
    r.Use(middleware.GinLogger(logger))

    r.GET("/users/:id", func(c *gin.Context) {
        // All logs in this request will have the same trace_id in content
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

### 5. Querying Logs by Trace ID

Trace IDs are stored in log **content**, not as labels. Use LogQL content search:

```logql
# ✅ CORRECT - Search in log content
{app="my-api"} |= "550e8400-e29b-41d4-a716-446655440000"

# ✅ CORRECT - Filter by level, then search trace
{app="my-api", level="error"} |= "550e8400-e29b-41d4-a716-446655440000"

# ❌ INCORRECT - trace_id is not a label
{app="my-api", trace_id="550e8400-e29b-41d4-a716-446655440000"}
```

**Why not use trace_id as a label?**
- Trace IDs have **extremely high cardinality** (one per request)
- High cardinality labels cause severe performance issues in Loki
- Content search with `|=` is sufficient for finding traces

## Performance Considerations

### Low Cardinality Design

✅ **This library follows Loki best practices:**

- Trace IDs are stored in **log content**, not as labels
- No cardinality issues regardless of traffic volume
- Scales to millions of requests per minute
- Works with any retention period

### Query Performance

Searching trace IDs in content is fast and efficient:

```logql
# Fast content search with label pre-filtering
{app="my-api", level="error"} |= "trace-id-here"
```

**Tips for faster queries:**
1. Always filter by labels first (`app`, `level`, etc.)
2. Use the `|=` operator for exact trace ID matching
3. Add time ranges to limit the search scope

### No Special Configuration Needed

Unlike label-based tracing, you don't need:
- ❌ Cardinality limits configuration
- ❌ Retention policies for high-cardinality labels
- ❌ Sampling strategies
- ❌ Special Loki tuning

Everything works out of the box with optimal performance.

## See Also

- [LABELS.md](./LABELS.md) - Understanding Loki labels
- [W3C Trace Context](https://www.w3.org/TR/trace-context/) - Standard specification
- [README.md](./README.md) - Complete usage examples
