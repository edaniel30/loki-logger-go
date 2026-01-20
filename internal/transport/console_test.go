package transport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/edaniel30/loki-logger-go/types"
	"github.com/stretchr/testify/assert"
)

func TestConsoleTransport(t *testing.T) {
	ct := NewConsoleTransport()

	// Test Name
	assert.Equal(t, "console", ct.Name())

	// Test formatLevel for all levels
	assert.Equal(t, colorCyan+"[DEBUG]"+colorReset, ct.formatLevel(types.LevelDebug))
	assert.Equal(t, colorGreen+"[INFO]"+colorReset, ct.formatLevel(types.LevelInfo))
	assert.Equal(t, colorYellow+"[WARN]"+colorReset, ct.formatLevel(types.LevelWarn))
	assert.Equal(t, colorRed+"[ERROR]"+colorReset, ct.formatLevel(types.LevelError))
	assert.Equal(t, colorMagenta+"[FATAL]"+colorReset, ct.formatLevel(types.LevelFatal))

	// Test formatFields
	assert.Equal(t, "", ct.formatFields(types.Fields{}))
	assert.Equal(t, "key=value", ct.formatFields(types.Fields{"key": "value"}))

	// Test sorted fields
	result := ct.formatFields(types.Fields{
		"zebra": "last",
		"apple": "first",
		"moon":  "middle",
	})
	assert.Equal(t, "apple=first moon=middle zebra=last", result)

	// Test fields with different types
	result = ct.formatFields(types.Fields{
		"string": "text",
		"number": 42,
		"float":  3.14,
		"bool":   true,
	})
	assert.Contains(t, result, "bool=true")
	assert.Contains(t, result, "float=3.14")
	assert.Contains(t, result, "number=42")
	assert.Contains(t, result, "string=text")

	// Test format with fields
	entry := &types.Entry{
		Level:     types.LevelInfo,
		Message:   "test message",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		Fields:    types.Fields{"user": "john"},
	}
	result = ct.format(entry)
	assert.Contains(t, result, "2024-01-15 10:30:45")
	assert.Contains(t, result, "[INFO]")
	assert.Contains(t, result, "test message")
	assert.Contains(t, result, "user=john")
	assert.True(t, strings.HasSuffix(result, "\n"))

	// Test format without fields
	entry = &types.Entry{
		Level:     types.LevelError,
		Message:   "error occurred",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
		Fields:    types.Fields{},
	}
	result = ct.format(entry)
	assert.Contains(t, result, "2024-01-15 10:30:45")
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "error occurred")
	assert.True(t, strings.HasSuffix(result, "\n"))

	ctx := context.Background()
	assert.NoError(t, ct.Write(ctx, entry))
	assert.NoError(t, ct.Flush(ctx))
	assert.NoError(t, ct.Close())
}
