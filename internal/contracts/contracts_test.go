package contracts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestValidateExamples(t *testing.T) {
	root := repoRoot(t)
	count, err := contracts.ValidateExamplesDir(filepath.Join(root, "docs", "schemas", "examples"))
	require.NoError(t, err)
	require.Equal(t, 13, count)
}

func TestValidateErrorRegistryDocsBidirectional(t *testing.T) {
	root := repoRoot(t)
	docsPath := filepath.Join(root, "docs", "ERROR_CODES.md")
	require.NoError(t, contracts.ValidateErrorRegistryDocs(docsPath, apperrors.IsKnown))

	codes := make([]string, 0, len(apperrors.KnownCodes()))
	for code := range apperrors.KnownCodes() {
		codes = append(codes, string(code))
	}
	require.NoError(t, contracts.ValidateRegistryCoveredByDocs(docsPath, codes))
}

func TestValidateDataRequiresKnownSchema(t *testing.T) {
	err := contracts.ValidateData(map[string]any{"name": "missing"})
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)

	err = contracts.ValidateData(map[string]any{"schema": "onemoreffmpeg.typo/v1"})
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestValidateExampleRejectsInvalidResult(t *testing.T) {
	cases := []string{
		`{"schema":"onemoreffmpeg.result/v1","ok":false,"command":"version","data":{"schema":"onemoreffmpeg.version/v1"}}`,
		`{"schema":"onemoreffmpeg.result/v1","ok":true,"command":"bogus","data":{"schema":"onemoreffmpeg.version/v1"}}`,
		`{"schema":"onemoreffmpeg.result/v1","ok":true,"command":"version","data":{"schema":"onemoreffmpeg.typo/v1"}}`,
	}
	for _, body := range cases {
		path := filepath.Join(t.TempDir(), "case.json")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		err := contracts.ValidateExample(path)
		typed, ok := apperrors.As(err)
		require.True(t, ok)
		require.Equal(t, apperrors.E7003, typed.Code)
	}
}

func TestValidateExampleRejectsIncompleteCapabilities(t *testing.T) {
	body := `{"schema":"onemoreffmpeg.result/v1","ok":true,"command":"capabilities","data":{"schema":"onemoreffmpeg.capabilities/v1"}}`
	path := filepath.Join(t.TempDir(), "case.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
	err := contracts.ValidateExample(path)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestValidateExampleRejectsInvalidError(t *testing.T) {
	cases := []string{
		`{"schema":"onemoreffmpeg.error/v1","ok":true,"code":"OMF-E7001","message":"bad","details":{}}`,
		`{"schema":"onemoreffmpeg.error/v1","ok":false,"code":"OMF-E9999","message":"bad","details":{}}`,
		`{"schema":"onemoreffmpeg.error/v1","ok":false,"code":"OMF-E7001","message":"bad"}`,
	}
	for _, body := range cases {
		path := filepath.Join(t.TempDir(), "case.json")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
		err := contracts.ValidateExample(path)
		typed, ok := apperrors.As(err)
		require.True(t, ok)
		require.Equal(t, apperrors.E7003, typed.Code)
	}
}
