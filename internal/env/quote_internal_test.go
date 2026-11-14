package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Single-quote escaping must neutralize single quotes, $ and backticks so a
// hostile or unusual install path cannot expand or inject into ~/.profile.
func TestShellQuoteEscapesSingleQuotesAndExpansion(t *testing.T) {
	require.Equal(t, `'plain'`, shellQuote("plain"))
	require.Equal(t, `'a'\''b'`, shellQuote("a'b"))
	require.Equal(t, `'$HOME'`, shellQuote("$HOME"))
	require.Equal(t, `'a"b'`, shellQuote(`a"b`))
	require.Equal(t, "'a`b'", shellQuote("a`b"))
	require.Equal(t, `'$(rm -rf /)'`, shellQuote("$(rm -rf /)"))
}

func TestWriteShimQuotesTargetWithSingleQuote(t *testing.T) {
	dir := t.TempDir()
	shim := filepath.Join(dir, "ffmpeg")
	target := filepath.Join(dir, "it's", "bin", "ffmpeg")
	require.NoError(t, writeShim(shim, target))
	body, err := os.ReadFile(shim)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(body), "#!/bin/sh\n"))
	require.Contains(t, string(body), "exec "+shellQuote(target)+` "$@"`)
}