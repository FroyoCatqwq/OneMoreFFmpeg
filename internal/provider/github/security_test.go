package github_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func TestNewClientRejectsUserinfo(t *testing.T) {
	for _, base := range []string{
		"https://api.github.com@evil.example",
		"https://user:pass@api.github.com",
		"http://user@localhost:8080",
	} {
		_, err := github.NewClient(base, "ghp_secret")
		require.Error(t, err, "must reject base URL with userinfo: %s", base)
		require.NotContains(t, err.Error(), "ghp_secret", "error must not echo token")
	}
}

func TestNewClientErrorDoesNotEchoCredentialsInURL(t *testing.T) {
	// A rejected non-loopback http URL must not echo any embedded credentials.
	_, err := github.NewClient("http://user:pass@remote.example", "token")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "user:pass")
}
