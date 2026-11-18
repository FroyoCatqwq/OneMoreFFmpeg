package selfupdate_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
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

func selfupdateAssetName(t *testing.T, version string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "one-more-ffmpeg-v" + version + "-windows-amd64.exe"
	}
	return "one-more-ffmpeg-v" + version + "-linux-amd64"
}
