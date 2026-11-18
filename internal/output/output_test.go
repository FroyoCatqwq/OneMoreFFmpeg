package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/output"
)

func TestEncodeJSONMatchesV1Shape(t *testing.T) {
	// The stdout contract uses structs (model.Result/ErrorObject), not maps.
	// json/v2 preserves struct field order and does not HTML-escape, matching
	// the v1 Encoder with SetEscapeHTML(false).
	type payload struct {
		Schema string `json:"schema"`
		OK     bool   `json:"ok"`
		Text   string `json:"text"`
	}
	value := payload{Schema: "onemoreffmpeg.result/v1", OK: true, Text: "<tag>&\"quote\""}
	var v1 bytes.Buffer
	enc := json.NewEncoder(&v1)
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(value))

	got, err := output.EncodeJSON(value)
	require.NoError(t, err)
	require.Equal(t, v1.Bytes(), got, "json/v2 output must be byte-identical to the v1 stdout contract")
}

func TestWriteJSONAndPretty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, output.WriteJSON(&buf, map[string]any{"ok": true}))
	require.Equal(t, "{\"ok\":true}\n", buf.String())
	require.Contains(t, output.PrettyJSON([]byte(`{"a":1}`)), "\n")
}

func TestProgressNoPanic(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewProgress(&buf, "download", 100)
	p.Update(50)
	p.Update(100)
	p.Complete()
	p.Complete() // idempotent
	p.Close()
}
