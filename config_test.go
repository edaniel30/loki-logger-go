package loki

import (
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWithAllOptions(t *testing.T) {
	// Start with default config
	cfg := DefaultConfig()

	// Verify defaults
	assert.Equal(t, "app", cfg.AppName)
	assert.Equal(t, "http://localhost:3100", cfg.LokiHost)
	assert.Equal(t, types.LevelInfo, cfg.LogLevel)
	assert.NotNil(t, cfg.Labels)
	assert.True(t, cfg.IncludeStackTrace)
	assert.False(t, cfg.OnlyConsole)
	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 5*time.Second, cfg.FlushInterval)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Nil(t, cfg.ErrorHandler)

	// Apply ALL options
	testHandler := func(transport string, err error) {}

	WithAppName("test-app")(&cfg)
	WithLokiHost("http://loki:3100")(&cfg)
	WithLokiBasicAuth("admin", "password")(&cfg)
	WithLogLevel(types.LevelDebug)(&cfg)
	WithLabels(types.Labels{"env": "test", "region": "us-east"})(&cfg)
	WithIncludeStackTrace(false)(&cfg)
	WithOnlyConsole(true)(&cfg)
	WithBatchSize(200)(&cfg)
	WithFlushInterval(10 * time.Second)(&cfg)
	WithMaxRetries(5)(&cfg)
	WithTimeout(30 * time.Second)(&cfg)
	WithErrorHandler(testHandler)(&cfg)

	// Verify all options were applied
	assert.Equal(t, "test-app", cfg.AppName)
	assert.Equal(t, "http://loki:3100", cfg.LokiHost)
	assert.Equal(t, "admin", cfg.LokiUsername)
	assert.Equal(t, "password", cfg.LokiPassword)
	assert.Equal(t, types.LevelDebug, cfg.LogLevel)
	assert.Equal(t, "test", cfg.Labels["env"])
	assert.Equal(t, "us-east", cfg.Labels["region"])
	assert.False(t, cfg.IncludeStackTrace)
	assert.True(t, cfg.OnlyConsole)
	assert.Equal(t, 200, cfg.BatchSize)
	assert.Equal(t, 10*time.Second, cfg.FlushInterval)
	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.NotNil(t, cfg.ErrorHandler)
}

func TestConfigValidate(t *testing.T) {
	baseConfig := Config{
		AppName:       "test-app",
		LokiHost:      "http://localhost:3100",
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	}

	tests := []struct {
		name       string
		modify     func(*Config)
		errorField string
		errorMsg   string
	}{
		{
			name:       "empty AppName",
			modify:     func(c *Config) { c.AppName = "" },
			errorField: "AppName",
			errorMsg:   "is required",
		},
		{
			name:       "empty LokiHost when not OnlyConsole",
			modify:     func(c *Config) { c.LokiHost = "" },
			errorField: "LokiHost",
			errorMsg:   "is required when OnlyConsole is false",
		},
		{
			name:       "invalid BatchSize",
			modify:     func(c *Config) { c.BatchSize = 0 },
			errorField: "BatchSize",
			errorMsg:   "must be greater than 0",
		},
		{
			name:       "invalid FlushInterval",
			modify:     func(c *Config) { c.FlushInterval = 0 },
			errorField: "FlushInterval",
			errorMsg:   "must be greater than 0",
		},
		{
			name:       "invalid MaxRetries",
			modify:     func(c *Config) { c.MaxRetries = -1 },
			errorField: "MaxRetries",
			errorMsg:   "cannot be negative",
		},
		{
			name:       "invalid Timeout",
			modify:     func(c *Config) { c.Timeout = 0 },
			errorField: "Timeout",
			errorMsg:   "must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseConfig
			tt.modify(&cfg)

			err := cfg.validate()

			require.Error(t, err)
			var configErr *ConfigError
			require.ErrorAs(t, err, &configErr)
			assert.Equal(t, tt.errorField, configErr.Field)
			assert.Contains(t, configErr.Message, tt.errorMsg)
		})
	}
}
