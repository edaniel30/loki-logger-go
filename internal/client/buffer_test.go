package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferPool(t *testing.T) {
	buf := Get()
	assert.NotNil(t, buf)
	assert.Equal(t, 0, buf.Len())

	// Test buffer reuse (Put and Get)
	buf.WriteString("test data")
	Put(buf)

	buf2 := Get()
	assert.Equal(t, 0, buf2.Len()) // Should be reset

	// Test buffer is reusable
	buf2.WriteString("second")
	assert.Equal(t, "second", buf2.String())
	Put(buf2)

	// Test Put discards oversized buffers
	buf3 := Get()
	largeData := strings.Repeat("x", maxBufferSize+1)
	buf3.WriteString(largeData)
	assert.Greater(t, buf3.Cap(), maxBufferSize)
	Put(buf3) // Should discard

	buf4 := Get()
	defer Put(buf4)
	assert.LessOrEqual(t, buf4.Cap(), maxBufferSize)

	assert.NotPanics(t, func() {
		Put(nil)
	})
}
