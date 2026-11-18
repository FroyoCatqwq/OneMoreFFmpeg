package download_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/download"
)

func TestFetchProgressCompleteOnSuccess(t *testing.T) {
	t.Setenv("CI", "true")
	fixture := newFixture(t, []byte("payload-bytes"))
	dir := t.TempDir()
	var progress bytes.Buffer
	instance, err := download.New(download.Options{Dir: dir, MaxBytes: 1 << 20, ProgressWriter: &progress})
	require.NoError(t, err)
	_, err = instance.Fetch(context.Background(), fixture.release())
	require.NoError(t, err)
	require.Contains(t, progress.String(), "completed")
	require.Contains(t, progress.String(), "100%")
}

func TestFetchProgressFailureNeverShowsComplete(t *testing.T) {
	t.Setenv("CI", "true")
	fixture := newFixture(t, []byte("payload-bytes"))
	release := fixture.release()
	release.AssetSHA256 = strings.Repeat("0", 64)
	release.ChecksumsURL = ""
	dir := t.TempDir()
	var progress bytes.Buffer
	instance, err := download.New(download.Options{Dir: dir, MaxBytes: 1 << 20, ProgressWriter: &progress})
	require.NoError(t, err)
	_, err = instance.Fetch(context.Background(), release)
	require.Error(t, err)
	out := progress.String()
	require.Contains(t, out, "failed")
	require.NotContains(t, out, "completed")
	require.NotContains(t, out, "100%")
}

func TestFetchProgressCancelNeverShowsComplete(t *testing.T) {
	t.Setenv("CI", "true")
	fixture := newFixture(t, bytes.Repeat([]byte("x"), 4096))
	dir := t.TempDir()
	var progress bytes.Buffer
	instance, err := download.New(download.Options{Dir: dir, MaxBytes: 1 << 20, ProgressWriter: &progress})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before fetch
	_, err = instance.Fetch(ctx, fixture.release())
	require.Error(t, err)
	out := progress.String()
	require.NotContains(t, out, "completed")
	require.NotContains(t, out, "100%")
}
