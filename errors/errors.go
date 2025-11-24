package errors

import "fmt"

// ConfigError represents an error in logger configuration.
type ConfigError struct {
	message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: %s", e.message)
}

func ErrInvalidConfig(message string) error {
	return &ConfigError{message: message}
}

// TransportError represents an error during log transport.
type TransportError struct {
	message string
	cause   error
}

func (e *TransportError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("transport error: %s: %v", e.message, e.cause)
	}
	return fmt.Sprintf("transport error: %s", e.message)
}

func (e *TransportError) Unwrap() error {
	return e.cause
}

func ErrTransport(message string, cause error) error {
	return &TransportError{message: message, cause: cause}
}
