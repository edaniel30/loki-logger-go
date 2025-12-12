package transport

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
)

// ConsoleTransport writes log entries to stdout or stderr.
type ConsoleTransport struct {
	disableColors    bool
	disableTimestamp bool
	writer           io.Writer
	mu               sync.Mutex
}

type ConsoleOption func(*ConsoleTransport)

func NewConsoleTransport(opts ...ConsoleOption) *ConsoleTransport {
	ct := &ConsoleTransport{
		disableColors:    false,
		disableTimestamp: false,
		writer:           os.Stdout,
	}

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}

func WithDisableColors(disable bool) ConsoleOption {
	return func(ct *ConsoleTransport) {
		ct.disableColors = disable
	}
}

func WithDisableTimestamp(disable bool) ConsoleOption {
	return func(ct *ConsoleTransport) {
		ct.disableTimestamp = disable
	}
}

func WithWriter(w io.Writer) ConsoleOption {
	return func(ct *ConsoleTransport) {
		ct.writer = w
	}
}

func (ct *ConsoleTransport) Name() string {
	return "console"
}

func (ct *ConsoleTransport) Write(ctx context.Context, entries ...*Entry) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	for _, entry := range entries {
		formatted := ct.format(entry)

		if _, err := ct.writer.Write([]byte(formatted)); err != nil {
			return fmt.Errorf("failed to write to console: %w", err)
		}
	}

	return nil
}

// format converts a log entry to text format with colors
func (ct *ConsoleTransport) format(entry *Entry) string {
	var b strings.Builder

	if !ct.disableTimestamp {
		b.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05"))
		b.WriteString(" ")
	}

	levelStr := ct.formatLevel(entry.Level)
	b.WriteString(levelStr)
	b.WriteString(" ")

	b.WriteString(entry.Message)

	if len(entry.Fields) > 0 {
		b.WriteString(" ")
		b.WriteString(ct.formatFields(entry.Fields))
	}

	b.WriteString("\n")

	return b.String()
}

func (ct *ConsoleTransport) formatLevel(level string) string {
	levelStr := strings.ToUpper(level)

	if ct.disableColors {
		return fmt.Sprintf("[%s]", levelStr)
	}

	var color string
	switch level {
	case "debug":
		color = "\033[36m" // Cyan
	case "info":
		color = "\033[32m" // Green
	case "warn":
		color = "\033[33m" // Yellow
	case "error":
		color = "\033[31m" // Red
	case "fatal":
		color = "\033[35m" // Magenta
	default:
		color = "\033[0m" // Reset
	}

	return fmt.Sprintf("%s[%s]\033[0m", color, levelStr)
}

func (ct *ConsoleTransport) formatFields(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return strings.Join(parts, " ")
}

func (ct *ConsoleTransport) Flush(ctx context.Context) error {
	return nil
}

func (ct *ConsoleTransport) Close() error {
	return nil
}
