package journal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

func TestListRefusesToHideCorruptRecords(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.json"),
		[]byte("{not json"), 0o600))

	_, err := journal.List(dir)
	require.Error(t, err, "a corrupt journal must not be silently skipped")

	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E6005, typed.Code)
	require.Equal(t, "broken.json", typed.Details["file"])
}

func TestListOfCleanDirectoryStillSucceeds(t *testing.T) {
	records, err := journal.List(t.TempDir())
	require.NoError(t, err)
	require.Empty(t, records)
}
