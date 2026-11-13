package archive_test

import (
	"archive/tar"
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/archive"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func writeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	for name, body := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())
}

func writeTarXZ(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	xzWriter, err := xz.NewWriter(file)
	require.NoError(t, err)
	tarWriter := tar.NewWriter(xzWriter)
	for name, body := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}))
		_, err := tarWriter.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, xzWriter.Close())
	require.NoError(t, file.Close())
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "ffmpeg.zip")
	writeZip(t, archivePath, map[string]string{
		"ffmpeg-9.0/bin/ffmpeg":  "ffmpeg-bytes",
		"ffmpeg-9.0/LICENSE.txt": "license",
	})
	destination := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(destination, 0o755))

	require.NoError(t, archive.Extract(context.Background(), archivePath, destination))
	body, err := os.ReadFile(filepath.Join(destination, "ffmpeg-9.0", "bin", "ffmpeg"))
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-bytes", string(body))
}

func TestExtractTarXZ(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "ffmpeg.tar.xz")
	writeTarXZ(t, archivePath, map[string]string{"pkg/bin/ffmpeg": "data"})
	destination := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(destination, 0o755))

	require.NoError(t, archive.Extract(context.Background(), archivePath, destination))
	_, err := os.Stat(filepath.Join(destination, "pkg", "bin", "ffmpeg"))
	require.NoError(t, err)
}

func TestExtractRejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../evil.txt", "/etc/passwd", "pkg/../../evil.txt", `C:\evil.txt`} {
		dir := t.TempDir()
		archivePath := filepath.Join(dir, "bad.zip")
		writeZip(t, archivePath, map[string]string{name: "x"})
		destination := filepath.Join(dir, "out")
		require.NoError(t, os.MkdirAll(destination, 0o755))

		err := archive.Extract(context.Background(), archivePath, destination)
		typed, ok := apperrors.As(err)
		require.True(t, ok, name)
		require.Equal(t, apperrors.E5002, typed.Code, name)
	}
}

func TestExtractRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "link.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)
	writer := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "pkg/link"}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	require.NoError(t, err)
	_, err = entry.Write([]byte("../target"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, file.Close())

	destination := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(destination, 0o755))
	err = archive.Extract(context.Background(), archivePath, destination)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E5002, typed.Code)
}

func TestExtractRejectsUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ffmpeg.7z")
	require.NoError(t, os.WriteFile(path, []byte("7z"), 0o644))
	destination := filepath.Join(dir, "out")
	require.NoError(t, os.MkdirAll(destination, 0o755))

	err := archive.Extract(context.Background(), path, destination)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E5001, typed.Code)
}

func TestFormatFor(t *testing.T) {
	require.Equal(t, "zip", archive.FormatFor("a.zip"))
	require.Equal(t, "tar.xz", archive.FormatFor("A.TAR.XZ"))
	require.Equal(t, "tar.gz", archive.FormatFor("a.tgz"))
	require.Equal(t, "", archive.FormatFor("a.rar"))
	require.True(t, strings.HasSuffix(archive.FormatFor("a.tar.zst"), "zst"))
}
