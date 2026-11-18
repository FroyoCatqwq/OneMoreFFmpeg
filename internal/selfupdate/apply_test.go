package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// hookOps fails renames matching a predicate. It is used to inject failures
// into the exact replace/restore transaction on temp files.
type hookOps struct {
	base FileOps
	hook func(from, to string) error
}

func (h *hookOps) Stat(p string) (os.FileInfo, error) { return h.base.Stat(p) }
func (h *hookOps) Remove(p string) error              { return h.base.Remove(p) }
func (h *hookOps) WriteFile(p string, d []byte, m os.FileMode) error {
	return h.base.WriteFile(p, d, m)
}
func (h *hookOps) ReadFile(p string) ([]byte, error) { return h.base.ReadFile(p) }
func (h *hookOps) Rename(from, to string) error {
	if h.hook != nil {
		if err := h.hook(from, to); err != nil {
			return err
		}
	}
	return h.base.Rename(from, to)
}

func applyFixture(t *testing.T, payload []byte, hook func(from, to string) error) (Options, *Update, *hookOps) {
	t.Helper()
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	exe := filepath.Join(dir, "omf")
	require.NoError(t, os.WriteFile(exe, []byte("old"), 0o755))
	ops := &hookOps{base: realFileOps{}, hook: hook}
	u := &Update{
		Version: "9.9.9", AssetName: "one-more-ffmpeg-v9.9.9-linux-amd64",
		AssetURL: server.URL, SHA256: hex.EncodeToString(sum[:]),
	}
	return Options{
		Executable: exe, Dir: dir, FileOps: ops,
		HTTPClient: server.Client(), GOOS: "linux", GOARCH: "amd64",
	}, u, ops
}

func TestApplySuccessReplacesExecutable(t *testing.T) {
	payload := []byte("new-binary")
	o, u, _ := applyFixture(t, payload, nil)
	require.NoError(t, o.Apply(context.Background(), u))
	data, err := os.ReadFile(o.Executable)
	require.NoError(t, err)
	require.Equal(t, payload, data)
}

func TestApplyPreservesExecutableMode(t *testing.T) {
	o, u, _ := applyFixture(t, []byte("new-binary"), nil)
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(o.Executable, 0o750))
	}
	require.NoError(t, o.Apply(context.Background(), u))
	info, err := os.Stat(o.Executable)
	require.NoError(t, err)
	if runtime.GOOS != "windows" {
		// Unix permission bits are preserved; Windows uses ACLs and Go reports
		// a synthetic 0666 regardless of Chmod, so only check Unix here.
		require.Equal(t, os.FileMode(0o750), info.Mode().Perm(), "original mode must be preserved")
	}
}

func TestApplyInstallFailureRestoresBackup(t *testing.T) {
	payload := []byte("new-binary")
	o, u, _ := applyFixture(t, payload, func(from, to string) error {
		// Fail only the staging -> executable install rename (not the backup rename).
		if strings.HasPrefix(filepath.Base(from), "one-more-ffmpeg-v") && filepath.Base(to) == "omf" {
			return os.ErrPermission
		}
		return nil
	})
	err := o.Apply(context.Background(), u)
	require.Error(t, err)
	data, readErr := os.ReadFile(o.Executable)
	require.NoError(t, readErr)
	require.Equal(t, "old", string(data), "previous executable must be restored")
}

func TestApplyDoubleFailureRetainsBackup(t *testing.T) {
	payload := []byte("new-binary")
	o, u, _ := applyFixture(t, payload, func(from, to string) error {
		base := filepath.Base(from)
		if strings.HasPrefix(base, "one-more-ffmpeg-v") && filepath.Base(to) == "omf" {
			return os.ErrPermission // install fails
		}
		if base == "omf.old" && filepath.Base(to) == "omf" {
			return os.ErrPermission // restore also fails
		}
		return nil
	})
	err := o.Apply(context.Background(), u)
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "install updated executable")
	require.Contains(t, msg, "restore previous executable")
	require.Contains(t, msg, ".old", "backup path must be reported")
	_, statErr := os.Stat(o.Executable + ".old")
	require.NoError(t, statErr, "unique recoverable backup must be retained")
}

func TestApplyRejectsExistingBackup(t *testing.T) {
	o, u, _ := applyFixture(t, []byte("new"), nil)
	require.NoError(t, os.WriteFile(o.Executable+".old", []byte("previous"), 0o755))
	err := o.Apply(context.Background(), u)
	require.Error(t, err)
	require.Contains(t, err.Error(), "backup still exists")
}
