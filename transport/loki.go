package transport

import (
	"context"
	"fmt"
	"github.com/edaniel30/loki-logger-go/internal/client"
	"sync"
	"time"
)

// LokiTransport sends log entries to a Grafana Loki server.
// It batches entries for efficiency and flushes periodically.
type LokiTransport struct {
	client          *client.Client
	buffer          []*Entry
	batchSize       int
	flushInterval   time.Duration
	flushTimeout    time.Duration
	shutdownTimeout time.Duration
	mu              sync.Mutex
	stopCh          chan struct{}
	flushCh         chan struct{}
	doneCh          chan struct{} // signals when background flusher is done
	wg              sync.WaitGroup
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

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// FlushTimeout is the timeout for flush operations
	FlushTimeout time.Duration

	// ShutdownTimeout is the timeout for graceful shutdown
	ShutdownTimeout time.Duration
}

// NewLokiTransport creates a new Loki transport with the given configuration.
func NewLokiTransport(config LokiTransportConfig) *LokiTransport {
	lt := &LokiTransport{
		client:          client.NewClient(config.LokiURL, config.LokiUsername, config.LokiPassword, config.Timeout, config.MaxRetries),
		buffer:          make([]*Entry, 0, config.BatchSize),
		batchSize:       config.BatchSize,
		flushInterval:   config.FlushInterval,
		flushTimeout:    config.FlushTimeout,
		shutdownTimeout: config.ShutdownTimeout,
		stopCh:          make(chan struct{}),
		flushCh:         make(chan struct{}, 1),
		doneCh:          make(chan struct{}),
	}

	// Start background flusher
	lt.wg.Add(1)
	go lt.backgroundFlusher()

	return lt
}

func (lt *LokiTransport) Name() string {
	return "loki"
}

// Write adds entries to the buffer and flushes if batch size is reached.
func (lt *LokiTransport) Write(ctx context.Context, entries ...*Entry) error {
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
	lt.buffer = make([]*Entry, 0, lt.batchSize)
	lt.mu.Unlock()

	// Convert transport.Entry to client.Entry
	clientEntries := make([]*client.Entry, len(toSend))
	for i, entry := range toSend {
		clientEntries[i] = &client.Entry{
			Level:     entry.Level,
			Message:   entry.Message,
			Fields:    entry.Fields,
			Timestamp: entry.Timestamp,
			Labels:    entry.Labels,
		}
	}

	// Send to Loki
	if err := lt.client.Push(ctx, clientEntries); err != nil {
		return fmt.Errorf("failed to push to Loki: %w", err)
	}

	return nil
}

// Close stops the background flusher and flushes remaining entries.
// It waits up to the configured ShutdownTimeout for graceful shutdown.
// If shutdown takes longer, it returns an error but the goroutine will continue to completion.
func (lt *LokiTransport) Close() error {
	// Signal shutdown
	close(lt.stopCh)

	// Wait for background flusher to finish with timeout
	shutdownTimer := time.NewTimer(lt.shutdownTimeout)
	defer shutdownTimer.Stop()

	select {
	case <-lt.doneCh:
		// Clean shutdown completed
		return nil
	case <-shutdownTimer.C:
		// Timeout waiting for shutdown
		return fmt.Errorf("timeout waiting for background flusher to shutdown after %v", lt.shutdownTimeout)
	}
}

// backgroundFlusher periodically flushes buffered entries.
func (lt *LokiTransport) backgroundFlusher() {
	defer lt.wg.Done()
	defer close(lt.doneCh) // Signal completion when done

	ticker := time.NewTicker(lt.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), lt.flushTimeout)
			_ = lt.Flush(ctx) // Ignore errors in background flush
			cancel()

		case <-lt.flushCh:
			ctx, cancel := context.WithTimeout(context.Background(), lt.flushTimeout)
			_ = lt.Flush(ctx)
			cancel()

		case <-lt.stopCh:
			// Perform final flush before exiting
			ctx, cancel := context.WithTimeout(context.Background(), lt.flushTimeout)
			_ = lt.Flush(ctx) // Ignore error - Close() will report timeout if needed
			cancel()
			return
		}
	}
}
