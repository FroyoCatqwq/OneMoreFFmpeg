package provider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

const testSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func customResolver(fetch provider.ChecksumsFetcher) *provider.Resolver {
	return &provider.Resolver{
		Fetch: fetch,
		Now:   func() time.Time { return time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC) },
	}
}

func customEntry() provider.Custom {
	return provider.Custom{
		ID: "my-ffmpeg", URL: "https://example.invalid/ffmpeg-custom-win64.zip",
		SHA256: testSHA, LicenseHint: "gpl",
	}
}

func TestResolveCustomProvider(t *testing.T) {
	result, err := customResolver(nil).Resolve(context.Background(), nil, provider.Request{
		Provider: "my-ffmpeg", Channel: "daily", Target: "win64", Variant: "gpl",
	}, []provider.Custom{customEntry()})
	require.NoError(t, err)
	require.Equal(t, "my-ffmpeg", result.Provider)
	require.Equal(t, "custom", result.Repository)
	require.Equal(t, "direct-aaaaaaaaaaaa", result.ReleaseTag)
	require.Equal(t, testSHA, result.AssetSHA256)
	require.Equal(t, "ffmpeg-custom-win64.zip", result.AssetName)
	require.Equal(t, "gpl", result.LicenseHint)
}

func TestNormalizeRequestRequiresProvider(t *testing.T) {
	_, err := provider.NormalizeRequest(provider.Request{})
	require.Error(t, err) // provider id is required; no built-in default
}

func TestNormalizeRequestDefaults(t *testing.T) {
	request, err := provider.NormalizeRequest(provider.Request{Provider: "my-ffmpeg"})
	require.NoError(t, err)
	require.Equal(t, "daily", request.Channel)
	require.Equal(t, "gpl", request.Variant)
	require.Equal(t, "static", request.Linkage)
	require.NotEmpty(t, request.Target)
}

func TestNormalizeRequestRejectsPinOptions(t *testing.T) {
	for _, req := range []provider.Request{
		{Provider: "x", ReleaseTag: "latest"},
		{Provider: "x", ReleaseLine: "n9.0"},
		{Provider: "x", Channel: "stable"},
		{Provider: "x", Variant: "banana"},
		{Provider: "x", Linkage: "dynamic"},
	} {
		_, err := provider.NormalizeRequest(req)
		require.Error(t, err, "request %+v must be rejected", req)
	}
}

func TestResolveCustomRejectsUnknownProvider(t *testing.T) {
	_, err := customResolver(nil).Resolve(context.Background(), nil, provider.Request{
		Provider: "missing", Channel: "daily", Target: "win64",
	}, []provider.Custom{customEntry()})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3001, typed.Code)
}

func TestResolveCustomVerifiesChecksumsFile(t *testing.T) {
	fetch := func(context.Context, string) ([]byte, error) {
		return []byte(testSHA + "  ffmpeg-custom-win64.zip\n"), nil
	}
	entry := customEntry()
	entry.ChecksumsURL = "https://example.invalid/checksums.sha256"
	result, err := customResolver(fetch).Resolve(context.Background(), nil, provider.Request{
		Provider: entry.ID, Channel: "daily", Target: "win64",
	}, []provider.Custom{entry})
	require.NoError(t, err)
	require.Len(t, result.ChecksumsSHA256, 64, "checksums file digest is recorded")
}

func TestResolveCustomChecksumsMismatch(t *testing.T) {
	fetch := func(context.Context, string) ([]byte, error) {
		return []byte("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff  ffmpeg-custom-win64.zip\n"), nil
	}
	entry := customEntry()
	entry.ChecksumsURL = "https://example.invalid/checksums.sha256"
	_, err := customResolver(fetch).Resolve(context.Background(), nil, provider.Request{
		Provider: entry.ID, Channel: "daily", Target: "win64",
	}, []provider.Custom{entry})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4002, typed.Code)
}

func TestResolveCustomFetchError(t *testing.T) {
	fetch := func(context.Context, string) ([]byte, error) { return nil, errors.New("network down") }
	entry := customEntry()
	entry.ChecksumsURL = "https://example.invalid/checksums.sha256"
	_, err := customResolver(fetch).Resolve(context.Background(), nil, provider.Request{
		Provider: entry.ID, Channel: "daily", Target: "win64",
	}, []provider.Custom{entry})
	require.Error(t, err)
}
