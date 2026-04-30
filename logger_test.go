package loki

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/mocks"
	"github.com/edaniel30/loki-logger-go/internal/transport"
	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig() *Config {
	return &Config{
		AppName:           "test-app",
		AppVersion:        "1.0.0",
		AppEnv:            "local",
		OnlyConsole: true,
		BatchSize:   100,
		FlushInterval:     5 * time.Second,
		MaxRetries:        3,
		Timeout:           10 * time.Second,
		Labels:            make(types.Labels),
	}
}

func newTestLogger(t *testing.T) *Logger {
	logger, err := New(newTestConfig())
	require.NoError(t, err)
	return logger
}

func newTestLoggerWithMock(t *testing.T) (*Logger, *mocks.MockTransport) {
	logger := newTestLogger(t)
	mock := mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}
	return logger, mock
}

func TestNew(t *testing.T) {
	logger, err := New(&Config{
		AppName:       "test-app",
		LokiHost:      "http://localhost:3100",
		OnlyConsole:   false,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, logger)
	assert.Equal(t, "test-app", logger.config.AppName)
	assert.Equal(t, 2, len(logger.transports)) // console + loki

	logger, err = New(&Config{
		AppName:       "test-app",
		OnlyConsole:   true,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, len(logger.transports)) // console only

	logger, err = New(&Config{
		AppName:       "test-app",
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	},
		WithLokiHost("http://loki:3100"),
		WithLogLevel(types.LevelWarn),
		WithBatchSize(200),
	)
	require.NoError(t, err)
	assert.Equal(t, "http://loki:3100", logger.config.LokiHost)
	assert.Equal(t, types.LevelWarn, logger.config.LogLevel)
	assert.Equal(t, 200, logger.config.BatchSize)

	logger, err = New(&Config{AppName: ""})
	require.Error(t, err)
	assert.Nil(t, logger)
	var configErr *ConfigError
	require.ErrorAs(t, err, &configErr)
	assert.Equal(t, "AppName", configErr.Field)
}

func TestLoggerLogLevels(t *testing.T) {
	logger, mock := newTestLoggerWithMock(t)
	ctx := context.Background()

	logger.Debug(ctx, "debug", nil)
	logger.Info(ctx, "info", nil)
	logger.Warn(ctx, "warn", nil)
	logger.Error(ctx, "error", nil)
	logger.Fatal(ctx, "fatal", nil)

	entries := mock.GetEntries()
	require.Len(t, entries, 5)
	assert.Equal(t, types.LevelDebug, entries[0].Level)
	assert.Equal(t, types.LevelInfo, entries[1].Level)
	assert.Equal(t, types.LevelWarn, entries[2].Level)
	assert.Equal(t, types.LevelError, entries[3].Level)
	assert.Equal(t, types.LevelFatal, entries[4].Level)
}

func TestLoggerLabels(t *testing.T) {
	cfg := newTestConfig()
	cfg.AppName = "my-app"
	logger, _ := New(cfg)
	mock := mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	logger.Info(context.Background(), "test", nil)
	entries := mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "my-app", entries[0].Labels["app"])

	mock.Reset()
	logger.Error(context.Background(), "error", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Labels["level"])

	cfg = newTestConfig()
	cfg.Labels = types.Labels{
		"team":   "backend",
		"region": "us-east-1",
	}
	logger, _ = New(cfg)
	mock = mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	logger.Info(context.Background(), "test", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	// System labels are always set and cannot be overridden
	assert.Equal(t, "local", entries[0].Labels["environment"]) // default value
	assert.Equal(t, "test-app", entries[0].Labels["app"])      // from config
	// User labels work normally
	assert.Equal(t, "backend", entries[0].Labels["team"])
	assert.Equal(t, "us-east-1", entries[0].Labels["region"])
}

func TestLoggerFields(t *testing.T) {
	logger, mock := newTestLoggerWithMock(t)

	logger.Info(context.Background(), "test", map[string]any{
		"user_id":  12345,
		"action":   "login",
		"duration": 1.5,
	})
	entries := mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, 12345, entries[0].Fields["user_id"])
	assert.Equal(t, "login", entries[0].Fields["action"])
	assert.Equal(t, 1.5, entries[0].Fields["duration"])

	mock.Reset()
	logger.Info(context.Background(), "test", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.NotNil(t, entries[0].Fields)
	// Verify automatic file and line fields are added
	assert.Contains(t, entries[0].Fields, "file")
	assert.Contains(t, entries[0].Fields, "line")
	// Verify the types and format
	assert.IsType(t, "", entries[0].Fields["file"])
	assert.IsType(t, 0, entries[0].Fields["line"])
	assert.NotEmpty(t, entries[0].Fields["file"])
	assert.Greater(t, entries[0].Fields["line"], 0)

	// Test that user-provided file and line fields are not overwritten
	mock.Reset()
	logger.Info(context.Background(), "test", map[string]any{
		"file": "custom_file.go",
		"line": 999,
	})
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "custom_file.go", entries[0].Fields["file"])
	assert.Equal(t, 999, entries[0].Fields["line"])
}

func TestLoggerStackTrace(t *testing.T) {
	cfg := newTestConfig()
	logger, _ := New(cfg)
	mock := mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	logger.Error(context.Background(), "error message", nil)
	entries := mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "Stack trace:")
	assert.Contains(t, entries[0].Message, "error message")

	mock.Reset()
	logger.Fatal(context.Background(), "fatal error", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "Stack trace:")

	// Info and Debug should NOT have stack traces
	mock.Reset()
	logger.Info(context.Background(), "info message", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.NotContains(t, entries[0].Message, "Stack trace:")
	assert.Equal(t, "info message", entries[0].Message)

	mock.Reset()
	logger.Debug(context.Background(), "debug message", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.NotContains(t, entries[0].Message, "Stack trace:")
}

func TestLoggerWithOnFlushError(t *testing.T) {
	// Use a deterministic error server instead of a fixed unreachable port.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	received := make(chan error, 1)

	cfg := DefaultConfig()
	cfg.MaxRetries = 0
	cfg.Timeout = 100 * time.Millisecond

	logger, err := New(
		cfg,
		WithAppName("test-app"),
		WithLokiHost(srv.URL),
		WithBatchSize(1),
		WithFlushInterval(1*time.Hour),
		WithOnFlushError(func(err error) {
			select {
			case received <- err:
			default:
			}
		}),
	)
	require.NoError(t, err)
	defer func() { _ = logger.Close() }()

	// Remove the console transport so no colored output is written to stdout during tests.
	// transports[0] is always ConsoleTransport, transports[1] is LokiTransport.
	logger.transports = logger.transports[1:]

	ctx := context.Background()
	logger.Info(ctx, "hello", nil)

	select {
	case err := <-received:
		assert.ErrorContains(t, err, "failed to push to Loki")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected OnFlushError to be called")
	}
}

func TestLoggerClose(t *testing.T) {
	logger, mock := newTestLoggerWithMock(t)
	err := logger.Close()
	assert.NoError(t, err)
	assert.Equal(t, 1, mock.FlushCalled)
	assert.Equal(t, 1, mock.CloseCalled)

	logger, mock = newTestLoggerWithMock(t)
	mock.FlushErr = errors.New("flush failed")
	err = logger.Close()
	assert.EqualError(t, err, "flush failed")

	logger, mock = newTestLoggerWithMock(t)
	mock.CloseErr = errors.New("close failed")
	err = logger.Close()
	assert.EqualError(t, err, "close failed")
}

func TestLoggerWithLabels(t *testing.T) {
	cfg := newTestConfig()
	cfg.Labels = types.Labels{"team": "platform", "region": "us-east"}
	logger, _ := New(cfg)
	mock := mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	childLogger := logger.WithLabels(types.Labels{"component": "auth", "service": "api"})
	childLogger.transports = []transport.Transport{mock}

	childLogger.Info(context.Background(), "test", nil)
	entries := mock.GetEntries()
	require.Len(t, entries, 1)

	// User-provided labels
	assert.Equal(t, "platform", entries[0].Labels["team"])
	assert.Equal(t, "us-east", entries[0].Labels["region"])
	assert.Equal(t, "auth", entries[0].Labels["component"])
	assert.Equal(t, "api", entries[0].Labels["service"])
	// System labels (reserved, cannot be overridden)
	assert.Equal(t, "test-app", entries[0].Labels["app"])
	assert.Equal(t, "info", entries[0].Labels["level"])
	assert.Equal(t, "1.0.0", entries[0].Labels["version"]) // default value
	assert.Equal(t, "local", entries[0].Labels["environment"]) // default value

	cfg = newTestConfig()
	cfg.Labels = types.Labels{"env": "prod"}
	logger, _ = New(cfg)

	childLogger = logger.WithLabels(types.Labels{"component": "auth"})
	assert.Equal(t, "prod", logger.config.Labels["env"])
	assert.NotContains(t, logger.config.Labels, "component")
	assert.Equal(t, "prod", childLogger.config.Labels["env"])
	assert.Equal(t, "auth", childLogger.config.Labels["component"])

	cfg = newTestConfig()
	cfg.Labels = types.Labels{"env": "dev"}
	logger, _ = New(cfg)
	mock = mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	childLogger = logger.WithLabels(types.Labels{"env": "prod"})
	childLogger.transports = []transport.Transport{mock}

	childLogger.Info(context.Background(), "test", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "prod", entries[0].Labels["env"])
}
