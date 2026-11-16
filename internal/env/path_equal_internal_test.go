package env

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathEntryEqualPlatformSensitive(t *testing.T) {
	lower := "/home/u/.local/bin"
	upper := "/home/u/.LOCAL/bin"
	if runtime.GOOS == "windows" {
		require.True(t, pathEntryEqual(lower, upper), "Windows PATH is case-insensitive")
	} else {
		require.False(t, pathEntryEqual(lower, upper), "Unix PATH is case-sensitive")
	}
	require.True(t, pathEntryEqual(lower, lower))
}

func TestAppendPathUnixDoesNotSkipCaseVariant(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows PATH is case-insensitive")
	}
	previous := "/home/u/.LOCAL/bin"
	entry := "/home/u/.local/bin"
	got := appendPath(previous, entry)
	require.Contains(t, got, entry, "distinct-case Unix directory must be appended")
}
