package loki

import (
	"context"
	"fmt"
	"github.com/edaniel30/loki-logger-go/models"
	"github.com/edaniel30/loki-logger-go/transport"
	"runtime/debug"
	"sync"
	"time"
)

type Logger struct {
	config     models.Config
	transports []transport.Transport
	mu         sync.RWMutex
}

func New(config models.Config, opts ...models.Option) (*Logger, error) {
	for _, opt := range opts {
		opt(&config)
	}

	if err := config.Validate(); err != nil {
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

func (l *Logger) Debug(message string, fields ...models.Fields) {
	l.log(models.LevelDebug, message, fields...)
}

func (l *Logger) Info(message string, fields ...models.Fields) {
	l.log(models.LevelInfo, message, fields...)
}

func (l *Logger) Warn(message string, fields ...models.Fields) {
	l.log(models.LevelWarn, message, fields...)
}

func (l *Logger) Error(message string, fields ...models.Fields) {
	l.log(models.LevelError, message, fields...)
}

func (l *Logger) Fatal(message string, fields ...models.Fields) {
	l.log(models.LevelFatal, message, fields...)
}

func (l *Logger) Log(level models.Level, message string, fields ...models.Fields) {
	l.log(level, message, fields...)
}

func (l *Logger) log(level models.Level, message string, fields ...models.Fields) {
	if !level.IsEnabled(l.config.LogLevel) {
		return
	}

	mergedFields := make(models.Fields)
	for _, f := range fields {
		for k, v := range f {
			mergedFields[k] = v
		}
	}

	// Check if stack trace should be skipped
	skipStackTrace := false
	if skip, ok := mergedFields["_skip_stack_trace"].(bool); ok && skip {
		skipStackTrace = true
		delete(mergedFields, "_skip_stack_trace") // Remove internal field
	}

	// Include stack trace automatically for error and fatal levels if configured
	if !skipStackTrace && l.config.IncludeStackTrace && (level == models.LevelError || level == models.LevelFatal) {
		stack := string(debug.Stack())
		message = fmt.Sprintf("%s\n\nStack trace:\n%s", message, stack)
	}

	labels := make(map[string]string)
	labels["app"] = l.config.AppName
	labels["level"] = level.String() // Add level as label for Loki indexing

	for k, v := range l.config.Labels {
		labels[k] = v
	}

	transportEntry := &transport.Entry{
		Level:     level.String(),
		Message:   message,
		Fields:    mergedFields,
		Timestamp: time.Now(),
		Labels:    labels,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, t := range l.transports {
		_ = t.Write(ctx, transportEntry)
	}
}

// Flush ensures all buffered log entries are sent to their destinations.
// Should be called before application shutdown.
func (l *Logger) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, t := range l.transports {
		if err := t.Flush(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Close releases all resources held by the logger.
// After calling Close, the logger should not be used.
func (l *Logger) Close() error {
	// Flush before closing
	if err := l.Flush(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, t := range l.transports {
		if err := t.Close(); err != nil {
			return err
		}
	}

	return nil
}

// WithFields creates a new logger with additional default fields.
// This is useful for adding context to all logs from a specific component.
func (l *Logger) WithFields(fields models.Fields) *Logger {
	// Deep copy the config to avoid modifying the original logger
	newConfig := l.config

	// Deep copy the Labels map
	newConfig.Labels = make(map[string]string)
	for k, v := range l.config.Labels {
		newConfig.Labels[k] = v
	}

	// Add new fields as labels
	for k, v := range fields {
		if strVal, ok := v.(string); ok {
			newConfig.Labels[k] = strVal
		}
	}

	newLogger := &Logger{
		config:     newConfig,
		transports: l.transports,
	}

	return newLogger
}
