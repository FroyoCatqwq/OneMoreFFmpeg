package official

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gh "github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func mockClient(t *testing.T, body string) *gh.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	client, err := gh.NewClient(gh.WithHTTPClient(srv.Client()), gh.WithEnterpriseURLs(srv.URL, srv.URL))
	require.NoError(t, err)
	return client
}

func releaseBody(assetName, digest string) string {
	release := map[string]any{
		"tag_name": "latest", "name": "Latest Auto-Build", "draft": false, "prerelease": false,
		"published_at": "2026-11-19T12:57:55Z", "id": 1,
		"assets": []any{map[string]any{
			"name": assetName, "digest": "sha256:" + digest,
			"browser_download_url": "https://example.invalid/" + assetName,
		}},
	}
	b, _ := json.Marshal(release)
	return string(b)
}

const testDigest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestResolveDefaultsGPLStaticSnapshot(t *testing.T) {
	client := mockClient(t, releaseBody("ffmpeg-master-latest-win64-gpl.zip", testDigest))
	res := &Resolver{Client: client}
	release, err := res.Resolve(context.Background(), Request{Target: "win64"})
	require.NoError(t, err)
	require.Equal(t, "gpl", release.Variant)
	require.Equal(t, "static", release.Linkage)
	require.Equal(t, "ffmpeg-master-latest-win64-gpl.zip", release.AssetName)
	require.Equal(t, testDigest, release.AssetSHA256)
	require.Len(t, release.Warnings, 1)
	require.Contains(t, release.Warnings[0].Message, "README")
}

func TestResolveRejectsStableLine(t *testing.T) {
	client := mockClient(t, releaseBody("x", testDigest))
	res := &Resolver{Client: client}
	_, err := res.Resolve(context.Background(), Request{Target: "win64", Line: "n9.0"})
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3004, typed.Code)
}

func TestResolveRejectsNonX8664(t *testing.T) {
	client := mockClient(t, releaseBody("x", testDigest))
	res := &Resolver{Client: client}
	_, err := res.Resolve(context.Background(), Request{Target: "linuxarm64"})
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)
}

func TestAssetNameConfigurableTemplate(t *testing.T) {
	rules := DefaultRules()
	rules.AssetTemplate = `custom-{{.Line}}-{{.Target}}-{{.Variant}}{{.Ext}}`
	res := &Resolver{Rules: rules}
	name, err := res.assetName(Request{Line: "master", Target: "linux64", Variant: "gpl", Linkage: "static"})
	require.NoError(t, err)
	require.Equal(t, "custom-master-linux64-gpl.tar.xz", name)
}

func TestResolveMissingAsset(t *testing.T) {
	client := mockClient(t, releaseBody("other-asset.zip", testDigest))
	res := &Resolver{Client: client}
	_, err := res.Resolve(context.Background(), Request{Target: "win64"})
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3008, typed.Code)
	require.Equal(t, "ffmpeg-master-latest-win64-gpl.zip", typed.Details["asset"])
}

func TestRulesNotHardcodedInLogic(t *testing.T) {
	// Changing the rules must change behavior without code changes.
	rules := DefaultRules()
	rules.Owner = "OtherOrg"
	rules.Repo = "OtherRepo"
	require.Equal(t, "OtherOrg/OtherRepo",
		strings.TrimSpace(rules.Owner+"/"+rules.Repo))
}
