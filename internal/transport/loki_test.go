package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newErrorServer returns an httptest.Server that always responds with 500 Internal Server Error.
// This deterministically triggers flush errors without relying on unreachable ports.
func newErrorServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLokiTransport(t *testing.T) {
	config := LokiTransportConfig{
		LokiURL:       "http://localhost:3100",
		BatchSize:     10,
		FlushInterval: 1 * time.Hour,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	}
	lt := NewLokiTransport(&config)
	defer func() { _ = lt.Close() }()

	assert.Equal(t, "loki", lt.Name())

	entry := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "test",
		Timestamp: time.Now(),
		Labels:    types.Labels{"app": "test"},
		Fields:    map[string]any{},
	}

	ctx := context.Background()
	err := lt.Write(ctx, entry)
	require.NoError(t, err)

	lt.mu.Lock()
	assert.Equal(t, 1, len(lt.buffer))
	lt.mu.Unlock()

	_ = lt.Flush(ctx) // Will fail but clears buffer

	lt.mu.Lock()
	assert.Equal(t, 0, len(lt.buffer))
	lt.mu.Unlock()

	err = lt.Flush(ctx)
	assert.NoError(t, err)
}

func TestLokiTransport_BatchFlush(t *testing.T) {
	config := LokiTransportConfig{
		LokiURL:       "http://localhost:3100",
		BatchSize:     2,
		FlushInterval: 1 * time.Hour,
		MaxRetries:    0,
		Timeout:       10 * time.Second,
	}
	lt := NewLokiTransport(&config)
	defer func() { _ = lt.Close() }()

	entry := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "test",
		Timestamp: time.Now(),
		Labels:    types.Labels{"app": "test"},
		Fields:    map[string]any{},
	}

	ctx := context.Background()

	err := lt.Write(ctx, entry)
	require.NoError(t, err)

	lt.mu.Lock()
	bufferLen := len(lt.buffer)
	lt.mu.Unlock()
	assert.Equal(t, 1, bufferLen)

	_ = lt.Write(ctx, entry)

	lt.mu.Lock()
	bufferLen = len(lt.buffer)
	lt.mu.Unlock()
	assert.Equal(t, 0, bufferLen)
}

func TestLokiTransport_OnFlushError(t *testing.T) {
	t.Run("callback is invoked on flush error", func(t *testing.T) {
		srv := newErrorServer(t)
		var called []error
		config := LokiTransportConfig{
			LokiURL:       srv.URL,
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			MaxRetries:    0,
			Timeout:       100 * time.Millisecond,
			OnFlushError: func(err error) {
				called = append(called, err)
			},
		}
		lt := NewLokiTransport(&config)
		defer func() { _ = lt.Close() }()

		entry := &types.Entry{
			Level:     types.LevelInfo,
			Message:   "test",
			Timestamp: time.Now(),
			Labels:    types.Labels{"app": "test"},
			Fields:    map[string]any{},
		}

		// Write triggers immediate flush (batch size = 1)
		ctx := context.Background()
		_ = lt.Write(ctx, entry)

		// Flush errors arrive asynchronously via background flusher —
		// but Write-triggered flush is synchronous, so callback fires inline.
		assert.Len(t, called, 1)
		assert.ErrorContains(t, called[0], "failed to push to Loki")
	})

	t.Run("no callback does not panic", func(t *testing.T) {
		srv := newErrorServer(t)
		config := LokiTransportConfig{
			LokiURL:       srv.URL,
			BatchSize:     1,
			FlushInterval: 1 * time.Hour,
			MaxRetries:    0,
			Timeout:       100 * time.Millisecond,
			OnFlushError:  nil,
		}
		lt := NewLokiTransport(&config)
		defer func() { _ = lt.Close() }()

		entry := &types.Entry{
			Level:     types.LevelInfo,
			Message:   "test",
			Timestamp: time.Now(),
			Labels:    types.Labels{"app": "test"},
			Fields:    map[string]any{},
		}

		ctx := context.Background()
		assert.NotPanics(t, func() {
			_ = lt.Write(ctx, entry)
		})
	})

	t.Run("background flush calls callback", func(t *testing.T) {
		srv := newErrorServer(t)
		received := make(chan error, 1)
		config := LokiTransportConfig{
			LokiURL:       srv.URL,
			BatchSize:     100,
			FlushInterval: 50 * time.Millisecond,
			MaxRetries:    0,
			Timeout:       100 * time.Millisecond,
			OnFlushError: func(err error) {
				select {
				case received <- err:
				default:
				}
			},
		}
		lt := NewLokiTransport(&config)
		defer func() { _ = lt.Close() }()

		ctx := context.Background()
		entry := &types.Entry{
			Level:     types.LevelInfo,
			Message:   "test",
			Timestamp: time.Now(),
			Labels:    types.Labels{"app": "test"},
			Fields:    map[string]any{},
		}
		_ = lt.Write(ctx, entry)

		select {
		case err := <-received:
			assert.ErrorContains(t, err, "failed to push to Loki")
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected OnFlushError to be called by background flusher")
		}
	})
}

func TestLokiTransport_Close(t *testing.T) {
	config := LokiTransportConfig{
		LokiURL:       "http://localhost:3100",
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
		MaxRetries:    3,
		Timeout:       1 * time.Second,
	}
	lt := NewLokiTransport(&config)

	time.Sleep(100 * time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- lt.Close()
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
		select {
		case <-lt.doneCh:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("background flusher did not stop")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close() hung")
	}
}
