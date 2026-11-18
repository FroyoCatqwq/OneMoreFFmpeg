// Package output is the single presentation boundary for OneMoreFFmpeg.
//
// No CLI command imports a presentation third-party library directly: JSON
// encoding (encoding/json/v2), pretty-printing (tidwall/pretty), tables and
// progress bars (go-pretty) are all hidden behind this package. Command code
// only calls output.JSON / output.Table / output.NewProgress.
package output

import (
	"encoding/json/v2"
	"io"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/tidwall/pretty"
)

// EncodeJSON serializes v as a single compact JSON value terminated by a
// newline, using encoding/json/v2. HTML escaping is off to preserve the v1
// stdout contract (values with <, >, & are emitted literally).
func EncodeJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(raw)+1)
	out = append(out, raw...)
	out = append(out, '\n')
	return out, nil
}

// WriteJSON encodes v and writes it to w in one shot so a partial object can
// never be followed by an error envelope.
func WriteJSON(w io.Writer, v any) error {
	payload, err := EncodeJSON(v)
	if err != nil {
		return err
	}
	_, err = w.Write(payload)
	return err
}

// PrettyJSON returns an indented, human-readable form of raw JSON. Used for
// --in-detail-usage examples and debug output; never for the stdout contract.
func PrettyJSON(raw []byte) string {
	return string(pretty.Pretty(raw))
}

// Table renders rows as an aligned plain-text table.
func Table(headers []string, rows [][]string) string {
	t := table.NewWriter()
	head := make(table.Row, 0, len(headers))
	for _, h := range headers {
		head = append(head, h)
	}
	t.AppendHeader(head)
	for _, row := range rows {
		r := make(table.Row, 0, len(row))
		for _, cell := range row {
			r = append(r, cell)
		}
		t.AppendRow(r)
	}
	t.SetStyle(table.StyleLight)
	return t.Render()
}

// Bytes formats a byte count for human display (delegates to go-pretty).
func Bytes(n int64) string {
	return progress.FormatBytes(n)
}
