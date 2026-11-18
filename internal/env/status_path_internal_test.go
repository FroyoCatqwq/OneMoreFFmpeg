package env

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathContainsUsesPlatformCaseSensitivity(t *testing.T) {
	dir := "/opt/omf/current/bin"
	if runtime.GOOS == "windows" {
		dir = `C:\omf\current\bin`
	}
	lower := "PATH=" + dir
	wrongCase := "path=" + dir
	if runtime.GOOS == "windows" {
		require.True(t, pathContains([]string{wrongCase}, dir), "Windows env names are case-insensitive")
	} else {
		require.False(t, pathContains([]string{wrongCase}, dir), "lowercase 'path' is not PATH on Unix")
	}
	require.True(t, pathContains([]string{lower}, dir))
}
