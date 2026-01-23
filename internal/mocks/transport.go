package mocks

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
)

// MockTransport is a test transport that captures log entries
type MockTransport struct {
	name        string
	entries     []*types.Entry
	WriteErr    error
	FlushErr    error
	CloseErr    error
	mu          sync.Mutex
	WriteCalled int
	FlushCalled int
	CloseCalled int
	WriteDelay  time.Duration // Simulate slow writes
}

func NewMockTransport(name string) *MockTransport {
	return &MockTransport{
		name:    name,
		entries: make([]*types.Entry, 0),
	}
}

func (m *MockTransport) Name() string {
	return m.name
}

func (m *MockTransport) Write(ctx context.Context, entries ...*types.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.WriteCalled++

	if m.WriteDelay > 0 {
		select {
		case <-time.After(m.WriteDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if m.WriteErr != nil {
		return m.WriteErr
	}

	// Deep copy entries to avoid race conditions
	for _, entry := range entries {
		entryCopy := &types.Entry{
			Level:     entry.Level,
			Message:   entry.Message,
			Timestamp: entry.Timestamp,
			Fields:    make(map[string]any),
			Labels:    make(types.Labels),
		}

		maps.Copy(entryCopy.Fields, entry.Fields)
		maps.Copy(entryCopy.Labels, entry.Labels)

		m.entries = append(m.entries, entryCopy)
	}
	return nil
}

func (m *MockTransport) Flush(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.FlushCalled++

	if m.FlushErr != nil {
		return m.FlushErr
	}

	return nil
}

func (m *MockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CloseCalled++

	if m.CloseErr != nil {
		return m.CloseErr
	}

	return nil
}

func (m *MockTransport) GetEntries() []*types.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*types.Entry, len(m.entries))
	copy(result, m.entries)
	return result
}

func (m *MockTransport) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = make([]*types.Entry, 0)
	m.WriteCalled = 0
	m.FlushCalled = 0
	m.CloseCalled = 0
}
