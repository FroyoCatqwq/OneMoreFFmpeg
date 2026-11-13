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
	require.Contains(t, found, binDir)
}
