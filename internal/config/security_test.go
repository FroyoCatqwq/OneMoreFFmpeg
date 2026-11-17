package config_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
)

func TestParseRejectsUserinfoInGitHubAPI(t *testing.T) {
	for _, url := range []string{
		"https://api.github.com@evil.example",
		"https://user:pass@api.github.com",
	} {
		cfg := `schema = "onemoreffmpeg.config/v1"
[network]
github_api = "` + url + `"`
		_, err := config.Parse(strings.NewReader(cfg))
		require.Error(t, err, "must reject userinfo: %s", url)
	}
}

func TestParseRejectsExcessiveTimeoutAndBytes(t *testing.T) {
	base := `schema = "onemoreffmpeg.config/v1"
[network]
timeout = "%s"
[download]
max_bytes = %d`
	for _, tc := range []struct {
		timeout string
		bytes   int64
	}{
		{"1000000h", 536870912},
		{"30s", 1 << 62},
	} {
		cfg := fmt.Sprintf(base, tc.timeout, tc.bytes)
		_, err := config.Parse(strings.NewReader(cfg))
		require.Error(t, err, "must reject timeout=%s bytes=%d", tc.timeout, tc.bytes)
	}
}

func TestParseRejectsOversizedConfig(t *testing.T) {
	huge := `schema = "onemoreffmpeg.config/v1"
[log]
level = "` + strings.Repeat("a", (1<<20)+1) + `"`
	_, err := config.Parse(strings.NewReader(huge))
	require.Error(t, err, "oversized config must be refused")
}
