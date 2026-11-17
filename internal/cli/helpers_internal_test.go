package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
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

func TestHasHelpFlag(t *testing.T) {
	require.True(t, hasHelpFlag([]string{"--json", "--help"}))
	require.True(t, hasHelpFlag([]string{"provider", "-h"}))
	require.False(t, hasHelpFlag([]string{"--help-topic", "--help"}), "flag value must be skipped")
	require.False(t, hasHelpFlag([]string{"--", "--help"}), "args after -- are not flags")
}

func TestHasHelpSubcommand(t *testing.T) {
	require.True(t, hasHelpSubcommand([]string{"one-more-ffmpeg", "--json", "help"}))
	require.True(t, hasHelpSubcommand([]string{"one-more-ffmpeg", "--json", "help", "version"}))
	require.False(t, hasHelpSubcommand([]string{"one-more-ffmpeg", "--dir", "help", "detect"}), "flag value must not count")
	require.False(t, hasHelpSubcommand([]string{"one-more-ffmpeg", "--", "help"}), "after -- is not a command")
	require.False(t, hasHelpSubcommand([]string{"one-more-ffmpeg", "version"}))
}

func TestHasInDetailUsageStopsAtSeparator(t *testing.T) {
	require.False(t, hasInDetailUsage([]string{"--", "--in-detail-usage"}))
	require.True(t, hasInDetailUsage([]string{"version", "--in-detail-usage"}))
}

func TestErrorObjectDegradesOffContractEnvelope(t *testing.T) {
	long := strings.Repeat("x", 600)
	obj := errorObject(apperrors.New(apperrors.E7001, long))
	require.Equal(t, model.ErrorSchema, obj.Schema)
	require.Equal(t, string(apperrors.E9001), obj.Code)
	require.NotContains(t, obj.Message, long)
	require.Equal(t, map[string]any{}, obj.Details)
	require.Empty(t, obj.Cause)
}

func TestTakesValueHasCompleteSet(t *testing.T) {
	for _, name := range []string{"--dir", "--roots", "--max-depth", "--max-files", "--timeout", "--probe-timeout", "--config", "-c"} {
		require.True(t, takesValue[name], name)
	}
}
