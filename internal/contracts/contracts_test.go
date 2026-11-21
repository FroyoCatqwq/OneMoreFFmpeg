package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/contracts"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestValidateExamples(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "docs", "reference", "schemas", "examples")
	_, err := contracts.ValidateExamplesDir(dir)
	require.NoError(t, err)

	// A schema without a golden example has no field-level guard at all, which
	// is how uninstall / journal-* / capability-list / release-catalog went
	// unchecked until now. Require one example per frozen data schema instead of
	// pinning a count that silently drifts.
	knownSchemas := model.KnownDataSchemas()
	covered := make(map[string]bool, len(knownSchemas))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, readErr)
		var envelope struct {
			Data struct {
				Schema string `json:"schema"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(payload, &envelope), entry.Name())
		if envelope.Data.Schema != "" {
			covered[envelope.Data.Schema] = true
		}
	}
	var missing []string
	for schema := range knownSchemas {
		if !covered[schema] {
			missing = append(missing, schema)
		}
	}
	require.Empty(t, missing, "data schemas without an example: %v", missing)
}

// The command->schema registry accessor lives in production code (contracts.CommandDataSchemas()) so the
// runtime validator, golden examples and freeze tests cannot drift apart.
func schemaAllowed(schemas []string, candidate string) bool {
	for _, schema := range schemas {
		if schema == candidate {
			return true
		}
	}
	return false
}

func TestGoldenExamplesCoverEveryFrozenCommand(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "docs", "reference", "schemas", "examples")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	registry := contracts.CommandDataSchemas()
	seen := make(map[string]bool, len(registry))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, readErr)
		var envelope struct {
			Schema  string `json:"schema"`
			Command string `json:"command"`
			Data    struct {
				Schema string `json:"schema"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(payload, &envelope), entry.Name())
		if envelope.Schema != model.ResultSchema {
			continue
		}
		allowed, known := registry[envelope.Command]
		require.True(t, known, "unknown command in %s: %s", entry.Name(), envelope.Command)
		require.True(t, schemaAllowed(allowed, envelope.Data.Schema), "command/schema mismatch in %s: %s", entry.Name(), envelope.Data.Schema)
		seen[envelope.Command] = true
	}
	for command := range registry {
		require.True(t, seen[command], "frozen command has no golden example: %s", command)
	}
}

func TestValidateErrorRegistryDocsBidirectional(t *testing.T) {
	root := repoRoot(t)
	docsPath := filepath.Join(root, "docs", "reference", "ERROR_CODES.md")
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

func TestValidateCommand(t *testing.T) {
	for _, name := range []string{"journal.list", "journal.rollback", "uninstall"} {
		require.NoError(t, contracts.ValidateCommand(name), name)
	}
	err := contracts.ValidateCommand("not-a-command")
	typed, ok := apperrors.As(err)
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

func writeResultExample(t *testing.T, command string, data map[string]any) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"schema":  model.ResultSchema,
		"ok":      true,
		"command": command,
		"data":    data,
	})
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "case.json")
	require.NoError(t, os.WriteFile(path, body, 0o644))
	return path
}

func validRelease() map[string]any {
	return map[string]any{
		"schema":          model.ReleaseSchema,
		"provider":        "btbn",
		"repository":      "FFmpeg-Builds/FFmpeg-Builds",
		"channel":         "daily",
		"release_tag":     "autobuild-2026-11-11-12-39",
		"published_at":    "2026-11-11T12:39:00Z",
		"asset_name":      "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip",
		"asset_url":       "https://example.com/asset.zip",
		"asset_sha256":    strings.Repeat("a", 64),
		"variant":         "gpl",
		"linkage":         "static",
		"target":          "win64",
		"license_hint":    "gpl",
		"redistributable": true,
	}
}

func validArtifact() map[string]any {
	return map[string]any{
		"schema":        model.ArtifactSchema,
		"artifact_id":   "art-0123456789abcdef",
		"provider":      "btbn",
		"release_tag":   "autobuild-2026-11-11-12-39",
		"asset_name":    "ffmpeg.zip",
		"asset_sha256":  strings.Repeat("b", 64),
		"local_path":    "C:/data/downloads/ffmpeg.zip",
		"verified":      true,
		"downloaded_at": "2026-11-11T00:00:00Z",
	}
}

func TestValidateReleaseAndArtifactExamples(t *testing.T) {
	require.NoError(t, contracts.ValidateExample(writeResultExample(t, "resolve", validRelease())))
	require.NoError(t, contracts.ValidateExample(writeResultExample(t, "download", validArtifact())))
}

func TestValidateReleaseLatestWithExplicitPin(t *testing.T) {
	data := validRelease()
	data["release_tag"] = "latest"
	data["release_id"] = "250311488"
	data["pinned_at"] = "2026-11-11T12:39:00Z"
	require.NoError(t, contracts.ValidateExample(writeResultExample(t, "resolve", data)))

	data = validRelease()
	data["release_tag"] = "latest"
	data["release_id"] = "250311488"
	data["pinned_at"] = "not-a-time"
	err := contracts.ValidateExample(writeResultExample(t, "resolve", data))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)
}

func TestValidateExampleRejectsInvalidRelease(t *testing.T) {
	cases := []struct {
		mutate func(map[string]any)
		code   apperrors.Code
	}{
		{func(data map[string]any) { data["release_tag"] = "latest" }, apperrors.E3010},
		{func(data map[string]any) { data["asset_sha256"] = "short" }, apperrors.E7003},
		{func(data map[string]any) { data["asset_url"] = "https://user@example.com/asset.zip" }, apperrors.E7003},
		{func(data map[string]any) { data["channel"] = "beta" }, apperrors.E7003},
		{func(data map[string]any) { data["published_at"] = "yesterday" }, apperrors.E7003},
		{func(data map[string]any) { delete(data, "license_hint") }, apperrors.E7003},
		{func(data map[string]any) { data["linkage"] = "dynamic" }, apperrors.E7003},
		{func(data map[string]any) { data["target"] = "darwin-arm64" }, apperrors.E7003},
		{func(data map[string]any) {
			data["license_hint"] = "nonfree"
			data["redistributable"] = true
		}, apperrors.E7003},
		{func(data map[string]any) { data["checksums_url"] = "ftp://example.com/checksums.sha256" }, apperrors.E7003},
		{func(data map[string]any) { data["checksums_sha256"] = strings.Repeat("Z", 64) }, apperrors.E7003},
	}
	for index, testCase := range cases {
		data := validRelease()
		testCase.mutate(data)
		err := contracts.ValidateExample(writeResultExample(t, "resolve", data))
		typed, ok := apperrors.As(err)
		require.True(t, ok, index)
		require.Equal(t, testCase.code, typed.Code, index)
	}
}

func TestValidateExampleRejectsContractHoles(t *testing.T) {
	cases := []func(map[string]any){
		func(data map[string]any) { data["asset_name"] = "../evil.zip" },
		func(data map[string]any) {
			data["warnings"] = []any{map[string]any{"code": "OMF-W9999", "message": "x"}}
		},
		func(data map[string]any) { data["checksums_sha256"] = strings.Repeat("a", 64) }, // sha256 without url is invalid
	}
	for index, mutate := range cases {
		data := validRelease()
		mutate(data)
		err := contracts.ValidateExample(writeResultExample(t, "resolve", data))
		typed, ok := apperrors.As(err)
		require.True(t, ok, index)
		require.Equal(t, apperrors.E7003, typed.Code, index)
	}
}

func TestValidateExampleRejectsArtifactContractHoles(t *testing.T) {
	cases := []func(map[string]any){
		func(data map[string]any) { data["artifact_id"] = "whatever" },
		func(data map[string]any) { data["local_path"] = "relative/a.zip" },
		func(data map[string]any) {
			data["warnings"] = []any{map[string]any{"code": "OMF-W9999", "message": "x"}}
		},
	}
	for index, mutate := range cases {
		data := validArtifact()
		mutate(data)
		err := contracts.ValidateExample(writeResultExample(t, "download", data))
		typed, ok := apperrors.As(err)
		require.True(t, ok, index)
		require.Equal(t, apperrors.E7003, typed.Code, index)
	}
}

func TestValidateDataFieldsRejectsMalformedCapabilities(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "docs", "reference", "schemas", "examples", "capabilities.example.json"))
	require.NoError(t, err)
	var wrapper map[string]any
	require.NoError(t, json.Unmarshal(body, &wrapper))
	data := wrapper["data"].(map[string]any)
	require.NoError(t, contracts.ValidateDataFields(data))

	commands := data["commands"].(map[string]any)
	commands["extra"] = map[string]any{"ok": true}
	require.Error(t, contracts.ValidateDataFields(data))
	delete(commands, "extra")

	delete(data, "warnings")
	require.Error(t, contracts.ValidateDataFields(data))
}

func TestValidateExampleAcceptsUNCPath(t *testing.T) {
	data := validArtifact()
	data["local_path"] = `\\server\share\downloads\ffmpeg.zip`
	require.NoError(t, contracts.ValidateExample(writeResultExample(t, "download", data)))
}

func TestValidateExampleRejectsInvalidArtifact(t *testing.T) {
	cases := []func(map[string]any){
		func(data map[string]any) { data["verified"] = false },
		func(data map[string]any) { delete(data, "verified") },
		func(data map[string]any) { data["asset_sha256"] = "not-a-hash" },
		func(data map[string]any) { data["downloaded_at"] = "2026-11-11" },
		func(data map[string]any) { delete(data, "local_path") },
	}
	for index, mutate := range cases {
		data := validArtifact()
		mutate(data)
		err := contracts.ValidateExample(writeResultExample(t, "download", data))
		typed, ok := apperrors.As(err)
		require.True(t, ok, index)
		require.Equal(t, apperrors.E7003, typed.Code, index)
	}
}

func TestValidateExampleRejectsInvalidProviderData(t *testing.T) {
	// Empty providers is valid now (no built-ins); a missing/invalid field is not.
	malformed := map[string]any{"schema": model.ProviderListSchema, "providers": "not-an-array"}
	err := contracts.ValidateExample(writeResultExample(t, "provider.list", malformed))
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)

	incomplete := map[string]any{
		"schema": model.ProviderListSchema,
		"providers": []any{map[string]any{
			"id": "my-ffmpeg", "repository": "custom",
		}},
	}
	err = contracts.ValidateExample(writeResultExample(t, "provider.list", incomplete))
	typed, ok = apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E7003, typed.Code)

	for _, added := range []map[string]any{
		{"schema": model.ProviderAddSchema, "id": "x", "url": "https://user@example.com/a.zip", "status": "added"},
		{"schema": model.ProviderAddSchema, "id": "x", "url": "https://example.com/a.zip", "status": "pending"},
		{"schema": model.ProviderAddSchema, "id": "x", "url": "https://example.com/a.zip", "status": "added", "sha256": "abc"},
	} {
		err = contracts.ValidateExample(writeResultExample(t, "provider.add", added))
		typed, ok = apperrors.As(err)
		require.True(t, ok, added)
		require.Equal(t, apperrors.E7003, typed.Code, added)
	}
}

func TestValidateCapabilitiesExampleWithCustomHelpTopic(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "reference", "schemas", "examples", "capabilities.example.json"))
	require.NoError(t, err)
	var obj map[string]any
	require.NoError(t, json.Unmarshal(raw, &obj))
	data, ok := obj["data"].(map[string]any)
	require.True(t, ok)

	commands := data["commands"].(map[string]any)
	commands["-h decoder=h264"] = commands["-h full"]
	delete(commands, "-h full")
	help := data["help"].(map[string]any)
	entry := help["full"].(map[string]any)
	entry["topic"] = "decoder=h264"
	help["decoder=h264"] = entry
	delete(help, "full")

	require.NoError(t, contracts.ValidateExample(writeResultExample(t, "capabilities", data)))
}

func TestValidateErrorExampleCountsRunes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "error.json")
	body, err := json.Marshal(map[string]any{
		"schema":  model.ErrorSchema,
		"ok":      false,
		"code":    "OMF-E7001",
		"message": strings.Repeat("中", 200),
		"details": map[string]any{},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, body, 0o644))
	require.NoError(t, contracts.ValidateExample(path))

	tooLong, err := json.Marshal(map[string]any{
		"schema":  model.ErrorSchema,
		"ok":      false,
		"code":    "OMF-E7001",
		"message": strings.Repeat("中", 201),
		"details": map[string]any{},
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, tooLong, 0o644))
	err = contracts.ValidateExample(path)
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
