package cli_test

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

	"github.com/stretchr/testify/require"

	appcli "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cli"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
)

const (
	phase4AssetName = "ffmpeg-n9.0-latest-win64-gpl-9.0.zip"
	phase4SHA       = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func phase4Home(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(paths.HomeEnv, home)
	return home
}

func runJSON(t *testing.T, args ...string) (int, map[string]any, string) {
	t.Helper()
	opts, stdout, _ := testOptions(args...)
	code := appcli.Run(context.Background(), opts)
	var object map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &object), stdout.String())
	return code, object, stdout.String()
}

func TestProviderAddAndList(t *testing.T) {
	home := phase4Home(t)
	sha := strings.Repeat("c", 64)

	code, object, raw := runJSON(t, "provider", "add", "--id", "my-ffmpeg",
		"--url", "https://example.invalid/ffmpeg.zip", "--sha256", sha, "--apply", "--json")
	require.Equal(t, 0, code, raw)
	data := object["data"].(map[string]any)
	require.Equal(t, "my-ffmpeg", data["id"])
	require.Equal(t, "added", data["status"])
	require.FileExists(t, filepath.Join(home, "providers.toml"))

	code, object, raw = runJSON(t, "provider", "list", "--json")
	require.Equal(t, 0, code, raw)
	providers := object["data"].(map[string]any)["providers"].([]any)
	require.Len(t, providers, 1)
	require.Equal(t, "my-ffmpeg", providers[0].(map[string]any)["id"])

	code, object, _ = runJSON(t, "provider", "add", "--id", "my-ffmpeg",
		"--url", "https://example.invalid/other.zip", "--sha256", sha, "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7005", object["code"])

	code, object, _ = runJSON(t, "provider", "add", "--id", "another",
		"--url", "https://example.invalid/other.zip", "--sha256", sha, "--dry-run", "--json")
	require.Equal(t, 0, code)
	require.Equal(t, "dry-run", object["data"].(map[string]any)["status"])
}

func TestDownloadOfflineRelease(t *testing.T) {
	payload := []byte("asset-payload")
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	checksums := digest + "  " + phase4AssetName + "\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/asset", func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write(payload) })
	mux.HandleFunc("/checksums", func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte(checksums)) })
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	phase4Home(t)
	downloadDir := t.TempDir()
	release := model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "custom",
		Repository:      "custom",
		Channel:         "daily",
		ReleaseTag:      "latest",
		ReleaseID:       "441698707",
		PublishedAt:     "2026-11-11T13:00:32Z",
		AssetName:       phase4AssetName,
		AssetURL:        server.URL + "/asset",
		AssetSHA256:     digest,
		ChecksumsURL:    server.URL + "/checksums",
		ChecksumsSHA256: shaOfString(checksums),
		Variant:         "gpl",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "gpl",
		Redistributable: true,
	}
	raw, err := json.Marshal(map[string]any{
		"schema":  model.ResultSchema,
		"ok":      true,
		"command": "resolve",
		"data":    release,
	})
	require.NoError(t, err)
	releasePath := filepath.Join(t.TempDir(), "release.json")
	require.NoError(t, os.WriteFile(releasePath, raw, 0o644))

	code, object, out := runJSON(t, "download", "--release", releasePath, "--dir", downloadDir, "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, true, data["verified"])
	require.Equal(t, digest, data["asset_sha256"])
	require.FileExists(t, filepath.Join(downloadDir, digest[:16], phase4AssetName))

	// Nonfree offline release must still require consent.
	release.LicenseHint = "nonfree"
	release.Redistributable = false
	raw, err = json.Marshal(release)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(releasePath, raw, 0o644))
	code, object, _ = runJSON(t, "download", "--release", releasePath, "--dir", downloadDir, "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7001", object["code"])

	code, object, rawOut := runJSON(t, "download", "--release", releasePath, "--dir", downloadDir, "--allow-nonfree", "--json")
	require.Equal(t, 0, code, rawOut)
	warnings := object["data"].(map[string]any)["warnings"].([]any)
	require.Len(t, warnings, 1)
	require.Equal(t, "OMF-W1006", warnings[0].(map[string]any)["code"])

	code, object, _ = runJSON(t, "download", "--release", releasePath, "--dir", downloadDir, "--max-bytes", "0", "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7005", object["code"])

	// A nonfree variant is enough even when license_hint claims otherwise.
	release.LicenseHint = "gpl"
	release.Variant = "nonfree"
	raw, err = json.Marshal(release)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(releasePath, raw, 0o644))
	code, object, _ = runJSON(t, "download", "--release", releasePath, "--dir", downloadDir, "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7001", object["code"])
}

func shaOfString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestDownloadRejectsMalformedReleaseFile(t *testing.T) {
	phase4Home(t)
	base := model.Release{
		Schema:      model.ReleaseSchema,
		Provider:    "btbn",
		ReleaseTag:  "v1",
		AssetName:   phase4AssetName,
		AssetURL:    "https://example.invalid/a.zip",
		AssetSHA256: phase4SHA,
	}
	path := filepath.Join(t.TempDir(), "release.json")
	raw, err := json.Marshal(base)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	// Missing repository/channel/published/variant/... is a contract error.
	code, object, _ := runJSON(t, "download", "--release", path, "--dir", t.TempDir(), "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7003", object["code"])

	code, object, _ = runJSON(t, "download", "--release", filepath.Join(t.TempDir(), "missing.json"),
		"--dir", t.TempDir(), "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7007", object["code"])

	base.Repository = "FFmpeg-Builds/FFmpeg-Builds"
	base.Channel = "stable"
	base.PublishedAt = "2026-11-11T13:00:32Z"
	base.Variant = "gpl"
	base.Linkage = "static"
	base.Target = "win64"
	base.LicenseHint = "gpl"
	base.Redistributable = true
	raw, err = json.Marshal(base)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	code, object, _ = runJSON(t, "download", "--release", path, "--dir", "relative", "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7003", object["code"])
}

func TestDownloadRejectsUnpinnedReleaseFile(t *testing.T) {
	phase4Home(t)
	release := map[string]any{
		"schema":       model.ReleaseSchema,
		"provider":     "btbn",
		"release_tag":  "latest",
		"asset_name":   phase4AssetName,
		"asset_url":    "https://example.invalid/a.zip",
		"asset_sha256": phase4SHA,
	}
	raw, err := json.Marshal(release)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "release.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	code, object, _ := runJSON(t, "download", "--release", path, "--dir", t.TempDir(), "--json")
	require.Equal(t, 1, code)
	require.Equal(t, "OMF-E3010", object["code"])
}
