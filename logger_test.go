package loki

import (
	"context"
	"errors"
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
		AppName:       "test-app",
		OnlyConsole:   true,
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
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

	mock.Reset()
	logger.Log(ctx, types.LevelWarn, "warning", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, types.LevelWarn, entries[0].Level)
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
		"environment": "production",
		"region":      "us-east-1",
	}
	logger, _ = New(cfg)
	mock = mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	logger.Info(context.Background(), "test", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, "production", entries[0].Labels["environment"])
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
	assert.Equal(t, 0, len(entries[0].Fields))
}

func TestLoggerStackTrace(t *testing.T) {
	cfg := newTestConfig()
	cfg.IncludeStackTrace = true
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

	mock.Reset()
	logger.Info(context.Background(), "info message", nil)
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.NotContains(t, entries[0].Message, "Stack trace:")
	assert.Equal(t, "info message", entries[0].Message)

	logger2, mock2 := newTestLoggerWithMock(t)
	logger2.Error(context.Background(), "error message", nil)
	entries = mock2.GetEntries()
	require.Len(t, entries, 1)
	assert.NotContains(t, entries[0].Message, "Stack trace:")

	mock.Reset()
	logger.Error(context.Background(), "error message", map[string]any{
		"_skip_stack_trace": true,
	})
	entries = mock.GetEntries()
	require.Len(t, entries, 1)
	assert.NotContains(t, entries[0].Message, "Stack trace:")
	assert.NotContains(t, entries[0].Fields, "_skip_stack_trace")
}

func TestLoggerErrorHandler(t *testing.T) {
	var handlerCalled bool
	var handlerTransport string
	var handlerErr error

	cfg := newTestConfig()
	cfg.ErrorHandler = func(transport string, err error) {
		handlerCalled = true
		handlerTransport = transport
		handlerErr = err
	}
	logger, _ := New(cfg)
	mock := mocks.NewMockTransport("mock")
	mock.WriteErr = errors.New("write failed")
	logger.transports = []transport.Transport{mock}

	logger.Info(context.Background(), "test", nil)
	assert.True(t, handlerCalled)
	assert.Equal(t, "mock", handlerTransport)
	assert.EqualError(t, handlerErr, "write failed")

	logger2, mock2 := newTestLoggerWithMock(t)
	mock2.WriteErr = errors.New("write failed")
	assert.NotPanics(t, func() {
		logger2.Info(context.Background(), "test", nil)
	})
}

func TestLoggerFlush(t *testing.T) {
	logger := newTestLogger(t)
	mock1 := mocks.NewMockTransport("mock1")
	mock2 := mocks.NewMockTransport("mock2")
	logger.transports = []transport.Transport{mock1, mock2}

	err := logger.Flush()
	assert.NoError(t, err)
	assert.Equal(t, 1, mock1.FlushCalled)
	assert.Equal(t, 1, mock2.FlushCalled)

	logger = newTestLogger(t)
	mock1 = mocks.NewMockTransport("mock1")
	mock1.FlushErr = errors.New("flush error 1")
	mock2 = mocks.NewMockTransport("mock2")
	mock2.FlushErr = errors.New("flush error 2")
	logger.transports = []transport.Transport{mock1, mock2}

	err = logger.Flush()
	assert.EqualError(t, err, "flush error 1")

	var errorCount int
	cfg := newTestConfig()
	cfg.ErrorHandler = func(transport string, err error) {
		errorCount++
	}
	logger, _ = New(cfg)
	mock := mocks.NewMockTransport("mock")
	mock.FlushErr = errors.New("flush failed")
	logger.transports = []transport.Transport{mock}

	_ = logger.Flush()
	assert.Equal(t, 1, errorCount)
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
	cfg.Labels = types.Labels{"env": "prod", "region": "us-east"}
	logger, _ := New(cfg)
	mock := mocks.NewMockTransport("mock")
	logger.transports = []transport.Transport{mock}

	childLogger := logger.WithLabels(types.Labels{"component": "auth", "version": "v1"})
	childLogger.transports = []transport.Transport{mock}

	childLogger.Info(context.Background(), "test", nil)
	entries := mock.GetEntries()
	require.Len(t, entries, 1)

	assert.Equal(t, "prod", entries[0].Labels["env"])
	assert.Equal(t, "us-east", entries[0].Labels["region"])
	assert.Equal(t, "auth", entries[0].Labels["component"])
	assert.Equal(t, "v1", entries[0].Labels["version"])
	assert.Equal(t, "test-app", entries[0].Labels["app"])
	assert.Equal(t, "info", entries[0].Labels["level"])

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
