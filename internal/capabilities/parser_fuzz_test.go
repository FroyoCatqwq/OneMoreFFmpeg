package capabilities

import "testing"

// FuzzParseVersion feeds arbitrary FFmpeg -version stdout into the parser.
func FuzzParseVersion(f *testing.F) {
	for _, seed := range []string{
		"ffmpeg version 9.9.9 Copyright (c) 2026",
		"ffmpeg version N-128790-gabcdef Copyright",
		"",
		"configuration: --enable-gpl\nffmpeg version 8.1.3",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		result := parseVersion(raw)
		if result.Version == "" && len(raw) > 0 {
			// Parsers are allowed to reject malformed input, but must not panic.
			_ = result
		}
	})
}

// FuzzParseList covers the flag-based and simple list parsers used by the 9
// capability list commands; both must handle arbitrary FFmpeg output safely.
func FuzzParseList(f *testing.F) {
	for _, seed := range []string{
		"Encoders:\n V....D h264 H.264 / AVC\n",
		"Pixel formats:\nFLAGS NAME NB_COMPONENTS\nIO... yuv420p 3 12 8-8-8\n",
		"",
		" garbled \x00 line\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		parseList("encoders", raw)
		parseFlagList(raw)
		parseSimpleList(raw)
		parseChannelLayouts(raw)
	})
}

// FuzzValidateHelpTopic ensures topic parsing never panics on hostile input.
func FuzzValidateHelpTopic(f *testing.F) {
	for _, seed := range []string{"full", "decoder=h264", "encoder=", "bogus", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, topic string) {
		_ = ValidateHelpTopic(topic)
	})
}
