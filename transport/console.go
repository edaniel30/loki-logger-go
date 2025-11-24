package transport

import (
	"context"
	"fmt"
	"io"
	"loki-logger-go/formatter"
	"os"
	"sync"
)

// ConsoleTransport writes log entries to stdout or stderr.
type ConsoleTransport struct {
	formatter formatter.Formatter
	writer    io.Writer
	mu        sync.Mutex
}

type ConsoleOption func(*ConsoleTransport)

func NewConsoleTransport(opts ...ConsoleOption) *ConsoleTransport {
	ct := &ConsoleTransport{
		formatter: formatter.NewTextFormatter(),
		writer:    os.Stdout,
	}

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}

func WithFormatter(f formatter.Formatter) ConsoleOption {
	return func(ct *ConsoleTransport) {
		ct.formatter = f
	}
}

func WithWriter(w io.Writer) ConsoleOption {
	return func(ct *ConsoleTransport) {
		ct.writer = w
	}
}

func (ct *ConsoleTransport) Write(ctx context.Context, entries ...*Entry) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	for _, entry := range entries {
		formatterEntry := &formatter.Entry{
			Level:     entry.Level,
			Message:   entry.Message,
			Fields:    entry.Fields,
			Timestamp: entry.Timestamp,
			Labels:    entry.Labels,
		}

		data, err := ct.formatter.Format(formatterEntry)
		if err != nil {
			return fmt.Errorf("failed to format entry: %w", err)
		}

		if _, err := ct.writer.Write(data); err != nil {
			return fmt.Errorf("failed to write to console: %w", err)
		}
	}

	return nil
}

func (ct *ConsoleTransport) Flush(ctx context.Context) error {
	return nil
}

func (ct *ConsoleTransport) Close() error {
	return nil
}
