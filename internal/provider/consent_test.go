package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func TestIsNonfreeVariant(t *testing.T) {
	for _, variant := range []string{"nonfree", "nonfree-shared", "NonFree", " NONFREE-SHARED "} {
		require.True(t, provider.IsNonfreeVariant(variant), variant)
	}
	for _, variant := range []string{"gpl", "lgpl", "gpl-shared", "lgpl-shared", "", "nonfreeish"} {
		require.False(t, provider.IsNonfreeVariant(variant), variant)
	}
}

func customSummary() model.ProviderSummary {
	return model.ProviderSummary{ID: "my-ffmpeg", Repository: "custom"}
}

func TestCheckNonfreeConsentRequiresFlag(t *testing.T) {
	_, err := provider.CheckNonfreeConsent(customSummary(), "nonfree", false)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7001, typed.Code)
	require.Equal(t, "my-ffmpeg", typed.Details["provider"])
}

func TestCheckNonfreeConsentEmitsWarning(t *testing.T) {
	warnings, err := provider.CheckNonfreeConsent(customSummary(), "nonfree-shared", true)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Equal(t, "OMF-W1006", warnings[0].Code)
}

func TestCheckNonfreeConsentSkipsFreeVariants(t *testing.T) {
	warnings, err := provider.CheckNonfreeConsent(customSummary(), "gpl", false)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
