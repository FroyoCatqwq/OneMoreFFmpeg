package capabilities_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appcaps "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/capabilities"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func testInstallation(fake *testutil.FakeFFmpeg) model.Installation {
	hash := strings.Repeat("a", 64)
	return model.Installation{
		ID:                  "inst-test",
		RootDir:             fake.Dir,
		FFmpeg:              fake.FFmpegPath,
		FFprobe:             fake.FFprobePath,
		FFplay:              fake.FFplayPath,
		Sources:             []string{"explicit_dir"},
		Status:              "healthy",
		Version:             "n9.0.2",
		Fingerprint:         strings.Repeat("b", 64),
		VersionStdoutSHA256: hash,
		LicenseStdoutSHA256: hash,
		Warnings:            []model.Warning{},
		BinarySHA:           map[string]string{"ffmpeg": hash, "ffprobe": hash, "ffplay": hash},
	}
}

func newCollector(fake *testutil.FakeFFmpeg) *appcaps.Collector {
	collector := appcaps.New()
	collector.Runner.Env = fake.CapabilityEnv()
	return collector
}

func TestCollectAllCommands(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.Equal(t, model.CapabilitiesSchema, set.Schema)
	require.Len(t, set.Commands, 13)
	for _, key := range appcaps.CommandKeys("full") {
		require.Contains(t, set.Commands, key)
		require.True(t, set.Commands[key].OK, key)
	}

	require.Equal(t, "n9.0.2", set.Version.Version)
	require.Contains(t, set.License.Summary, "GPL")
	require.NotEmpty(t, set.Muxers)
	require.NotEmpty(t, set.Demuxers)
	require.NotEmpty(t, set.Devices)
	require.NotEmpty(t, set.Decoders)
	require.NotEmpty(t, set.Encoders)
	require.NotEmpty(t, set.Filters)
	require.NotEmpty(t, set.PixelFormats)
	require.NotEmpty(t, set.ChannelLayouts)
	require.NotEmpty(t, set.SampleFormats)
	require.Contains(t, set.Help, "full")
	require.Empty(t, set.Warnings)

	foundH264 := false
	for _, decoder := range set.Decoders {
		if decoder.Name == "h264" {
			foundH264 = true
		}
	}
	require.True(t, foundH264)
}

func TestCollectFFplayMissingWarning(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	installation := testInstallation(fake)
	installation.FFplay = ""
	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: installation,
		HelpTopic:    "full",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.NotEmpty(t, set.Warnings)
	require.Equal(t, "OMF-W1001", set.Warnings[0].Code)
}

func TestInvalidHelpTopic(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	_, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "decoder=../evil",
		NoCache:      true,
	})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E2005, typed.Code)
}

func TestCacheHitRefreshAndCorruption(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	installation := testInstallation(fake)
	opts := appcaps.Options{
		Installation: installation,
		HelpTopic:    "full",
		CacheDir:     cacheDir,
	}

	first, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.False(t, first.CacheHit)
	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_EXIT=3")
	second, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, second.CacheHit)
	require.NotEmpty(t, second.Decoders)

	refreshOpts := opts
	refreshOpts.Refresh = true
	third, err := collector.Collect(context.Background(), refreshOpts)
	require.NoError(t, err)
	require.False(t, third.CacheHit)
	require.False(t, third.Commands["-decoders"].OK)

	files, err = os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, files[0].Name()), []byte("{corrupt"), 0o644))

	collector.Runner.Env = fake.CapabilityEnv()
	fourth, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.False(t, fourth.CacheHit)
	require.True(t, fourth.Commands["-decoders"].OK)
	foundCacheWarning := false
	for _, warning := range fourth.Warnings {
		if warning.Code == "OMF-W1003" {
			foundCacheWarning = true
		}
	}
	require.True(t, foundCacheWarning)
}

func TestCommandFailureWarningsNotDeduped(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_EXIT=3")
	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.False(t, set.Commands["-decoders"].OK)
	count := 0
	for _, warning := range set.Warnings {
		if warning.Code == "OMF-W1002" {
			count++
		}
	}
	require.GreaterOrEqual(t, count, 13)
}

func TestNoCacheDoesNotWrite(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	_, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		CacheDir:     cacheDir,
		NoCache:      true,
	})
	require.NoError(t, err)
	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Empty(t, files)
}
