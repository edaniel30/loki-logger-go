package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelJSONSerialization(t *testing.T) {
	t.Run("marshal level in struct", func(t *testing.T) {
		type testStruct struct {
			Level Level `json:"level"`
		}

		data := testStruct{Level: LevelError}
		result, err := json.Marshal(data)

		require.NoError(t, err)
		assert.Equal(t, `{"level":"error"}`, string(result))
	})

	t.Run("unmarshal level in struct", func(t *testing.T) {
		type testStruct struct {
			Level Level `json:"level"`
		}

		input := `{"level":"warn"}`
		var result testStruct
		err := json.Unmarshal([]byte(input), &result)

		require.NoError(t, err)
		assert.Equal(t, LevelWarn, result.Level)
	})

	t.Run("unmarshal invalid level in struct", func(t *testing.T) {
		type testStruct struct {
			Level Level `json:"level"`
		}

		input := `{"level":"invalid"}`
		var result testStruct
		err := json.Unmarshal([]byte(input), &result)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid log level")
	})

	t.Run("round trip all levels", func(t *testing.T) {
		type testStruct struct {
			Level Level `json:"level"`
		}

		levels := []Level{LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}

		for _, originalLevel := range levels {
			data := testStruct{Level: originalLevel}

			// Marshal
			marshaled, err := json.Marshal(data)
			require.NoError(t, err)

			// Unmarshal
			var unmarshaled testStruct
			err = json.Unmarshal(marshaled, &unmarshaled)
			require.NoError(t, err)

			// Verify round trip
			assert.Equal(t, originalLevel, unmarshaled.Level)
		}
	})
}

func TestLevelIsEnabled(t *testing.T) {
	assert.True(t, LevelDebug.IsEnabled(LevelDebug))
	assert.True(t, LevelInfo.IsEnabled(LevelDebug))
	assert.True(t, LevelWarn.IsEnabled(LevelDebug))
	assert.True(t, LevelError.IsEnabled(LevelDebug))
	assert.True(t, LevelFatal.IsEnabled(LevelDebug))
}
