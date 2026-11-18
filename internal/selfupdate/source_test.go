package selfupdate

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetectSourceRPM(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("RPM source detection is Linux-only")
	}
	old := rpmQuery
	t.Cleanup(func() { rpmQuery = old })
	rpmQuery = func(string) (string, error) { return "one-more-ffmpeg-1.1.0-1.x86_64", nil }
	source, info := DetectSource("/usr/bin/one-more-ffmpeg")
	require.Equal(t, InstallSourceRPM, source)
	require.Equal(t, "rpm", info.Source)
	require.Contains(t, info.Advice, "dnf")
}

func TestDetectSourceUnknownWhenRPMFails(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("RPM source detection is Linux-only")
	}
	old := rpmQuery
	t.Cleanup(func() { rpmQuery = old })
	rpmQuery = func(string) (string, error) { return "", errors.New("command not found") }
	source, info := DetectSource("/usr/bin/one-more-ffmpeg")
	require.Equal(t, InstallSourceUnknown, source)
	require.Contains(t, info.Advice, "refusing in-place replacement")
}

func TestDetectSourceManualInHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	exe := filepath.Join(home, ".local", "bin", "one-more-ffmpeg")
	source, info := DetectSource(exe)
	require.Equal(t, InstallSourceManual, source)
	require.Equal(t, "manual", info.Source)
}

func TestGuardApplyBlocksManagedAndUnknown(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("RPM source detection is Linux-only")
	}
	old := rpmQuery
	t.Cleanup(func() { rpmQuery = old })
	rpmQuery = func(string) (string, error) { return "omf", nil }
	err := GuardApply("/usr/bin/one-more-ffmpeg", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "package-managed")

	rpmQuery = func(string) (string, error) { return "", errors.New("not found") }
	err = GuardApply("/usr/bin/one-more-ffmpeg", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "source unknown")

	// Explicit override only relaxes Unknown, not managed packages.
	err = GuardApply("/usr/bin/one-more-ffmpeg", true)
	require.NoError(t, err)

	rpmQuery = func(string) (string, error) { return "omf", nil }
	err = GuardApply("/usr/bin/one-more-ffmpeg", true)
	require.Error(t, err, "override must not bypass package-managed protection")
}

func TestScoopShimsDirectoryIsManaged(t *testing.T) {
	root := t.TempDir()
	t.Setenv("SCOOP", root)
	exe := filepath.Join(root, "shims", "one-more-ffmpeg.exe")
	source, info := DetectSource(exe)
	require.Equal(t, InstallSourceScoop, source, "Scoop shims are managed files: %+v", info)
	require.Contains(t, info.Advice, "scoop update")
}

func TestScoopEnvAloneDoesNotMisclassify(t *testing.T) {
	t.Setenv("SCOOP", t.TempDir())
	exe := filepath.Join(t.TempDir(), "one-more-ffmpeg.exe")
	if runtime.GOOS == "windows" {
		// On Windows a temp dir may still be under the user's home -> manual.
		source, _ := DetectSource(exe)
		require.NotEqual(t, InstallSourceScoop, source)
	}
}
