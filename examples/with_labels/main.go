// Set environment variables before running:
// export LOKI_HOST="http://localhost:3100"
// export LOKI_USERNAME=""  # Optional, leave empty if no auth
// export LOKI_PASSWORD=""  # Optional, leave empty if no auth

package main

import (
	"context"
	"log"
	"os"
	"time"

	loki "github.com/edaniel30/loki-logger-go"
	"github.com/edaniel30/loki-logger-go/types"
)

func main() {
	// Create a logger with global labels
	logger, err := loki.New(
		loki.DefaultConfig(),
		loki.WithAppName("labels-example"),
		loki.WithLokiHost(os.Getenv("LOKI_HOST")),
		loki.WithLokiBasicAuth(os.Getenv("LOKI_USERNAME"), os.Getenv("LOKI_PASSWORD")),
		loki.WithLabels(types.Labels{
			"environment": "production",
			"region":      "us-east-1",
			"team":        "backend",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	// Parent logger with global labels
	logger.Info(ctx, "Application started", types.Fields{
		"version": "1.0.0",
	})

	// Create child logger for API component
	apiLogger := logger.WithLabels(types.Labels{
		"component": "api",
		"service":   "users",
	})

	apiLogger.Info(ctx, "API request received", types.Fields{
		"method":     "GET",
		"path":       "/api/users",
		"request_id": "req-001",
	})

	// Simulate API processing
	time.Sleep(100 * time.Millisecond)

	apiLogger.Info(ctx, "API request completed", types.Fields{
		"request_id":  "req-001",
		"status_code": 200,
		"duration_ms": 95,
	})

	// Create child logger for database component
	dbLogger := logger.WithLabels(types.Labels{
		"component": "database",
		"service":   "postgres",
	})

	dbLogger.Info(ctx, "Database query executed", types.Fields{
		"query":        "SELECT * FROM users",
		"duration_ms":  45,
		"rows_fetched": 150,
	})

	// Create child logger for cache component
	cacheLogger := logger.WithLabels(types.Labels{
		"component": "cache",
		"service":   "redis",
	})

	cacheLogger.Info(ctx, "Cache hit", types.Fields{
		"key": "user:12345",
		"ttl": 3600,
	})

	// Each child logger has its own labels, plus inherits parent labels
	// In Grafana, you can query:
	// {app="labels-example", component="api"} - for API logs only
	// {app="labels-example", component="database"} - for DB logs only
	// {app="labels-example", environment="production"} - for all logs

	logger.Info(ctx, "Application shutting down", nil)
}
