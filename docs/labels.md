# Labels Guide

Labels in Loki are indexed metadata that allow you to efficiently query and filter your logs. This guide covers best practices for using labels effectively.

## What are Labels?

Labels are key-value pairs attached to log entries that Loki indexes. They are used for:
- **Filtering**: Quickly find logs from specific services, environments, or components
- **Grouping**: Aggregate logs by common attributes
- **Querying**: Use LogQL to query logs based on label combinations

## Setting Labels

### Global Labels

Set labels that apply to all log entries from your logger:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLokiHost("http://localhost:3100"),
    loki.WithLabels(types.Labels{
        "environment": "production",
        "region":      "us-east-1",
        "team":        "backend",
    }),
)
```

### Contextual Labels with WithLabels

Create child loggers with additional labels for specific components:

```go
// Logger for user service component
userLogger := logger.WithLabels(types.Labels{
    "component": "user-service",
    "version":   "2.1.0",
})

// Logger for payment component
paymentLogger := logger.WithLabels(types.Labels{
    "component": "payment",
    "provider":  "stripe",
})
```

**Note**: `WithLabels()` only accepts string values as labels. Non-string values are ignored and a warning is logged.

## Label Cardinality

**Cardinality** is the number of unique label combinations. High cardinality severely impacts Loki's performance.

### What is High Cardinality?

High cardinality occurs when labels have many unique values:

```go
// ❌ BAD: High cardinality (millions of unique values)
logger.WithLabels(types.Labels{
    "user_id":       "12345",        // Unique per user
    "request_id":    "abc-def-ghi",  // Unique per request
    "timestamp":     "2024-01-15",   // Changes frequently
    "session_token": "xyz123",       // Unique per session
})
```

### Why is High Cardinality Bad?

- **Performance**: Loki creates a separate stream for each unique label combination
- **Memory**: Each stream consumes memory and storage
- **Query Speed**: More streams = slower queries
- **Resource Cost**: Increased CPU and disk usage

### Label Cardinality Best Practices

#### ✅ Good: Low Cardinality Labels

Use labels with limited, predictable values:

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLabels(types.Labels{
        "environment": "production",     // 3-4 values: dev, staging, prod
        "region":      "us-east-1",      // ~10-20 regions
        "datacenter":  "dc-1",           // Limited datacenters
        "cluster":     "prod-cluster-1", // Few clusters
    }),
)

// Component-level labels
apiLogger := logger.WithLabels(types.Labels{
    "component": "api",        // Limited components
    "service":   "auth",       // Limited services
    "version":   "v2.1.0",     // Few versions at a time
})
```

**Recommended label cardinality**:
- Per label: < 20-50 unique values
- Total combinations: < 100-200 streams

#### ❌ Bad: High Cardinality Labels

Never use labels with unbounded unique values:

```go
// ❌ DON'T DO THIS
logger.WithLabels(types.Labels{
    "user_id":       "12345",           // Millions of users
    "request_id":    "abc-def",         // Every request
    "session_id":    "xyz-123",         // Every session
    "ip_address":    "192.168.1.1",     // Many IPs
    "email":         "user@example.com", // Per user
    "timestamp":     time.Now().String(), // Always unique
    "trace_id":      "...",              // Per request
})
```

## Labels vs Fields

Understanding when to use labels versus structured fields is crucial:

| Aspect | Labels | Fields |
|--------|--------|--------|
| **Purpose** | Indexing and filtering | Structured log data |
| **Indexed** | Yes (by Loki) | No |
| **Query Performance** | Fast filtering | Requires parsing |
| **Cardinality Impact** | HIGH (creates streams) | None |
| **Examples** | environment, service, region | user_id, request_id, duration |

### Use Labels For:

- **Static attributes**: Environment, region, cluster
- **Low cardinality dimensions**: Service name, component, version
- **Filtering criteria**: Things you query by frequently
- **Grouping dimensions**: Aggregation boundaries

### Use Fields For:

- **Request-specific data**: request_id, user_id, session_id
- **Variable data**: Durations, counts, error messages
- **High cardinality attributes**: IPs, emails, tokens
- **Structured context**: Any data you want in JSON format

### Example: Labels + Fields

```go
// ✅ GOOD: Proper separation
logger.Info(ctx, "User logged in", types.Fields{
    // Fields (high cardinality, not indexed)
    "user_id":      12345,
    "email":        "user@example.com",
    "ip_address":   "192.168.1.1",
    "request_id":   "abc-123",
    "duration_ms":  45,
    "user_agent":   "Mozilla/5.0...",
})

// Labels are set at logger level (low cardinality, indexed)
authLogger := logger.WithLabels(types.Labels{
    "service":   "auth",
    "component": "login",
})
```

## Querying with Labels

In Grafana/Loki, you can query by labels:

```logql
# All logs from production API service
{app="my-service", environment="production", service="api"}

# Auth component logs only
{app="my-service", component="authentication"}

# Multi-label query
{app="my-service", environment="production", service="api", region="us-east-1"}
```

## Label Naming Conventions

Follow these conventions for consistency:

### Recommended Names

- **environment**: `dev`, `staging`, `production`
- **region**: `us-east-1`, `eu-west-1`, `ap-southeast-1`
- **service**: Service name within your app
- **component**: Subcomponent or module name
- **version**: Semantic version (e.g., `v2.1.0`)
- **cluster**: Cluster identifier
- **datacenter**: Physical location

### Naming Guidelines

- Use **lowercase** with **underscores**: `service_name`, not `ServiceName` or `serviceName`
- Be **specific but not verbose**: `db_connection`, not `database_connection_pool_manager`
- Avoid **redundancy**: `user_service`, not `user_service_service`
- Use **singular form**: `environment`, not `environments`

## Performance Tips

1. **Limit total labels**: 3-6 labels per stream is ideal
2. **Keep values bounded**: Each label should have < 50 unique values
3. **Monitor cardinality**: Use Loki's cardinality API
4. **Use label matchers**: Query specific label combinations
5. **Avoid dynamic labels**: Never use timestamps, IDs, or random values

For more information:
- [Loki Best Practices](https://grafana.com/docs/loki/latest/best-practices/)
