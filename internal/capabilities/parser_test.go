package capabilities

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(data)
}

func TestParseVersion(t *testing.T) {
	raw := "ffmpeg version n9.0.2\nconfiguration: --enable-gpl --enable-version3\n"
	info := parseVersion(raw)
	require.Equal(t, "n9.0.2", info.Version)
	require.Equal(t, "--enable-gpl --enable-version3", info.Configuration)
	require.Equal(t, raw, info.Raw)
}

func TestParseLicense(t *testing.T) {
	info := parseLicense("", "GPL version 3 or later\nThis program is free software.\n")
	require.Equal(t, "GPL version 3 or later", info.Summary)
	require.Contains(t, info.Raw, "free software")
}

func TestParseLicenseRealOutput(t *testing.T) {
	gpl := parseLicense(readTestdata(t, "license_gpl.txt"), "")
	require.Contains(t, gpl.Summary, "free software")

	nonfree := parseLicense(readTestdata(t, "license_nonfree.txt"), "")
	require.Contains(t, nonfree.Summary, "nonfree parts compiled in")

	banner := "ffmpeg version N-128713-g0f89d936b3 Copyright (c) 2000-2026\n" +
		"  built with gcc 16.2.0\n" +
		"  configuration: --enable-gpl\n" +
		readTestdata(t, "license_gpl.txt")
	leaked := parseLicense(banner, "")
	require.Contains(t, leaked.Summary, "free software")
}

func TestHelpTopicMissing(t *testing.T) {
	require.True(t, helpTopicMissing(readTestdata(t, "help_not_found.txt"), ""))
	require.True(t, helpTopicMissing("", ""))
	require.True(t, helpTopicMissing("Unknown filter 'nosuch'.\n", ""))
	require.True(t, helpTopicMissing("Unknown format 'nosuch'.\n", ""))
	require.True(t, helpTopicMissing("", "Codec 'nosuch' is not recognized by FFmpeg.\n"))
	require.True(t, helpTopicMissing("", "Unknown encoder 'nosuch'.\n"))
	require.False(t, helpTopicMissing("Decoder h264 [H.264 / AVC / MPEG-4 AVC]:\n    General capabilities: dr1 delay threads\n", "ffmpeg version 9\n"))
	require.False(t, helpTopicMissing("Help text\n", ""))
}

func TestParseFlagList(t *testing.T) {
	raw := `Decoders:
 V..... = Video
 ------
 V....D h264 H.264 / AVC
 A....D aac AAC (Advanced Audio Coding)
`
	entries, unknown := parseFlagList(raw)
	require.Zero(t, unknown)
	require.Len(t, entries, 2)
	require.Equal(t, "aac", entries[0].Name)
	require.Equal(t, "h264", entries[1].Name)
	require.Equal(t, "H.264 / AVC", entries[1].Description)
	require.Equal(t, "V....D", entries[1].Flags)
}

func TestParseSimpleList(t *testing.T) {
	raw := "name depth\n u8 8\n s16 16\n"
	entries, unknown := parseSimpleList(raw)
	require.Zero(t, unknown)
	require.Len(t, entries, 2)
	require.Equal(t, "s16", entries[0].Name)
	require.Equal(t, "16", entries[0].Description)
}

func TestParsePixelFormatsSkipsHeader(t *testing.T) {
	entries, unknown := parseList("pixel_formats", readTestdata(t, "pix_fmts.txt"))
	require.Zero(t, unknown)
	require.Len(t, entries, 4)
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
		require.NotEqual(t, "FLAGS", entry.Flags)
		if entry.Name == "yuv420p" {
			require.Equal(t, "3 12 8-8-8", entry.Description)
		}
	}
	require.False(t, names["NAME"])
	require.True(t, names["yuv420p"])
	require.True(t, names["cuda"])
}

func TestParseChannelLayoutsOnlyStandardSection(t *testing.T) {
	entries, unknown := parseList("channel_layouts", readTestdata(t, "layouts.txt"))
	require.Zero(t, unknown)
	require.Len(t, entries, 4)
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name] = true
	}
	require.True(t, names["mono"])
	require.True(t, names["stereo"])
	require.True(t, names["5.1"])
	require.False(t, names["FL"])
	require.False(t, names["BR"])
}

func TestValidateHelpTopic(t *testing.T) {
	for _, topic := range []string{
		"full",
		"decoder=h264",
		"encoder=libx264",
		"filter=scale",
		"muxer=mp4",
		"demuxer=mov,mp4",
	} {
		require.NoError(t, ValidateHelpTopic(topic), topic)
	}
	for _, topic := range []string{
		"",
		"bogus",
		"decoder=",
		"decoder=h264&bad",
		"decoder=../evil",
	} {
		err := ValidateHelpTopic(topic)
		typed, ok := apperrors.As(err)
		require.True(t, ok)
		require.Equal(t, apperrors.E2005, typed.Code)
	}
}
