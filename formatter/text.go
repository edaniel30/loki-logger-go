package formatter

import (
	"fmt"
	"sort"
	"strings"
)

// TextFormatter formats log entries as human-readable text.
type TextFormatter struct {
	// DisableColors disables ANSI color codes in output
	DisableColors bool
	// DisableTimestamp removes timestamp from output
	DisableTimestamp bool
}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		DisableColors:    false,
		DisableTimestamp: false,
	}
}

// Format converts a log entry to text format.
func (f *TextFormatter) Format(entry *Entry) ([]byte, error) {
	var b strings.Builder

	if !f.DisableTimestamp {
		b.WriteString(entry.Timestamp.Format("2006-01-02 15:04:05"))
		b.WriteString(" ")
	}

	levelStr := f.formatLevel(entry.Level)
	b.WriteString(levelStr)
	b.WriteString(" ")

	b.WriteString(entry.Message)

	if len(entry.Fields) > 0 {
		b.WriteString(" ")
		b.WriteString(f.formatFields(entry.Fields))
	}

	b.WriteString("\n")

	return []byte(b.String()), nil
}

func (f *TextFormatter) formatLevel(level string) string {
	levelStr := strings.ToUpper(level)

	if f.DisableColors {
		return fmt.Sprintf("[%s]", levelStr)
	}

	var color string
	switch level {
	case "debug":
		color = "\033[36m" // Cyan
	case "info":
		color = "\033[32m" // Green
	case "warn":
		color = "\033[33m" // Yellow
	case "error":
		color = "\033[31m" // Red
	case "fatal":
		color = "\033[35m" // Magenta
	default:
		color = "\033[0m" // Reset
	}

	return fmt.Sprintf("%s[%s]\033[0m", color, levelStr)
}

func (f *TextFormatter) formatFields(fields map[string]interface{}) string {
	if len(fields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return strings.Join(parts, " ")
}
