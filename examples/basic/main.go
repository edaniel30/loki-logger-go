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
	// Create a basic logger that sends logs to both console and Loki
	logger, err := loki.New(
		loki.DefaultConfig(),
		loki.WithAppName("basic-example"),
		loki.WithLokiHost(os.Getenv("LOKI_HOST")),
		loki.WithLokiBasicAuth(os.Getenv("LOKI_USERNAME"), os.Getenv("LOKI_PASSWORD")),
		loki.WithLogLevel(types.LevelDebug), // Show all log levels including Debug
	)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	// Log at different levels
	logger.Info(ctx, "Application started", nil)
	logger.Debug(ctx, "Debug information", types.Fields{
		"environment": "development",
	})

	// Log with structured fields
	logger.Info(ctx, "User action", types.Fields{
		"user_id": 12345,
		"action":  "login",
		"ip":      "192.168.1.1",
	})

	// Simulate some work
	time.Sleep(1 * time.Second)

	logger.Warn(ctx, "Warning message", types.Fields{
		"reason": "high memory usage",
		"usage":  85.5,
	})

	// Log an error
	logger.Error(ctx, "Failed to process request", types.Fields{
		"error":      "connection timeout",
		"request_id": "abc-123",
		"retries":    3,
	})

	logger.Info(ctx, "Application shutting down", nil)
}
