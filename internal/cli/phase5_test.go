package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestInstallDryRunJSON(t *testing.T) {
	phase4Home(t)
	release := model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "btbn",
		Repository:      "FFmpeg-Builds/FFmpeg-Builds",
		Channel:         "daily",
		ReleaseTag:      "autobuild-2026-11-12-12-00",
		ReleaseID:       "1",
		PublishedAt:     "2026-11-12T12:00:00Z",
		AssetName:       "ffmpeg.zip",
		AssetURL:        "https://example.invalid/ffmpeg.zip",
		AssetSHA256:     strings.Repeat("a", 64),
		Variant:         "gpl",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "gpl",
		Redistributable: true,
	}
	raw, err := json.Marshal(release)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "release.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	code, object, out := runJSON(t, "install", "--release", path, "--json")
	require.Equal(t, 0, code, out)
	data := object["data"].(map[string]any)
	require.Equal(t, model.InstallSchema, data["schema"])
	require.Equal(t, true, data["dry_run"])
	require.Equal(t, "missing", data["current_pointer"])
	require.Contains(t, data["root_dir"], filepath.Join("apps", "btbn"))
	require.NotEmpty(t, data["installation_id"])
}

func TestInstallRejectsNonfreeWithoutConsent(t *testing.T) {
	phase4Home(t)
	release := model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        "bnqdzj-nonfree",
		Repository:      "BnqDzj/FFmpeg-Builds-nonfree",
		Channel:         "daily",
		ReleaseTag:      "autobuild-2026-11-12-12-00",
		ReleaseID:       "1",
		PublishedAt:     "2026-11-12T12:00:00Z",
		AssetName:       "ffmpeg.zip",
		AssetURL:        "https://example.invalid/ffmpeg.zip",
		AssetSHA256:     strings.Repeat("b", 64),
		Variant:         "nonfree",
		Linkage:         "static",
		Target:          "win64",
		LicenseHint:     "nonfree",
		Redistributable: false,
	}
	raw, err := json.Marshal(release)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "release.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))

	code, object, _ := runJSON(t, "install", "--release", path, "--json")
	require.Equal(t, 2, code)
	require.Equal(t, "OMF-E7001", object["code"])
}
