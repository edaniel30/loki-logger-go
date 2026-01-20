package loki

import "fmt"

// Public Error Types
// These types are exported so users can use errors.As() to inspect them
// and access their fields for better error handling.

// ConfigError represents a configuration validation error.
// Users can access the Field and Message to understand what configuration is invalid.
type ConfigError struct {
	Field   string // The configuration field that caused the error (optional)
	Message string // Human-readable error message
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("loki: config error [%s]: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("loki: config error: %s", e.Message)
}

// TransportError represents an error that occurred during log transport.
// The Transport field identifies which transport failed (e.g., "console", "loki"),
// Op identifies the operation (e.g., "write", "flush"), and Cause contains the underlying error.
type TransportError struct {
	Transport string // The name of the transport that failed (e.g., "console", "loki")
	Op        string // The operation that failed (e.g., "write", "flush", "close")
	Cause     error  // The underlying error
}

func (e *TransportError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("loki: transport '%s' operation '%s' failed: %v", e.Transport, e.Op, e.Cause)
	}
	return fmt.Sprintf("loki: transport '%s' operation '%s' failed", e.Transport, e.Op)
}

func (e *TransportError) Unwrap() error {
	return e.Cause
}

// ClientError represents an error that occurred in the HTTP client when communicating with Loki.
// The URL and Method fields provide context about the failed request.
type ClientError struct {
	Method string // HTTP method (e.g., "POST")
	URL    string // The URL that was being accessed
	Cause  error  // The underlying error
}

func (e *ClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("loki: client error [%s %s]: %v", e.Method, e.URL, e.Cause)
	}
	return fmt.Sprintf("loki: client error [%s %s]", e.Method, e.URL)
}

func (e *ClientError) Unwrap() error {
	return e.Cause
}

// Internal constructor functions

// newConfigFieldError creates a configuration error with a specific field.
func newConfigFieldError(field, message string) error {
	return &ConfigError{Field: field, Message: message}
}
