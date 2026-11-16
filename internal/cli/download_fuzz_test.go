package cli

import "testing"

// FuzzNormalizeJSONInput covers UTF-8 BOM / UTF-16 boundary handling used when
// reading PowerShell-redirected release JSON. Invalid input must never panic.
func FuzzNormalizeJSONInput(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`{"schema":"x"}`),
		{0xEF, 0xBB, 0xBF, '{', '}'},
		{0xFF, 0xFE, '{', 0x00, '}'},
		{0xFE, 0xFF, 0x00, '{', 0x00, '}'},
		{0x7B, 0x00},
		{},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		_ = normalizeJSONInput(raw)
	})
}
