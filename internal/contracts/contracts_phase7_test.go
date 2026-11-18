package contracts_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func TestValidateCommandDataRejectsMismatch(t *testing.T) {
	data := map[string]any{"schema": model.DoctorSchema}
	require.NoError(t, contracts.ValidateCommandData("doctor", data))
	err := contracts.ValidateCommandData("version", data)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
	require.Equal(t, "version", typed.Details["command"])
}

func TestValidateCommandDataRejectsUnknownCommand(t *testing.T) {
	err := contracts.ValidateCommandData("not-a-command", map[string]any{"schema": model.VersionSchema})
	require.Error(t, err)
}

func TestCommandDataSchemasCoverAllFrozenCommands(t *testing.T) {
	require.Len(t, contracts.CommandDataSchemas, 19)
	for _, schema := range []string{model.CapabilitiesSchema, model.CapabilityListSchema} {
		require.Contains(t, contracts.CommandDataSchemas["capabilities"], schema)
	}
}

func TestErrorObjectFieldsAreValid(t *testing.T) {
	require.NoError(t, contracts.ErrorObjectFieldsAreValid("OMF-E7001", "bad usage", map[string]any{}, ""))
	require.Error(t, contracts.ErrorObjectFieldsAreValid("OMF-X9999", "x", map[string]any{}, ""))
	require.Error(t, contracts.ErrorObjectFieldsAreValid("OMF-E7001", "", map[string]any{}, ""))
	require.Error(t, contracts.ErrorObjectFieldsAreValid("OMF-E7001", "ok", nil, ""))
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'a'
	}
	require.Error(t, contracts.ErrorObjectFieldsAreValid("OMF-E7001", "ok", map[string]any{}, string(long)))
	require.Error(t, contracts.ErrorObjectFieldsAreValid("OMF-E7001", "ok", map[string]any{}, "token ghp_abcdefghijklmnopqrstuvwxyz"))
}
