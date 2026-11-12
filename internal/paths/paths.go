// Package paths resolves the OneMoreFFmpeg data root and the directories that
// live under it. Every command must use this package instead of assembling
// platform paths locally.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// HomeEnv overrides the data root. It must be an absolute path.
const HomeEnv = "ONE_MORE_FFMPEG_HOME"

// Layout is the resolved on-disk layout. All fields are absolute.
type Layout struct {
	DataRoot     string
	CacheDir     string
	DownloadsDir string
	ConfigFile   string
}

// Resolve returns the layout for the current process.
func Resolve() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return resolve(runtime.GOOS, home, os.Getenv)
}

// DataRoot returns the root of the managed directory tree.
func DataRoot() (string, error) {
	layout, err := Resolve()
	if err != nil {
		return "", err
	}
	return layout.DataRoot, nil
}

// CacheDir returns the directory for disposable caches.
func CacheDir() (string, error) {
	layout, err := Resolve()
	if err != nil {
		return "", err
	}
	return layout.CacheDir, nil
}

// DownloadsDir returns the directory for downloaded artifacts.
func DownloadsDir() (string, error) {
	layout, err := Resolve()
	if err != nil {
		return "", err
	}
	return layout.DownloadsDir, nil
}

// ConfigFile returns the default config.toml path.
func ConfigFile() (string, error) {
	layout, err := Resolve()
	if err != nil {
		return "", err
	}
	return layout.ConfigFile, nil
}

func resolve(goos, home string, getenv func(string) string) (Layout, error) {
	if override := strings.TrimSpace(getenv(HomeEnv)); override != "" {
		if !filepath.IsAbs(override) {
			return Layout{}, apperrors.New(apperrors.E7007, "data root override must be an absolute path").WithDetails(map[string]any{"env": HomeEnv, "value": override})
		}
		return layoutFor(filepath.Clean(override)), nil
	}
	switch goos {
	case "windows":
		base := strings.TrimSpace(getenv("LOCALAPPDATA"))
		if !filepath.IsAbs(base) {
			return Layout{}, apperrors.New(apperrors.E9003, "cannot determine LOCALAPPDATA").WithDetails(map[string]any{"env": "LOCALAPPDATA"})
		}
		return layoutFor(filepath.Join(base, "OneMoreFFmpeg")), nil
	default:
		if dataHome := strings.TrimSpace(getenv("XDG_DATA_HOME")); filepath.IsAbs(dataHome) {
			return layoutFor(filepath.Join(dataHome, "onemoreffmpeg")), nil
		}
		if !filepath.IsAbs(home) {
			return Layout{}, apperrors.New(apperrors.E9003, "cannot determine home directory")
		}
		return layoutFor(filepath.Join(home, ".local", "share", "onemoreffmpeg")), nil
	}
}

func layoutFor(root string) Layout {
	return Layout{
		DataRoot:     root,
		CacheDir:     filepath.Join(root, "cache"),
		DownloadsDir: filepath.Join(root, "downloads"),
		ConfigFile:   filepath.Join(root, "config.toml"),
	}
}
