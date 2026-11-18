package selfupdate_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	ggithub "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/selfupdate"
)

func TestCheckFindsNewerStable(t *testing.T) {
	version := "1.0.3"
	name := selfupdateAssetName(t, version)
	digest := sha256.Sum256([]byte("fake")) // digest value only checked for length
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"tag_name":"v1.0.3","draft":false,"prerelease":false,
		  "assets":[{"name":"` + name + `","digest":"sha256:` + hex.EncodeToString(digest[:]) + `",
		  "browser_download_url":"https://example.invalid/x"}]}]`))
	}))
	defer api.Close()
	client, err := ggithub.NewClient(api.URL, "")
	require.NoError(t, err)

	u, err := (selfupdate.Options{
		CurrentVersion: "1.0.2", GOOS: runtime.GOOS, GOARCH: "amd64", Client: client,
	}).Check(context.Background())
	require.NoError(t, err)
	require.NotNil(t, u)
	require.Equal(t, "1.0.3", u.Version)
}

func TestCheckNoUpdateWhenCurrentIsLatest(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"tag_name":"v1.0.3","draft":false,"prerelease":false,"assets":[]}]`))
	}))
	defer api.Close()
	client, err := ggithub.NewClient(api.URL, "")
	require.NoError(t, err)
	u, err := (selfupdate.Options{
		CurrentVersion: "1.0.3", GOOS: runtime.GOOS, GOARCH: "amd64", Client: client,
	}).Check(context.Background())
	require.NoError(t, err)
	require.Nil(t, u)
}

func TestApplyDownloadsVerifiesAndReplaces(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable replacement tested on Linux path")
	}
	payload := []byte("#!/bin/sh\necho updated\n")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	exe := filepath.Join(dir, "one-more-ffmpeg")
	require.NoError(t, os.WriteFile(exe, []byte("old"), 0o755))
	name := "one-more-ffmpeg-v9.9.9-linux-amd64"
	u := &selfupdate.Update{
		Version: "9.9.9", AssetName: name, AssetURL: server.URL,
		SHA256: hex.EncodeToString(sum[:]),
	}
	err := (selfupdate.Options{
		Executable: exe, Dir: dir, GOOS: "linux", GOARCH: "amd64",
	}).Apply(context.Background(), u)
	require.NoError(t, err)
	body, err := os.ReadFile(exe)
	require.NoError(t, err)
	require.Equal(t, payload, body)
}

func selfupdateAssetName(t *testing.T, version string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "one-more-ffmpeg-v" + version + "-windows-amd64.exe"
	}
	return "one-more-ffmpeg-v" + version + "-linux-amd64"
}
