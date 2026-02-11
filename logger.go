package loki

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/transport"
	"github.com/edaniel30/loki-logger-go/types"
	"github.com/edaniel30/loki-logger-go/utils"
)

type Logger struct {
	config     Config
	transports []transport.Transport
	mu         sync.RWMutex
}

func New(config *Config, opts ...Option) (*Logger, error) {
	for _, opt := range opts {
		opt(config)
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	logger := &Logger{
		config:     *config,
		transports: make([]transport.Transport, 0),
	}

	logger.setupTransports()

	return logger, nil
}

func (l *Logger) setupTransports() {
	// always add console transport
	consoleTransport := transport.NewConsoleTransport()
	l.transports = append(l.transports, consoleTransport)

	// if not only console, add loki transport
	if !l.config.OnlyConsole {
		lokiTransport := transport.NewLokiTransport(&transport.LokiTransportConfig{
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
}

// Debug logs a message at debug level with optional structured fields.
// Debug logs are typically used for detailed diagnostic information during development.
func (l *Logger) Debug(ctx context.Context, message string, fields map[string]any) {
	l.log(ctx, types.LevelDebug, message, fields)
}

// Info logs a message at info level with optional structured fields.
// Info logs are used for general informational messages about application state.
func (l *Logger) Info(ctx context.Context, message string, fields map[string]any) {
	l.log(ctx, types.LevelInfo, message, fields)
}

// Warn logs a message at warning level with optional structured fields.
// Warn logs indicate potentially harmful situations that should be reviewed.
func (l *Logger) Warn(ctx context.Context, message string, fields map[string]any) {
	l.log(ctx, types.LevelWarn, message, fields)
}

// Error logs a message at error level with optional structured fields.
// Error logs indicate error conditions that should be investigated.
// If IncludeStackTrace is enabled, automatically includes a stack trace.
func (l *Logger) Error(ctx context.Context, message string, fields map[string]any) {
	l.log(ctx, types.LevelError, message, fields)
}

// Fatal logs a message at fatal level with optional structured fields.
// Fatal logs indicate severe errors that may cause application failure.
// Stack traces are automatically included for fatal logs.
func (l *Logger) Fatal(ctx context.Context, message string, fields map[string]any) {
	l.log(ctx, types.LevelFatal, message, fields)
}

func (l *Logger) log(ctx context.Context, level types.Level, message string, fields map[string]any) {
	if !level.IsEnabled(l.config.LogLevel) {
		return
	}

	// Use fields directly (no merge needed since we only accept one map now)
	if fields == nil {
		fields = make(map[string]any)
	}

	// Automatically add caller information (file and line) if not already present
	// Uses utils.GetCaller() to dynamically find the first caller outside the logger package
	if file, line, ok := utils.GetCaller(); ok {
		if _, exists := fields["file"]; !exists {
			fields["file"] = filepath.Base(file)
		}
		if _, exists := fields["line"]; !exists {
			fields["line"] = line
		}
	}

	// Include stack trace automatically for error and fatal levels
	// Stack traces are always enabled (hardcoded to true)
	if level == types.LevelError || level == types.LevelFatal {
		stack := string(debug.Stack())
		message = fmt.Sprintf("%s\n\nStack trace:\n%s", message, stack)
	}

	labels := make(types.Labels)

	// Copy user-provided labels first
	maps.Copy(labels, l.config.Labels)

	// Set system labels last to prevent user overrides
	// These are reserved keys that ensure consistent Loki indexing
	labels["app"] = l.config.AppName
	labels["level"] = level.String()
	labels["version"] = l.config.AppVersion
	labels["environment"] = l.config.AppEnv

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
		// Write to transport, errors are logged but don't stop execution
		_ = t.Write(writeCtx, transportEntry)
	}
}

// Close releases all resources held by the logger.
// After calling Close, the logger should not be used.
// Flushes buffered logs before closing transports.
func (l *Logger) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	l.mu.RLock()
	// Flush all transports first
	var flushErr error
	for _, t := range l.transports {
		if err := t.Flush(ctx); err != nil && flushErr == nil {
			flushErr = err
		}
	}
	l.mu.RUnlock()

	// Now close all transports
	l.mu.Lock()
	defer l.mu.Unlock()

	var closeErr error
	for _, t := range l.transports {
		if err := t.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}

	// Return flush error if it occurred, otherwise close error
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

// WithLabels creates a new logger with additional default labels.
// This is useful for adding context to all logs from a specific component.
// Labels are indexed by Loki and should have low cardinality (< 50 unique values per label).
func (l *Logger) WithLabels(labels types.Labels) *Logger {
	// Deep copy the config to avoid modifying the original logger
	newConfig := l.config

	// Deep copy the Labels map
	newConfig.Labels = make(types.Labels)
	maps.Copy(newConfig.Labels, l.config.Labels)

	// Add new labels (already string type)
	maps.Copy(newConfig.Labels, labels)

	// Share transports with parent logger (they are thread-safe and designed to be shared)
	newLogger := &Logger{
		config:     newConfig,
		transports: l.transports, // Shared (thread-safe)
	}

	return newLogger
}
