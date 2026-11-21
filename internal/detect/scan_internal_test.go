package detect

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestScanExplicitRootUnderTmpIsNotExcluded(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "ffmpeg-build", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "ffmpeg"), []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "ffprobe"), []byte("#!/bin/sh\n"), 0o755))

	found := []string{}
	files := 0
	opts := Options{MaxDepth: 8, MaxFiles: 1000, Timeout: time.Second, ProbeTimeout: time.Second}
	err := scanOne(context.Background(), root, opts,
		func(dir, source string) { found = append(found, dir) },
		func(model.DetectError) {}, &files)
	require.NoError(t, err)
	// Paths may be reported in canonical (e.g. Windows 8.3 short-name)
	// form; compare by file identity, not by literal string.
	wantInfo, err := os.Stat(binDir)
	require.NoError(t, err)
	matched := false
	for _, dir := range found {
		gotInfo, statErr := os.Stat(dir)
		if statErr == nil && os.SameFile(wantInfo, gotInfo) {
			matched = true
			break
		}
	}
	require.True(t, matched, "expected %q in discovered roots, got %v", binDir, found)
}
