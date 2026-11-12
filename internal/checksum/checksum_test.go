package checksum_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func testdata(t *testing.T, name string) *os.File {
	t.Helper()
	file, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	return file
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

func TestStreamSHA256(t *testing.T) {
	digest, size, err := checksum.StreamSHA256(context.Background(), strings.NewReader("hello\n"))
	require.NoError(t, err)
	require.Equal(t, int64(6), size)
	require.Equal(t, "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03", digest)
}

func TestStreamSHA256LimitExceeded(t *testing.T) {
	_, size, err := checksum.StreamSHA256Limit(context.Background(), strings.NewReader("0123456789"), 5)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4004, typed.Code)
	require.Greater(t, size, int64(5))
}

func TestStreamSHA256Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := checksum.StreamSHA256(ctx, strings.NewReader("data"))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4006, typed.Code)
}

func TestStreamSHA256DeadlineIsDownloadFailure(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, _, err := checksum.StreamSHA256(ctx, strings.NewReader("data"))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
}

func TestStreamSHA256Deadline(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, _, err := checksum.StreamSHA256(ctx, strings.NewReader("data"))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
}

func TestStreamSHA256ReaderError(t *testing.T) {
	_, _, err := checksum.StreamSHA256(context.Background(), failingReader{})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
}

func TestParseChecksums(t *testing.T) {
	digest, err := checksum.ParseChecksums(testdata(t, "checksums.sha256"), "ffmpeg-master-latest-win64-gpl.zip")
	require.NoError(t, err)
	require.Equal(t, "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03", digest)

	digest, err = checksum.ParseChecksums(strings.NewReader(
		"BA7816BF8F01CFEA414140DE5DAE2223B00361A396177A9CB410FF61F20015AD *ffmpeg.zip\n"), "ffmpeg.zip")
	require.NoError(t, err)
	require.Equal(t, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad", digest)
}

func TestParseChecksumsRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../ffmpeg.zip", `C:\ffmpeg.zip`, "dir/ffmpeg.zip"} {
		_, err := checksum.ParseChecksums(strings.NewReader(""), name)
		typed, ok := apperrors.As(err)
		require.True(t, ok, name)
		require.Equal(t, apperrors.E4002, typed.Code, name)
	}
}

func TestParseChecksumsAcceptsRelativePaths(t *testing.T) {
	const digest = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	for _, line := range []string{
		digest + "  ./ffmpeg.zip\n",
		digest + "  dist/ffmpeg.zip\n",
	} {
		got, err := checksum.ParseChecksums(strings.NewReader(line), "ffmpeg.zip")
		require.NoError(t, err, line)
		require.Equal(t, digest, got, line)
	}
}

func TestParseChecksumsRejectsConflictingEntries(t *testing.T) {
	conflict := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  ffmpeg.zip\n" +
		"5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03  ./ffmpeg.zip\n"
	_, err := checksum.ParseChecksums(strings.NewReader(conflict), "ffmpeg.zip")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestParseChecksumsRejectsParentTraversal(t *testing.T) {
	body := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  ../ffmpeg.zip\n"
	_, err := checksum.ParseChecksums(strings.NewReader(body), "ffmpeg.zip")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestParseChecksumsReadErrorIsDownloadFailure(t *testing.T) {
	reader := io.MultiReader(
		strings.NewReader("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad  ffmpeg.zip\n"),
		failingReader{},
	)
	_, err := checksum.ParseChecksums(reader, "ffmpeg.zip")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
}

func TestParseChecksumsMissingEntry(t *testing.T) {
	_, err := checksum.ParseChecksums(testdata(t, "checksums.sha256"), "ffmpeg-master-latest-win64-lgpl.zip")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestParseChecksumsRejectsEmptyInput(t *testing.T) {
	_, err := checksum.ParseChecksums(io.LimitReader(strings.NewReader(""), 0), "ffmpeg.zip")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}
