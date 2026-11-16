package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutsideAppsOnlyRejectsTrueParent(t *testing.T) {
	require.False(t, outsideApps("..stable"))
	require.False(t, outsideApps(filepath.Join("..cache", "bin")))
	require.False(t, outsideApps(filepath.Join("btbn", "tag", "abc")))
	require.True(t, outsideApps(".."))
	require.True(t, outsideApps(filepath.Join("..", "escape")))
}

func TestSanitizePanicDropsValue(t *testing.T) {
	secret := "ghp_supersecrettoken"
	got := sanitizePanic(secret)
	require.False(t, strings.Contains(got, secret))
	require.Contains(t, got, "string")
}

func TestWarnToStderrUsesInjectedWriter(t *testing.T) {
	var buf strings.Builder
	ctx := withStderr(context.Background(), &buf)
	warnToStderr(ctx)("degraded")
	require.Contains(t, buf.String(), "warning: degraded")
}

func TestTakesValueHasCompleteSet(t *testing.T) {
	for _, name := range []string{"--dir", "--roots", "--max-depth", "--max-files", "--timeout", "--probe-timeout", "--config", "-c"} {
		require.True(t, takesValue[name], name)
	}
}
