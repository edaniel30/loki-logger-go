package loki

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An error that keeps an HTTP request, as AWS SDK errors do: encoding it
// fails on the request's GetBody func
type sdkError struct{ Request *http.Request }

func (e *sdkError) Error() string { return "GetObject: 403 AccessDenied" }

func TestEncodableFields(t *testing.T) {
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://r2.example/x", nil)
	require.NoError(t, err)
	req.GetBody = func() (io.ReadCloser, error) { return nil, nil }
	wrapped := fmt.Errorf("failed to download the email: %w", &sdkError{Request: req})

	fields := map[string]any{
		"error":   wrapped,
		"plain":   errors.New("boom"),
		"user_id": 42,
		"handler": func() {},
		"nothing": nil,
	}
	_, err = json.Marshal(fields)
	require.Error(t, err)

	out := encodableFields(fields)
	_, err = json.Marshal(out)
	require.NoError(t, err)
	assert.Equal(t, "failed to download the email: GetObject: 403 AccessDenied", out["error"])
	assert.Equal(t, "boom", out["plain"], "an error is its message, not {}")
	assert.Equal(t, 42, out["user_id"])
	assert.IsType(t, "", out["handler"])
	assert.Nil(t, out["nothing"])
	assert.Equal(t, wrapped, fields["error"], "the caller's map is not changed")
	assert.Nil(t, encodableFields(nil))
}
