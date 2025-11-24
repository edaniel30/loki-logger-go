# Labels and Indexing in Loki Logger Go

## Overview

This document explains how labels work in this library and how they're indexed in Grafana Loki.

## What Gets Indexed?

### Automatic Labels (Always Added)

Every log entry automatically includes these indexed labels:

| Label     | Description                    | Example Values        |
|-----------|--------------------------------|-----------------------|
| `app`     | Application name               | `my-app`, `api-gateway` |
| `level`   | Log level ✨                   | `debug`, `info`, `warn`, `error`, `fatal` |

### Custom Labels (Optional)

You can add custom labels when creating the logger:

```go
logger, err := loki.New(
    models.DefaultConfig(),
    models.WithAppName("my-app"),
    models.WithLabels(map[string]string{
        "environment": "production",
        "region":      "us-east-1",
        "datacenter":  "dc1",
    }),
)
```

## Performance Considerations

### Good Label Candidates ✅

- **environment**: `development`, `staging`, `production` (3 values)
- **region**: `us-east-1`, `us-west-2`, `eu-west-1` (5-10 values)
- **datacenter**: `dc1`, `dc2`, `dc3` (3-5 values)
- **cluster**: `k8s-prod-01`, `k8s-prod-02` (5-10 values)
- **level**: `debug`, `info`, `warn`, `error`, `fatal` (5 values)

### Bad Label Candidates ❌

- **user_id**: `12345`, `67890`, ... (thousands/millions of values)
- **request_id**: UUID (millions of unique values)
- **ip_address**: IP addresses (thousands of values)
- **timestamp**: Changes every millisecond
- **session_id**: UUID (millions of unique values)

## Label Cardinality Impact

Loki creates a **separate stream** for each unique combination of labels.

### Example: Good Cardinality

```
Labels: app, environment, level

Combinations:
- app=my-app, environment=prod, level=error
- app=my-app, environment=prod, level=info
- app=my-app, environment=staging, level=error
...

Total streams: 3 environments × 5 levels = 15 streams ✅
```

### Example: Bad Cardinality

```
Labels: app, user_id, level

Combinations:
- app=my-app, user_id=1, level=error
- app=my-app, user_id=2, level=error
- app=my-app, user_id=3, level=error
...

Total streams: 1,000,000 users × 5 levels = 5M streams ❌
```

## Tips

1. **Keep it simple**: Start with just `app` and `level`
2. **Monitor cardinality**: Check Loki metrics for number of streams
3. **Use fields for high-cardinality data**: User IDs, IPs, etc.
4. **Test queries**: Use Grafana Explore to test label combinations
5. **Document your labels**: Keep a list of all labels used across services

## Questions?

- How many labels should I use? → Start with 3-5, max 10
- What's good cardinality? → < 10,000 total streams
- Can I change labels later? → Yes, but creates new streams
- Should I use level as label? → Yes! It's now automatic ✅
