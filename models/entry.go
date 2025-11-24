package models

import "time"

// Fields is a map of key-value pairs for structured logging.
// It allows attaching arbitrary metadata to log entries.
type Fields map[string]any

// Entry represents a single log record with all its associated data.
type Entry struct {
	// Level is the severity of this log entry
	Level Level

	// Message is the main log message
	Message string

	// Fields contains structured data attached to this entry
	Fields Fields

	// Timestamp is when this log entry was created
	Timestamp time.Time

	// Labels are key-value pairs used for indexing in Loki
	Labels map[string]string
}

func NewEntry(level Level, message string, fields Fields) *Entry {
	return &Entry{
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
		Labels:    make(map[string]string),
	}
}

// WithLabel adds a label to the entry for Loki indexing.
// Labels should be used sparingly as they impact cardinality.
func (e *Entry) WithLabel(key, value string) *Entry {
	e.Labels[key] = value
	return e
}

// WithLabels adds multiple labels to the entry.
func (e *Entry) WithLabels(labels map[string]string) *Entry {
	for k, v := range labels {
		e.Labels[k] = v
	}
	return e
}
