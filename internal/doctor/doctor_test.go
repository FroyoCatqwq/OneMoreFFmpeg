package doctor_test

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/doctor"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func TestDoctorReportsUnhealthyWithoutCurrent(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, journal.Write(filepath.Join(root, "journal"), journal.Record{
		ID: "journal-pending", Action: "install", Status: journal.StatusPending,
	}))

	report, err := doctor.Run(context.Background(), doctor.Options{DataRoot: root, Env: []string{"PATH=/usr/bin"}})
	require.NoError(t, err)
	require.False(t, report.Healthy)
	require.Len(t, report.Checks, 5)
	require.Equal(t, []string{"journal-pending"}, report.JournalPending)

	byName := map[string]bool{}
	for _, check := range report.Checks {
		byName[check.Name] = check.OK
	}
	require.False(t, byName["current_exists"])
	require.False(t, byName["trio_valid"])
	require.False(t, byName["no_pending_journal"])
}

// installFakeTrio installs the fake ffmpeg/ffprobe/ffplay and switches current.
func installFakeTrio(t *testing.T, root string, fake *testutil.FakeFFmpeg) string {
	t.Helper()
	t.Setenv("OMF_FAKE_VERSION_FFMPEG", "9.9.9")
	t.Setenv("OMF_FAKE_VERSION_FFPROBE", "9.9.9")
	t.Setenv("OMF_FAKE_VERSION_FFPLAY", "9.9.9")
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "ffmpeg.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for _, source := range []string{fake.FFmpegPath, fake.FFprobePath, fake.FFplayPath} {
		body, err := os.ReadFile(source)
		require.NoError(t, err)
		entry, err := writer.Create("ffmpeg-build/bin/" + filepath.Base(source))
		require.NoError(t, err)
		_, err = entry.Write(body)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	payload, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	sum := sha256.Sum256(payload)
	release := model.Release{
		Schema: model.ReleaseSchema, Provider: "btbn", Repository: "FFmpeg-Builds/FFmpeg-Builds",
		Channel: "daily", ReleaseTag: "autobuild-2026-11-12-12-00", ReleaseID: "1",
		PublishedAt: "2026-11-12T12:00:00Z", AssetName: "ffmpeg.zip",
		AssetURL: "https://example.invalid/ffmpeg.zip", AssetSHA256: hex.EncodeToString(sum[:]),
		Variant: "gpl", Linkage: "static", Target: "win64", LicenseHint: "gpl", Redistributable: true,
	}
	data, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: release, ArchivePath: archivePath, Apply: true,
	})
	require.NoError(t, err)
	return data.RootDir
}

// ensureCurrentLink guarantees current is a real link so the trio probe runs.
// Some hosts cannot create junctions and install falls back to a pointer file;
// recreate a link there, and skip only if the host supports no link at all.
func ensureCurrentLink(t *testing.T, root, rootDir string) {
	t.Helper()
	kind, _ := install.InspectCurrent(root)
	if kind == model.CurrentJunction || kind == model.CurrentSymlink {
		return
	}
	current := filepath.Join(root, "current")
	_ = platform.RemoveLink(current)
	_ = os.Remove(filepath.Join(root, "current.pointer"))
	if err := platform.CreateLink(current, rootDir); err != nil {
		if err := os.Symlink(rootDir, current); err != nil {
			t.Skipf("cannot create a current link on this host: %v", err)
		}
	}
}

func trioCheck(t *testing.T, report model.DoctorData) model.Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == "trio_valid" {
			return check
		}
	}
	t.Fatal("trio_valid check not found")
	return model.Check{}
}

func TestDoctorTrioHealthyWithFFplay(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	root := t.TempDir()
	rootDir := installFakeTrio(t, root, fake)
	ensureCurrentLink(t, root, rootDir)

	report, err := doctor.Run(context.Background(), doctor.Options{
		DataRoot: root,
		Env:      []string{"PATH=" + filepath.Join(root, "current", "bin")},
		Runner:   ffexec.Runner{Timeout: 5 * time.Second},
	})
	require.NoError(t, err)
	check := trioCheck(t, report)
	require.True(t, check.OK)
	require.Contains(t, check.Detail, "ffplay")
	require.True(t, report.Healthy)
	for _, warning := range report.Warnings {
		require.NotEqual(t, "OMF-W1001", warning.Code)
	}
}

func TestDoctorTrioWarnsWhenFFplayMissing(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	root := t.TempDir()
	rootDir := installFakeTrio(t, root, fake)
	ensureCurrentLink(t, root, rootDir)
	require.NoError(t, os.Remove(filepath.Join(rootDir, "bin", filepath.Base(fake.FFplayPath))))

	report, err := doctor.Run(context.Background(), doctor.Options{
		DataRoot: root,
		Env:      []string{"PATH=" + filepath.Join(root, "current", "bin")},
		Runner:   ffexec.Runner{Timeout: 5 * time.Second},
	})
	require.NoError(t, err)
	check := trioCheck(t, report)
	// ffplay missing is a warning, not a failed check.
	require.True(t, check.OK)
	require.True(t, report.Healthy)
	require.NotContains(t, check.Detail, "ffplay")
	found := false
	for _, warning := range report.Warnings {
		if warning.Code == "OMF-W1001" {
			found = true
		}
	}
	require.True(t, found, "expected an OMF-W1001 warning for missing ffplay")
}
