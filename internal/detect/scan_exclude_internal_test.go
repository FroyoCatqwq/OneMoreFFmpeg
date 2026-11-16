package detect

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsExcludedDirAppliesToExplicitRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "builds", "ffmpeg")
	require.True(t, isExcludedDir(root, filepath.Join(root, ".git")))
	require.True(t, isExcludedDir(root, filepath.Join(root, "node_modules")))
	require.True(t, isExcludedDir(root, filepath.Join(root, "vendor", "$Recycle.Bin")))
	require.False(t, isExcludedDir(root, filepath.Join(root, "bin")))
}

func TestIsExcludedDirSkipsPrefixContainingExplicitRoot(t *testing.T) {
	if filepath.Separator != '/' {
		t.Skip("POSIX prefix rules")
	}
	root := filepath.Join("/tmp", "my-builds")
	require.False(t, isExcludedDir(root, filepath.Join(root, "ffmpeg", "bin")))
}

func TestIsExcludedDirRejectsSystemPrefixesOnFullScan(t *testing.T) {
	if filepath.Separator != '/' {
		t.Skip("POSIX prefix rules")
	}
	root := string(filepath.Separator)
	require.True(t, isExcludedDir(root, "/proc/1"))
	require.True(t, isExcludedDir(root, "/var/lib/docker/overlay2"))
}
