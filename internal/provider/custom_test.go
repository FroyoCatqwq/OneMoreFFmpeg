package provider_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func validCustom() provider.Custom {
	return provider.Custom{
		ID:           "my-ffmpeg",
		URL:          "https://example.invalid/ffmpeg.zip",
		SHA256:       strings.Repeat("a", 64),
		ChecksumsURL: "https://example.invalid/checksums.sha256",
		LicenseHint:  "gpl",
	}
}

func TestCustomFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.toml")

	entries, err := provider.LoadCustomFile(path)
	require.NoError(t, err)
	require.Empty(t, entries)

	entries = append(entries, validCustom())
	require.NoError(t, provider.SaveCustomFile(path, entries))

	loaded, err := provider.LoadCustomFile(path)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Equal(t, "my-ffmpeg", loaded[0].ID)
	require.Equal(t, "gpl", loaded[0].LicenseHint)
	require.Equal(t, strings.Repeat("a", 64), loaded[0].SHA256)
}

func TestCustomFileRejectsUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.toml")
	body := "schema = \"onemoreffmpeg.providers/v1\"\n[[provider]]\nid = \"my-ffmpeg\"\nurl = \"https://example.invalid/a.zip\"\nsha256 = \"" +
		strings.Repeat("a", 64) + "\"\nunexpected = true\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	_, err := provider.LoadCustomFile(path)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7004, typed.Code)
}

func TestAddCustomRejectsReservedAndDuplicateIDs(t *testing.T) {
	for _, id := range []string{"btbn", "bnqdzj-nonfree", "custom", "Bad_ID"} {
		entry := validCustom()
		entry.ID = id
		_, err := provider.AddCustom(nil, entry)
		typed, ok := apperrors.As(err)
		require.True(t, ok, id)
		require.Equal(t, apperrors.E7005, typed.Code, id)
	}

	entries, err := provider.AddCustom(nil, validCustom())
	require.NoError(t, err)
	_, err = provider.AddCustom(entries, validCustom())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}

func TestAddCustomValidatesURLAndHash(t *testing.T) {
	entry := validCustom()
	entry.URL = "http://example.invalid/ffmpeg.zip"
	_, err := provider.AddCustom(nil, entry)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3008, typed.Code)

	entry = validCustom()
	entry.SHA256 = "abc"
	_, err = provider.AddCustom(nil, entry)
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3009, typed.Code)

	entry = validCustom()
	entry.ChecksumsURL = "ftp://example.invalid/checksums"
	_, err = provider.AddCustom(nil, entry)
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3008, typed.Code)

	entry = validCustom()
	entry.LicenseHint = "proprietary"
	_, err = provider.AddCustom(nil, entry)
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}

func TestAddCustomNormalizesEntry(t *testing.T) {
	entry := validCustom()
	entry.ID = "  My-FFmpeg  "
	entry.SHA256 = strings.ToUpper(entry.SHA256)
	entry.LicenseHint = "GPL"
	entry.URL = "  https://example.invalid/ffmpeg.zip  "

	entries, err := provider.AddCustom(nil, entry)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "my-ffmpeg", entries[0].ID)
	require.Equal(t, strings.ToLower(entry.SHA256), entries[0].SHA256)
	require.Equal(t, "gpl", entries[0].LicenseHint)
	require.Equal(t, "https://example.invalid/ffmpeg.zip", entries[0].URL)
}

func TestCustomFileRejectsDuplicateIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.toml")
	sha := strings.Repeat("a", 64)
	body := "schema = \"onemoreffmpeg.providers/v1\"\n" +
		"[[provider]]\nid = \"dup\"\nurl = \"https://example.invalid/a.zip\"\nsha256 = \"" + sha + "\"\n" +
		"[[provider]]\nid = \"dup\"\nurl = \"https://example.invalid/b.zip\"\nsha256 = \"" + sha + "\"\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))

	_, err := provider.LoadCustomFile(path)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}

func TestAddCustomRejectsCredentialAndNamelessURLs(t *testing.T) {
	for _, rawURL := range []string{
		"https://user:pass@example.invalid/a.zip",
		"https://example.invalid/",
	} {
		entry := validCustom()
		entry.URL = rawURL
		_, err := provider.AddCustom(nil, entry)
		typed, ok := apperrors.As(err)
		require.True(t, ok, rawURL)
		require.Equal(t, apperrors.E3008, typed.Code, rawURL)
	}
}

func TestCustomSummary(t *testing.T) {
	summary := provider.CustomSummary(validCustom())
	require.Equal(t, "my-ffmpeg", summary.ID)
	require.Equal(t, "custom", summary.Repository)
	require.Equal(t, []string{"daily"}, summary.Channels)
	require.False(t, summary.Redistributable)
}

func TestFindCustom(t *testing.T) {
	entry, ok := provider.FindCustom([]provider.Custom{validCustom()}, "my-ffmpeg")
	require.True(t, ok)
	require.Equal(t, "my-ffmpeg", entry.ID)

	_, ok = provider.FindCustom([]provider.Custom{validCustom()}, "missing")
	require.False(t, ok)
}
