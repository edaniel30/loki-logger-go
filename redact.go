package loki

import (
	"fmt"
	"reflect"
	"strings"
)

// sensitiveKeyTerms are substrings that mark a field key as sensitive. A key
// is redacted if its lowercased form contains any of these terms, so e.g.
// "phone" alone also catches "display_phone_number" and "phone_number_id"
// without listing every variant. Matching applies at any depth inside nested
// maps and slices (e.g. an HTTP header map or a JSON request body
// unmarshalled into map[string]any).
var sensitiveKeyTerms = []string{
	"phone",
	"wa_id",
	"from",
	"body",
	"name",
	"email",
	"password",
	"token",
	"secret",
	"key",
	"authorization",
	"cookie",
	"signature",
}

// maskRunLength is the fixed number of mask characters used regardless of
// the original string length, so redacted output doesn't leak length.
const maskRunLength = 3

const maskChar = "*"

// redactFields returns a copy of fields with the values of sensitive keys
// masked, keeping only the first and last character of each string value.
// Nested maps and slices (of any concrete type, e.g. map[string][]string
// from http.Header) are walked recursively.
func redactFields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	redacted, ok := redactAny(fields).(map[string]any)
	if !ok {
		return fields
	}
	return redacted
}

func redactAny(v any) any {
	rv := reflect.ValueOf(v)
	//exhaustive:ignore // only Map/Slice/Array/Ptr/Interface need special handling; default covers all other kinds
	switch rv.Kind() {
	case reflect.Map:
		return redactMap(rv)
	case reflect.Slice, reflect.Array:
		return redactSlice(rv)
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return v
		}
		return redactAny(rv.Elem().Interface())
	default:
		return v
	}
}

func redactMap(rv reflect.Value) any {
	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		key := fmt.Sprintf("%v", iter.Key().Interface())
		value := iter.Value().Interface()
		if isSensitiveKey(key) {
			out[key] = redactLeaf(value)
			continue
		}
		out[key] = redactAny(value)
	}
	return out
}

func redactSlice(rv reflect.Value) any {
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = redactAny(rv.Index(i).Interface())
	}
	return out
}

// redactLeaf masks string values directly. A slice/array under a sensitive
// key (e.g. http.Header's []string values) has each string element masked.
// Any other structured value (e.g. a nested map) is walked normally instead
// of being blanket-masked, so sensitive keys further down are still caught.
func redactLeaf(v any) any {
	if s, ok := v.(string); ok {
		return maskString(s)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = redactLeaf(rv.Index(i).Interface())
		}
		return out
	}

	return redactAny(v)
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, term := range sensitiveKeyTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

// maskString keeps the first and last character of s and replaces everything
// in between with a fixed-length run of mask characters.
func maskString(s string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return s
	}
	if n <= 2 {
		return strings.Repeat(maskChar, n)
	}
	return string(runes[0]) + strings.Repeat(maskChar, maskRunLength) + string(runes[n-1])
}
