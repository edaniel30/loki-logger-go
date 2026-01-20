package transport

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/client"
	"github.com/edaniel30/loki-logger-go/types"
)

// LokiTransport sends log entries to a Grafana Loki server.
// It batches entries for efficiency and flushes periodically.
type LokiTransport struct {
	client        *client.Client
	buffer        []*types.Entry
	batchSize     int
	flushInterval time.Duration
	timeout       time.Duration
	mu            sync.Mutex
	stopCh        chan struct{}
	doneCh        chan struct{} // signals when background flusher is done
}

// LokiTransportConfig configures a LokiTransport instance.
type LokiTransportConfig struct {
	// LokiURL is the base URL of the Loki server
	LokiURL string

	// LokiUsername is the username for basic auth (optional)
	LokiUsername string

	// LokiPassword is the password for basic auth (optional)
	LokiPassword string

	// BatchSize is the number of entries to batch before sending
	BatchSize int

	// FlushInterval is how often to flush regardless of batch size
	FlushInterval time.Duration

	// MaxRetries is the number of retry attempts for failed requests
	MaxRetries int

	// Timeout is the timeout for all operations (HTTP requests, flush, shutdown)
	Timeout time.Duration
}

// NewLokiTransport creates a new Loki transport with the given configuration.
func NewLokiTransport(config LokiTransportConfig) *LokiTransport {
	lt := &LokiTransport{
		client:        client.NewClient(config.LokiURL, config.LokiUsername, config.LokiPassword, config.Timeout, config.MaxRetries),
		buffer:        make([]*types.Entry, 0, config.BatchSize),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		timeout:       config.Timeout,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}

	// Start background flusher
	go lt.backgroundFlusher()

	return lt
}

func (lt *LokiTransport) Name() string {
	return "loki"
}

// Write adds entries to the buffer and flushes if batch size is reached.
func (lt *LokiTransport) Write(ctx context.Context, entries ...*types.Entry) error {
	lt.mu.Lock()
	lt.buffer = append(lt.buffer, entries...)
	shouldFlush := len(lt.buffer) >= lt.batchSize
	lt.mu.Unlock()

	if shouldFlush {
		return lt.Flush(ctx)
	}

	return nil
}

// Flush sends all buffered entries to Loki immediately.
func (lt *LokiTransport) Flush(ctx context.Context) error {
	lt.mu.Lock()
	if len(lt.buffer) == 0 {
		lt.mu.Unlock()
		return nil
	}

	// Take ownership of current buffer and allocate new one
	// This avoids race conditions by not reusing the underlying array
	toSend := lt.buffer
	lt.buffer = make([]*types.Entry, 0, lt.batchSize)
	lt.mu.Unlock()

	// Send to Loki - no conversion needed, both use types.Entry
	if err := lt.client.Push(ctx, toSend); err != nil {
		return fmt.Errorf("failed to push to Loki: %w", err)
	}

	return nil
}

// Close stops the background flusher and flushes remaining entries.
// It waits up to the configured Timeout for graceful shutdown.
func (lt *LokiTransport) Close() error {
	// Signal shutdown
	close(lt.stopCh)

	// Wait for background flusher to finish with timeout
	shutdownTimer := time.NewTimer(lt.timeout)
	defer shutdownTimer.Stop()

	select {
	case <-lt.doneCh:
		// Clean shutdown completed
		return nil
	case <-shutdownTimer.C:
		// Timeout waiting for shutdown
		return fmt.Errorf("timeout waiting for background flusher to shutdown after %v", lt.timeout)
	}
}

// backgroundFlusher periodically flushes buffered entries.
func (lt *LokiTransport) backgroundFlusher() {
	defer close(lt.doneCh) // Signal completion when done

	ticker := time.NewTicker(lt.flushInterval)
	defer ticker.Stop()

	// Helper to flush with timeout
	doFlush := func() {
		ctx, cancel := context.WithTimeout(context.Background(), lt.timeout)
		defer cancel()
		_ = lt.Flush(ctx) // Ignore errors in background flush
	}

	for {
		select {
		case <-ticker.C:
			doFlush()

		case <-lt.stopCh:
			// Perform final flush before exiting
			doFlush()
			return
		}
	}
}
