package models

import "strings"

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
// Returns LevelInfo if the string is not recognized.
func ParseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// IsEnabled checks whether this level should be logged given the configured minimum level.
func (l Level) IsEnabled(configuredLevel Level) bool {
	return l >= configuredLevel
}
