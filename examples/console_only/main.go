package main

import (
	"context"
	"log"
	"time"

	loki "github.com/edaniel30/loki-logger-go"
	"github.com/edaniel30/loki-logger-go/types"
)

func main() {
	// Create a console-only logger (no Loki connection required)
	// This is useful for local development and testing
	logger, err := loki.New(
		loki.DefaultConfig(),
		loki.WithAppName("console-only-example"),
		loki.WithOnlyConsole(true), // Disable Loki transport
		loki.WithLogLevel(types.LevelDebug),
	)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	ctx := context.Background()

	logger.Info(ctx, "Running in console-only mode", nil)
	logger.Debug(ctx, "This is a debug message", types.Fields{
		"mode": "development",
	})

	// Simulate some processing
	for i := 1; i <= 5; i++ {
		logger.Info(ctx, "Processing item", types.Fields{
			"item_number": i,
			"total":       5,
		})
		time.Sleep(500 * time.Millisecond)
	}

	logger.Info(ctx, "Processing complete", nil)
}
