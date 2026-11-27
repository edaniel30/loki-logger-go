package loki

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// defaultRedactKeys are substrings that mark a field key as sensitive. A key
// is redacted if its lowercased form contains any of these terms, so e.g.
// "phone" alone also catches "display_phone_number" and "phone_number_id"
// without listing every variant. Matching applies at any depth inside nested
// maps and slices (e.g. an HTTP header map or a JSON request body
// unmarshalled into map[string]any).
//
// Add terms with WithRedactKeys; exempt exact keys that contain a term but
// are not sensitive (e.g. "app_name" contains "name") with WithKeepKeys.
var defaultRedactKeys = []string{
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
	"user_id",
}

// maskRunLength is the fixed number of mask characters used regardless of
// the original string length, so redacted output doesn't leak length.
const maskRunLength = 3

const maskChar = "*"

// redactor masks the values of sensitive keys in log fields.
type redactor struct {
	// terms mark a key as sensitive when the lowercased key contains one
	terms []string
	// keep are lowercased keys that are never redacted, even when they
	// contain a term: they match the whole key, not a part of it
	keep map[string]bool
}

// newRedactor builds a redactor from defaultRedactKeys plus extraTerms, with
// keepKeys exempted. Terms and keys are case-insensitive; blank ones are
// ignored.
func newRedactor(extraTerms, keepKeys []string) *redactor {
	r := &redactor{keep: make(map[string]bool, len(keepKeys))}
	seen := make(map[string]bool)
	for _, term := range append(append([]string{}, defaultRedactKeys...), extraTerms...) {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		r.terms = append(r.terms, term)
	}
	for _, key := range keepKeys {
		if key = strings.ToLower(strings.TrimSpace(key)); key != "" {
			r.keep[key] = true
		}
	}
	return r
}

// fields returns a copy of fields with the values of sensitive keys masked,
// keeping only the first and last character of each string value. Nested
// maps and slices (of any concrete type, e.g. map[string][]string from
// http.Header) are walked recursively.
func (r *redactor) fields(fields map[string]any) map[string]any {
	if fields == nil {
		return nil
	}
	redacted, ok := r.value(fields).(map[string]any)
	if !ok {
		return fields
	}
	return redacted
}

func (r *redactor) value(v any) any {
	rv := reflect.ValueOf(v)
	// Only Map/Slice/Array/Ptr/Interface/String need special handling; the
	// default covers every other kind
	//exhaustive:ignore
	switch rv.Kind() {
	case reflect.Map:
		return r.mapValue(rv)
	case reflect.Slice, reflect.Array:
		return r.slice(rv)
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return v
		}
		return r.value(rv.Elem().Interface())
	case reflect.String:
		// A named string type (type Protocol string) is a String kind but
		// not a string: v.(string) would panic. Those are enums, never a
		// JSON document, and keep their value
		if s, ok := v.(string); ok {
			return r.jsonString(s)
		}
		return v
	default:
		return v
	}
}

// jsonString handles fields that carry an entire JSON document encoded as a
// string (e.g. a raw webhook payload logged as raw_payload: "{...}").
// Without this, sensitive keys inside that string are invisible to the
// key-based walk above, since the whole thing looks like one opaque string
// value. Non-JSON strings are returned unchanged.
func (r *redactor) jsonString(s string) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return s
	}

	var decoded any
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		return s
	}

	redactedJSON, err := json.Marshal(r.value(decoded))
	if err != nil {
		return s
	}

	return string(redactedJSON)
}

func (r *redactor) mapValue(rv reflect.Value) any {
	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		key := fmt.Sprintf("%v", iter.Key().Interface())
		value := iter.Value().Interface()
		if r.isSensitive(key) {
			out[key] = r.leaf(value)
			continue
		}
		out[key] = r.value(value)
	}
	return out
}

func (r *redactor) slice(rv reflect.Value) any {
	out := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		out[i] = r.value(rv.Index(i).Interface())
	}
	return out
}

// leaf masks string values directly. A slice/array under a sensitive key
// (e.g. http.Header's []string values) has each string element masked. Any
// other structured value (e.g. a nested map) is walked normally instead of
// being blanket-masked, so sensitive keys further down are still caught.
func (r *redactor) leaf(v any) any {
	rv := reflect.ValueOf(v)
	// Any string kind, named string types included (type Token string)
	if rv.Kind() == reflect.String {
		return maskString(rv.String())
	}

	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = r.leaf(rv.Index(i).Interface())
		}
		return out
	}

	return r.value(v)
}

// isSensitive reports whether a key's value must be masked: it contains a
// term and is not one of the kept keys. A kept key is still walked, so
// sensitive keys nested under it are masked.
func (r *redactor) isSensitive(key string) bool {
	lower := strings.ToLower(key)
	if r.keep[lower] {
		return false
	}
	for _, term := range r.terms {
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
