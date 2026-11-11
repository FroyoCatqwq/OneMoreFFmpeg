package capabilities

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

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
