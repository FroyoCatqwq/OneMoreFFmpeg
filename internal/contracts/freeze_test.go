package contracts

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// Phase 6 contract freeze. These tests pin the external CLI contract: the
// command set, the data-schema set (all major version 1) and the exit-code
// mapping. Any change here is a deliberate contract change and must bump the
// schema major version / follow docs/VERSIONING.md, not slip in silently.

var frozenCommands = []string{
	"capabilities", "detect", "doctor", "download", "env.rollback", "env.set",
	"env.status", "env.unset", "install", "journal.list", "journal.rollback",
	"provider.add", "provider.list", "release.list", "resolve", "uninstall",
	"validate", "version",
}

var frozenDataSchemas = []string{
	"onemoreffmpeg.artifact/v1",
	"onemoreffmpeg.capabilities/v1",
	"onemoreffmpeg.capability-list/v1",
	"onemoreffmpeg.detect/v1",
	"onemoreffmpeg.doctor/v1",
	"onemoreffmpeg.env-set/v1",
	"onemoreffmpeg.env-status/v1",
	"onemoreffmpeg.install/v1",
	"onemoreffmpeg.journal-action/v1",
	"onemoreffmpeg.journal-list/v1",
	"onemoreffmpeg.provider-add/v1",
	"onemoreffmpeg.provider-list/v1",
	"onemoreffmpeg.release-catalog/v1",
	"onemoreffmpeg.release/v1",
	"onemoreffmpeg.uninstall/v1",
	"onemoreffmpeg.validate/v1",
	"onemoreffmpeg.version/v1",
}

func TestFrozenCommandSet(t *testing.T) {
	actual := make([]string, 0, len(knownCommands))
	for command := range knownCommands {
		actual = append(actual, command)
	}
	sort.Strings(actual)
	require.Equal(t, frozenCommands, actual, "the CLI command set is frozen; change it only via a contract revision")
}

func TestFrozenSchemaSet(t *testing.T) {
	actual := make([]string, 0, len(model.KnownDataSchemas))
	for schema := range model.KnownDataSchemas {
		actual = append(actual, schema)
	}
	sort.Strings(actual)
	require.Equal(t, frozenDataSchemas, actual, "the data-schema set is frozen")
	for _, schema := range actual {
		require.True(t, strings.HasSuffix(schema, "/v1"), "all schemas must stay major version 1: %s", schema)
	}
	require.Equal(t, "onemoreffmpeg.result/v1", model.ResultSchema)
	require.Equal(t, "onemoreffmpeg.error/v1", model.ErrorSchema)
}

func TestSchemaMajorVersionBumpIsRejected(t *testing.T) {
	err := ValidateData(map[string]any{"schema": "onemoreffmpeg.version/v2"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestUnknownCommandIsRejected(t *testing.T) {
	err := ValidateCommand("not-a-command")
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestExitCodeContract(t *testing.T) {
	exit2 := []apperrors.Code{
		apperrors.E2005, apperrors.E3001, apperrors.E3004, apperrors.E3008, apperrors.E3009,
		apperrors.E6003, apperrors.E6004, apperrors.E7001, apperrors.E7002, apperrors.E7003,
		apperrors.E7004, apperrors.E7005, apperrors.E7007,
	}
	for _, code := range exit2 {
		require.Equal(t, 2, apperrors.ExitCodeFor(code), string(code))
	}
	exit130 := []apperrors.Code{apperrors.E1011, apperrors.E4006, apperrors.E9004}
	for _, code := range exit130 {
		require.Equal(t, 130, apperrors.ExitCodeFor(code), string(code))
	}
	exit1 := []apperrors.Code{
		apperrors.E1001, apperrors.E2001, apperrors.E3002, apperrors.E4001, apperrors.E4002,
		apperrors.E5004, apperrors.E5005, apperrors.E5009, apperrors.E6005, apperrors.E6006, apperrors.E6009,
		apperrors.E7006, apperrors.E9001,
	}
	for _, code := range exit1 {
		require.Equal(t, 1, apperrors.ExitCodeFor(code), string(code))
	}
	for _, code := range []apperrors.Code{apperrors.W1001, apperrors.W1002, apperrors.W1003} {
		require.Equal(t, 0, apperrors.ExitCodeFor(code), string(code))
	}
}