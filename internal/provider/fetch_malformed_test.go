package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func TestFetchURLMalformedNoPanic(t *testing.T) {
	// net/url.Parse returns (nil, err) on these; the function must check err
	// BEFORE dereferencing parsed (used to panic).
	for _, raw := range []string{
		"https://example.com/%zz",
		"http://[::1",
		"https://exa\nmple.com/a",
		"://bad",
	} {
		t.Run(raw, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := fetchURL(context.Background(), raw, 5, 0)
				typed, ok := apperrors.As(err)
				require.True(t, ok)
				require.Equal(t, apperrors.E3008, typed.Code)
			})
		})
	}
}
