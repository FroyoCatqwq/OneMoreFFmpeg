package contracts_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestKnownDataSchemasReturnsDefensiveCopy(t *testing.T) {
	first := model.KnownDataSchemas()
	delete(first, model.VersionSchema)
	first["forged/v1"] = struct{}{}

	second := model.KnownDataSchemas()
	require.Contains(t, second, model.VersionSchema, "mutating a returned map must not change the registry")
	require.NotContains(t, second, "forged/v1")
	require.True(t, model.IsKnownDataSchema(model.VersionSchema))
}

func TestCommandDataSchemasReturnsDeepCopy(t *testing.T) {
	first := contracts.CommandDataSchemas()
	delete(first, "version")
	first["forged"] = []string{"onemoreffmpeg.forged/v1"}
	// Deep-copy check: mutating a returned []string must not affect the registry.
	first["capabilities"][0] = "onemoreffmpeg.forged/v1"

	second := contracts.CommandDataSchemas()
	require.Contains(t, second, "version")
	require.NotContains(t, second, "forged")
	require.Contains(t, second["capabilities"], model.CapabilitiesSchema,
		"mutating a returned slice must not change the registry")
	require.NotContains(t, second["capabilities"], "onemoreffmpeg.forged/v1")
}
