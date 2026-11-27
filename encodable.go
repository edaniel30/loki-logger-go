package loki

import (
	"encoding/json"
	"fmt"
)

// encodableFields returns fields with values a log line can carry as JSON.
// An error becomes its message: encoded as is, a plain error is {} and one
// that keeps an HTTP response (AWS SDK errors do, with a func inside) cannot
// be encoded at all. Any other value that cannot be encoded (a func, a
// channel, a struct holding one) becomes its text (%v). The caller's map is
// not changed.
func encodableFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return fields
	}
	out := make(map[string]any, len(fields))
	for k, v := range fields {
		switch value := v.(type) {
		case nil:
			out[k] = nil
		case error:
			out[k] = value.Error()
		default:
			if _, err := json.Marshal(value); err != nil {
				out[k] = fmt.Sprintf("%v", value)
			} else {
				out[k] = value
			}
		}
	}
	return out
}
