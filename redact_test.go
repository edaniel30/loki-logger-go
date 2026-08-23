package loki

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaskString(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"single char", "a", "*"},
		{"two chars", "ab", "**"},
		{"three chars", "abc", "a***c"},
		{"phone number", "+5215512345678", "+***8"},
		{"unicode", "héllo", "h***o"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, maskString(tt.in))
		})
	}
}

func TestIsSensitiveKey(t *testing.T) {
	assert.True(t, isSensitiveKey("phone"))
	assert.True(t, isSensitiveKey("Phone"))
	assert.True(t, isSensitiveKey("AUTHORIZATION"))
	assert.True(t, isSensitiveKey("X-Hub-Signature-256"))
	assert.False(t, isSensitiveKey("event_id"))
	assert.False(t, isSensitiveKey("status"))
}

func TestRedactFields_Nil(t *testing.T) {
	assert.Nil(t, redactFields(nil))
}

func TestRedactFields_TopLevelString(t *testing.T) {
	out := redactFields(map[string]any{
		"phone":  "+5215512345678",
		"status": "delivered",
	})
	assert.Equal(t, "+***8", out["phone"])
	assert.Equal(t, "delivered", out["status"])
}

func TestRedactFields_NestedMap(t *testing.T) {
	// Mirrors the shape of a WhatsApp webhook body.
	body := map[string]any{
		"messaging_product": "whatsapp",
		"metadata": map[string]any{
			"display_phone_number": "5215512345678",
			"phone_number_id":      "1234567890",
		},
		"contacts": []any{
			map[string]any{
				"profile": map[string]any{"name": "John Doe"},
				"wa_id":   "5215512345678",
			},
		},
		"messages": []any{
			map[string]any{
				"from": "5215512345678",
				"id":   "wamid.ABC123",
				"text": map[string]any{"body": "hello there"},
				"type": "text",
			},
		},
	}

	out := redactFields(map[string]any{"body": body})
	redactedBody := out["body"].(map[string]any)

	assert.Equal(t, "whatsapp", redactedBody["messaging_product"])

	metadata := redactedBody["metadata"].(map[string]any)
	assert.Equal(t, "5***8", metadata["display_phone_number"])
	assert.Equal(t, "1***0", metadata["phone_number_id"]) // "phone" substring catches this too

	contacts := redactedBody["contacts"].([]any)
	contact := contacts[0].(map[string]any)
	assert.Equal(t, "5***8", contact["wa_id"])
	profile := contact["profile"].(map[string]any)
	assert.Equal(t, "J***e", profile["name"])

	messages := redactedBody["messages"].([]any)
	msg := messages[0].(map[string]any)
	assert.Equal(t, "5***8", msg["from"])
	assert.Equal(t, "wamid.ABC123", msg["id"])
	text := msg["text"].(map[string]any)
	assert.Equal(t, "h***e", text["body"])
}

func TestRedactFields_HTTPHeaderShape(t *testing.T) {
	// http.Header is map[string][]string under the hood.
	headers := map[string][]string{
		"Authorization": {"Bearer abc123xyz"},
		"Content-Type":  {"application/json"},
	}

	out := redactFields(map[string]any{"headers": headers})
	redactedHeaders := out["headers"].(map[string]any)

	auth := redactedHeaders["Authorization"].([]any)
	assert.Equal(t, "B***z", auth[0])

	contentType := redactedHeaders["Content-Type"].([]any)
	assert.Equal(t, "application/json", contentType[0])
}

func TestRedactFields_NonStringLeafUnderSensitiveKey(t *testing.T) {
	// "to" is sensitive but here its value is a number; masking only
	// applies to strings, so it should pass through unchanged.
	out := redactFields(map[string]any{"to": 12345})
	assert.Equal(t, 12345, out["to"])
}

func TestRedactFields_DoesNotMutateOriginal(t *testing.T) {
	original := map[string]any{"phone": "+5215512345678"}
	_ = redactFields(original)
	assert.Equal(t, "+5215512345678", original["phone"])
}

func TestRedactFields_JSONEncodedStringValue(t *testing.T) {
	// Mirrors a real production log: the whole webhook body is logged as a
	// single JSON-encoded string under a key that isn't itself sensitive
	// (raw_payload), so the key-based walk alone never sees "name",
	// "text.body", "user_id", etc. living inside that string.
	rawPayload := `{"messaging_product":"whatsapp","metadata":{"display_phone_number":"573212418580","phone_number_id":"1304723552721923"},"contacts":[{"profile":{"name":"Majo","username":"mariahormazab"},"user_id":"CO.1942991563042707"}],"messages":[{"from_user_id":"CO.1942991563042707","id":"wamid.ABC123","timestamp":"1787524845","text":{"body":"secret message"},"type":"text"}]}`

	out := redactFields(map[string]any{
		"event_id":    "54dc8e97",
		"raw_payload": rawPayload,
	})

	assert.Equal(t, "54dc8e97", out["event_id"])

	var redactedPayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(out["raw_payload"].(string)), &redactedPayload))

	metadata := redactedPayload["metadata"].(map[string]any)
	assert.Equal(t, "5***0", metadata["display_phone_number"])
	assert.Equal(t, "1***3", metadata["phone_number_id"])

	contacts := redactedPayload["contacts"].([]any)
	contact := contacts[0].(map[string]any)
	assert.Equal(t, "C***7", contact["user_id"])
	profile := contact["profile"].(map[string]any)
	assert.Equal(t, "M***o", profile["name"])
	assert.Equal(t, "m***b", profile["username"]) // "username" also contains "name"

	messages := redactedPayload["messages"].([]any)
	msg := messages[0].(map[string]any)
	assert.Equal(t, "C***7", msg["from_user_id"])
	assert.Equal(t, "wamid.ABC123", msg["id"]) // not a sensitive key
	text := msg["text"].(map[string]any)
	assert.Equal(t, "s***e", text["body"])
}

func TestRedactJSONString_NonJSONStringUnchanged(t *testing.T) {
	out := redactFields(map[string]any{"status": "delivered"})
	assert.Equal(t, "delivered", out["status"])
}

func TestRedactJSONString_InvalidJSONLookingStringUnchanged(t *testing.T) {
	out := redactFields(map[string]any{"note": "{not valid json"})
	assert.Equal(t, "{not valid json", out["note"])
}
