package transport

import (
	"context"
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLokiTransport(t *testing.T) {
	config := LokiTransportConfig{
		LokiURL:       "http://localhost:3100",
		BatchSize:     10,
		FlushInterval: 1 * time.Hour,
		MaxRetries:    3,
		Timeout:       10 * time.Second,
	}
	lt := NewLokiTransport(config)
	defer lt.Close()

	assert.Equal(t, "loki", lt.Name())

	entry := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "test",
		Timestamp: time.Now(),
		Labels:    types.Labels{"app": "test"},
		Fields:    types.Fields{},
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
	lt := NewLokiTransport(config)
	defer lt.Close()

	entry := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "test",
		Timestamp: time.Now(),
		Labels:    types.Labels{"app": "test"},
		Fields:    types.Fields{},
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

func TestLokiTransport_Close(t *testing.T) {
	config := LokiTransportConfig{
		LokiURL:       "http://localhost:3100",
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
		MaxRetries:    3,
		Timeout:       1 * time.Second,
	}
	lt := NewLokiTransport(config)

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
