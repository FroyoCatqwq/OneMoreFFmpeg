//go:build !windows

package env_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
)

// User-scope shims live in ~/.local/bin (system scope in /usr/local/bin), both
// OUTSIDE the data root. The rollback path guard must accept those legitimate
// shim directories; rejecting them broke every Linux env rollback with shims.
func TestUserScopeShimRollbackIsAllowed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()

	res, err := appenv.Apply(context.Background(), appenv.ApplyOptions{
		DataRoot: root, Apply: true, NoVariables: true, NoShellRC: true,
	})
	require.NoError(t, err)
	require.True(t, res.Applied)
	shim := filepath.Join(home, ".local", "bin", "ffmpeg")
	_, err = os.Stat(shim)
	require.NoError(t, err, "shim must be created by apply")

	back, err := appenv.Rollback(context.Background(), root, res.JournalID)
	require.NoError(t, err, "rollback must accept the ~/.local/bin shim directory")
	require.True(t, back.Applied)
	_, err = os.Stat(shim)
	require.True(t, os.IsNotExist(err), "rollback must remove the created shim")
}