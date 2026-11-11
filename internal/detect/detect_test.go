package detect_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func newDetector(t *testing.T, fake *testutil.FakeFFmpeg) *appdetect.Detector {
	t.Helper()
	detector := appdetect.New()
	detector.Runner.Env = fake.VersionEnv("n9.0.2", "n9.0.2", "n9.0.2")
	return detector
}

func isolatedOptions() appdetect.Options {
	opts := appdetect.DefaultOptions()
	opts.NoPath = true
	opts.NoEnv = true
	opts.NoCommon = true
	return opts
}

func TestDetectExplicitDirHealthy(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := newDetector(t, fake)
	opts := isolatedOptions()
	opts.Dir = fake.Dir

	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	installation := result.Installations[0]
	require.Equal(t, "healthy", installation.Status)
	require.Equal(t, "n9.0.2", installation.Version)
	require.Contains(t, installation.Sources, "explicit_dir")
	require.NotEmpty(t, installation.Fingerprint)
	report, ok := result.Reports[installation.ID]
	require.True(t, ok)
	require.Equal(t, "healthy", report.Status)
}

func TestDetectEnvVar(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	t.Setenv("ONE_MORE_FFMPEG_DIR", fake.Dir)
	t.Setenv("FFMPEG_DIR", "")
	t.Setenv("LAE_FFMPEG_DIR", "")
	opts := isolatedOptions()
	opts.NoEnv = false
	opts.NoValidate = true

	detector := appdetect.New()
	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	require.Contains(t, result.Installations[0].Sources, "env")
	require.Equal(t, "unknown", result.Installations[0].Status)
}

func TestDetectPATH(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	t.Setenv("PATH", fake.Dir)
	opts := isolatedOptions()
	opts.NoPath = false
	opts.NoValidate = true

	detector := appdetect.New()
	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	require.Contains(t, result.Installations[0].Sources, "PATH")
}

func TestDetectScan(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	opts := isolatedOptions()
	opts.Scan = true
	opts.Roots = []string{fake.Dir}
	opts.MaxDepth = 2
	opts.MaxFiles = 1000
	opts.NoValidate = true

	detector := appdetect.New()
	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	require.Contains(t, result.Installations[0].Sources, "scan")
}

func TestScanMaxFiles(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		path := filepath.Join(root, fmt.Sprintf("file-%02d.txt", i))
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	}
	opts := isolatedOptions()
	opts.Scan = true
	opts.Roots = []string{root}
	opts.MaxDepth = 2
	opts.MaxFiles = 5
	opts.NoValidate = true

	detector := appdetect.New()
	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.NotEmpty(t, result.Errors)
	require.Equal(t, "OMF-E1008", result.Errors[0].Code)
}

func TestValidateDir(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := newDetector(t, fake)
	report, err := detector.ValidateDir(context.Background(), fake.Dir, isolatedOptions())
	require.NoError(t, err)
	require.Equal(t, "onemoreffmpeg.validate/v1", report.Schema)
	require.Equal(t, "healthy", report.Status)
	require.NotEmpty(t, report.Checks)
}

func TestValidateDirMissingFFplay(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	require.NoError(t, os.Remove(fake.FFplayPath))
	detector := newDetector(t, fake)
	report, err := detector.ValidateDir(context.Background(), fake.Dir, isolatedOptions())
	require.NoError(t, err)
	require.Equal(t, "degraded", report.Status)
	require.NotEmpty(t, report.Warnings)
	require.Equal(t, "OMF-W1001", report.Warnings[0].Code)
}

func TestValidateVersionMismatch(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := appdetect.New()
	detector.Runner.Env = fake.VersionEnv("n9.0.2", "n9.0.3", "n9.0.2")
	report, err := detector.ValidateDir(context.Background(), fake.Dir, isolatedOptions())
	require.NoError(t, err)
	require.Equal(t, "unusable", report.Status)
}

func TestValidateID(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	t.Setenv("ONE_MORE_FFMPEG_DIR", fake.Dir)
	t.Setenv("FFMPEG_DIR", "")
	t.Setenv("LAE_FFMPEG_DIR", "")
	opts := isolatedOptions()
	opts.NoEnv = false

	detector := newDetector(t, fake)
	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	id := result.Installations[0].ID
	report, err := detector.ValidateID(context.Background(), id, opts)
	require.NoError(t, err)
	require.Equal(t, id, report.InstallationID)
}

func TestValidateTimeout(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := appdetect.New()
	detector.Runner.Env = append(fake.VersionEnv("n9.0.2", "n9.0.2", "n9.0.2"), "OMF_FAKE_SLEEP_MS=500")
	opts := isolatedOptions()
	opts.ProbeTimeout = 50 * time.Millisecond
	report, err := detector.ValidateDir(context.Background(), fake.Dir, opts)
	require.NoError(t, err)
	require.Equal(t, "unusable", report.Status)
}

func TestValidatePermissionErrorUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable bit is not enforced the same way on Windows")
	}
	fake := testutil.NewFakeFFmpeg(t)
	require.NoError(t, os.Chmod(fake.FFmpegPath, 0o000))
	detector := newDetector(t, fake)
	report, err := detector.ValidateDir(context.Background(), fake.Dir, isolatedOptions())
	require.NoError(t, err)
	require.Equal(t, "unusable", report.Status)
}
