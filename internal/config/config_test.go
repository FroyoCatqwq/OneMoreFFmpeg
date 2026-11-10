package config_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func TestParseValidConfig(t *testing.T) {
	cfg, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"

[log]
level = "debug"
json = true
color = "never"
`))
	require.NoError(t, err)
	require.Equal(t, config.SchemaVersion, cfg.Schema)
	require.Equal(t, "debug", cfg.Log.Level)
	require.True(t, cfg.Log.JSON)
	require.Equal(t, "never", cfg.Log.Color)
}

func TestRejectUnknownKey(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"
unknown = true
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7004, typed.Code)
}

func TestRejectSchemaMismatch(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`schema = "wrong/v1"`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestRejectMissingSchema(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
[log]
level = "info"
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestLoadRejectsDirectory(t *testing.T) {
	_, err := config.Load(t.TempDir())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7007, typed.Code)
}

func TestRejectTypeMismatch(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"

[log]
level = 123
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}

func TestExampleConfigParses(t *testing.T) {
	_, err := config.Load("../../config.example.toml")
	require.NoError(t, err)
}

func TestRejectInvalidLevel(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"

[log]
level = "verbose"
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7005, typed.Code)
}
