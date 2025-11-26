package models

import (
	"time"

	"github.com/edaniel30/loki-logger-go/errors"
)

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

	// EnableConsole enables logging to console in addition to other transports
	OnlyConsole bool
}

func DefaultConfig() Config {
	return Config{
		AppName:       "app",
		LokiHost:      "http://localhost:3100",
		LogLevel:      LevelInfo,
		Labels:        make(map[string]string),
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
		OnlyConsole:   true,
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

func WithOnlyConsole(enabled bool) Option {
	return func(c *Config) {
		c.OnlyConsole = enabled
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
