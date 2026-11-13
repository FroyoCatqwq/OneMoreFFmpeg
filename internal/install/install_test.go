package install_test

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func buildArchive(t *testing.T, fake *testutil.FakeFFmpeg) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "ffmpeg.zip")
	file, err := os.Create(path)
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

	payload, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(payload)
	return path, hex.EncodeToString(sum[:])
}

func testRelease(sha string) model.Release {
	return model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "btbn",
		Repository:      "FFmpeg-Builds/FFmpeg-Builds",
		Channel:         "daily",
		ReleaseTag:      "autobuild-2026-11-12-12-00",
		ReleaseID:       "1",
		PublishedAt:     "2026-11-12T12:00:00Z",
		AssetName:       "ffmpeg.zip",
		AssetURL:        "https://example.invalid/ffmpeg.zip",
		AssetSHA256:     sha,
		Variant:         "gpl",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "gpl",
		Redistributable: true,
	}
}

func enableFakeVersion(t *testing.T) {
	t.Helper()
	t.Setenv("OMF_FAKE_VERSION_FFMPEG", "9.9.9")
	t.Setenv("OMF_FAKE_VERSION_FFPROBE", "9.9.9")
	t.Setenv("OMF_FAKE_VERSION_FFPLAY", "9.9.9")
}

func TestInstallDryRunWritesNothing(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	archivePath, sha := buildArchive(t, fake)
	root := t.TempDir()

	data, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(sha), ArchivePath: archivePath,
	})
	require.NoError(t, err)
	require.True(t, data.DryRun)
	require.Contains(t, data.RootDir, filepath.Join("apps", "btbn"))
	require.NoDirExists(t, data.RootDir)
	require.Equal(t, model.CurrentMissing, data.CurrentPointer)
}

func TestInstallApplySwitchesCurrentAndJournals(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	enableFakeVersion(t)
	archivePath, sha := buildArchive(t, fake)
	root := t.TempDir()

	data, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(sha), ArchivePath: archivePath,
		Apply: true, EmitLock: true,
	})
	require.NoError(t, err)
	require.False(t, data.DryRun)
	require.NotEmpty(t, data.InstallationID)
	require.Equal(t, sha, data.AssetSHA256)
	require.FileExists(t, filepath.Join(data.RootDir, "bin", filepath.Base(fake.FFmpegPath)))
	require.FileExists(t, filepath.Join(data.RootDir, "install.json"))
	require.FileExists(t, filepath.Join(data.RootDir, "ffmpeg-trio.lock"))

	switch data.CurrentPointer {
	case model.CurrentSymlink:
		target, err := os.Readlink(filepath.Join(root, "current"))
		require.NoError(t, err)
		require.NotEmpty(t, target)
	case model.CurrentPointerFile:
		body, err := os.ReadFile(filepath.Join(root, "current.pointer"))
		require.NoError(t, err)
		require.Contains(t, string(body), data.RootDir)
	default:
		t.Fatalf("unexpected current pointer: %s", data.CurrentPointer)
	}

	records, err := journal.List(filepath.Join(root, "journal"))
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, journal.StatusCommitted, records[0].Status)

	// Applying the same archive is idempotent.
	second, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(sha), ArchivePath: archivePath, Apply: true,
	})
	require.NoError(t, err)
	require.Equal(t, data.InstallationID, second.InstallationID)

	recordBody, err := os.ReadFile(filepath.Join(data.RootDir, "install.json"))
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(recordBody, &stored))
	require.Equal(t, "onemoreffmpeg.install-record/v1", stored["schema"])
}

func TestInstallForceReplacesExistingDirectory(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	enableFakeVersion(t)
	archivePath, sha := buildArchive(t, fake)
	root := t.TempDir()
	release := testRelease(sha)
	target := filepath.Join(root, "apps", "btbn", release.ReleaseTag, sha[:16])
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "old.txt"), []byte("old"), 0o644))

	data, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: release, ArchivePath: archivePath, Apply: true, Force: true,
	})
	require.NoError(t, err)
	require.Equal(t, target, data.RootDir)
	_, err = os.Stat(filepath.Join(target, "old.txt"))
	require.True(t, os.IsNotExist(err))
}

func TestInstallRollbackRestoresPreviousVersion(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	enableFakeVersion(t)
	archiveOne, shaOne := buildArchive(t, fake)
	second := filepath.Join(filepath.Dir(archiveOne), "ffmpeg-fake-2.zip")
	sourceZip, err := zip.OpenReader(archiveOne)
	require.NoError(t, err)
	destFile, err := os.Create(second)
	require.NoError(t, err)
	destZip := zip.NewWriter(destFile)
	for _, entry := range sourceZip.File {
		reader, err := entry.Open()
		require.NoError(t, err)
		body, err := io.ReadAll(reader)
		_ = reader.Close()
		require.NoError(t, err)
		writer, err := destZip.Create(entry.Name)
		require.NoError(t, err)
		_, err = writer.Write(body)
		require.NoError(t, err)
	}
	writer, err := destZip.Create("ffmpeg-build/version.txt")
	require.NoError(t, err)
	_, err = writer.Write([]byte("second"))
	require.NoError(t, err)
	require.NoError(t, destZip.Close())
	require.NoError(t, destFile.Close())
	require.NoError(t, sourceZip.Close())
	payload, err := os.ReadFile(second)
	require.NoError(t, err)
	secondSHA := sha256.Sum256(payload)
	shaTwo := hex.EncodeToString(secondSHA[:])
	require.NotEqual(t, shaOne, shaTwo)

	root := t.TempDir()
	first, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(shaOne), ArchivePath: archiveOne, Apply: true,
	})
	require.NoError(t, err)
	secondData, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(shaTwo), ArchivePath: second, Apply: true, Force: true,
	})
	require.NoError(t, err)
	require.NotEqual(t, first.RootDir, secondData.RootDir)
	require.NoDirExists(t, first.RootDir)
	require.DirExists(t, secondData.RootDir)

	record, err := journal.Read(filepath.Join(root, "journal"), secondData.JournalID)
	require.NoError(t, err)
	require.Equal(t, journal.StatusCommitted, record.Status)
	require.Equal(t, first.RootDir, record.Details["quarantine_original"])

	detail, err := install.Rollback(root, record)
	require.NoError(t, err)
	require.Contains(t, detail, "rolled back")
	require.DirExists(t, first.RootDir)
	require.NoDirExists(t, secondData.RootDir)
}

func TestInstallFailureRestoresQuarantinedVersion(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	enableFakeVersion(t)
	archive, sha := buildArchive(t, fake)
	root := t.TempDir()
	first, err := install.Run(context.Background(), install.Options{
		DataRoot: root, Release: testRelease(sha), ArchivePath: archive, Apply: true,
	})
	require.NoError(t, err)

	bad := filepath.Join(filepath.Dir(archive), "bad.zip")
	file, err := os.Create(bad)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	entry, err := writer.Create("readme.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("no binaries"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	payload, err := os.ReadFile(bad)
	require.NoError(t, err)
	badSum := sha256.Sum256(payload)
	badRelease := testRelease(hex.EncodeToString(badSum[:]))

	_, err = install.Run(context.Background(), install.Options{
		DataRoot: root, Release: badRelease, ArchivePath: bad, Apply: true, Force: true,
	})
	require.Error(t, err)
	require.DirExists(t, first.RootDir)
	require.NoDirExists(t, install.TargetDir(root, badRelease))
}

func TestInstallRejectsMissingBinaries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.zip")
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	entry, err := writer.Create("readme.txt")
	require.NoError(t, err)
	_, err = entry.Write([]byte("no binaries"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
	payload, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(payload)

	_, err = install.Run(context.Background(), install.Options{
		DataRoot: t.TempDir(), Release: testRelease(hex.EncodeToString(sum[:])), ArchivePath: path, Apply: true,
	})
	require.Error(t, err)
}
