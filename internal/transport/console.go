package transport

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/edaniel30/loki-logger-go/types"
)

// ANSI color codes for console output
const (
	colorReset   = "\033[0m"
	colorCyan    = "\033[36m" // Debug
	colorGreen   = "\033[32m" // Info
	colorYellow  = "\033[33m" // Warn
	colorRed     = "\033[31m" // Error
	colorMagenta = "\033[35m" // Fatal
)

// ConsoleTransport writes log entries to stdout with colored output.
// This is a minimal implementation with sensible defaults:
// - Always writes to stdout
// - Always includes timestamp
// - Always includes colors
// - Thread-safe for concurrent use
type ConsoleTransport struct {
	mu sync.Mutex
}

// NewConsoleTransport creates a new console transport with default settings.
func NewConsoleTransport() *ConsoleTransport {
	return &ConsoleTransport{}
}

func (ct *ConsoleTransport) Name() string {
	return "console"
}

func (ct *ConsoleTransport) Write(ctx context.Context, entries ...*types.Entry) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	for _, entry := range entries {
		formatted := ct.format(entry)

		if _, err := os.Stdout.WriteString(formatted); err != nil {
			return fmt.Errorf("failed to write to console: %w", err)
		}
	}

	return nil
}

// format converts a log entry to text format with colors and timestamp.
func (ct *ConsoleTransport) format(entry *types.Entry) string {
	var b strings.Builder

	// Timestamp (always included)
	b.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05"))
	b.WriteString(" ")

	// Level with color
	b.WriteString(ct.formatLevel(entry.Level))
	b.WriteString(" ")

	// Message
	b.WriteString(entry.Message)

	// Fields (if any)
	if len(entry.Fields) > 0 {
		b.WriteString(" ")
		b.WriteString(ct.formatFields(entry.Fields))
	}

	b.WriteString("\n")

	return b.String()
}

// formatLevel returns a colored level string.
func (ct *ConsoleTransport) formatLevel(level types.Level) string {
	levelStr := strings.ToUpper(level.String())

	var color string
	switch level {
	case types.LevelDebug:
		color = colorCyan
	case types.LevelInfo:
		color = colorGreen
	case types.LevelWarn:
		color = colorYellow
	case types.LevelError:
		color = colorRed
	case types.LevelFatal:
		color = colorMagenta
	default:
		color = colorReset
	}

	return fmt.Sprintf("%s[%s]%s", color, levelStr, colorReset)
}

// formatFields formats structured fields as key=value pairs, sorted alphabetically.
func (ct *ConsoleTransport) formatFields(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return strings.Join(parts, " ")
}

// Flush returns nil because console writes are not buffered.
// Logs are written immediately to stdout in Write().
func (ct *ConsoleTransport) Flush(ctx context.Context) error {
	return nil
}

// Close returns nil because console transport has no resources to release.
// stdout is managed by the OS and should not be closed.
func (ct *ConsoleTransport) Close() error {
	return nil
}
