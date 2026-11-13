package env_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
)

type fakeStore struct {
	values map[string]string
	sets   int
	failOn map[string]bool
}

func (f *fakeStore) Get(name string) (string, bool, error) {
	value, ok := f.values[name]
	return value, ok, nil
}

func (f *fakeStore) Set(name, value string) error {
	if f.failOn[name] {
		return apperrors.New(apperrors.E6002, "injected write failure")
	}
	f.sets++
	f.values[name] = value
	return nil
}

func (f *fakeStore) Remove(name string) error {
	f.sets++
	delete(f.values, name)
	return nil
}

func TestApplyDryRunDoesNotWrite(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("persistent env writes are Windows-only")
	}
	store := &fakeStore{values: map[string]string{"PATH": `C:\Windows`}}
	root := t.TempDir()

	result, err := appenv.Apply(context.Background(), appenv.ApplyOptions{DataRoot: root, Store: store})
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.False(t, result.Applied)
	require.Zero(t, store.sets)
	require.NotEmpty(t, result.Operations)
	require.NoFileExists(t, filepath.Join(root, "journal"))
}

func TestApplyWritesAndJournals(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("persistent env writes are Windows-only")
	}
	store := &fakeStore{values: map[string]string{"PATH": `C:\Windows`}}
	root := t.TempDir()

	result, err := appenv.Apply(context.Background(), appenv.ApplyOptions{DataRoot: root, Store: store, Apply: true})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.NotEmpty(t, result.JournalID)
	require.Contains(t, store.values["PATH"], filepath.Join(root, "current", "bin"))
	require.Equal(t, filepath.Join(root, "current"), store.values["ONE_MORE_FFMPEG_DIR"])
	record, err := journal.Read(filepath.Join(root, "journal"), result.JournalID)
	require.NoError(t, err)
	require.Equal(t, journal.StatusCommitted, record.Status)
	require.NotEmpty(t, record.Details["path_previous"])

	_, err = appenv.RollbackWithStore(context.Background(), root, result.JournalID, store)
	require.NoError(t, err)
	require.Equal(t, `C:\Windows`, store.values["PATH"])
	_, exists := store.values["ONE_MORE_FFMPEG_DIR"]
	require.False(t, exists)
}

func TestApplyFailureWritesFailedJournalAndUndoes(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("persistent env writes are Windows-only")
	}
	store := &fakeStore{
		values: map[string]string{"PATH": `C:\Windows`},
		failOn: map[string]bool{"ONE_MORE_FFMPEG_DIR": true},
	}
	root := t.TempDir()

	_, err := appenv.Apply(context.Background(), appenv.ApplyOptions{DataRoot: root, Store: store, Apply: true})
	require.Error(t, err)
	require.Equal(t, `C:\Windows`, store.values["PATH"])

	records, err := journal.List(filepath.Join(root, "journal"))
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, journal.StatusFailed, records[0].Status)
	require.NotEmpty(t, records[0].Details["path_previous"])
}
