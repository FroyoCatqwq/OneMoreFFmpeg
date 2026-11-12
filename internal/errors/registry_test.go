package errors_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func TestRegistryCompleteness(t *testing.T) {
	codes := apperrors.KnownCodes()
	require.GreaterOrEqual(t, len(codes), 67)
	for code, entry := range codes {
		require.NotEmpty(t, entry.Name, code)
		require.NotEmpty(t, entry.Description, code)
	}
	require.True(t, apperrors.IsKnown(string(apperrors.E1001)))
	require.True(t, apperrors.IsKnown(string(apperrors.E1012)))
	require.False(t, apperrors.IsKnown("OMF-E9999"))
}

func TestExitCodeMapping(t *testing.T) {
	require.Equal(t, 2, apperrors.ExitCodeFor(apperrors.E7001))
	require.Equal(t, 2, apperrors.ExitCodeFor(apperrors.E7007))
	require.Equal(t, 1, apperrors.ExitCodeFor(apperrors.E7006))
	require.Equal(t, 130, apperrors.ExitCodeFor(apperrors.E1011))
	require.Equal(t, 0, apperrors.ExitCodeFor(apperrors.W1001))
}

func TestFromNil(t *testing.T) {
	typed := apperrors.From(nil)
	require.Equal(t, apperrors.E9003, typed.Code)
}

func TestErrorObjectDefaults(t *testing.T) {
	err := apperrors.New(apperrors.E7005, "invalid value").
		WithDetails(map[string]any{"field": "log.level"}).
		WithCause("expected enum")
	require.Equal(t, "OMF-E7005: invalid value", err.Error())
	require.Equal(t, 2, err.ExitCode())
	require.Equal(t, "invalid value", err.Message)
	require.Equal(t, "expected enum", err.Cause)
}
