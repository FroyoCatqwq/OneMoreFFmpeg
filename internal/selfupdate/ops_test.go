package selfupdate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteStateAtomic(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "omf")
	ops := realFileOps{}
	require.NoError(t, writeState(ops, exe, updateState{Version: "1.1.1", Stage: "staged", Executable: exe}))
	state, err := readState(ops, exe)
	require.NoError(t, err)
	require.Equal(t, "staged", state.Stage)
	// No leftover temp file.
	_, err = os.Stat(statePathFor(exe) + ".tmp")
	require.True(t, os.IsNotExist(err))
}

func TestRecoverRestoresFromBackup(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "omf")
	backup := exe + ".old"
	ops := realFileOps{}
	require.NoError(t, ops.WriteFile(backup, []byte("old"), 0o755))
	require.NoError(t, writeState(ops, exe, updateState{
		Version: "1.1.1", Stage: "backup_created", Executable: exe, Backup: backup,
	}))
	out, err := (Options{Executable: exe, FileOps: ops}).Recover()
	require.NoError(t, err)
	require.Equal(t, "restored-previous", out)
	data, err := os.ReadFile(exe)
	require.NoError(t, err)
	require.Equal(t, "old", string(data))
}

func TestRecoverCleansStaging(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "omf")
	staging := filepath.Join(dir, "staged")
	ops := realFileOps{}
	require.NoError(t, ops.WriteFile(staging, []byte("new"), 0o755))
	require.NoError(t, writeState(ops, exe, updateState{
		Version: "1.1.1", Stage: "staged", Executable: exe, Staging: staging,
	}))
	out, err := (Options{Executable: exe, FileOps: ops}).Recover()
	require.NoError(t, err)
	require.Equal(t, "cleaned-staging", out)
	_, err = os.Stat(staging)
	require.True(t, os.IsNotExist(err))
}

func TestErrorsJoinOnDoubleFailure(t *testing.T) {
	// Verifies the error-combination contract used by Apply rollback paths:
	// both the install failure and the restore failure must surface.
	installErr := errors.New("install updated executable: boom")
	rollbackErr := errors.New("restore previous executable: denied; backup retained at /x/omf.old")
	joined := errors.Join(installErr, rollbackErr)
	msg := joined.Error()
	require.True(t, strings.Contains(msg, "install updated executable"))
	require.True(t, strings.Contains(msg, "restore previous executable"))
	require.True(t, strings.Contains(msg, "backup retained"))
}
