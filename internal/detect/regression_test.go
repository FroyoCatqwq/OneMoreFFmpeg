package detect_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appdetect "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/detect"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func TestValidateDirFindsBinSubdirectory(t *testing.T) {
	root := t.TempDir()
	fake := testutil.NewFakeFFmpegAt(t, filepath.Join(root, "bin"))
	detector := newDetector(t, fake)

	report, err := detector.ValidateDir(context.Background(), root, isolatedOptions())
	require.NoError(t, err)
	require.Equal(t, "healthy", report.Status)

	var ffmpegCheck string
	for _, check := range report.Checks {
		if check.Name == "ffmpeg_exists" {
			ffmpegCheck = check.Detail
		}
	}
	require.Equal(t, fake.FFmpegPath, ffmpegCheck)
}

func TestValidateDirWithoutBinariesFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("nothing"), 0o644))
	detector := appdetect.New()

	_, err := detector.ValidateDir(context.Background(), root, isolatedOptions())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E1002, typed.Code)
}

func TestDetectDropsCandidatesWithoutBinaries(t *testing.T) {
	empty := t.TempDir()
	t.Setenv("ONE_MORE_FFMPEG_DIR", empty)
	t.Setenv("FFMPEG_DIR", "")
	t.Setenv("LAE_FFMPEG_DIR", "")
	opts := isolatedOptions()
	opts.NoEnv = false
	opts.NoValidate = true

	result, err := appdetect.New().Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Empty(t, result.Installations)
	require.Empty(t, result.Reports)
}

func TestDetectValidationIsNotCanceledByScanBudget(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := newDetector(t, fake)
	opts := isolatedOptions()
	opts.Dir = fake.Dir
	// The discovery budget is already exhausted before probing starts; the
	// probe must still run with its own timeout and produce a healthy report.
	opts.Timeout = time.Nanosecond
	opts.ProbeTimeout = 5 * time.Second

	result, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, result.Installations, 1)
	require.Equal(t, "healthy", result.Installations[0].Status)
	require.Equal(t, "n9.0.2", result.Installations[0].Version)
}

func TestScanMaxFilesIsGlobalAcrossRoots(t *testing.T) {
	first := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(first, "filler.txt"), []byte("x"), 0o644))
	second := t.TempDir()
	testutil.NewFakeFFmpegAt(t, second)

	opts := isolatedOptions()
	opts.Scan = true
	opts.Roots = []string{first, second}
	opts.MaxFiles = 1
	opts.MaxDepth = 4

	result, err := appdetect.New().Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Empty(t, result.Installations)

	foundLimit := false
	for _, item := range result.Errors {
		if item.Code == "OMF-E1008" {
			foundLimit = true
		}
	}
	require.True(t, foundLimit)
}

func TestDetectInstallationIDStableWithoutValidation(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	detector := newDetector(t, fake)
	opts := isolatedOptions()
	opts.Dir = fake.Dir
	opts.NoValidate = true

	first, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	second, err := detector.Detect(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, first.Installations, 1)
	require.Len(t, second.Installations, 1)
	require.Equal(t, first.Installations[0].ID, second.Installations[0].ID)
	require.Equal(t, first.Installations[0].Fingerprint, second.Installations[0].Fingerprint)
}
