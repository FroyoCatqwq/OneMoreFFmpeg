package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func TestRegistryIsEmptyByDefault(t *testing.T) {
	// Built-in providers were removed; only user-supplied custom providers are
	// supported, so a fresh registry has no entries.
	require.Empty(t, provider.NewRegistry().List())
}

func TestRegistryGetUnknownProvider(t *testing.T) {
	_, err := provider.NewRegistry().Get("nope")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3001, typed.Code)
}
