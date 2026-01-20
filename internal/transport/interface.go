package transport

import (
	"context"

	"github.com/edaniel30/loki-logger-go/types"
)

// Transport defines how log entries are sent to their destination.
// Implementations must be thread-safe for concurrent use.
type Transport interface {
	// Name returns the name of this transport (e.g., "console", "loki")
	Name() string

	// Write sends one or more log entries to the transport destination.
	// Returns an error if the write operation fails.
	Write(ctx context.Context, entries ...*types.Entry) error

	// Flush ensures all buffered entries are sent to the destination.
	// Should be called before application shutdown.
	Flush(ctx context.Context) error

	// Close releases any resources held by the transport.
	// After calling Close, the transport should not be used.
	Close() error
}
