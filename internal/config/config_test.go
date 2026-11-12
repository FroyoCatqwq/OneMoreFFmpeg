package config_test

import (
	"fmt"
	"path/filepath"
	"runtime"
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

func TestRejectSyntaxError(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"
[log
level = "info"
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7002, typed.Code)
}

func TestRejectDuplicateKey(t *testing.T) {
	_, err := config.Parse(strings.NewReader(`
schema = "onemoreffmpeg.config/v1"
schema = "onemoreffmpeg.config/v1"
`))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7002, typed.Code)
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

func TestDefaultsForNetworkAndDownload(t *testing.T) {
	cfg, err := config.Parse(strings.NewReader(`schema = "onemoreffmpeg.config/v1"`))
	require.NoError(t, err)
	require.Equal(t, "30s", cfg.Network.Timeout)
	require.Equal(t, 3, cfg.Network.MaxAttempts)
	require.Equal(t, "https://api.github.com", cfg.Network.GitHubAPI)
	require.Equal(t, int64(512<<20), cfg.Download.MaxBytes)
	require.Equal(t, 5, cfg.Download.MaxRedirects)
	require.Empty(t, cfg.Download.Dir)
}

func TestParseNetworkAndDownloadSections(t *testing.T) {
	root := "/"
	if runtime.GOOS == "windows" {
		root = `C:\`
	}
	dir := filepath.Join(root, "tmp", "omf-downloads")
	body := fmt.Sprintf(`
schema = "onemoreffmpeg.config/v1"

[network]
timeout = "45s"
max_attempts = 5
github_api = "https://github.example.invalid/api/v3"

[download]
max_bytes = 1048576
max_redirects = 2
dir = '%s'
`, dir)
	cfg, err := config.Parse(strings.NewReader(body))
	require.NoError(t, err)
	require.Equal(t, "45s", cfg.Network.Timeout)
	require.Equal(t, 5, cfg.Network.MaxAttempts)
	require.Equal(t, "https://github.example.invalid/api/v3", cfg.Network.GitHubAPI)
	require.Equal(t, int64(1048576), cfg.Download.MaxBytes)
	require.Equal(t, 2, cfg.Download.MaxRedirects)
	require.Equal(t, dir, cfg.Download.Dir)
}

func TestRejectInvalidNetworkSettings(t *testing.T) {
	cases := []string{
		`
schema = "onemoreffmpeg.config/v1"
[network]
timeout = "soon"
`,
		`
schema = "onemoreffmpeg.config/v1"
[network]
timeout = "0s"
`,
		`
schema = "onemoreffmpeg.config/v1"
[network]
max_attempts = 0
`,
		`
schema = "onemoreffmpeg.config/v1"
[network]
max_attempts = 11
`,
		`
schema = "onemoreffmpeg.config/v1"
[network]
github_api = "http://api.github.com"
`,
		`
schema = "onemoreffmpeg.config/v1"
[network]
github_api = "https://"
`,
	}
	for _, body := range cases {
		_, err := config.Parse(strings.NewReader(body))
		typed, ok := apperrors.As(err)
		require.True(t, ok, body)
		require.Equal(t, apperrors.E7005, typed.Code, body)
	}
}

func TestAllowsLoopbackHTTPGitHubAPI(t *testing.T) {
	for _, api := range []string{"http://localhost:8080", "http://127.0.0.1:8080/api", "http://[::1]:8080"} {
		body := fmt.Sprintf("schema = \"onemoreffmpeg.config/v1\"\n[network]\ngithub_api = %q\n", api)
		_, err := config.Parse(strings.NewReader(body))
		require.NoError(t, err, api)
	}
}

func TestRejectInvalidDownloadSettings(t *testing.T) {
	cases := []string{
		`
schema = "onemoreffmpeg.config/v1"
[download]
max_bytes = 0
`,
		`
schema = "onemoreffmpeg.config/v1"
[download]
max_bytes = -1
`,
		`
schema = "onemoreffmpeg.config/v1"
[download]
max_redirects = -1
`,
		`
schema = "onemoreffmpeg.config/v1"
[download]
max_redirects = 21
`,
		`
schema = "onemoreffmpeg.config/v1"
[download]
dir = "relative/downloads"
`,
		`
schema = "onemoreffmpeg.config/v1"
[download]
max_bytes = "big"
`,
	}
	for _, body := range cases {
		_, err := config.Parse(strings.NewReader(body))
		typed, ok := apperrors.As(err)
		require.True(t, ok, body)
		require.Equal(t, apperrors.E7005, typed.Code, body)
	}
}

func TestLoadOptionalMissingFileUsesDefaults(t *testing.T) {
	cfg, err := config.LoadOptional(filepath.Join(t.TempDir(), "missing.toml"))
	require.NoError(t, err)
	require.Equal(t, "30s", cfg.Network.Timeout)
	require.Equal(t, 3, cfg.Network.MaxAttempts)
}

func TestLoadOptionalDirectoryFails(t *testing.T) {
	_, err := config.LoadOptional(t.TempDir())
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7007, typed.Code)
}

func TestExampleConfigHasUsableDownloadSettings(t *testing.T) {
	cfg, err := config.Load("../../config.example.toml")
	require.NoError(t, err)
	if cfg.Download.Dir != "" {
		require.True(t, filepath.IsAbs(cfg.Download.Dir), "example dir must be absolute")
	}
	require.Positive(t, cfg.Download.MaxBytes)
	require.GreaterOrEqual(t, cfg.Download.MaxRedirects, 0)
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
