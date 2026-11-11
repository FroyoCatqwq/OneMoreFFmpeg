package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	appcli "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cli"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/testutil"
)

func testOptions(args ...string) (appcli.RunOptions, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return appcli.RunOptions{
		Args:   append([]string{"one-more-ffmpeg"}, args...),
		Stdout: stdout,
		Stderr: stderr,
		Version: buildinfo.Info{
			Version:   "0.1.0-test",
			Commit:    "testcommit",
			BuildTime: "2026-11-10T00:00:00Z",
			GoVersion: "go1.27.3",
			OS:        "windows",
			Arch:      "amd64",
		},
	}, stdout, stderr
}

func TestVersionJSON(t *testing.T) {
	opts, stdout, _ := testOptions("version", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)

	var result model.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, model.ResultSchema, result.Schema)
	require.True(t, result.OK)
	require.Equal(t, "version", result.Command)
	data, ok := result.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.VersionSchema, data["schema"])
	require.Equal(t, "0.1.0-test", data["version"])
}

func TestDetectJSONNoSources(t *testing.T) {
	opts, stdout, _ := testOptions("detect", "--json", "--no-path", "--no-env", "--no-common", "--no-validate")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)

	var result model.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "detect", result.Command)
	data, ok := result.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.DetectSchema, data["schema"])
	require.NotContains(t, data, "placeholder")
	require.Empty(t, data["installations"])
	require.Empty(t, data["errors"])
}

func TestJSONFlagBeforeCommand(t *testing.T) {
	opts, stdout, _ := testOptions("--json", "version")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)

	var result model.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.True(t, result.OK)
	require.Equal(t, "version", result.Command)
}

func TestJSONEqualsOneOnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.toml")
	opts, stdout, _ := testOptions("--json=1", "--config", missing, "version")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.False(t, obj.OK)
	require.Equal(t, "OMF-E7007", obj.Code)
}

func TestJSONFalseKeepsHumanOutput(t *testing.T) {
	opts, stdout, _ := testOptions("--json=false", "version")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)
	require.NotContains(t, stdout.String(), `"schema"`)
	require.Contains(t, stdout.String(), "OneMoreFFmpeg")
}

func TestUsageErrorJSON(t *testing.T) {
	opts, stdout, _ := testOptions("version", "--bogus", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)
	require.NotContains(t, stdout.String(), "Incorrect Usage")

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.False(t, obj.OK)
	require.Equal(t, "OMF-E7001", obj.Code)
}

func TestRootsRequireScan(t *testing.T) {
	opts, stdout, _ := testOptions("detect", "--roots", "/tmp", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.Equal(t, "OMF-E7001", obj.Code)
}

func TestValidateRequiresInput(t *testing.T) {
	opts, stdout, _ := testOptions("validate", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.Equal(t, "OMF-E7001", obj.Code)
}

func TestHelp(t *testing.T) {
	opts, stdout, _ := testOptions("--help")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "one-more-ffmpeg")
}

func TestMissingConfigJSONError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.toml")
	opts, stdout, _ := testOptions("--config", missing, "version", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.Equal(t, model.ErrorSchema, obj.Schema)
	require.False(t, obj.OK)
	require.Equal(t, "OMF-E7007", obj.Code)
}

func TestCapabilitiesRequiresInput(t *testing.T) {
	opts, stdout, _ := testOptions("capabilities", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.Equal(t, "OMF-E7001", obj.Code)
}

func TestCapabilitiesNoValidateRejected(t *testing.T) {
	opts, stdout, _ := testOptions("capabilities", "--dir", t.TempDir(), "--no-validate", "--json")
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 2, code)

	var obj model.ErrorObject
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &obj))
	require.Equal(t, "OMF-E7001", obj.Code)
}

func TestCapabilitiesHappyPath(t *testing.T) {
	fake := testutil.NewFakeFFmpeg(t)
	setCapabilityEnv(t, fake)

	opts, stdout, _ := testOptions(
		"capabilities",
		"--dir", fake.Dir,
		"--help-topic", "full",
		"--no-cache",
		"--no-path", "--no-env", "--no-common",
		"--json",
	)
	code := appcli.Run(context.Background(), opts)
	require.Equal(t, 0, code)

	var result model.Result
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Equal(t, "capabilities", result.Command)
	data, ok := result.Data.(map[string]any)
	require.True(t, ok)
	require.Equal(t, model.CapabilitiesSchema, data["schema"])
	commands, ok := data["commands"].(map[string]any)
	require.True(t, ok)
	require.Len(t, commands, 13)
}

func setCapabilityEnv(t *testing.T, fake *testutil.FakeFFmpeg) {
	t.Helper()
	for _, pair := range fake.CapabilityEnv() {
		key, value, ok := strings.Cut(pair, "=")
		if !ok {
			continue
		}
		t.Setenv(key, value)
	}
}
