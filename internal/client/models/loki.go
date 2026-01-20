package models

// PushRequest represents the JSON structure for Loki's push API.
type PushRequest struct {
	Streams []*Stream `json:"streams"`
}

// Stream represents a single log stream with labels and values.
// Each stream groups log entries with the same label set.
type Stream struct {
	Stream map[string]string `json:"stream"` // Labels for this stream
	Values [][]string        `json:"values"` // [[timestamp_ns, log_line], ...]
}
