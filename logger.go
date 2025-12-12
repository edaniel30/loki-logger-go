package loki

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/ratelimit"
	"github.com/edaniel30/loki-logger-go/models"
	"github.com/edaniel30/loki-logger-go/transport"
)

type Logger struct {
	config              models.Config
	transports          []transport.Transport
	mu                  sync.RWMutex
	labelCardinalityMap map[string]map[string]struct{} // tracks unique values per label
	cardinalityMu       *sync.Mutex                    // pointer to share across child loggers
	rateLimiter         *ratelimit.RateLimiter         // nil if rate limiting disabled
	rateLimitStopCh     chan struct{}                  // signals stats reporter to stop
	rateLimitDoneCh     chan struct{}                  // signals stats reporter finished
}

func New(config models.Config, opts ...models.Option) (*Logger, error) {
	for _, opt := range opts {
		opt(&config)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	logger := &Logger{
		config:              config,
		transports:          make([]transport.Transport, 0),
		labelCardinalityMap: make(map[string]map[string]struct{}),
		cardinalityMu:       &sync.Mutex{},
		rateLimiter:         ratelimit.NewRateLimiter(config.MaxLogsPerSecond, config.SamplingRatio),
		rateLimitStopCh:     make(chan struct{}),
		rateLimitDoneCh:     make(chan struct{}),
	}

	if err := logger.setupTransports(); err != nil {
		return nil, err
	}

	// Start rate limit stats reporter if rate limiting is enabled and handler is set
	if logger.rateLimiter != nil && config.RateLimitHandler != nil {
		go logger.rateLimitStatsReporter()
	} else {
		close(logger.rateLimitDoneCh) // No reporter, mark as done
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
			LokiURL:         l.config.LokiHost,
			LokiUsername:    l.config.LokiUsername,
			LokiPassword:    l.config.LokiPassword,
			BatchSize:       l.config.BatchSize,
			FlushInterval:   l.config.FlushInterval,
			MaxRetries:      l.config.MaxRetries,
			Timeout:         l.config.Timeout,
			FlushTimeout:    l.config.FlushTimeout,
			ShutdownTimeout: l.config.ShutdownTimeout,
		})
		l.transports = append(l.transports, lokiTransport)
	}

	return nil
}

func (l *Logger) Debug(ctx context.Context, message string, fields models.Fields) {
	l.log(ctx, models.LevelDebug, message, fields)
}

func (l *Logger) Info(ctx context.Context, message string, fields models.Fields) {
	l.log(ctx, models.LevelInfo, message, fields)
}

func (l *Logger) Warn(ctx context.Context, message string, fields models.Fields) {
	l.log(ctx, models.LevelWarn, message, fields)
}

func (l *Logger) Error(ctx context.Context, message string, fields models.Fields) {
	l.log(ctx, models.LevelError, message, fields)
}

func (l *Logger) Fatal(ctx context.Context, message string, fields models.Fields) {
	l.log(ctx, models.LevelFatal, message, fields)
}

func (l *Logger) Log(ctx context.Context, level models.Level, message string, fields models.Fields) {
	l.log(ctx, level, message, fields)
}

// trackLabelCardinality tracks unique values for a label and triggers warning if threshold exceeded
func (l *Logger) trackLabelCardinality(labelKey, labelValue string) {
	// Skip if cardinality checking is disabled
	if l.config.MaxLabelCardinality <= 0 {
		return
	}

	l.cardinalityMu.Lock()
	defer l.cardinalityMu.Unlock()

	// Initialize map for this label if doesn't exist
	if l.labelCardinalityMap[labelKey] == nil {
		l.labelCardinalityMap[labelKey] = make(map[string]struct{})
	}

	// Add value to set
	l.labelCardinalityMap[labelKey][labelValue] = struct{}{}

	// Check if threshold exceeded
	uniqueCount := len(l.labelCardinalityMap[labelKey])
	if uniqueCount > l.config.MaxLabelCardinality {
		// Only warn once when threshold is first exceeded
		if uniqueCount == l.config.MaxLabelCardinality+1 {
			if l.config.CardinalityWarningHandler != nil {
				l.config.CardinalityWarningHandler(labelKey, uniqueCount, l.config.MaxLabelCardinality)
			}
		}
	}
}

func (l *Logger) log(ctx context.Context, level models.Level, message string, fields models.Fields) {
	if !level.IsEnabled(l.config.LogLevel) {
		return
	}

	// Check if this level should always be logged (bypass rate limiting)
	alwaysLog := false
	for _, alwaysLevel := range l.config.AlwaysLogLevels {
		if level == alwaysLevel {
			alwaysLog = true
			break
		}
	}

	// Apply rate limiting if enabled and not an "always log" level
	if !alwaysLog && l.rateLimiter != nil {
		allowed, _ := l.rateLimiter.Allow()
		if !allowed {
			return // Drop this log due to rate limiting
		}
	}

	// Use fields directly (no merge needed since we only accept one map now)
	if fields == nil {
		fields = make(models.Fields)
	}

	// Check if stack trace should be skipped
	skipStackTrace := false
	if skip, ok := fields["_skip_stack_trace"].(bool); ok && skip {
		skipStackTrace = true
		delete(fields, "_skip_stack_trace") // Remove internal field
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

	// Track cardinality for all labels
	for k, v := range labels {
		l.trackLabelCardinality(k, v)
	}

	transportEntry := &transport.Entry{
		Level:     level.String(),
		Message:   message,
		Fields:    fields,
		Timestamp: time.Now(),
		Labels:    labels,
	}

	// Use provided context with a timeout if it doesn't already have a deadline
	writeCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		writeCtx, cancel = context.WithTimeout(ctx, l.config.WriteTimeout)
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
// Uses the configured FlushTimeout (default: 10 seconds).
func (l *Logger) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.config.FlushTimeout)
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
// Uses the configured ShutdownTimeout (default: 10 seconds) for graceful shutdown.
func (l *Logger) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), l.config.ShutdownTimeout)
	defer cancel()
	return l.CloseContext(ctx)
}

// CloseContext releases all resources held by the logger.
// Respects the provided context for cancellation and deadline during flush.
func (l *Logger) CloseContext(ctx context.Context) error {
	// Stop rate limit stats reporter if running
	select {
	case <-l.rateLimitStopCh:
		// Already closed
	default:
		close(l.rateLimitStopCh)
	}

	// Wait for stats reporter to finish
	<-l.rateLimitDoneCh

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

// WithFields creates a new logger with additional default fields.
// This is useful for adding context to all logs from a specific component.
// Note: Only string values are added as labels for Loki indexing.
// Non-string values trigger a warning and are ignored to prevent high cardinality issues.
func (l *Logger) WithFields(fields models.Fields) *Logger {
	// Deep copy the config to avoid modifying the original logger
	newConfig := l.config

	// Deep copy the Labels map
	newConfig.Labels = make(map[string]string)
	for k, v := range l.config.Labels {
		newConfig.Labels[k] = v
	}

	// Add new fields as labels (only strings)
	for k, v := range fields {
		if strVal, ok := v.(string); ok {
			newConfig.Labels[k] = strVal
			// Track cardinality immediately for the new label
			l.trackLabelCardinality(k, strVal)
		} else {
			// Warn about non-string values
			if l.config.ErrorHandler != nil {
				l.config.ErrorHandler("logger",
					fmt.Errorf("WithFields: field '%s' has non-string value (type: %T). Only string values are added as labels. Use Fields parameter in log methods for non-string values", k, v))
			}
		}
	}

	// Share transports with parent logger (they are thread-safe and designed to be shared)
	newLogger := &Logger{
		config:              newConfig,
		transports:          l.transports,               // Shared (thread-safe)
		labelCardinalityMap: l.labelCardinalityMap,      // Share cardinality tracking
		cardinalityMu:       l.cardinalityMu,            // Share mutex
		rateLimiter:         l.rateLimiter,              // Share rate limiter
		rateLimitStopCh:     l.rateLimitStopCh,          // Share stop channel
		rateLimitDoneCh:     l.rateLimitDoneCh,          // Share done channel
	}

	return newLogger
}

// rateLimitStatsReporter periodically reports rate limit statistics.
func (l *Logger) rateLimitStatsReporter() {
	defer close(l.rateLimitDoneCh)

	ticker := time.NewTicker(l.config.RateLimitStatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if l.rateLimiter != nil && l.config.RateLimitHandler != nil {
				dropped, sampled := l.rateLimiter.GetAndResetStats()
				if dropped > 0 || sampled > 0 {
					l.config.RateLimitHandler(dropped, sampled)
				}
			}
		case <-l.rateLimitStopCh:
			// Report final stats before exiting
			if l.rateLimiter != nil && l.config.RateLimitHandler != nil {
				dropped, sampled := l.rateLimiter.GetAndResetStats()
				if dropped > 0 || sampled > 0 {
					l.config.RateLimitHandler(dropped, sampled)
				}
			}
			return
		}
	}
}
