package download_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/download"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const assetName = "ffmpeg-n9.0-latest-win64-gpl-9.0.zip"

func shaOf(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

type fixture struct {
	server        *httptest.Server
	payload       []byte
	sha           string
	checksums     string
	requests      *int
	assetRequests *int
}

func newFixture(t *testing.T, payload []byte) *fixture {
	t.Helper()
	requests := 0
	assetRequests := 0
	f := &fixture{payload: payload, sha: shaOf(payload), requests: &requests, assetRequests: &assetRequests}
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(writer http.ResponseWriter, _ *http.Request) {
		requests++
		assetRequests++
		_, _ = writer.Write(payload)
	})
	mux.HandleFunc("/checksums", func(writer http.ResponseWriter, _ *http.Request) {
		requests++
		_, _ = writer.Write([]byte(f.checksums))
	})
	mux.HandleFunc("/broken", func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "boom", http.StatusInternalServerError)
	})
	mux.HandleFunc("/redirect", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/redirect", http.StatusFound)
	})
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	f.checksums = f.sha + "  " + assetName + "\n"
	return f
}

func (f *fixture) release() model.Release {
	return model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "btbn",
		Repository:      "FFmpeg-Builds/FFmpeg-Builds",
		Channel:         "stable",
		ReleaseTag:      "latest",
		ReleaseID:       "441698707",
		PublishedAt:     "2026-11-11T13:00:32Z",
		AssetName:       assetName,
		AssetURL:        f.server.URL + "/asset",
		AssetSHA256:     f.sha,
		ChecksumsURL:    f.server.URL + "/checksums",
		ChecksumsSHA256: shaOf([]byte(f.checksums)),
		Variant:         "gpl",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "gpl",
		Redistributable: true,
	}
}

func downloader(t *testing.T, dir string) *download.Downloader {
	t.Helper()
	instance, err := download.New(download.Options{Dir: dir, MaxBytes: 1 << 20})
	require.NoError(t, err)
	return instance
}

func TestFetchVerifiesAndRecordsArtifact(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	dir := t.TempDir()

	artifact, err := downloader(t, dir).Fetch(context.Background(), fixture.release())
	require.NoError(t, err)
	require.True(t, artifact.Verified)
	require.Equal(t, "art-"+fixture.sha[:16], artifact.ArtifactID)
	require.Equal(t, "btbn", artifact.Provider)

	expectedPath := filepath.Join(dir, fixture.sha[:16], assetName)
	require.Equal(t, expectedPath, artifact.LocalPath)
	content, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	require.Equal(t, fixture.payload, content)

	raw, err := os.ReadFile(filepath.Join(dir, fixture.sha[:16], "artifact.json"))
	require.NoError(t, err)
	var stored model.Artifact
	require.NoError(t, json.Unmarshal(raw, &stored))
	require.Equal(t, artifact, stored)
}

func TestFetchReusesExistingFile(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	dir := t.TempDir()
	instance := downloader(t, dir)

	_, err := instance.Fetch(context.Background(), fixture.release())
	require.NoError(t, err)
	requestsAfterFirst := *fixture.requests

	_, err = instance.Fetch(context.Background(), fixture.release())
	require.NoError(t, err)
	require.Equal(t, 1, *fixture.assetRequests, "matching file must not be downloaded again")
	require.Equal(t, requestsAfterFirst+1, *fixture.requests, "checksums source must be rechecked on reuse")
}

func TestFetchRejectsSHA256Mismatch(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	release := fixture.release()
	release.AssetSHA256 = strings.Repeat("0", 64)
	release.ChecksumsURL = ""
	dir := t.TempDir()

	_, err := downloader(t, dir).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)

	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	require.Empty(t, entries, "failed download must not leave files behind")
}

func TestFetchRejectsBackslashAssetName(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	release := fixture.release()
	release.ChecksumsURL = ""
	release.ChecksumsSHA256 = ""
	release.AssetName = `..\\evil.zip`

	_, err := downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4005, typed.Code)
}

func TestFetchRejectsChecksumsDisagreement(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	fixture.checksums = strings.Repeat("a", 64) + "  " + assetName + "\n"
	release := fixture.release()
	release.ChecksumsSHA256 = "" // only the parsed line can disagree
	dir := t.TempDir()

	_, err := downloader(t, dir).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
	_, statErr := os.Stat(filepath.Join(dir, fixture.sha[:16], assetName))
	require.True(t, os.IsNotExist(statErr), "unverified file must be removed")
}

func TestFetchRejectsChecksumsFileHashMismatch(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	release := fixture.release()
	release.ChecksumsSHA256 = strings.Repeat("b", 64)
	dir := t.TempDir()

	_, err := downloader(t, dir).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestFetchEnforcesSizeLimit(t *testing.T) {
	fixture := newFixture(t, []byte(strings.Repeat("x", 2048)))
	instance, err := download.New(download.Options{Dir: t.TempDir(), MaxBytes: 1024})
	require.NoError(t, err)

	_, err = instance.Fetch(context.Background(), fixture.release())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4004, typed.Code)
}

func TestFetchRejectsPlainHTTPRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "http://example.invalid/asset.zip", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	release := model.Release{
		Schema:      model.ReleaseSchema,
		Provider:    "btbn",
		ReleaseTag:  "latest",
		AssetName:   assetName,
		AssetURL:    server.URL,
		AssetSHA256: strings.Repeat("c", 64),
	}
	_, err := downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
	require.Contains(t, typed.Message, "redirect target must be https")
}

func TestFetchReplacesCorruptExistingFile(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	dir := t.TempDir()
	target := filepath.Join(dir, fixture.sha[:16], assetName)
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("corrupt"), 0o644))

	artifact, err := downloader(t, dir).Fetch(context.Background(), fixture.release())
	require.NoError(t, err)
	require.Equal(t, target, artifact.LocalPath)
	content, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, fixture.payload, content)
}

func TestFetchCleansUpWhenArtifactWriteFails(t *testing.T) {
	fixture := newFixture(t, []byte("payload-bytes"))
	dir := t.TempDir()
	targetDir := filepath.Join(dir, fixture.sha[:16])
	require.NoError(t, os.MkdirAll(filepath.Join(targetDir, "artifact.json"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(targetDir, "artifact.json.bak"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(targetDir, "artifact.json.bak", "keep"), []byte("x"), 0o644))

	_, err := downloader(t, dir).Fetch(context.Background(), fixture.release())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4005, typed.Code)
	_, statErr := os.Stat(filepath.Join(targetDir, assetName))
	require.True(t, os.IsNotExist(statErr), "newly downloaded asset must be removed when artifact.json cannot be written")
}

func TestFetchHonorsZeroRedirects(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	release := fixture.release()
	release.ChecksumsURL = ""
	release.ChecksumsSHA256 = ""
	release.AssetURL = fixture.server.URL + "/redirect"
	instance, err := download.New(download.Options{Dir: t.TempDir(), MaxBytes: 1 << 20, MaxRedirects: 0})
	require.NoError(t, err)

	_, err = instance.Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4003, typed.Code)
}

func TestFetchEnforcesRedirectLimit(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	release := fixture.release()
	release.AssetURL = fixture.server.URL + "/redirect"
	instance, err := download.New(download.Options{Dir: t.TempDir(), MaxBytes: 1 << 20, MaxRedirects: 1})
	require.NoError(t, err)

	_, err = instance.Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4003, typed.Code)
}

func TestFetchMapsHTTPFailure(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	release := fixture.release()
	release.AssetURL = fixture.server.URL + "/broken"

	_, err := downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)
}

func TestFetchCanceled(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := downloader(t, t.TempDir()).Fetch(ctx, fixture.release())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4006, typed.Code)
}

func TestFetchRejectsUnsafeNamesAndSchemes(t *testing.T) {
	fixture := newFixture(t, []byte("payload"))
	release := fixture.release()
	release.AssetName = "../escape.zip"
	_, err := downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4005, typed.Code)

	release = fixture.release()
	release.AssetURL = "http://example.com/asset.zip"
	_, err = downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4001, typed.Code)

	release = fixture.release()
	release.AssetSHA256 = "not-a-hash"
	_, err = downloader(t, t.TempDir()).Fetch(context.Background(), release)
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestNewRequiresAbsoluteDir(t *testing.T) {
	_, err := download.New(download.Options{Dir: "relative"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4005, typed.Code)
}

// A large asset on a slow link trickles its body over longer than the phase
// timeout. The timeout must bound connection phases, not the whole body read,
// so a steady download still completes.
func TestFetchSlowBodyCompletesDespiteShortTimeout(t *testing.T) {
	payload := []byte(strings.Repeat("x", 4096))
	sha := shaOf(payload)
	checksums := sha + "  " + assetName + "\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(writer http.ResponseWriter, _ *http.Request) {
		flusher := writer.(http.Flusher)
		chunk := len(payload) / 8
		for start := 0; start < len(payload); start += chunk {
			end := start + chunk
			if end > len(payload) {
				end = len(payload)
			}
			_, _ = writer.Write(payload[start:end])
			flusher.Flush()
			time.Sleep(60 * time.Millisecond)
		}
	})
	mux.HandleFunc("/checksums", func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(checksums))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	instance, err := download.New(download.Options{Dir: t.TempDir(), MaxBytes: 1 << 20, Timeout: 250 * time.Millisecond})
	require.NoError(t, err)
	release := model.Release{
		Schema: model.ReleaseSchema, Provider: "btbn", ReleaseTag: "latest",
		AssetName: assetName, AssetURL: server.URL + "/asset", AssetSHA256: sha,
		ChecksumsURL: server.URL + "/checksums", ChecksumsSHA256: shaOf([]byte(checksums)),
	}
	artifact, err := instance.Fetch(context.Background(), release)
	require.NoError(t, err, "a slow but steady body must not be aborted by the phase timeout")
	require.True(t, artifact.Verified)
}
