package types

import "time"

// Labels is a map of key-value pairs for indexing in Loki.
type Labels map[string]string

// Entry represents a single log record with all its associated data.
// This is the internal representation of a log entry before it's sent to transports.
type Entry struct {
	// Level is the severity of this log entry
	Level Level

	// Message is the main log message
	Message string

	// Fields contains structured data attached to this entry
	Fields map[string]any

	// Timestamp is when this log entry was created
	Timestamp time.Time

	// Labels are key-value pairs used for indexing in Loki
	Labels Labels
}
