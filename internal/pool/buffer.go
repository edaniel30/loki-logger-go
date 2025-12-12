package pool

import (
	"bytes"
	"sync"
)

const (
	// maxBufferSize is the maximum buffer size to keep in the pool.
	// Buffers larger than this will be discarded to prevent memory bloat.
	// Set to 256KB to accommodate logs with full stack traces (typically 10-50KB)
	// plus JSON encoding overhead and structured fields.
	maxBufferSize = 256 * 1024 // 256KB
)

// bufferPool provides a global pool of byte buffers to reduce allocations.
// Buffers are used for:
// - Formatting log lines as JSON (with message, fields, and stack traces)
// - Building label keys for stream grouping
// Typical usage sizes:
// - Simple log: ~1-2KB
// - Log with stack trace: 10-50KB
// - Log with many fields: 5-10KB
var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// Get retrieves a buffer from the pool and resets it for use.
// The caller must return the buffer using Put when done.
func Get() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns a buffer to the pool for reuse.
// Buffers exceeding maxBufferSize are discarded to prevent memory leaks.
func Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	// Discard oversized buffers to prevent pool bloat
	if buf.Cap() > maxBufferSize {
		return
	}

	bufferPool.Put(buf)
}
