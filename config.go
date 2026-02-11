package loki

import (
	"time"

	"github.com/edaniel30/loki-logger-go/types"
)

// Config holds the logger configuration.
// Use DefaultConfig() to get sensible defaults, then customize with Option functions.
type Config struct {
	// Required fields
	AppName string // Application name, used as a label in Loki (required)

	// Loki connection
	LokiHost     string // Loki server URL, e.g., "http://localhost:3100" (required if not OnlyConsole)
	LokiUsername string // Username for basic auth (optional)
	LokiPassword string // Password for basic auth (optional)

	// Logging behavior
	LogLevel          types.Level  // Minimum level to log (default: types.LevelInfo)
	Labels            types.Labels // Default labels attached to all log entries
	IncludeStackTrace bool         // Include stack trace in error and fatal logs (default: true)
	OnlyConsole       bool         // Only log to console, skip Loki (default: false)

	// Performance settings
	BatchSize     int           // Number of logs to accumulate before sending to Loki (default: 100)
	FlushInterval time.Duration // How often to flush logs to Loki regardless of batch size (default: 5s)
	MaxRetries    int           // Number of times to retry failed requests to Loki (default: 3)
	Timeout       time.Duration // Timeout for operations (connect, write, flush, shutdown) (default: 10s)

	AppVersion string // Version of the application (default: "1.0.0")
	AppEnv     string // Environment of the application (default: "local")
}

// DefaultConfig returns a Config with sensible default values.
// This is the recommended starting point for most applications.
//
// Default values:
//   - AppName: "app"
//   - LokiHost: "http://localhost:3100"
//   - AppVersion: "1.0.0"
//   - AppEnv: "local"
//   - LogLevel: LevelInfo
//   - Labels: empty map
//   - IncludeStackTrace: true
//   - OnlyConsole: false (logs to both console and Loki)
//   - BatchSize: 100
//   - FlushInterval: 5 seconds
//   - MaxRetries: 3
//   - Timeout: 10 seconds
//
// Example:
//
//	cfg := loki.DefaultConfig()
//	logger, err := loki.New(cfg,
//		loki.WithAppName("my-app"),
//		loki.WithLokiHost("http://loki:3100"),
//	)
func DefaultConfig() *Config {
	return &Config{
		AppName:           "app",
		AppVersion:        "1.0.0",
		AppEnv:            "local",
		LokiHost:          "http://localhost:3100",
		LogLevel:          types.LevelInfo,
		Labels:            make(types.Labels),
		IncludeStackTrace: true,
		OnlyConsole:       false,
		BatchSize:         100,
		FlushInterval:     5 * time.Second,
		MaxRetries:        3,
		Timeout:           10 * time.Second,
	}
}

// Option is a function that modifies a Config.
// Use Option functions with New() to customize the configuration.
type Option func(*Config)

// WithAppName sets the application name used as a label in Loki.
//
// Example:
//
//	loki.WithAppName("my-service")
func WithAppName(name string) Option {
	return func(c *Config) {
		c.AppName = name
	}
}

// WithAppVersion sets the application version used as a label in Loki.
//
// Example:
//
//	loki.WithAppVersion("1.0.0")
func WithAppVersion(version string) Option {
	return func(c *Config) {
		c.AppVersion = version
	}
}

// WithAppEnv sets the application environment used as a label in Loki.
//
// Example:
//
//	loki.WithAppEnv("local")
func WithAppEnv(env string) Option {
	return func(c *Config) {
		c.AppEnv = env
	}
}

// WithLokiHost sets the Loki server URL.
//
// Example:
//
//	loki.WithLokiHost("http://loki:3100")
//	loki.WithLokiHost("https://logs.example.com")
func WithLokiHost(host string) Option {
	return func(c *Config) {
		c.LokiHost = host
	}
}

// WithLokiBasicAuth sets username and password for basic authentication.
//
// Example:
//
//	loki.WithLokiBasicAuth("user", "password")
func WithLokiBasicAuth(username, password string) Option {
	return func(c *Config) {
		c.LokiUsername = username
		c.LokiPassword = password
	}
}

// WithLogLevel sets the minimum log level that will be logged.
// Logs below this level will be discarded.
//
// Example:
//
//	loki.WithLogLevel(loki.LevelDebug)
//	loki.WithLogLevel(loki.LevelWarn)
func WithLogLevel(level types.Level) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// WithLabels sets default labels that will be attached to all log entries.
// These labels are useful for filtering and querying in Loki.
//
// Example:
//
//	loki.WithLabels(types.Labels{
//		"environment": "production",
//		"region":      "us-east-1",
//	})
func WithLabels(labels types.Labels) Option {
	return func(c *Config) {
		c.Labels = labels
	}
}

// WithOnlyConsole enables console-only mode, skipping Loki transport entirely.
// Useful for local development or testing.
//
// Example:
//
//	loki.WithOnlyConsole(true) // Only log to console
func WithOnlyConsole(enabled bool) Option {
	return func(c *Config) {
		c.OnlyConsole = enabled
	}
}

// WithBatchSize sets the number of logs to accumulate before sending to Loki.
// Larger batches reduce network overhead but increase memory usage.
// Default is 100.
//
// Example:
//
//	loki.WithBatchSize(200)
func WithBatchSize(size int) Option {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithFlushInterval sets how often to flush logs to Loki regardless of batch size.
// This ensures logs are sent even if the batch isn't full.
// Default is 5 seconds.
//
// Example:
//
//	loki.WithFlushInterval(10 * time.Second)
func WithFlushInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.FlushInterval = interval
	}
}

// validate checks if the configuration is valid.
// Returns a ConfigError if any required field is missing or invalid.
func (c *Config) validate() error {
	if c.AppName == "" {
		return newConfigFieldError("AppName", "is required")
	}

	if !c.OnlyConsole && c.LokiHost == "" {
		return newConfigFieldError("LokiHost", "is required when OnlyConsole is false")
	}

	if c.BatchSize <= 0 {
		return newConfigFieldError("BatchSize", "must be greater than 0")
	}

	if c.FlushInterval <= 0 {
		return newConfigFieldError("FlushInterval", "must be greater than 0")
	}

	if c.MaxRetries < 0 {
		return newConfigFieldError("MaxRetries", "cannot be negative")
	}

	if c.Timeout <= 0 {
		return newConfigFieldError("Timeout", "must be greater than 0")
	}

	return nil
}
