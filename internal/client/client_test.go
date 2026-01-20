package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_formatLogLine(t *testing.T) {
	c := NewClient("http://localhost:3100", "", "", 10*time.Second, 3)

	entry := &types.Entry{
		Level:   types.LevelInfo,
		Message: "test message",
		Fields:  types.Fields{},
	}
	line, err := c.formatLogLine(entry)
	require.NoError(t, err)

	var data map[string]any
	err = json.Unmarshal([]byte(line), &data)
	require.NoError(t, err)
	assert.Equal(t, "info", data["level"])
	assert.Equal(t, "test message", data["message"])

	entry = &types.Entry{
		Level:   types.LevelError,
		Message: "error occurred",
		Fields: types.Fields{
			"user_id": 123,
			"action":  "login",
		},
	}
	line, err = c.formatLogLine(entry)
	require.NoError(t, err)

	err = json.Unmarshal([]byte(line), &data)
	require.NoError(t, err)
	assert.Equal(t, "error", data["level"])
	assert.Equal(t, "error occurred", data["message"])
	assert.Equal(t, float64(123), data["user_id"])
	assert.Equal(t, "login", data["action"])
}

func TestClient_labelsToKey(t *testing.T) {
	c := NewClient("http://localhost:3100", "", "", 10*time.Second, 3)

	assert.Equal(t, "", c.labelsToKey(types.Labels{}))
	assert.Equal(t, "env=prod;", c.labelsToKey(types.Labels{"env": "prod"}))

	key := c.labelsToKey(types.Labels{
		"env":    "prod",
		"app":    "myapp",
		"region": "us-east",
	})
	assert.Equal(t, "app=myapp;env=prod;region=us-east;", key)

	key1 := c.labelsToKey(types.Labels{"a": "1", "b": "2", "c": "3"})
	key2 := c.labelsToKey(types.Labels{"c": "3", "a": "1", "b": "2"})
	assert.Equal(t, key1, key2)
}

func TestClient_buildPayload(t *testing.T) {
	c := NewClient("http://localhost:3100", "", "", 10*time.Second, 3)

	payload, err := c.buildPayload([]*types.Entry{})
	require.NoError(t, err)
	assert.NotNil(t, payload)

	entry := &types.Entry{
		Level:     types.LevelWarn,
		Message:   "warning",
		Timestamp: time.Unix(1234567890, 123456789),
		Labels:    types.Labels{"app": "test"},
		Fields:    types.Fields{"key": "value"},
	}
	payload, err = c.buildPayload([]*types.Entry{entry})
	require.NoError(t, err)

	var data map[string]any
	err = json.Unmarshal(payload, &data)
	require.NoError(t, err)

	streams := data["streams"].([]any)
	require.Equal(t, 1, len(streams))

	stream := streams[0].(map[string]any)
	labels := stream["stream"].(map[string]any)
	assert.Equal(t, "test", labels["app"])

	values := stream["values"].([]any)
	require.Equal(t, 1, len(values))

	logEntry := values[0].([]any)
	require.Equal(t, 2, len(logEntry))
	assert.NotEmpty(t, logEntry[0].(string)) // timestamp

	logLine := logEntry[1].(string)
	var logData map[string]any
	err = json.Unmarshal([]byte(logLine), &logData)
	require.NoError(t, err)
	assert.Equal(t, "warn", logData["level"])
	assert.Equal(t, "warning", logData["message"])
	assert.Equal(t, "value", logData["key"])

	entry1 := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "message1",
		Timestamp: time.Unix(1000, 0),
		Labels:    types.Labels{"app": "test", "env": "prod"},
		Fields:    types.Fields{},
	}
	entry2 := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "message2",
		Timestamp: time.Unix(1001, 0),
		Labels:    types.Labels{"app": "test", "env": "prod"}, // Same labels
		Fields:    types.Fields{},
	}
	entry3 := &types.Entry{
		Level:     types.LevelError,
		Message:   "message3",
		Timestamp: time.Unix(1002, 0),
		Labels:    types.Labels{"app": "test", "env": "dev"}, // Different labels
		Fields:    types.Fields{},
	}

	payload, err = c.buildPayload([]*types.Entry{entry1, entry2, entry3})
	require.NoError(t, err)

	err = json.Unmarshal(payload, &data)
	require.NoError(t, err)

	streams = data["streams"].([]any)
	assert.Equal(t, 2, len(streams)) // 2 streams: prod and dev
}

func TestClient_Push(t *testing.T) {
	c := NewClient("http://localhost:3100", "", "", 10*time.Second, 3)
	err := c.Push(context.Background(), []*types.Entry{})
	assert.NoError(t, err)
}

func TestClient_send(t *testing.T) {
	// Test successful send
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/loki/api/v1/push", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, "", "", 10*time.Second, 3)
	err := c.send(context.Background(), []byte(`{"test":"data"}`))
	assert.NoError(t, err)

	// Test with basic auth
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "user", username)
		assert.Equal(t, "pass", password)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c = NewClient(server.URL, "user", "pass", 10*time.Second, 3)
	err = c.send(context.Background(), []byte(`{"test":"data"}`))
	assert.NoError(t, err)

	// Test server error
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	c = NewClient(server.URL, "", "", 10*time.Second, 3)
	err = c.send(context.Background(), []byte(`{"test":"data"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "server error")
}

func TestClient_sendWithRetry(t *testing.T) {
	// Test success on first attempt
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClient(server.URL, "", "", 10*time.Second, 3)
	err := c.sendWithRetry(context.Background(), []byte(`{"test":"data"}`))
	assert.NoError(t, err)

	// Test retry and eventual success
	attempts := 0
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	c = NewClient(server.URL, "", "", 10*time.Second, 3)
	err = c.sendWithRetry(context.Background(), []byte(`{"test":"data"}`))
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)

	// Test failure after max retries
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c = NewClient(server.URL, "", "", 10*time.Second, 2)
	err = c.sendWithRetry(context.Background(), []byte(`{"test":"data"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed after 2 retries")
}
