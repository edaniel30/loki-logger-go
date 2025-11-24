package formatter

import (
	"encoding/json"
)

// JSONFormatter formats log entries as JSON.
type JSONFormatter struct {
	// PrettyPrint enables indented JSON output for readability
	PrettyPrint bool
}

// NewJSONFormatter creates a new JSON formatter.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		PrettyPrint: false,
	}
}

// Format converts a log entry to JSON format.
func (f *JSONFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(map[string]interface{})

	// Add standard fields
	data["level"] = entry.Level
	data["message"] = entry.Message
	data["timestamp"] = entry.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")

	// Add custom fields
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			data[k] = v
		}
	}

	// Add labels if present
	if len(entry.Labels) > 0 {
		data["labels"] = entry.Labels
	}

	if f.PrettyPrint {
		return json.MarshalIndent(data, "", "  ")
	}

	return json.Marshal(data)
}
