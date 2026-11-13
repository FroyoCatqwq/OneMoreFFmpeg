package cli

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
)

func TestNormalizeJSONInput(t *testing.T) {
	text := `{"schema":"onemoreffmpeg.release/v1"}`
	require.Equal(t, []byte(text), normalizeJSONInput([]byte(text)))
	require.Equal(t, []byte(text), normalizeJSONInput(append([]byte{0xEF, 0xBB, 0xBF}, []byte(text)...)))

	units := utf16.Encode([]rune(text))
	raw := []byte{0xFF, 0xFE}
	for _, unit := range units {
		buffer := make([]byte, 2)
		binary.LittleEndian.PutUint16(buffer, unit)
		raw = append(raw, buffer...)
	}
	require.Equal(t, []byte(text), normalizeJSONInput(raw))
}
