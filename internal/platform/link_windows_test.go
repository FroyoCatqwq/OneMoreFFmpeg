//go:build windows

package platform_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

// The sandbox may block junction reparse points; CreateLink must still succeed
// via the directory-symlink fallback (developer mode / sudo / gsudo).
func TestCreateLinkFallsBackToSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "marker.txt"), []byte("ok"), 0o644))

	link := filepath.Join(dir, "current")
	require.NoError(t, platform.CreateLink(link, target))

	resolved, err := filepath.EvalSymlinks(link)
	require.NoError(t, err)
	require.Equal(t, target, resolved)
	body, err := os.ReadFile(filepath.Join(link, "marker.txt"))
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

func TestReplaceLinkReportsActualKind(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, os.Mkdir(target, 0o755))

	link := filepath.Join(dir, "current")
	kind, err := platform.ReplaceLink(link, target)
	require.NoError(t, err)
	require.Contains(t, []string{model.CurrentSymlink, model.CurrentJunction}, kind)

	resolved, err := filepath.EvalSymlinks(link)
	require.NoError(t, err)
	require.Equal(t, target, resolved)

	other := filepath.Join(dir, "other")
	require.NoError(t, os.Mkdir(other, 0o755))
	kind2, err := platform.ReplaceLink(link, other)
	require.NoError(t, err)
	require.Contains(t, []string{model.CurrentSymlink, model.CurrentJunction}, kind2)
	resolved, err = filepath.EvalSymlinks(link)
	require.NoError(t, err)
	require.Equal(t, other, resolved)
}