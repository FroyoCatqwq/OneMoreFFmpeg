package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

const (
	stableSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	masterSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func asset(name, digest string) map[string]any {
	return map[string]any{
		"name":                 name,
		"size":                 1024,
		"browser_download_url": "https://example.invalid/" + name,
		"digest":               digest,
		"updated_at":           "2026-11-11T13:00:00Z",
	}
}

func releasePayload(id int64, tag, published string, assets ...map[string]any) map[string]any {
	if assets == nil {
		assets = []map[string]any{}
	}
	return map[string]any{
		"id":           id,
		"tag_name":     tag,
		"name":         tag,
		"published_at": published,
		"assets":       assets,
	}
}

func btbnServer(t *testing.T, includeChecksums bool, digest string) *httptest.Server {
	t.Helper()
	stableAssets := []map[string]any{
		asset("ffmpeg-n9.0-latest-win64-gpl-9.0.zip", digest),
		asset("ffmpeg-n8.1-latest-win64-gpl-8.1.zip", ""),
		asset("ffmpeg-master-latest-win64-gpl.zip", ""),
	}
	if includeChecksums {
		stableAssets = append(stableAssets, asset("checksums.sha256", ""))
	}
	stable := releasePayload(441698707, "latest", "2026-11-11T13:00:32Z", stableAssets...)
	master := releasePayload(441698708, "autobuild-2026-11-11-12-39", "2026-11-11T12:39:00Z",
		asset("ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip", "sha256:"+masterSHA))

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(request.URL.Path, "/releases/latest"):
			_ = json.NewEncoder(writer).Encode(stable)
		case strings.HasSuffix(request.URL.Path, "/releases"):
			_ = json.NewEncoder(writer).Encode([]map[string]any{master})
		case strings.Contains(request.URL.Path, "/releases/tags/"):
			_ = json.NewEncoder(writer).Encode(master)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func checksumsFetcher(line string) provider.ChecksumsFetcher {
	return func(context.Context, string) ([]byte, error) {
		return []byte(line), nil
	}
}

func resolver(t *testing.T, server *httptest.Server, fetch provider.ChecksumsFetcher) *provider.Resolver {
	t.Helper()
	client, err := github.NewClient(server.URL, "")
	require.NoError(t, err)
	if fetch == nil {
		fetch = checksumsFetcher(stableSHA + "  ffmpeg-n9.0-latest-win64-gpl-9.0.zip\n")
	}
	return &provider.Resolver{
		Client: client,
		Fetch:  fetch,
		Now:    func() time.Time { return time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC) },
	}
}

func TestResolveStablePrefersDigest(t *testing.T) {
	server := btbnServer(t, true, "sha256:"+stableSHA)
	result, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "latest", result.ReleaseTag)
	require.Equal(t, "441698707", result.ReleaseID)
	require.Equal(t, "2026-11-11T13:00:32Z", result.PublishedAt)
	require.Equal(t, "ffmpeg-n9.0-latest-win64-gpl-9.0.zip", result.AssetName)
	require.Equal(t, stableSHA, result.AssetSHA256)
	require.Equal(t, "gpl", result.LicenseHint)
	require.True(t, result.Redistributable)
	require.Equal(t, "2026-11-12T00:00:00Z", result.PinnedAt)
	require.NotEmpty(t, result.ChecksumsURL)
}

func TestResolveStableFallsBackToChecksums(t *testing.T) {
	server := btbnServer(t, true, "")
	result, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, stableSHA, result.AssetSHA256)
	require.NotEmpty(t, result.ChecksumsSHA256)
}

func TestResolveDailyUsesConcreteAutobuildTag(t *testing.T) {
	server := btbnServer(t, false, "")
	result, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "daily", Target: "win64", Variant: "gpl",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, "autobuild-2026-11-11-12-39", result.ReleaseTag)
	require.Equal(t, "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip", result.AssetName)
}

func TestResolveNonfreeConsent(t *testing.T) {
	nonfreeAssets := []map[string]any{
		asset("ffmpeg-N-128756-gd0b9c63c9f-win64-nonfree.zip", "sha256:"+masterSHA),
		asset("ffmpeg-N-128756-gd0b9c63c9f-win64-nonfree-shared.zip", ""),
	}
	nonfreeRelease := releasePayload(99, "autobuild-2026-11-11-12-39", "2026-11-11T12:39:00Z", nonfreeAssets...)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(request.URL.Path, "/releases") {
			_ = json.NewEncoder(writer).Encode([]map[string]any{nonfreeRelease})
			return
		}
		_ = json.NewEncoder(writer).Encode(nonfreeRelease)
	}))
	t.Cleanup(server.Close)

	_, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "bnqdzj-nonfree", Channel: "daily", Target: "win64", Variant: "nonfree",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7001, typed.Code)

	result, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "bnqdzj-nonfree", Channel: "daily", Target: "win64", Variant: "nonfree", AllowNonfree: true,
	}, nil)
	require.NoError(t, err)
	require.False(t, result.Redistributable)
	require.Equal(t, "nonfree", result.LicenseHint)
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "OMF-W1006", result.Warnings[0].Code)
}

func TestResolveStableUnsupportedForBnqdzj(t *testing.T) {
	server := btbnServer(t, false, "")
	_, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "bnqdzj-nonfree", Channel: "stable", Target: "win64", Variant: "nonfree",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3004, typed.Code)
}

func TestResolveWithoutChecksumsFails(t *testing.T) {
	server := btbnServer(t, false, "")
	_, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3005, typed.Code)
}

func TestResolveDigestSurvivesChecksumsFetchFailure(t *testing.T) {
	server := btbnServer(t, true, "sha256:"+stableSHA)
	failing := func(context.Context, string) ([]byte, error) {
		return nil, apperrors.New(apperrors.E3005, "checksums download failed")
	}
	result, err := resolver(t, server, failing).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, stableSHA, result.AssetSHA256)
	require.Empty(t, result.ChecksumsURL)

	// Without a digest the checksums failure is fatal.
	serverNoDigest := btbnServer(t, true, "")
	_, err = resolver(t, serverNoDigest, failing).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3005, typed.Code)
}

func TestResolveDigestAndChecksumsMismatch(t *testing.T) {
	server := btbnServer(t, true, "sha256:"+stableSHA)
	fetch := checksumsFetcher(masterSHA + "  ffmpeg-n9.0-latest-win64-gpl-9.0.zip\n")
	_, err := resolver(t, server, fetch).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestResolveCustomProvider(t *testing.T) {
	custom := []provider.Custom{{
		ID:          "my-ffmpeg",
		URL:         "https://example.invalid/ffmpeg-custom-win64.zip",
		SHA256:      stableSHA,
		LicenseHint: "gpl",
	}}
	server := btbnServer(t, false, "")
	result, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl",
	}, custom)
	require.NoError(t, err)
	require.Equal(t, "my-ffmpeg", result.Provider)
	require.Equal(t, "custom", result.Repository)
	require.Equal(t, "direct-aaaaaaaaaaaa", result.ReleaseTag)
	require.Equal(t, stableSHA, result.AssetSHA256)
	require.Equal(t, "ffmpeg-custom-win64.zip", result.AssetName)
	require.Equal(t, "gpl", result.LicenseHint)
	require.False(t, result.Redistributable)
}

func TestNormalizeRequestRejectsConflictingPin(t *testing.T) {
	_, err := provider.NormalizeRequest(provider.Request{ReleaseTag: "latest", ReleaseLine: "n9.0"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7001, typed.Code)
}

func TestNormalizeRequestDefaults(t *testing.T) {
	request, err := provider.NormalizeRequest(provider.Request{})
	require.NoError(t, err)
	require.Equal(t, "btbn", request.Provider)
	require.Equal(t, "daily", request.Channel)
	require.Equal(t, "gpl", request.Variant)
	require.Equal(t, "static", request.Linkage)
	require.NotEmpty(t, request.Target)

	request, err = provider.NormalizeRequest(provider.Request{Variant: "nonfree-shared"})
	require.NoError(t, err)
	require.Equal(t, "nonfree", request.Variant)
	require.Equal(t, "shared", request.Linkage)
}

func TestNormalizeRequestRejectsLineChannelMismatch(t *testing.T) {
	cases := []provider.Request{
		{Channel: "stable", ReleaseLine: "master"},
		{Channel: "daily", ReleaseLine: "n9.0"},
		{Channel: "daily", ReleaseTag: "latest"},
		{Channel: "daily", Variant: "banana"},
		{Channel: "daily", Linkage: "dynamic"},
	}
	for _, request := range cases {
		_, err := provider.NormalizeRequest(request)
		typed, ok := apperrors.As(err)
		require.True(t, ok, request)
		require.Equal(t, apperrors.E7001, typed.Code, request)
	}
}

func TestResolveCustomRejectsExplicitPinOptions(t *testing.T) {
	custom := []provider.Custom{{
		ID: "my-ffmpeg", URL: "https://example.invalid/ffmpeg.zip", SHA256: stableSHA, LicenseHint: "gpl",
	}}
	for _, request := range []provider.Request{
		{Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl", ReleaseTag: "v1"},
		{Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl", ReleaseLine: "master"},
	} {
		_, err := (&provider.Resolver{}).Resolve(context.Background(), provider.NewRegistry(), request, custom)
		typed, ok := apperrors.As(err)
		require.True(t, ok)
		require.Equal(t, apperrors.E7001, typed.Code)
	}
}

func TestResolveCustomVerifiesChecksumsFile(t *testing.T) {
	custom := []provider.Custom{{
		ID:           "my-ffmpeg",
		URL:          "https://example.invalid/ffmpeg.zip",
		SHA256:       stableSHA,
		ChecksumsURL: "https://example.invalid/checksums.sha256",
		LicenseHint:  "gpl",
	}}
	fetch := checksumsFetcher(stableSHA + "  ffmpeg.zip\n")
	result, err := (&provider.Resolver{Fetch: fetch}).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl",
	}, custom)
	require.NoError(t, err)
	require.NotEmpty(t, result.ChecksumsSHA256)

	fetch = checksumsFetcher(masterSHA + "  ffmpeg.zip\n")
	_, err = (&provider.Resolver{Fetch: fetch}).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl",
	}, custom)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestResolveCustomRejectsInvalidPinAndURL(t *testing.T) {
	cases := []struct {
		name   string
		custom provider.Custom
		code   apperrors.Code
	}{
		{
			name:   "short sha",
			custom: provider.Custom{ID: "bad", URL: "https://example.invalid/a.zip", SHA256: "abc", LicenseHint: "gpl"},
			code:   apperrors.E3009,
		},
		{
			name:   "http url",
			custom: provider.Custom{ID: "bad", URL: "http://example.invalid/a.zip", SHA256: stableSHA, LicenseHint: "gpl"},
			code:   apperrors.E3008,
		},
		{
			name:   "credential url",
			custom: provider.Custom{ID: "bad", URL: "https://user:pass@example.invalid/a.zip", SHA256: stableSHA, LicenseHint: "gpl"},
			code:   apperrors.E3008,
		},
	}
	for _, testCase := range cases {
		_, err := (&provider.Resolver{}).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
			Provider: "bad", Channel: "daily", Target: "win64", Variant: "gpl",
		}, []provider.Custom{testCase.custom})
		typed, ok := apperrors.As(err)
		require.True(t, ok, testCase.name)
		require.Equal(t, testCase.code, typed.Code, testCase.name)
	}
}

func TestResolveCustomRejectsStableChannel(t *testing.T) {
	custom := []provider.Custom{{
		ID:          "my-ffmpeg",
		URL:         "https://example.invalid/ffmpeg.zip",
		SHA256:      stableSHA,
		LicenseHint: "gpl",
	}}
	_, err := (&provider.Resolver{}).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "my-ffmpeg", Channel: "stable", Target: "win64", Variant: "gpl",
	}, custom)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3004, typed.Code)
}

func TestResolveRejectsInsecureAssetURL(t *testing.T) {
	payload := releasePayload(1, "latest", "2026-11-11T13:00:32Z", map[string]any{
		"name":                 "ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"browser_download_url": "http://example.invalid/ffmpeg.zip",
		"digest":               "sha256:" + stableSHA,
		"size":                 1,
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(payload)
	}))
	t.Cleanup(server.Close)

	_, err := resolver(t, server, nil).Resolve(context.Background(), provider.NewRegistry(), provider.Request{
		Provider: "btbn", Channel: "stable", Target: "win64", Variant: "gpl",
	}, nil)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3007, typed.Code)
}
