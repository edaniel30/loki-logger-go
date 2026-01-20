package types

import (
	"fmt"
	"strings"
)

// Level represents the severity of a log entry.
// Levels are ordered from least to most severe.
type Level int

const (
	// LevelDebug provides detailed information for diagnosing problems.
	LevelDebug Level = iota
	// LevelInfo provides general informational messages.
	LevelInfo
	// LevelWarn indicates potentially harmful situations.
	LevelWarn
	// LevelError represents error events that might still allow the application to continue.
	LevelError
	// LevelFatal represents severe errors that lead to application termination.
	LevelFatal
)

// String returns the string representation of the Level.
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	case LevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLevel converts a string to its corresponding Level.
// Returns an error if the string is not a valid level.
// Valid values: "debug", "info", "warn", "warning", "error", "fatal" (case-insensitive)
func ParseLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("invalid log level: %s", level)
	}
}

// IsEnabled checks whether this level should be logged given the configured minimum level.
// Returns true if the level is equal to or more severe than the configured level.
func (l Level) IsEnabled(configuredLevel Level) bool {
	return l >= configuredLevel
}

// MarshalText implements encoding.TextMarshaler for JSON/YAML serialization.
// This allows Level to be marshaled as a string instead of an integer.
func (l Level) MarshalText() ([]byte, error) {
	return []byte(l.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/YAML deserialization.
// This allows Level to be unmarshaled from a string.
func (l *Level) UnmarshalText(text []byte) error {
	level, err := ParseLevel(string(text))
	if err != nil {
		return err
	}
	*l = level
	return nil
}
