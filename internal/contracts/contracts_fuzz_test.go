package contracts

import "testing"

// FuzzValidateHelpTopic is intentionally not here (it lives in capabilities).
// These targets cover the URL/asset-name guards that protect release handling.

func FuzzIsHTTPS(f *testing.F) {
	for _, seed := range []string{
		"https://api.github.com", "http://api.github.com", "",
		"HTTPS://example.com", "https://user:pass@example.com", "//example.com",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_ = isHTTPS(value)
	})
}

func FuzzSafeAssetName(f *testing.F) {
	for _, seed := range []string{
		"ffmpeg.zip", "../escape.zip", "a/b.zip", "",
		"C:\\secret.zip", "..", "a\x00b",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, name string) {
		_ = safeAssetName(name)
	})
}

func FuzzIsAbsoluteLike(f *testing.F) {
	for _, seed := range []string{"/tmp/x", "C:\\x", "relative", "", "\\\\server\\share"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_ = isAbsoluteLike(value)
	})
}
