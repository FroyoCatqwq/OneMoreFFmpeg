package provider_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func TestRegistryListsBuiltinProviders(t *testing.T) {
	registry := provider.NewRegistry()
	list := registry.List()
	require.Len(t, list, 2)
	require.Equal(t, provider.IDBtbn, list[0].ID, "registry keeps documented order")
	require.Equal(t, provider.IDBnqdzjNonfree, list[1].ID)

	byID := map[string]int{}
	for index, item := range list {
		byID[item.ID] = index
	}
	btbn := list[byID[provider.IDBtbn]]
	require.Equal(t, "FFmpeg-Builds/FFmpeg-Builds", btbn.Repository)
	require.ElementsMatch(t, []string{"stable", "daily"}, btbn.Channels)
	require.True(t, btbn.Redistributable)

	nonfree := list[byID[provider.IDBnqdzjNonfree]]
	require.Equal(t, "BnqDzj/FFmpeg-Builds-nonfree", nonfree.Repository)
	require.Equal(t, []string{"daily"}, nonfree.Channels)
	require.False(t, nonfree.Redistributable)
}

func TestRegistryGetUnknownProvider(t *testing.T) {
	_, err := provider.NewRegistry().Get("nope")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E3001, typed.Code)
}

func TestProviderSummaryMarshalsEmptySlices(t *testing.T) {
	registry := provider.NewRegistry()
	list := registry.List()
	for _, item := range list {
		require.NotNil(t, item.Channels)
		require.NotNil(t, item.Variants)
		require.NotNil(t, item.Linkages)
	}
}
