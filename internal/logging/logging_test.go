package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"error": slog.LevelError,
		"warn":  slog.LevelWarn,
		"info":  slog.LevelInfo,
		"debug": slog.LevelDebug,
	}
	for input, want := range cases {
		got, err := parseLevel(input)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := parseLevel("bogus")
	require.Error(t, err)
}

func TestNewTextLogger(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, config.LogConfig{Level: "debug", Color: "never"})
	require.NoError(t, err)
	logger.Debug("hello", "key", "value")
	require.Contains(t, buf.String(), "hello")
	require.Contains(t, buf.String(), "key=value")
}

func TestNewJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(&buf, config.LogConfig{Level: "debug", JSON: true})
	require.NoError(t, err)
	logger.Debug("hello", "key", "value")
	require.Contains(t, buf.String(), `"msg":"hello"`)
	require.Contains(t, buf.String(), `"key":"value"`)
}
