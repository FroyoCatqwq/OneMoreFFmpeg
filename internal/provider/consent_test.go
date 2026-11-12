package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
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

func TestCheckNonfreeConsentRequiresFlag(t *testing.T) {
	registry := provider.NewRegistry()
	nonfree, err := registry.Get(provider.IDBnqdzjNonfree)
	require.NoError(t, err)

	_, err = provider.CheckNonfreeConsent(nonfree.Summary(), "nonfree", false)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7001, typed.Code)
	require.Equal(t, "bnqdzj-nonfree", typed.Details["provider"])
}

func TestCheckNonfreeConsentEmitsWarning(t *testing.T) {
	registry := provider.NewRegistry()
	nonfree, err := registry.Get(provider.IDBnqdzjNonfree)
	require.NoError(t, err)
	summary := nonfree.Summary()

	warnings, err := provider.CheckNonfreeConsent(summary, "nonfree-shared", true)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	require.Equal(t, "OMF-W1006", warnings[0].Code)
	require.False(t, summary.Redistributable)
}

func TestCheckNonfreeConsentSkipsFreeVariants(t *testing.T) {
	registry := provider.NewRegistry()
	free, err := registry.Get(provider.IDBtbn)
	require.NoError(t, err)

	warnings, err := provider.CheckNonfreeConsent(free.Summary(), "gpl", false)
	require.NoError(t, err)
	require.Empty(t, warnings)
}
