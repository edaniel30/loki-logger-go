package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"github.com/edaniel30/loki-logger-go/internal/pool"
	"net/http"
	"strconv"
	"time"
)

// Entry represents a single log record.
// This is duplicated here to avoid import cycles.
type Entry struct {
	Level     string
	Message   string
	Fields    map[string]any
	Timestamp time.Time
	Labels    map[string]string
}

const (
	// lokiPushEndpoint is the API endpoint for pushing logs to Loki
	lokiPushEndpoint = "/loki/api/v1/push"
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
func (c *Client) Push(ctx context.Context, entries []*Entry) error {
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
func (c *Client) buildPayload(entries []*Entry) ([]byte, error) {
	// Group entries by label set
	streams := make(map[string]*stream)

	for _, entry := range entries {
		labelKey := c.labelsToKey(entry.Labels)

		s, exists := streams[labelKey]
		if !exists {
			s = &stream{
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
	payload := pushRequest{
		Streams: make([]*stream, 0, len(streams)),
	}

	for _, s := range streams {
		payload.Streams = append(payload.Streams, s)
	}

	return json.Marshal(payload)
}

// formatLogLine converts an entry to a JSON log line.
func (c *Client) formatLogLine(entry *Entry) (string, error) {
	buf := pool.Get()
	defer pool.Put(buf)

	data := make(map[string]any)
	data["level"] = entry.Level
	data["message"] = entry.Message

	// Add all fields
	for k, v := range entry.Fields {
		data[k] = v
	}

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(data); err != nil {
		return "", err
	}

	// Remove trailing newline added by encoder
	line := buf.String()
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}

	return line, nil
}

// labelsToKey creates a unique key from labels for grouping.
func (c *Client) labelsToKey(labels map[string]string) string {
	buf := pool.Get()
	defer pool.Put(buf)

	for k, v := range labels {
		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(v)
		buf.WriteString(";")
	}

	return buf.String()
}

// sendWithRetry attempts to send the payload with exponential backoff.
func (c *Client) sendWithRetry(ctx context.Context, payload []byte) error {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 100ms, 200ms, 400ms, etc.
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
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
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// pushRequest represents the JSON structure for Loki's push API.
type pushRequest struct {
	Streams []*stream `json:"streams"`
}

// stream represents a single log stream with labels and values.
type stream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"`
}
