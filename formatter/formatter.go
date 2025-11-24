package formatter

import (
	"time"
)

// Entry represents a single log record with all its associated data.
// This is duplicated here to avoid import cycles.
type Entry struct {
	Level     string
	Message   string
	Fields    map[string]any
	Timestamp time.Time
	Labels    map[string]string
}

// Formatter defines how log entries are converted to byte arrays for output.
// Implementations should be thread-safe as they may be called concurrently.
type Formatter interface {
	// Format converts a log entry into bytes ready for output.
	// Returns an error if formatting fails.
	Format(entry *Entry) ([]byte, error)
}
