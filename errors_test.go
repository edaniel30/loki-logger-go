package loki

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "AppName",
		Message: "is required",
	}
	assert.Equal(t, "loki: config error [AppName]: is required", err.Error())

	errNoField := &ConfigError{
		Field:   "",
		Message: "invalid configuration",
	}
	assert.Equal(t, "loki: config error: invalid configuration", errNoField.Error())

	var genericErr error = err
	var configErr *ConfigError
	require.True(t, errors.As(genericErr, &configErr))
	assert.Equal(t, "AppName", configErr.Field)
	assert.Equal(t, "is required", configErr.Message)
}

func TestTransportError(t *testing.T) {
	cause := errors.New("network timeout")
	err := &TransportError{
		Transport: "loki",
		Op:        "write",
		Cause:     cause,
	}

	assert.Equal(t, "loki: transport 'loki' operation 'write' failed: network timeout", err.Error())

	errNoCause := &TransportError{
		Transport: "console",
		Op:        "flush",
		Cause:     nil,
	}
	assert.Equal(t, "loki: transport 'console' operation 'flush' failed", errNoCause.Error())

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))

	innerErr := errors.New("disk full")
	wrappedErr := fmt.Errorf("write failed: %w", innerErr)
	errWrapped := &TransportError{
		Transport: "console",
		Op:        "write",
		Cause:     wrappedErr,
	}
	assert.True(t, errors.Is(errWrapped, innerErr))

	var genericErr error = err
	var transportErr *TransportError
	require.True(t, errors.As(genericErr, &transportErr))
	assert.Equal(t, "loki", transportErr.Transport)
	assert.Equal(t, "write", transportErr.Op)
}

func TestClientError(t *testing.T) {
	cause := errors.New("connection timeout")
	err := &ClientError{
		Method: "POST",
		URL:    "http://localhost:3100/loki/api/v1/push",
		Cause:  cause,
	}

	// Test Error() with cause
	assert.Equal(t, "loki: client error [POST http://localhost:3100/loki/api/v1/push]: connection timeout", err.Error())

	// Test Error() without cause
	errNoCause := &ClientError{
		Method: "GET",
		URL:    "http://localhost:3100/ready",
		Cause:  nil,
	}
	assert.Equal(t, "loki: client error [GET http://localhost:3100/ready]", errNoCause.Error())

	// Test Unwrap
	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))

	// Test wrapped error chain
	innerErr := errors.New("no route to host")
	wrappedErr := fmt.Errorf("HTTP request failed: %w", innerErr)
	errWrapped := &ClientError{
		Method: "POST",
		URL:    "http://unreachable:3100/push",
		Cause:  wrappedErr,
	}
	assert.True(t, errors.Is(errWrapped, innerErr))

	// Test type assertion
	var genericErr error = err
	var clientErr *ClientError
	require.True(t, errors.As(genericErr, &clientErr))
	assert.Equal(t, "POST", clientErr.Method)
	assert.Equal(t, "http://localhost:3100/loki/api/v1/push", clientErr.URL)
}
