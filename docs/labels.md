# Labels

Best practices for using Loki labels effectively.

## What are Labels?

Labels are **indexed** key-value pairs that Loki uses for:
- **Filtering** - Find logs from specific services, environments, or components
- **Grouping** - Aggregate logs by common attributes
- **Querying** - Use LogQL to query by label combinations

**Critical**: Labels create separate streams in Loki. High cardinality = poor performance.

## Setting Labels

### Global Labels

```go
logger, err := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLabels(types.Labels{
        "environment": "production",
        "region":      "us-east-1",
        "team":        "backend",
    }),
)
```

### Component Labels with WithLabels

```go
// User service logger
userLogger := logger.WithLabels(types.Labels{
    "component": "user-service",
    "version":   "2.1.0",
})

// Payment logger
paymentLogger := logger.WithLabels(types.Labels{
    "component": "payment",
    "provider":  "stripe",
})
```

**Note**: `WithLabels()` only accepts string values. Non-string values are ignored.

## Label Cardinality

**Cardinality** = number of unique label combinations

High cardinality **severely** impacts Loki performance.

### Why High Cardinality is Bad

| Impact | Description |
|--------|-------------|
| **Performance** | Loki creates separate stream for each unique combination |
| **Memory** | Each stream consumes memory and storage |
| **Query Speed** | More streams = slower queries |
| **Cost** | Increased CPU, disk usage, and cloud costs |

### Cardinality Guidelines

| Guideline | Recommendation |
|-----------|----------------|
| **Per label** | < 20-50 unique values |
| **Total combinations** | < 100-200 streams |
| **Total labels per stream** | 3-6 labels |

###  Good: Low Cardinality

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
    "component": "api",      // Limited components
    "service":   "auth",     // Limited services
    "version":   "v2.1.0",   // Few versions at a time
})
```

### L Bad: High Cardinality

```go
// DON'T DO THIS - Creates millions of streams
logger.WithLabels(types.Labels{
    "user_id":       "12345",           // L Millions of users
    "request_id":    "abc-def",         // L Every request unique
    "session_id":    "xyz-123",         // L Every session unique
    "ip_address":    "192.168.1.1",     // L Many IPs
    "email":         "user@example.com", // L Per user
    "timestamp":     time.Now().String(), // L Always unique
    "trace_id":      "...",              // L Per request
})
```

## Labels vs Fields

| Aspect | Labels | Fields |
|--------|--------|--------|
| **Purpose** | Indexing and filtering | Structured log data |
| **Indexed** |  Yes (by Loki) | L No |
| **Query Performance** | Fast filtering | Requires parsing |
| **Cardinality Impact** | HIGH (creates streams) | None |
| **Searchable** | Instantly | Via log parsing |
| **Examples** | `environment`, `service`, `region` | `user_id`, `request_id`, `duration` |

### Use Labels For

| Category | Examples | Reason |
|----------|----------|--------|
| **Static attributes** | `environment`, `region`, `cluster` | Don't change often |
| **Low cardinality** | `service`, `component`, `version` | Limited unique values |
| **Filtering criteria** | Things you query by frequently | Fast lookups |
| **Grouping dimensions** | Aggregation boundaries | Stream organization |

### Use Fields For

| Category | Examples | Reason |
|----------|----------|--------|
| **Request-specific** | `request_id`, `user_id`, `session_id` | Unique per request |
| **Variable data** | Durations, counts, error messages | Constantly changing |
| **High cardinality** | IPs, emails, tokens, UUIDs | Too many unique values |
| **Structured context** | Any data in JSON format | Not for indexing |

### Example: Proper Separation

```go
//  GOOD: Labels for low cardinality, fields for high cardinality
authLogger := logger.WithLabels(types.Labels{
    "service":   "auth",     // Low cardinality label
    "component": "login",    // Low cardinality label
})

authLogger.Info(ctx, "User logged in", types.Fields{
    // High cardinality fields (not indexed)
    "user_id":      12345,
    "email":        "user@example.com",
    "ip_address":   "192.168.1.1",
    "request_id":   "abc-123",
    "duration_ms":  45,
    "user_agent":   "Mozilla/5.0...",
})
```

## Querying with Labels

LogQL queries in Grafana/Loki:

```logql
# All logs from production API service
{app="my-service", environment="production", service="api"}

# Auth component only
{app="my-service", component="authentication"}

# Multi-label query
{app="my-service", environment="production", region="us-east-1"}

# With filter
{app="my-service", environment="production"} |= "error"
```

## Label Naming Conventions

### Recommended Label Names

| Label | Values | Usage |
|-------|--------|-------|
| `environment` | `dev`, `staging`, `production` | Deployment environment |
| `region` | `us-east-1`, `eu-west-1` | Geographic region |
| `service` | `api`, `worker`, `auth` | Service name |
| `component` | `database`, `cache`, `queue` | Subcomponent |
| `version` | `v2.1.0`, `v2.2.0` | Semantic version |
| `cluster` | `prod-cluster-1` | Cluster identifier |
| `datacenter` | `dc-1`, `dc-2` | Physical location |

### Naming Guidelines

| Guideline | Example | Anti-pattern |
|-----------|---------|--------------|
| Use **lowercase + underscores** | `service_name` | `ServiceName`, `serviceName` |
| Be **specific but concise** | `db_connection` | `database_connection_pool_manager` |
| Avoid **redundancy** | `user_service` | `user_service_service` |
| Use **singular form** | `environment` | `environments` |
| Keep **short** | `env` or `environment` | `application_environment_type` |

## Common Patterns

### Pattern 1: Global + Component Labels

```go
// Global labels (apply to all logs)
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("my-service"),
    loki.WithLabels(types.Labels{
        "environment": "production",
        "region":      "us-east-1",
    }),
)

// Component-specific labels
authLogger := logger.WithLabels(types.Labels{"component": "auth"})
dbLogger := logger.WithLabels(types.Labels{"component": "database"})
```

### Pattern 2: Service + Version Labels

```go
logger, _ := loki.New(
    loki.DefaultConfig(),
    loki.WithAppName("user-api"),
    loki.WithLabels(types.Labels{
        "service":     "user-api",
        "version":     "v2.1.0",
        "environment": "production",
    }),
)
```

### Pattern 3: Avoid Request-Level Labels

```go
// L DON'T: Request ID as label (high cardinality)
reqLogger := logger.WithLabels(types.Labels{
    "request_id": requestID,  // Creates new stream per request!
})

//  DO: Request ID as field
logger.Info(ctx, "Request completed", types.Fields{
    "request_id": requestID,  // Searchable but not indexed
})
```

## Performance Tips

1. **Limit total labels**: 3-6 labels per stream ideal
2. **Keep values bounded**: Each label < 50 unique values
3. **Monitor cardinality**: Use Loki's cardinality API
4. **Use label matchers**: Query specific combinations
5. **Never use dynamic labels**: No timestamps, IDs, or random values
6. **Prefer fields for variables**: High cardinality data goes in fields

## Troubleshooting High Cardinality

### Symptoms

- Slow queries in Grafana
- High memory usage in Loki
- "too many streams" errors
- Increased Loki disk usage

### Diagnosis

```bash
# Check cardinality (Loki API)
curl http://localhost:3100/loki/api/v1/labels

# Check label values
curl http://localhost:3100/loki/api/v1/label/<label_name>/values
```

### Fix

1. Identify labels with > 50 unique values
2. Move high-cardinality labels to fields
3. Reduce label combinations
4. Use static values instead of dynamic

## Best Practices Summary

###  Do

```go
// Use low cardinality labels
loki.WithLabels(types.Labels{
    "environment": "production",  // 3-4 values
    "service":     "api",          // 10-20 services
    "region":      "us-east-1",    // 10-20 regions
})

// Put dynamic data in fields
logger.Info(ctx, "Event", types.Fields{
    "user_id":    12345,
    "request_id": "abc-123",
})

// Only string values in labels
loki.WithLabels(types.Labels{
    "version": "2.1.0",  // String
})
```

### L Don't

```go
// Don't use high cardinality labels
loki.WithLabels(types.Labels{
    "user_id":    "12345",     // Millions of users
    "request_id": "abc-123",   // Every request
    "timestamp":  "...",       // Always changing
})

// Don't use non-string values in labels
loki.WithLabels(types.Labels{
    "user_id": 12345,  // Ignored (not string)
    "active":  true,   // Ignored (not string)
})

// Don't create labels per request
reqLogger := logger.WithLabels(types.Labels{
    "request_id": uuid.New().String(),  // Creates new stream!
})
```

## Resources

- [Loki Best Practices](https://grafana.com/docs/loki/latest/best-practices/)
- [Understanding Label Cardinality](https://grafana.com/docs/loki/latest/best-practices/#use-dynamic-labels-sparingly)
