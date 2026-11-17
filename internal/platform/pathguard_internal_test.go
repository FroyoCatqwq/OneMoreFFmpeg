package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexicallyInsideRejectsEscapes(t *testing.T) {
	require.True(t, LexicallyInside("/data", "/data/apps/a"))
	require.False(t, LexicallyInside("/data", "/data"))
	require.False(t, LexicallyInside("/data", "/data/../etc"))
	require.False(t, LexicallyInside("/data", "/etc"))
	require.True(t, LexicallyInside("/data", "/data/..stable"), "names starting with two dots are fine")
}

func TestResolvedInsideRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	root := t.TempDir()
	outside := t.TempDir()
	apps := filepath.Join(root, "apps")
	require.NoError(t, os.MkdirAll(apps, 0o755))
	require.NoError(t, os.Symlink(outside, filepath.Join(apps, "escape")))

	lexical := filepath.Join(apps, "escape", "victim")
	require.True(t, LexicallyInside(apps, lexical), "lexically it looks inside")
	require.False(t, ResolvedInside(apps, lexical), "symlink resolves outside apps")
	require.False(t, NoSymlinkComponents(apps, lexical))
}

func TestNoSymlinkComponentsAllowsPlainDirs(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "apps", "btbn", "tag", "abc")
	require.True(t, NoSymlinkComponents(filepath.Join(root, "apps"), target))
}
