package paths

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func abs(parts ...string) string {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func stubEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func TestResolveWindowsDefault(t *testing.T) {
	localAppData := abs("Users", "tester", "AppData", "Local")
	layout, err := resolve("windows", "", stubEnv(map[string]string{"LOCALAPPDATA": localAppData}))
	require.NoError(t, err)

	root := filepath.Join(localAppData, "OneMoreFFmpeg")
	require.Equal(t, root, layout.DataRoot)
	require.Equal(t, filepath.Join(root, "cache"), layout.CacheDir)
	require.Equal(t, filepath.Join(root, "downloads"), layout.DownloadsDir)
	require.Equal(t, filepath.Join(root, "config.toml"), layout.ConfigFile)
	require.Equal(t, filepath.Join(root, "providers.toml"), layout.ProvidersFile)
}

func TestResolveWindowsRequiresLocalAppData(t *testing.T) {
	_, err := resolve("windows", "", stubEnv(nil))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E9003, typed.Code)
}

func TestResolveLinuxXDGDataHome(t *testing.T) {
	dataHome := abs("xdg", "data")
	layout, err := resolve("linux", abs("home", "tester"), stubEnv(map[string]string{
		"XDG_DATA_HOME": dataHome,
	}))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dataHome, "onemoreffmpeg"), layout.DataRoot)
}

func TestResolveLinuxFallsBackToHome(t *testing.T) {
	home := abs("home", "tester")
	layout, err := resolve("linux", home, stubEnv(map[string]string{
		"XDG_DATA_HOME": "relative/path",
	}))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "share", "onemoreffmpeg"), layout.DataRoot)
}

func TestResolveLinuxRequiresHome(t *testing.T) {
	_, err := resolve("linux", "", stubEnv(nil))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E9003, typed.Code)
}

func TestResolveOverrideWins(t *testing.T) {
	override := abs("srv", "omf")
	layout, err := resolve("windows", "", stubEnv(map[string]string{
		HomeEnv:        override,
		"LOCALAPPDATA": abs("ignored"),
	}))
	require.NoError(t, err)
	require.Equal(t, override, layout.DataRoot)
	require.Equal(t, filepath.Join(override, "cache"), layout.CacheDir)
	require.Equal(t, filepath.Join(override, "downloads"), layout.DownloadsDir)
}

func TestResolveOverrideMustBeAbsolute(t *testing.T) {
	_, err := resolve("linux", abs("home", "tester"), stubEnv(map[string]string{
		HomeEnv: "relative/root",
	}))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7007, typed.Code)
}

func TestResolveUsesProcessEnvironment(t *testing.T) {
	override := abs("tmp", "omf-home")
	t.Setenv(HomeEnv, override)

	layout, err := Resolve()
	require.NoError(t, err)
	require.Equal(t, override, layout.DataRoot)

	cacheDir, err := CacheDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(override, "cache"), cacheDir)

	downloads, err := DownloadsDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(override, "downloads"), downloads)

	configFile, err := ConfigFile()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(override, "config.toml"), configFile)

	providersFile, err := ProvidersFile()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(override, "providers.toml"), providersFile)

	root, err := DataRoot()
	require.NoError(t, err)
	require.Equal(t, override, root)
}
