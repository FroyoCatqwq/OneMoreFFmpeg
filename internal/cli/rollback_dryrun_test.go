package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

// A dry-run must validate the journal id and record just like --apply does;
// previously it returned ok:true for unknown, traversal and non-rollbackable ids.
func TestJournalRollbackDryRunRejectsUnknownID(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "journal", "rollback", "--journal", "does-not-exist", "--json")
	require.Equal(t, 1, code, out)
	require.Equal(t, "OMF-E6005", object["code"])
}

func TestJournalRollbackDryRunRejectsTraversalID(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "journal", "rollback", "--journal", "../evil", "--json")
	require.Equal(t, 1, code, out)
	require.Equal(t, "OMF-E6005", object["code"])
}

func TestEnvRollbackDryRunRejectsUnknownID(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "env", "rollback", "--journal", "does-not-exist", "--json")
	require.Equal(t, 1, code, out)
	require.Equal(t, "OMF-E6005", object["code"])
}

func TestEnvRollbackDryRunRejectsTraversalID(t *testing.T) {
	phase4Home(t)
	code, object, out := runJSON(t, "env", "rollback", "--journal", "../evil", "--json")
	require.Equal(t, 1, code, out)
	require.Equal(t, "OMF-E6005", object["code"])
}

func TestJournalRollbackDryRunPreviewsRecordAction(t *testing.T) {
	home := phase4Home(t)
	require.NoError(t, journal.Write(filepath.Join(home, "journal"), journal.Record{
		ID: "journal-preview", Action: "install", Status: journal.StatusCommitted, Target: filepath.Join(home, "apps", "x"),
	}))
	code, object, out := runJSON(t, "journal", "rollback", "--journal", "journal-preview", "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, "install", data["action"])
	require.Equal(t, false, data["applied"])
	require.Equal(t, "journal-preview", data["journal_id"])
}

func TestEnvRollbackDryRunPreviewsEnvRecord(t *testing.T) {
	home := phase4Home(t)
	require.NoError(t, journal.Write(filepath.Join(home, "journal"), journal.Record{
		ID: "journal-env", Action: "env-set", Status: journal.StatusCommitted, Target: home,
	}))
	code, object, out := runJSON(t, "env", "rollback", "--journal", "journal-env", "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, true, data["dry_run"])
	require.Equal(t, false, data["applied"])
	require.Equal(t, true, data["rollback_supported"])
}

func TestJournalRollbackRejectsAlreadyRolledBack(t *testing.T) {
	home := phase4Home(t)
	require.NoError(t, journal.Write(filepath.Join(home, "journal"), journal.Record{
		ID: "journal-done", Action: "install", Status: journal.StatusRolledBack,
	}))
	code, object, out := runJSON(t, "journal", "rollback", "--journal", "journal-done", "--json")
	require.Equal(t, 2, code, out)
	require.Equal(t, "OMF-E7001", object["code"])
}

func TestJournalRollbackRejectsUnsupportedAction(t *testing.T) {
	home := phase4Home(t)
	require.NoError(t, journal.Write(filepath.Join(home, "journal"), journal.Record{
		ID: "journal-bogus", Action: "bogus", Status: journal.StatusCommitted,
	}))
	code, object, out := runJSON(t, "journal", "rollback", "--journal", "journal-bogus", "--json")
	require.Equal(t, 2, code, out)
	require.Equal(t, "OMF-E7001", object["code"])
}

func TestEnvRollbackRejectsNonEnvAction(t *testing.T) {
	home := phase4Home(t)
	require.NoError(t, journal.Write(filepath.Join(home, "journal"), journal.Record{
		ID: "journal-install", Action: "install", Status: journal.StatusCommitted,
	}))
	code, object, out := runJSON(t, "env", "rollback", "--journal", "journal-install", "--json")
	require.Equal(t, 2, code, out)
	require.Equal(t, "OMF-E7001", object["code"])
}