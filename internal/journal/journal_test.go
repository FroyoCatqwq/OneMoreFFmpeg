package journal_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

func TestJournalWriteReadAndPending(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "journal")
	id := journal.NewID()
	record := journal.Record{ID: id, Action: "install", Status: journal.StatusPending, Target: "target"}
	require.NoError(t, journal.Write(dir, record))

	loaded, err := journal.Read(dir, id)
	require.NoError(t, err)
	require.Equal(t, journal.Schema, loaded.Schema)
	require.Equal(t, journal.StatusPending, loaded.Status)
	require.NotEmpty(t, loaded.CreatedAt)

	pending, err := journal.Pending(dir)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	loaded.Status = journal.StatusCommitted
	require.NoError(t, journal.Write(dir, loaded))
	pending, err = journal.Pending(dir)
	require.NoError(t, err)
	require.Empty(t, pending)
}

func TestJournalListMissingDirectory(t *testing.T) {
	records, err := journal.List(filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, err)
	require.Empty(t, records)
}
