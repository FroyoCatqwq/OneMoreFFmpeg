package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestInspectCurrentPlainFileIsMissing(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "current"), []byte("foreign"), 0o644))
	kind, target := InspectCurrent(root)
	require.Equal(t, model.CurrentMissing, kind)
	require.Empty(t, target)
}

func TestRemoveCurrentNeverDeletesForeignFile(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "current")
	require.NoError(t, os.WriteFile(current, []byte("foreign"), 0o644))
	require.Error(t, removeCurrent(root))
	body, err := os.ReadFile(current)
	require.NoError(t, err)
	require.Equal(t, "foreign", string(body), "foreign current must survive rollback")
}

func TestRestoreCurrentEmptyTargetWritesPointerFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, restoreCurrentState(root, "", model.CurrentPointerFile))
	body, err := os.ReadFile(filepath.Join(root, "current.pointer"))
	require.NoError(t, err)
	require.Equal(t, "\n", string(body))
}

func TestRollbackRejectsTamperedJournalPaths(t *testing.T) {
	root := t.TempDir()
	appsRoot := filepath.Join(root, "apps")
	target := filepath.Join(appsRoot, "btbn", "tag", "abc")
	record := journal.Record{
		ID: "journal-tamper", Action: "install", Status: journal.StatusCommitted,
		Target: target,
		Details: map[string]string{
			"quarantine": filepath.Join(t.TempDir(), "elsewhere"),
		},
	}
	_, err := Rollback(root, record)
	require.Error(t, err)
}
