package download_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/download"
)

func TestFetchRejectsSymlinkTargetDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privileges on Windows")
	}
	fixture := newFixture(t, []byte("payload"))
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	require.NoError(t, os.MkdirAll(outside, 0o755))
	// Pre-plant a symlink at the version directory derived from the pin sha.
	sha := fixture.release().AssetSHA256
	linkPath := filepath.Join(dir, sha[:16])
	require.NoError(t, os.Symlink(outside, linkPath))

	instance, err := download.New(download.Options{Dir: dir, MaxBytes: 1 << 20})
	require.NoError(t, err)
	_, err = instance.Fetch(context.Background(), fixture.release())
	require.Error(t, err)
	require.Contains(t, err.Error(), "symbolic link")
	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, entries, "nothing may be written outside the download cache")
}
