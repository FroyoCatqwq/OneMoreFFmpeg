package buildinfo_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
)

func TestNewDefaults(t *testing.T) {
	info := buildinfo.New("", "", "")
	require.Equal(t, "dev", info.Version)
	require.NotEmpty(t, info.Commit)
	require.Equal(t, runtime.Version(), info.GoVersion)
	require.Equal(t, runtime.GOOS, info.OS)
	require.Equal(t, runtime.GOARCH, info.Arch)
}

func TestNewExplicit(t *testing.T) {
	info := buildinfo.New("1.2.3", "abc123", "2026-11-10T00:00:00Z")
	require.Equal(t, "1.2.3", info.Version)
	require.Equal(t, "abc123", info.Commit)
	require.Equal(t, "2026-11-10T00:00:00Z", info.BuildTime)
}

func TestNewNormalizesMalformedBuildTime(t *testing.T) {
	for _, value := range []string{"2026-11-10", "yesterday", "not-a-time"} {
		info := buildinfo.New("1.2.3", "abc123", value)
		require.Equal(t, "unknown", info.BuildTime, value)
	}
}
