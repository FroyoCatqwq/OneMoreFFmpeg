package official

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRulesPartialOverride(t *testing.T) {
	// Regression: overriding only APIBase must not discard the other defaults.
	r := &Resolver{Rules: Rules{APIBase: "https://ghe.example.com"}}
	rules := r.rules()
	require.Equal(t, "https://ghe.example.com", rules.APIBase)
	require.Equal(t, DefaultOwner, rules.Owner)
	require.Equal(t, DefaultRepo, rules.Repo)
	require.Equal(t, DefaultTag, rules.Tag)
	require.Equal(t, "master", rules.SnapshotLine)
	require.NotEmpty(t, rules.AssetTemplate)
	require.NotEmpty(t, rules.StableLinePattern)
}

func TestEnterpriseBaseStripsAPIV3(t *testing.T) {
	// Regression: WithEnterpriseURLs adds /api/v3 itself; configured full REST
	// bases must not get it twice.
	require.Equal(t, "https://ghe.example.com", enterpriseBase("https://ghe.example.com/api/v3"))
	require.Equal(t, "https://ghe.example.com", enterpriseBase("https://ghe.example.com/api/v3/"))
	require.Equal(t, "https://ghe.example.com", enterpriseBase("https://ghe.example.com/"))
	require.Equal(t, "https://api.github.com", enterpriseBase("https://api.github.com"))
}

func TestIsLowerHex(t *testing.T) {
	require.True(t, isLowerHex("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	require.False(t, isLowerHex("0123456789ABCDEF0123456789abcdef0123456789abcdef0123456789abcdef"))
	require.False(t, isLowerHex("short"))
}
