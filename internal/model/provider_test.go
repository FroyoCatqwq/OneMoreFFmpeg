package model_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestProviderListDataNilSlicesMarshalAsArrays(t *testing.T) {
	raw, err := json.Marshal(model.ProviderListData{Schema: model.ProviderListSchema})
	require.NoError(t, err)
	require.Contains(t, string(raw), `"providers":[]`)

	raw, err = json.Marshal(model.ProviderListData{
		Schema:    model.ProviderListSchema,
		Providers: []model.ProviderSummary{{ID: "btbn"}},
	})
	require.NoError(t, err)
	text := string(raw)
	require.Contains(t, text, `"channels":[]`)
	require.Contains(t, text, `"variants":[]`)
	require.Contains(t, text, `"linkages":[]`)
	require.Contains(t, text, `"redistributable":false`)
}

func TestReleaseMarshalsContractFields(t *testing.T) {
	release := model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "btbn",
		Repository:      "FFmpeg-Builds/FFmpeg-Builds",
		Channel:         "daily",
		ReleaseTag:      "autobuild-2026-11-11-12-39",
		PublishedAt:     "2026-11-11T12:39:00Z",
		AssetName:       "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip",
		AssetURL:        "https://example.invalid/asset.zip",
		AssetSHA256:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Variant:         "gpl",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "gpl",
		Redistributable: true,
	}
	raw, err := json.Marshal(release)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, field := range []string{
		"schema", "provider", "repository", "channel", "release_tag", "published_at",
		"asset_name", "asset_url", "asset_sha256", "variant", "linkage", "target",
		"license_hint", "redistributable",
	} {
		require.Contains(t, decoded, field)
	}
	require.NotContains(t, decoded, "checksums_url")
	require.NotContains(t, decoded, "checksums_sha256")
}

func TestArtifactMarshalsContractFields(t *testing.T) {
	raw, err := json.Marshal(model.Artifact{
		Schema:       model.ArtifactSchema,
		ArtifactID:   "art-0123456789abcdef",
		Provider:     "btbn",
		ReleaseTag:   "autobuild-2026-11-11-12-39",
		AssetName:    "ffmpeg.zip",
		AssetSHA256:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		LocalPath:    "C:/data/downloads/ffmpeg.zip",
		Verified:     true,
		DownloadedAt: "2026-11-11T00:00:00Z",
	})
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	for _, field := range []string{
		"schema", "artifact_id", "provider", "release_tag", "asset_name",
		"asset_sha256", "local_path", "verified", "downloaded_at",
	} {
		require.Contains(t, decoded, field)
	}
}
