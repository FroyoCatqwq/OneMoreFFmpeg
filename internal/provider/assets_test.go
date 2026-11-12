package provider

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func assets(names ...string) []github.Asset {
	out := make([]github.Asset, 0, len(names))
	for _, name := range names {
		out = append(out, github.Asset{Name: name})
	}
	return out
}

func TestSelectAssetStablePrefersHighestLine(t *testing.T) {
	list := assets(
		"checksums.sha256",
		"ffmpeg-master-latest-win64-gpl.zip",
		"ffmpeg-n8.1-latest-win64-gpl-8.1.zip",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"ffmpeg-n9.0-latest-win64-gpl-shared-9.0.zip",
		"ffmpeg-n9.0-latest-linux64-gpl-9.0.tar.xz",
		"ffmpeg-n9.0-latest-winarm64-gpl-9.0.zip",
	)
	asset, line, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", Linkage: "static"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n9.0-latest-win64-gpl-9.0.zip", asset.Name)
	require.Equal(t, "n9.0", line)
}

func TestSelectAssetDailyRequiresMaster(t *testing.T) {
	list := assets(
		"ffmpeg-n8.1-latest-win64-gpl-8.1.zip",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
	)
	_, _, err := selectAsset(list, assetQuery{Channel: "daily", Target: "win64", Variant: "gpl", Linkage: "static"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)
}

func TestSelectAssetPrefersTargetArchiveFormat(t *testing.T) {
	list := assets(
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.tar.xz",
		"ffmpeg-n9.0-latest-linux64-gpl-9.0.zip",
		"ffmpeg-n9.0-latest-linux64-gpl-9.0.tar.xz",
	)
	windows, _, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", Linkage: "static"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n9.0-latest-win64-gpl-9.0.zip", windows.Name)

	linux, _, err := selectAsset(list, assetQuery{Channel: "stable", Target: "linux64", Variant: "gpl", Linkage: "static"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n9.0-latest-linux64-gpl-9.0.tar.xz", linux.Name)
}

func TestSelectAssetRejectsAmbiguousSameLine(t *testing.T) {
	list := assets(
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"ffmpeg-n9.0-latest-win64-gpl-copy.zip",
	)
	_, _, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", Linkage: "static"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)
}

func TestSelectAssetStableHonorsReleaseLine(t *testing.T) {
	list := assets(
		"ffmpeg-n8.1-latest-win64-gpl-8.1.zip",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
	)
	asset, line, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", ReleaseLine: "n8.1"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n8.1-latest-win64-gpl-8.1.zip", asset.Name)
	require.Equal(t, "n8.1", line)

	_, _, err = selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", ReleaseLine: "n7.0"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)
}

func TestSelectAssetDailyPrefersMaster(t *testing.T) {
	list := assets(
		"ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"ffmpeg-N-128790-g9bee1e603c-win64-gpl-shared.zip",
	)
	asset, line, err := selectAsset(list, assetQuery{Channel: "daily", Target: "win64", Variant: "gpl"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip", asset.Name)
	require.Equal(t, "master", line)
}

func TestSelectAssetSharedVariants(t *testing.T) {
	list := assets(
		"ffmpeg-n9.0-latest-win64-gpl-9.0.zip",
		"ffmpeg-n9.0-latest-win64-gpl-shared-9.0.zip",
	)
	asset, _, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl-shared"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n9.0-latest-win64-gpl-shared-9.0.zip", asset.Name)

	asset, _, err = selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl", Linkage: "shared"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-n9.0-latest-win64-gpl-shared-9.0.zip", asset.Name)
}

func TestSelectAssetLinuxNonfree(t *testing.T) {
	list := assets(
		"ffmpeg-N-128756-gd0b9c63c9f-linux64-nonfree.tar.xz",
		"ffmpeg-N-128756-gd0b9c63c9f-linux64-nonfree-shared.tar.xz",
		"ffmpeg-master-latest-win64-nonfree.zip",
	)
	asset, line, err := selectAsset(list, assetQuery{Channel: "daily", Target: "linux64", Variant: "nonfree-shared"})
	require.NoError(t, err)
	require.Equal(t, "ffmpeg-N-128756-gd0b9c63c9f-linux64-nonfree-shared.tar.xz", asset.Name)
	require.Equal(t, "master", line)
}

func TestSelectAssetNoMatch(t *testing.T) {
	list := assets("ffmpeg-n9.0-latest-win64-lgpl-9.0.zip")
	_, _, err := selectAsset(list, assetQuery{Channel: "stable", Target: "win64", Variant: "gpl"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)

	// Stable must not silently fall back to master builds.
	_, _, err = selectAsset(assets("ffmpeg-master-latest-win64-gpl.zip"), assetQuery{Channel: "stable", Target: "win64", Variant: "gpl"})
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3003, typed.Code)
}

func TestParseAssetNameRejectsUnknownShapes(t *testing.T) {
	for _, name := range []string{
		"checksums.sha256",
		"ffmpeg-n9.0-latest-win64-gpl-9.0.7z",
		"ffmpeg-latest-win64.zip",
		"ffprobe-n9.0-latest-win64-gpl-9.0.zip",
	} {
		_, ok := parseAssetName(name)
		require.False(t, ok, name)
	}
}
