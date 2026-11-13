package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestStatusReportsPointerFileAndVariables(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "current.pointer"), []byte(filepath.Join(root, "apps", "target")+"\n"), 0o644))

	status := appenv.Status(appenv.Options{
		DataRoot: root,
		Env: []string{
			"PATH=" + filepath.Join(root, "current", "bin") + string(os.PathListSeparator) + "/usr/bin",
			"ONE_MORE_FFMPEG_DIR=" + filepath.Join(root, "current"),
			"FFMPEG_DIR=",
		},
	})
	require.Equal(t, model.CurrentPointerFile, status.CurrentType)
	require.Equal(t, filepath.Join(root, "apps", "target"), status.CurrentTarget)
	require.True(t, status.PathContainsCurrent)
	require.NotNil(t, status.Variables["ONE_MORE_FFMPEG_DIR"])
	require.Nil(t, status.Variables["FFMPEG_DIR"])
	require.Nil(t, status.Variables["LAE_FFMPEG_DIR"])
	require.Len(t, status.Warnings, 1)
}

func TestStatusMissingCurrent(t *testing.T) {
	status := appenv.Status(appenv.Options{DataRoot: t.TempDir(), Env: []string{"PATH=/usr/bin"}})
	require.Equal(t, model.CurrentMissing, status.CurrentType)
	require.False(t, status.PathContainsCurrent)
	require.Empty(t, status.Warnings)
}
