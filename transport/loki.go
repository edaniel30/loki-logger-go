package transport

import (
	"context"
	"fmt"
	"loki-logger-go/internal/client"
	"sync"
	"time"
)

// LokiTransport sends log entries to a Grafana Loki server.
// It batches entries for efficiency and flushes periodically.
type LokiTransport struct {
	client        *client.Client
	buffer        []*Entry
	batchSize     int
	flushInterval time.Duration
	mu            sync.Mutex
	stopCh        chan struct{}
	flushCh       chan struct{}
	wg            sync.WaitGroup
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
}

// NewLokiTransport creates a new Loki transport with the given configuration.
func NewLokiTransport(config LokiTransportConfig) *LokiTransport {
	lt := &LokiTransport{
		client:        client.NewClient(config.LokiURL, config.LokiUsername, config.LokiPassword, config.Timeout, config.MaxRetries),
		buffer:        make([]*Entry, 0, config.BatchSize),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		stopCh:        make(chan struct{}),
		flushCh:       make(chan struct{}, 1),
	}

	// Start background flusher
	lt.wg.Add(1)
	go lt.backgroundFlusher()

	return lt
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

	// Copy buffer and reset
	toSend := make([]*Entry, len(lt.buffer))
	copy(toSend, lt.buffer)
	lt.buffer = lt.buffer[:0]
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
func (lt *LokiTransport) Close() error {
	close(lt.stopCh)
	lt.wg.Wait()

	// Final flush
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return lt.Flush(ctx)
}

// backgroundFlusher periodically flushes buffered entries.
func (lt *LokiTransport) backgroundFlusher() {
	defer lt.wg.Done()

	ticker := time.NewTicker(lt.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = lt.Flush(ctx) // Ignore errors in background flush
			cancel()

		case <-lt.flushCh:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = lt.Flush(ctx)
			cancel()

		case <-lt.stopCh:
			return
		}
	}
}
