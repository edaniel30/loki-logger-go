package loki

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/transport"
	"github.com/edaniel30/loki-logger-go/types"
)

type Logger struct {
	config     Config
	transports []transport.Transport
	mu         sync.RWMutex
}

func New(config Config, opts ...Option) (*Logger, error) {
	for _, opt := range opts {
		opt(&config)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	logger := &Logger{
		config:     config,
		transports: make([]transport.Transport, 0),
	}

	if err := logger.setupTransports(); err != nil {
		return nil, err
	}

	return logger, nil
}

func (l *Logger) setupTransports() error {
	// always add console transport
	consoleTransport := transport.NewConsoleTransport()
	l.transports = append(l.transports, consoleTransport)

	// if not only console, add loki transport
	if !l.config.OnlyConsole {
		lokiTransport := transport.NewLokiTransport(transport.LokiTransportConfig{
			LokiURL:       l.config.LokiHost,
			LokiUsername:  l.config.LokiUsername,
			LokiPassword:  l.config.LokiPassword,
			BatchSize:     l.config.BatchSize,
			FlushInterval: l.config.FlushInterval,
			MaxRetries:    l.config.MaxRetries,
			Timeout:       l.config.Timeout,
		})
		l.transports = append(l.transports, lokiTransport)
	}

	return nil
}

// Debug logs a message at debug level with optional structured fields.
// Debug logs are typically used for detailed diagnostic information during development.
func (l *Logger) Debug(ctx context.Context, message string, fields types.Fields) {
	l.log(ctx, types.LevelDebug, message, fields)
}

// Info logs a message at info level with optional structured fields.
// Info logs are used for general informational messages about application state.
func (l *Logger) Info(ctx context.Context, message string, fields types.Fields) {
	l.log(ctx, types.LevelInfo, message, fields)
}

// Warn logs a message at warning level with optional structured fields.
// Warn logs indicate potentially harmful situations that should be reviewed.
func (l *Logger) Warn(ctx context.Context, message string, fields types.Fields) {
	l.log(ctx, types.LevelWarn, message, fields)
}

// Error logs a message at error level with optional structured fields.
// Error logs indicate error conditions that should be investigated.
// If IncludeStackTrace is enabled, automatically includes a stack trace.
func (l *Logger) Error(ctx context.Context, message string, fields types.Fields) {
	l.log(ctx, types.LevelError, message, fields)
}

// Fatal logs a message at fatal level with optional structured fields.
// Fatal logs indicate severe errors that may cause application failure.
// If IncludeStackTrace is enabled, automatically includes a stack trace.
func (l *Logger) Fatal(ctx context.Context, message string, fields types.Fields) {
	l.log(ctx, types.LevelFatal, message, fields)
}

// Log logs a message at the specified level with optional structured fields.
// This method provides direct control over the log level.
func (l *Logger) Log(ctx context.Context, level types.Level, message string, fields types.Fields) {
	l.log(ctx, level, message, fields)
}

func (l *Logger) log(ctx context.Context, level types.Level, message string, fields types.Fields) {
	if !level.IsEnabled(l.config.LogLevel) {
		return
	}

	// Use fields directly (no merge needed since we only accept one map now)
	if fields == nil {
		fields = make(types.Fields)
	}

	// Check if stack trace should be skipped
	skipStackTrace := false
	if skip, ok := fields["_skip_stack_trace"].(bool); ok && skip {
		skipStackTrace = true
		delete(fields, "_skip_stack_trace") // Remove internal field
	}

	// Include stack trace automatically for error and fatal levels if configured
	if !skipStackTrace && l.config.IncludeStackTrace && (level == types.LevelError || level == types.LevelFatal) {
		stack := string(debug.Stack())
		message = fmt.Sprintf("%s\n\nStack trace:\n%s", message, stack)
	}

	labels := make(types.Labels)
	labels["app"] = l.config.AppName
	labels["level"] = level.String() // Add level as label for Loki indexing

	for k, v := range l.config.Labels {
		labels[k] = v
	}

	transportEntry := &types.Entry{
		Level:     level,
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
		Labels:    labels,
	}

	// Use provided context with a timeout if it doesn't already have a deadline
	writeCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		writeCtx, cancel = context.WithTimeout(ctx, l.config.Timeout)
		defer cancel()
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, t := range l.transports {
		if err := t.Write(writeCtx, transportEntry); err != nil {
			// Call error handler if configured
			if l.config.ErrorHandler != nil {
				l.config.ErrorHandler(t.Name(), err)
			}
		}
	}
}

// Flush ensures all buffered log entries are sent to their destinations.
// Should be called before application shutdown.
// Uses the configured Timeout.
func (l *Logger) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.config.Timeout)
	defer cancel()
	return l.FlushContext(ctx)
}

// FlushContext ensures all buffered log entries are sent to their destinations.
// Respects the provided context for cancellation and deadline.
func (l *Logger) FlushContext(ctx context.Context) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var firstErr error
	for _, t := range l.transports {
		if err := t.Flush(ctx); err != nil {
			// Call error handler
			if l.config.ErrorHandler != nil {
				l.config.ErrorHandler(t.Name(), err)
			}
			// Keep track of first error to return
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// Close releases all resources held by the logger.
// After calling Close, the logger should not be used.
// Uses the configured Timeout for graceful shutdown.
func (l *Logger) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.config.Timeout)
	defer cancel()
	return l.CloseContext(ctx)
}

// CloseContext releases all resources held by the logger.
// Respects the provided context for cancellation and deadline during flush.
func (l *Logger) CloseContext(ctx context.Context) error {
	// Flush before closing
	flushErr := l.FlushContext(ctx)

	l.mu.Lock()
	defer l.mu.Unlock()

	var firstErr error
	for _, t := range l.transports {
		if err := t.Close(); err != nil {
			// Call error handler
			if l.config.ErrorHandler != nil {
				l.config.ErrorHandler(t.Name(), err)
			}
			// Keep track of first error
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	// Return flush error if it occurred, otherwise close error
	if flushErr != nil {
		return flushErr
	}
	return firstErr
}

// WithLabels creates a new logger with additional default labels.
// This is useful for adding context to all logs from a specific component.
// Labels are indexed by Loki and should have low cardinality (< 50 unique values per label).
func (l *Logger) WithLabels(labels types.Labels) *Logger {
	// Deep copy the config to avoid modifying the original logger
	newConfig := l.config

	// Deep copy the Labels map
	newConfig.Labels = make(types.Labels)
	for k, v := range l.config.Labels {
		newConfig.Labels[k] = v
	}

	// Add new labels (already string type)
	for k, v := range labels {
		newConfig.Labels[k] = v
	}

	// Share transports with parent logger (they are thread-safe and designed to be shared)
	newLogger := &Logger{
		config:     newConfig,
		transports: l.transports, // Shared (thread-safe)
	}

	return newLogger
}
