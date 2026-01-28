package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/edaniel30/loki-logger-go/internal/client/models"
	"github.com/edaniel30/loki-logger-go/types"
)

const (
	// lokiPushEndpoint is the API endpoint for pushing logs to Loki
	lokiPushEndpoint = "/loki/api/v1/push"

	// initialBackoffMS is the initial backoff duration for retries in milliseconds
	initialBackoffMS = 100

	// maxErrorBodySize limits the size of error response bodies to prevent memory exhaustion
	maxErrorBodySize = 1024 // 1KB
)

// Client handles HTTP communication with Loki server.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	maxRetries int
}

// NewClient creates a new Loki HTTP client.
func NewClient(baseURL string, username string, password string, timeout time.Duration, maxRetries int) *Client {
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// Push sends log entries to Loki with automatic retries.
func (c *Client) Push(ctx context.Context, entries []*types.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	payload, err := c.buildPayload(entries)
	if err != nil {
		return fmt.Errorf("failed to build payload: %w", err)
	}

	return c.sendWithRetry(ctx, payload)
}

// buildPayload constructs the JSON payload expected by Loki's push API.
func (c *Client) buildPayload(entries []*types.Entry) ([]byte, error) {
	// Group entries by label set
	streams := make(map[string]*models.Stream)

	for _, entry := range entries {
		labelKey := c.labelsToKey(entry.Labels)

		s, exists := streams[labelKey]
		if !exists {
			s = &models.Stream{
				Stream: entry.Labels,
				Values: make([][]string, 0),
			}
			streams[labelKey] = s
		}

		// Format entry as JSON for the log line
		logLine, err := c.formatLogLine(entry)
		if err != nil {
			return nil, err
		}

		// Loki expects [timestamp_nanoseconds, log_line]
		timestamp := strconv.FormatInt(entry.Timestamp.UnixNano(), 10)
		s.Values = append(s.Values, []string{timestamp, logLine})
	}

	// Build final payload
	payload := models.PushRequest{
		Streams: make([]*models.Stream, 0, len(streams)),
	}

	for _, s := range streams {
		payload.Streams = append(payload.Streams, s)
	}

	return json.Marshal(payload)
}

// formatLogLine converts an entry to a JSON log line.
func (c *Client) formatLogLine(entry *types.Entry) (string, error) {
	buf := Get()
	defer Put(buf)

	data := make(map[string]any)
	data["level"] = entry.Level.String()
	data["message"] = entry.Message

	// Add all custom fields
	maps.Copy(data, entry.Fields)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	// Remove trailing newline added by encoder
	line := buf.String()
	if line != "" && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	return line, nil
}

// labelsToKey creates a unique key from labels for grouping.
// Keys are sorted alphabetically to ensure deterministic ordering,
// since Go map iteration order is non-deterministic.
func (c *Client) labelsToKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	buf := Get()
	defer Put(buf)

	// Extract and sort keys for deterministic ordering
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build key string with sorted labels
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(labels[k])
		buf.WriteString(";")
	}

	return buf.String()
}

// sendWithRetry attempts to send the payload with exponential backoff.
func (c *Client) sendWithRetry(ctx context.Context, payload []byte) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, 800ms, etc.
			backoff := time.Duration(initialBackoffMS*(1<<uint(attempt-1))) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if err := c.send(ctx, payload); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", c.maxRetries, lastErr)
}

// send performs the actual HTTP request to Loki.
func (c *Client) send(ctx context.Context, payload []byte) error {
	url := c.baseURL + lokiPushEndpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add basic auth if credentials are provided
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Limit error response body size to prevent memory exhaustion
		limitedReader := io.LimitReader(resp.Body, maxErrorBodySize)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return fmt.Errorf("loki returned status %d (failed to read response body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
