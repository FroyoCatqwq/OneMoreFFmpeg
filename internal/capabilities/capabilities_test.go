package capabilities_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appcaps "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/capabilities"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
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
	require.True(t, set.Help["full"].Found)
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

	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_FAIL_ARGS=-decoders")
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

func TestSingleCommandFailureKeepsOtherResults(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_FAIL_ARGS=-decoders")
	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.False(t, set.Commands["-decoders"].OK)
	require.True(t, set.Commands["-encoders"].OK)
	require.NotEmpty(t, set.Encoders)
	count := 0
	for _, warning := range set.Warnings {
		if warning.Code == "OMF-W1002" {
			count++
		}
	}
	require.Equal(t, 1, count)
}

func TestAllCommandFailureReturnsErrorAndSkipsCache(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_EXIT=3")
	_, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		CacheDir:     cacheDir,
	})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E2001, typed.Code)

	files, readErr := os.ReadDir(cacheDir)
	require.NoError(t, readErr)
	require.Empty(t, files)
}

func TestHelpTopicNotFoundIsReported(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	collector.Runner.Env = append(fake.CapabilityEnv(),
		"OMF_FAKE_STDOUT_HELP=Codec 'nope' is not recognized by FFmpeg.\n\nExiting with exit code 0\n")
	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "decoder=nope",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.False(t, set.Commands["-h decoder=nope"].OK)
	require.Equal(t, "help topic not found", set.Commands["-h decoder=nope"].Error)
	require.False(t, set.Help["decoder=nope"].Found)
	require.Contains(t, set.Help["decoder=nope"].Raw, "not recognized")
	found := false
	for _, warning := range set.Warnings {
		if warning.Code == "OMF-W1002" {
			found = true
		}
	}
	require.True(t, found)
}

func TestNoCacheDoesNotRequireResolvableDataRoot(t *testing.T) {
	t.Setenv(paths.HomeEnv, "relative-root")
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)

	set, err := collector.Collect(context.Background(), appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		NoCache:      true,
	})
	require.NoError(t, err)
	require.Len(t, set.Commands, 13)
}

func TestCacheKeyIncludesCommandTimeout(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	base := appcaps.Options{
		Installation:   testInstallation(fake),
		HelpTopic:      "full",
		CacheDir:       cacheDir,
		CommandTimeout: 2 * time.Second,
	}

	first, err := collector.Collect(context.Background(), base)
	require.NoError(t, err)
	require.False(t, first.CacheHit)

	again, err := collector.Collect(context.Background(), base)
	require.NoError(t, err)
	require.True(t, again.CacheHit)

	slower := base
	slower.CommandTimeout = 3 * time.Second
	third, err := collector.Collect(context.Background(), slower)
	require.NoError(t, err)
	require.False(t, third.CacheHit, "a different effective timeout must not reuse the cache")

	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestCollectListUsesCache(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	opts := appcaps.Options{
		Installation: testInstallation(fake),
		CacheDir:     cacheDir,
	}
	first, err := collector.CollectList(context.Background(), opts, "encoders")
	require.NoError(t, err)
	require.NotEmpty(t, first.Entries)

	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_FAIL_ARGS=-encoders")
	second, err := collector.CollectList(context.Background(), opts, "encoders")
	require.NoError(t, err, "second call must be served from cache")
	require.Equal(t, first.Fingerprint, second.Fingerprint)
	require.Equal(t, first.Entries, second.Entries)

	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestCommandTimeoutOverrideAppliesToEveryCommand(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	collector.Runner.Env = append(fake.CapabilityEnv(), "OMF_FAKE_SLEEP_MS=300")
	start := time.Now()
	_, err := collector.Collect(context.Background(), appcaps.Options{
		Installation:   testInstallation(fake),
		HelpTopic:      "full",
		NoCache:        true,
		CommandTimeout: 30 * time.Millisecond,
	})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E2001, typed.Code)
	require.Less(t, time.Since(start), 5*time.Second)
}

func TestCacheWithoutHelpEntryIsRejected(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	opts := appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		CacheDir:     cacheDir,
	}

	first, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.False(t, first.CacheHit)

	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	path := filepath.Join(cacheDir, files[0].Name())

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var entry map[string]any
	require.NoError(t, json.Unmarshal(raw, &entry))
	delete(entry, "help")
	broken, err := json.Marshal(entry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, broken, 0o644))

	second, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.False(t, second.CacheHit)
	require.Contains(t, second.Help, "full")

	third, err := collector.Collect(context.Background(), opts)
	require.NoError(t, err)
	require.True(t, third.CacheHit)
}

func TestCanceledContextDoesNotWriteCache(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	collector := newCollector(fake)
	cacheDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := collector.Collect(ctx, appcaps.Options{
		Installation: testInstallation(fake),
		HelpTopic:    "full",
		CacheDir:     cacheDir,
	})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E9004, typed.Code)

	files, readErr := os.ReadDir(cacheDir)
	require.NoError(t, readErr)
	require.Empty(t, files)
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
