package models

import (
	"fmt"
	"os"
	"time"

	"github.com/edaniel30/loki-logger-go/errors"
)

// ErrorHandler is called when a transport fails to write logs.
// It receives the transport name and the error that occurred.
type ErrorHandler func(transportName string, err error)

// CardinalityWarningHandler is called when a label exceeds the cardinality threshold.
// High cardinality labels can cause performance issues in Loki.
type CardinalityWarningHandler func(labelKey string, uniqueValues int, threshold int)

// RateLimitHandler is called periodically with rate limit statistics.
// It receives the number of logs dropped and sampled since last call.
type RateLimitHandler func(droppedCount int, sampledCount int)

// Config holds the configuration for a Logger instance.
type Config struct {
	// AppName identifies the application generating logs
	AppName string

	// LokiHost is the URL of the Loki server (e.g., "http://localhost:3100")
	LokiHost string

	// LokiUsername is the username for basic auth (optional)
	LokiUsername string

	// LokiPassword is the password for basic auth (optional)
	LokiPassword string

	// LogLevel is the minimum level that will be logged
	LogLevel Level

	// Labels are default labels attached to all log entries
	Labels map[string]string

	// BatchSize is the number of logs to accumulate before sending to Loki
	BatchSize int

	// FlushInterval is how often to flush logs to Loki regardless of batch size
	FlushInterval time.Duration

	// MaxRetries is the number of times to retry failed requests to Loki
	MaxRetries int

	// Timeout is the HTTP client timeout for requests to Loki
	Timeout time.Duration

	// WriteTimeout is the timeout for individual log write operations
	// Default: 5 seconds
	WriteTimeout time.Duration

	// FlushTimeout is the timeout for flush operations
	// Default: 10 seconds
	FlushTimeout time.Duration

	// ShutdownTimeout is the timeout for graceful shutdown
	// Default: 10 seconds
	ShutdownTimeout time.Duration

	// EnableConsole enables logging to console in addition to other transports
	OnlyConsole bool

	// IncludeStackTrace includes stack trace in error and fatal logs
	IncludeStackTrace bool

	// ErrorHandler is called when a transport fails to write logs
	// If nil, errors are written to stderr by default
	ErrorHandler ErrorHandler

	// MaxLabelCardinality is the maximum number of unique values allowed per label
	// before triggering a warning. Set to 0 to disable cardinality checking.
	// Default is 10.
	MaxLabelCardinality int

	// CardinalityWarningHandler is called when a label exceeds MaxLabelCardinality
	// If nil, warnings are written to stderr by default
	CardinalityWarningHandler CardinalityWarningHandler

	// MaxLogsPerSecond is the maximum number of logs allowed per second.
	// When exceeded, logs are sampled according to SamplingRatio.
	// Set to 0 to disable rate limiting. Default is 0 (disabled).
	MaxLogsPerSecond int

	// SamplingRatio is the ratio of logs to keep when rate limit is exceeded (0.0-1.0).
	// For example, 0.1 means keep 1 in 10 logs when over the rate limit.
	// Default is 0.1 (10%).
	SamplingRatio float64

	// AlwaysLogLevels are log levels that are never rate-limited or sampled.
	// Default is [LevelError, LevelFatal] to ensure critical logs are never dropped.
	AlwaysLogLevels []Level

	// RateLimitHandler is called periodically with rate limit statistics.
	// It receives the number of logs dropped and sampled since last report.
	// If nil, no statistics are reported.
	RateLimitHandler RateLimitHandler

	// RateLimitStatsInterval is how often to report rate limit statistics.
	// Default is 10 seconds.
	RateLimitStatsInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		AppName:                   "app",
		LokiHost:                  "http://localhost:3100",
		LogLevel:                  LevelInfo,
		Labels:                    make(map[string]string),
		BatchSize:                 100,
		FlushInterval:             5 * time.Second,
		MaxRetries:                3,
		Timeout:                   10 * time.Second,
		WriteTimeout:              5 * time.Second,
		FlushTimeout:              10 * time.Second,
		ShutdownTimeout:           10 * time.Second,
		OnlyConsole:               true,
		IncludeStackTrace:         true,
		ErrorHandler:              DefaultErrorHandler(),
		MaxLabelCardinality:       10,
		CardinalityWarningHandler: DefaultCardinalityWarningHandler(),
		MaxLogsPerSecond:          0,    // Disabled by default
		SamplingRatio:             0.1,  // 10% when rate limited
		AlwaysLogLevels:           []Level{LevelError, LevelFatal},
		RateLimitHandler:          nil,  // No handler by default
		RateLimitStatsInterval:    10 * time.Second,
	}
}

// DefaultErrorHandler returns an error handler that writes to stderr.
func DefaultErrorHandler() ErrorHandler {
	return func(transportName string, err error) {
		fmt.Fprintf(os.Stderr, "[loki-logger] transport error [%s]: %v\n", transportName, err)
	}
}

// DefaultCardinalityWarningHandler returns a cardinality warning handler that writes to stderr.
func DefaultCardinalityWarningHandler() CardinalityWarningHandler {
	return func(labelKey string, uniqueValues int, threshold int) {
		fmt.Fprintf(os.Stderr, "[loki-logger] WARNING: label '%s' has %d unique values (threshold: %d). High cardinality can impact Loki performance.\n",
			labelKey, uniqueValues, threshold)
	}
}

type Option func(*Config)

func WithAppName(name string) Option {
	return func(c *Config) {
		c.AppName = name
	}
}

func WithLokiHost(host string) Option {
	return func(c *Config) {
		c.LokiHost = host
	}
}

func WithLokiUsername(username string) Option {
	return func(c *Config) {
		c.LokiUsername = username
	}
}

func WithLokiPassword(password string) Option {
	return func(c *Config) {
		c.LokiPassword = password
	}
}

func WithLogLevel(level Level) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

func WithLabels(labels map[string]string) Option {
	return func(c *Config) {
		c.Labels = labels
	}
}

func WithBatchSize(size int) Option {
	return func(c *Config) {
		c.BatchSize = size
	}
}

func WithFlushInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.FlushInterval = interval
	}
}

func WithMaxRetries(retries int) Option {
	return func(c *Config) {
		c.MaxRetries = retries
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.WriteTimeout = timeout
	}
}

func WithFlushTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.FlushTimeout = timeout
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

func WithOnlyConsole(enabled bool) Option {
	return func(c *Config) {
		c.OnlyConsole = enabled
	}
}

func WithIncludeStackTrace(enabled bool) Option {
	return func(c *Config) {
		c.IncludeStackTrace = enabled
	}
}

func WithErrorHandler(handler ErrorHandler) Option {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

func WithMaxLabelCardinality(max int) Option {
	return func(c *Config) {
		c.MaxLabelCardinality = max
	}
}

func WithCardinalityWarningHandler(handler CardinalityWarningHandler) Option {
	return func(c *Config) {
		c.CardinalityWarningHandler = handler
	}
}

func WithRateLimit(maxLogsPerSecond int, samplingRatio float64) Option {
	return func(c *Config) {
		c.MaxLogsPerSecond = maxLogsPerSecond
		c.SamplingRatio = samplingRatio
	}
}

func WithMaxLogsPerSecond(max int) Option {
	return func(c *Config) {
		c.MaxLogsPerSecond = max
	}
}

func WithSamplingRatio(ratio float64) Option {
	return func(c *Config) {
		c.SamplingRatio = ratio
	}
}

func WithAlwaysLogLevels(levels []Level) Option {
	return func(c *Config) {
		c.AlwaysLogLevels = levels
	}
}

func WithRateLimitHandler(handler RateLimitHandler) Option {
	return func(c *Config) {
		c.RateLimitHandler = handler
	}
}

func WithRateLimitStatsInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.RateLimitStatsInterval = interval
	}
}

func (c *Config) Validate() error {
	if c.AppName == "" {
		return errors.ErrInvalidConfig("appName is required")
	}
	if c.BatchSize <= 0 {
		return errors.ErrInvalidConfig("batchSize must be positive")
	}
	if c.FlushInterval <= 0 {
		return errors.ErrInvalidConfig("flushInterval must be positive")
	}
	return nil
}
