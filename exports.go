package loki

import "github.com/edaniel30/loki-logger-go/models"

// config is the configuration for the logger.
type Config = models.Config

// option is a function that configures the logger.
type Option = models.Option

// level is the severity of a log entry.
type Level = models.Level

// Fields is a map of key-value pairs for structured logging.
// It allows attaching arbitrary metadata to log entries.
type Fields = models.Fields

// Entry represents a single log record with all its associated data.
type Entry = models.Entry

// ErrorHandler is called when a transport fails to write logs.
// It receives the transport name and the error that occurred.
type ErrorHandler = models.ErrorHandler

// CardinalityWarningHandler is called when a label exceeds the cardinality threshold.
// High cardinality labels can cause performance issues in Loki.
type CardinalityWarningHandler = models.CardinalityWarningHandler

// RateLimitHandler is called periodically with rate limit statistics.
// It receives the number of logs dropped and sampled since last call.
type RateLimitHandler = models.RateLimitHandler

// Re-export level constants
const (
	LevelDebug = models.LevelDebug
	LevelInfo  = models.LevelInfo
	LevelWarn  = models.LevelWarn
	LevelError = models.LevelError
	LevelFatal = models.LevelFatal
)

// Re-export functions
var (
	// DefaultConfig returns a Config with sensible defaults:
	// - AppName: "app"
	// - LokiHost: "http://localhost:3100"
	// - LogLevel: LevelInfo
	// - BatchSize: 100
	// - FlushInterval: 5 seconds
	// - MaxRetries: 3
	// - Timeout: 10 seconds
	// - WriteTimeout: 5 seconds
	// - FlushTimeout: 10 seconds
	// - ShutdownTimeout: 10 seconds
	// - OnlyConsole: true (Loki transport disabled by default)
	// - IncludeStackTrace: true
	// - MaxLabelCardinality: 10
	// - MaxLogsPerSecond: 0 (rate limiting disabled)
	// - SamplingRatio: 0.1 (10% when rate limited)
	// - AlwaysLogLevels: [LevelError, LevelFatal]
	DefaultConfig = models.DefaultConfig

	// DefaultErrorHandler returns an error handler that writes transport errors to stderr.
	// Format: "[loki-logger] transport error [transport_name]: error_message"
	DefaultErrorHandler = models.DefaultErrorHandler

	// DefaultCardinalityWarningHandler returns a handler that writes cardinality warnings to stderr.
	// Format: "[loki-logger] WARNING: label 'label_key' has N unique values (threshold: M)"
	DefaultCardinalityWarningHandler = models.DefaultCardinalityWarningHandler

	// NewEntry creates a new log Entry with the specified parameters.
	// Entries are the internal representation of log records.
	NewEntry = models.NewEntry

	// ParseLevel converts a string to a Level.
	// Valid values: "debug", "info", "warn", "error", "fatal" (case-insensitive)
	// Returns an error if the string is not a valid level.
	ParseLevel = models.ParseLevel
)

// Re-export configuration options
var (
	// WithAppName sets the application name (required)
	WithAppName = models.WithAppName

	// WithLokiHost sets the Loki server URL (e.g., "http://localhost:3100")
	WithLokiHost = models.WithLokiHost

	// WithLokiUsername sets the username for Loki basic auth (optional)
	WithLokiUsername = models.WithLokiUsername

	// WithLokiPassword sets the password for Loki basic auth (optional)
	WithLokiPassword = models.WithLokiPassword

	// WithLogLevel sets the minimum log level that will be logged
	WithLogLevel = models.WithLogLevel

	// WithLabels sets default labels attached to all log entries
	WithLabels = models.WithLabels

	// WithBatchSize sets the number of logs to accumulate before sending to Loki
	WithBatchSize = models.WithBatchSize

	// WithFlushInterval sets how often to flush logs to Loki regardless of batch size
	WithFlushInterval = models.WithFlushInterval

	// WithMaxRetries sets the number of times to retry failed requests to Loki
	WithMaxRetries = models.WithMaxRetries

	// WithTimeout sets the HTTP client timeout for requests to Loki
	WithTimeout = models.WithTimeout

	// WithWriteTimeout sets the timeout for individual log write operations
	// Default: 5 seconds
	WithWriteTimeout = models.WithWriteTimeout

	// WithFlushTimeout sets the timeout for flush operations
	// Default: 10 seconds
	WithFlushTimeout = models.WithFlushTimeout

	// WithShutdownTimeout sets the timeout for graceful shutdown
	// Default: 10 seconds
	WithShutdownTimeout = models.WithShutdownTimeout

	// WithOnlyConsole disables Loki transport and only logs to console
	WithOnlyConsole = models.WithOnlyConsole

	// WithIncludeStackTrace enables/disables automatic stack traces for error and fatal logs
	WithIncludeStackTrace = models.WithIncludeStackTrace

	// WithErrorHandler sets a custom error handler for transport failures
	// If not set, errors are written to stderr by default
	WithErrorHandler = models.WithErrorHandler

	// WithMaxLabelCardinality sets the maximum unique values per label before warning
	// Set to 0 to disable cardinality checking. Default is 10.
	WithMaxLabelCardinality = models.WithMaxLabelCardinality

	// WithCardinalityWarningHandler sets a custom handler for cardinality warnings
	// If not set, warnings are written to stderr by default
	WithCardinalityWarningHandler = models.WithCardinalityWarningHandler

	// WithRateLimit sets both the maximum logs per second and sampling ratio in one call.
	// maxLogsPerSecond: maximum logs allowed per second (0 to disable rate limiting)
	// samplingRatio: ratio of logs to keep when over limit (0.0-1.0, e.g., 0.1 = keep 1 in 10)
	// Default: 0 logs/sec (disabled), 0.1 sampling ratio
	WithRateLimit = models.WithRateLimit

	// WithMaxLogsPerSecond sets the maximum number of logs allowed per second.
	// When exceeded, logs are sampled according to SamplingRatio.
	// Set to 0 to disable rate limiting. Default is 0 (disabled).
	WithMaxLogsPerSecond = models.WithMaxLogsPerSecond

	// WithSamplingRatio sets the ratio of logs to keep when rate limit is exceeded.
	// Value should be between 0.0 and 1.0. For example, 0.1 means keep 1 in 10 logs.
	// Default is 0.1 (10%).
	WithSamplingRatio = models.WithSamplingRatio

	// WithAlwaysLogLevels sets log levels that are never rate-limited or sampled.
	// These levels will always be logged regardless of rate limits.
	// Default is [LevelError, LevelFatal] to ensure critical logs are never dropped.
	WithAlwaysLogLevels = models.WithAlwaysLogLevels

	// WithRateLimitHandler sets a custom handler for rate limit statistics.
	// The handler is called periodically with the number of logs dropped and sampled.
	// Use this to monitor rate limiting effectiveness and tune your configuration.
	// Default is nil (no handler).
	WithRateLimitHandler = models.WithRateLimitHandler

	// WithRateLimitStatsInterval sets how often to report rate limit statistics.
	// Default is 10 seconds.
	WithRateLimitStatsInterval = models.WithRateLimitStatsInterval
)
