package install

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

func TestShortSANeverContainsSeparators(t *testing.T) {
	for _, evil := range []string{
		"../../../../etc/passwdXX",
		"abcdef0123456789/../../x",
		`abcd\\ef01234567`,
		"................x",
	} {
		got := shortSHA(evil)
		require.NotContains(t, got, "/")
		require.NotContains(t, got, `\`)
		require.NotEqual(t, "..", got)
	}
	require.Equal(t, "0123456789abcdef", shortSHA("0123456789abcdef0123"))
}

func TestRollbackRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	dataRoot := t.TempDir()
	outside := t.TempDir()
	apps := filepath.Join(dataRoot, "apps")
	require.NoError(t, os.MkdirAll(apps, 0o755))
	// A symlink planted under apps/ makes the journal target lexically inside
	// but physically outside. Rollback must refuse it.
	require.NoError(t, os.Symlink(outside, filepath.Join(apps, "escape")))
	target := filepath.Join(apps, "escape", "victim")

	record := journal.Record{
		ID: "symlink-escape", Action: "install", Status: journal.StatusCommitted,
		Target: target,
	}
	_, err := Rollback(dataRoot, record)
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside apps")
}
