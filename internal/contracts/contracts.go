// Package contracts validates CLI JSON examples and the error-code registry.
package contracts

import (
	"encoding/json"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	appcaps "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/capabilities"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var (
	codePattern       = regexp.MustCompile(`\bOMF-[EW][0-9]{4}\b`)
	sha256Pattern     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	artifactIDPattern = regexp.MustCompile(`^art-[0-9a-f]{16}$`)
)

var knownCommands = map[string]struct{}{
	"version":       {},
	"detect":        {},
	"validate":      {},
	"capabilities":  {},
	"provider.list": {},
	"provider.add":  {},
	"resolve":       {},
	"download":      {},
	"install":       {},
	"env.status":    {},
	"env.set":       {},
	"env.unset":     {},
	"env.rollback":  {},
	"doctor":        {},
}

func ValidateData(data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return apperrors.New(apperrors.E7006, "data cannot be serialized").WithCause(err.Error())
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return apperrors.New(apperrors.E7003, "data is not a JSON object").WithCause(err.Error())
	}
	schema, ok := obj["schema"].(string)
	if !ok || schema == "" {
		return apperrors.New(apperrors.E7003, "data.schema is required")
	}
	if _, ok := model.KnownDataSchemas[schema]; !ok {
		return apperrors.New(apperrors.E7003, "unknown data.schema").WithDetails(map[string]any{"schema": schema})
	}
	return nil
}

// ValidateDataFields validates the schema-specific fields of a runtime data
// object. It is used by both example validation and CLI output so stdout JSON
// cannot silently violate the published contract.
func ValidateDataFields(data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return apperrors.New(apperrors.E7006, "data cannot be serialized").WithCause(err.Error())
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return apperrors.New(apperrors.E7003, "data is not a JSON object").WithCause(err.Error())
	}
	if err := ValidateData(data); err != nil {
		return err
	}
	schema, _ := object["schema"].(string)
	return validateDataFields(schema, object)
}

func ValidateExample(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return err
	}
	schema, _ := obj["schema"].(string)
	switch schema {
	case model.ResultSchema:
		ok, okType := obj["ok"].(bool)
		if !okType || !ok {
			return apperrors.New(apperrors.E7003, "result.ok must be true")
		}
		command, ok := obj["command"].(string)
		if !ok || command == "" {
			return apperrors.New(apperrors.E7003, "result.command is required")
		}
		if _, ok := knownCommands[command]; !ok {
			return apperrors.New(apperrors.E7003, "unknown result.command").WithDetails(map[string]any{"command": command})
		}
		data, ok := obj["data"].(map[string]any)
		if !ok {
			return apperrors.New(apperrors.E7003, "result.data is required")
		}
		dataSchema, ok := data["schema"].(string)
		if !ok || dataSchema == "" {
			return apperrors.New(apperrors.E7003, "result.data.schema is required")
		}
		if _, ok := model.KnownDataSchemas[dataSchema]; !ok {
			return apperrors.New(apperrors.E7003, "unknown result.data.schema").WithDetails(map[string]any{"schema": dataSchema})
		}
		if err := validateDataFields(dataSchema, data); err != nil {
			return err
		}
	case model.ErrorSchema:
		ok, okType := obj["ok"].(bool)
		if !okType || ok {
			return apperrors.New(apperrors.E7003, "error.ok must be false")
		}
		code, ok := obj["code"].(string)
		if !ok || code == "" {
			return apperrors.New(apperrors.E7003, "error.code is required")
		}
		if !apperrors.IsKnown(code) {
			return apperrors.New(apperrors.E7003, "unknown error.code").WithDetails(map[string]any{"code": code})
		}
		message, ok := obj["message"].(string)
		if !ok || message == "" || utf8.RuneCountInString(message) > 200 {
			return apperrors.New(apperrors.E7003, "error.message must be 1..200 characters")
		}
		if _, ok := obj["details"].(map[string]any); !ok {
			return apperrors.New(apperrors.E7003, "error.details must be an object")
		}
	default:
		return apperrors.New(apperrors.E7003, "unknown example schema")
	}
	return nil
}

func ValidateExamplesDir(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := ValidateExample(filepath.Join(dir, entry.Name())); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func validateDataFields(schema string, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return apperrors.New(apperrors.E7003, "data cannot be serialized").WithCause(err.Error())
	}
	switch schema {
	case model.ProviderListSchema:
		var list model.ProviderListData
		if err := json.Unmarshal(raw, &list); err != nil {
			return apperrors.New(apperrors.E7003, "provider list is malformed").WithCause(err.Error())
		}
		if len(list.Providers) == 0 {
			return apperrors.New(apperrors.E7003, "provider list must contain at least one provider")
		}
		for _, provider := range list.Providers {
			if provider.ID == "" || provider.Repository == "" {
				return apperrors.New(apperrors.E7003, "provider id and repository are required")
			}
			if len(provider.Channels) == 0 || len(provider.Variants) == 0 || len(provider.Linkages) == 0 {
				return apperrors.New(apperrors.E7003, "provider channels, variants and linkages are required").WithDetails(map[string]any{"provider": provider.ID})
			}
		}
	case model.ProviderAddSchema:
		var added model.ProviderAddData
		if err := json.Unmarshal(raw, &added); err != nil {
			return apperrors.New(apperrors.E7003, "provider add is malformed").WithCause(err.Error())
		}
		if added.ID == "" || added.URL == "" {
			return apperrors.New(apperrors.E7003, "provider add id and url are required")
		}
		if !isHTTPS(added.URL) {
			return apperrors.New(apperrors.E7003, "provider add url must be https")
		}
		switch added.Status {
		case "added", "dry-run":
		default:
			return apperrors.New(apperrors.E7003, "provider add status must be added or dry-run")
		}
		if added.SHA256 != "" && !sha256Pattern.MatchString(added.SHA256) {
			return apperrors.New(apperrors.E7003, "provider add sha256 must be lowercase hex")
		}
	case model.ReleaseSchema:
		return validateRelease(raw)
	case model.ArtifactSchema:
		return validateArtifact(raw, data)
	case model.CapabilitiesSchema:
		return validateCapabilities(data)
	case model.InstallSchema:
		return validateInstall(raw)
	}
	return nil
}

func validateInstall(raw []byte) error {
	var data model.InstallData
	if err := json.Unmarshal(raw, &data); err != nil {
		return apperrors.New(apperrors.E7003, "install data is malformed").WithCause(err.Error())
	}
	for field, value := range map[string]string{
		"installation_id": data.InstallationID,
		"root_dir":        data.RootDir,
		"provider":        data.Provider,
		"repository":      data.Repository,
		"release_tag":     data.ReleaseTag,
		"asset_sha256":    data.AssetSHA256,
		"license_hint":    data.LicenseHint,
	} {
		if value == "" {
			return apperrors.New(apperrors.E7003, "install field is required").WithDetails(map[string]any{"field": field})
		}
	}
	if !isAbsoluteLike(data.RootDir) {
		return apperrors.New(apperrors.E7003, "install root_dir must be absolute")
	}
	switch data.CurrentPointer {
	case model.CurrentJunction, model.CurrentSymlink, model.CurrentPointerFile, model.CurrentMissing:
	default:
		return apperrors.New(apperrors.E7003, "install current_pointer is invalid")
	}
	if !sha256Pattern.MatchString(data.AssetSHA256) {
		return apperrors.New(apperrors.E7003, "install asset_sha256 must be lowercase hex")
	}
	dryRun, _ := rawField(raw, "dry_run")
	isDryRun, _ := dryRun.(bool)
	if !isDryRun {
		if !sha256Pattern.MatchString(data.Binaries.FFmpegSHA256) || !sha256Pattern.MatchString(data.Binaries.FFprobeSHA256) {
			return apperrors.New(apperrors.E7003, "install binary hashes must be lowercase hex")
		}
		if data.Binaries.FFplaySHA256 != "" && !sha256Pattern.MatchString(data.Binaries.FFplaySHA256) {
			return apperrors.New(apperrors.E7003, "install ffplay hash must be lowercase hex")
		}
	}
	switch data.LicenseHint {
	case "gpl", "lgpl", "nonfree", "custom":
	default:
		return apperrors.New(apperrors.E7003, "install license_hint is invalid")
	}
	if data.LicenseHint == "nonfree" && data.Redistributable {
		return apperrors.New(apperrors.E7003, "nonfree install must not be redistributable")
	}
	if _, ok := rawField(raw, "dry_run"); !ok {
		return apperrors.New(apperrors.E7003, "install dry_run is required")
	}
	return nil
}

func rawField(raw []byte, name string) (any, bool) {
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, false
	}
	value, ok := object[name]
	return value, ok
}

func validateCapabilities(data map[string]any) error {
	commands, ok := data["commands"].(map[string]any)
	if !ok {
		return apperrors.New(apperrors.E7003, "capabilities.commands is required")
	}
	expected := appcaps.CommandKeys("full")
	if len(commands) != len(expected) {
		return apperrors.New(apperrors.E7003, "capabilities.commands must contain 13 commands").WithDetails(map[string]any{"expected": len(expected), "actual": len(commands)})
	}
	for _, key := range expected {
		if strings.HasPrefix(key, "-h ") {
			continue
		}
		if _, ok := commands[key]; !ok {
			return apperrors.New(apperrors.E7003, "capabilities command missing").WithDetails(map[string]any{"command": key})
		}
	}
	helpCommands := 0
	for key := range commands {
		if strings.HasPrefix(key, "-h ") {
			helpCommands++
			topic := strings.TrimPrefix(key, "-h ")
			help, ok := data["help"].(map[string]any)
			if !ok {
				return apperrors.New(apperrors.E7003, "capabilities.help is required")
			}
			if _, ok := help[topic]; !ok {
				return apperrors.New(apperrors.E7003, "capabilities help entry missing").WithDetails(map[string]any{"topic": topic})
			}
		}
	}
	if helpCommands != 1 {
		return apperrors.New(apperrors.E7003, "capabilities.commands must contain exactly one -h <topic> key").WithDetails(map[string]any{"count": helpCommands})
	}
	for _, field := range []string{
		"installation_id", "fingerprint", "version", "license",
		"muxers", "demuxers", "devices", "decoders", "encoders", "filters",
		"pixel_formats", "channel_layouts", "sample_formats", "help", "warnings",
	} {
		if _, ok := data[field]; !ok {
			return apperrors.New(apperrors.E7003, "capabilities field missing").WithDetails(map[string]any{"field": field})
		}
	}
	return nil
}

func validateRelease(raw []byte) error {
	var release model.Release
	if err := json.Unmarshal(raw, &release); err != nil {
		return apperrors.New(apperrors.E7003, "release is malformed").WithCause(err.Error())
	}
	for field, value := range map[string]string{
		"provider":     release.Provider,
		"repository":   release.Repository,
		"release_tag":  release.ReleaseTag,
		"published_at": release.PublishedAt,
		"asset_name":   release.AssetName,
		"asset_url":    release.AssetURL,
		"asset_sha256": release.AssetSHA256,
		"variant":      release.Variant,
		"linkage":      release.Linkage,
		"target":       release.Target,
		"license_hint": release.LicenseHint,
	} {
		if value == "" {
			return apperrors.New(apperrors.E7003, "release field is required").WithDetails(map[string]any{"field": field})
		}
	}
	switch release.Channel {
	case "stable", "daily":
	default:
		return apperrors.New(apperrors.E7003, "release channel must be stable or daily")
	}
	// D1: a floating `latest` tag is only acceptable when the record also
	// carries an immutable release id plus the published time and asset hash.
	if release.ReleaseTag == "latest" {
		if release.ReleaseID == "" || release.PublishedAt == "" || !sha256Pattern.MatchString(release.AssetSHA256) {
			return apperrors.New(apperrors.E3010, "release_tag=latest requires release_id, published_at and asset_sha256")
		}
	}
	if release.PinnedAt != "" {
		if _, err := time.Parse(time.RFC3339, release.PinnedAt); err != nil {
			return apperrors.New(apperrors.E7003, "release pinned_at must be RFC 3339").WithCause(err.Error())
		}
	}
	if _, err := time.Parse(time.RFC3339, release.PublishedAt); err != nil {
		return apperrors.New(apperrors.E7003, "release published_at must be RFC 3339").WithCause(err.Error())
	}
	if !isHTTPS(release.AssetURL) {
		return apperrors.New(apperrors.E7003, "release asset_url must be https")
	}
	if !sha256Pattern.MatchString(release.AssetSHA256) {
		return apperrors.New(apperrors.E7003, "release asset_sha256 must be lowercase hex")
	}
	if !safeAssetName(release.AssetName) {
		return apperrors.New(apperrors.E7003, "release asset_name must be a bare file name").WithDetails(map[string]any{"asset": release.AssetName})
	}
	if err := validateWarnings(release.Warnings); err != nil {
		return err
	}
	switch release.Linkage {
	case "static", "shared", "unknown":
	default:
		return apperrors.New(apperrors.E7003, "release linkage must be static, shared or unknown")
	}
	switch release.Target {
	case "win64", "linux64":
	default:
		return apperrors.New(apperrors.E7003, "release target must be win64 or linux64")
	}
	switch release.LicenseHint {
	case "gpl", "lgpl", "nonfree", "custom":
	default:
		return apperrors.New(apperrors.E7003, "release license_hint is invalid")
	}
	if release.LicenseHint == "nonfree" && release.Redistributable {
		return apperrors.New(apperrors.E7003, "nonfree release must not be redistributable")
	}
	if release.ChecksumsURL != "" && !isHTTPS(release.ChecksumsURL) {
		return apperrors.New(apperrors.E7003, "release checksums_url must be https")
	}
	if release.ChecksumsSHA256 != "" && !sha256Pattern.MatchString(release.ChecksumsSHA256) {
		return apperrors.New(apperrors.E7003, "release checksums_sha256 must be lowercase hex")
	}
	if (release.ChecksumsURL == "") != (release.ChecksumsSHA256 == "") {
		return apperrors.New(apperrors.E7003, "release checksums_url and checksums_sha256 must be paired")
	}
	return nil
}

func validateArtifact(raw []byte, data map[string]any) error {
	var artifact model.Artifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return apperrors.New(apperrors.E7003, "artifact is malformed").WithCause(err.Error())
	}
	for field, value := range map[string]string{
		"artifact_id":   artifact.ArtifactID,
		"provider":      artifact.Provider,
		"release_tag":   artifact.ReleaseTag,
		"asset_name":    artifact.AssetName,
		"asset_sha256":  artifact.AssetSHA256,
		"local_path":    artifact.LocalPath,
		"downloaded_at": artifact.DownloadedAt,
	} {
		if value == "" {
			return apperrors.New(apperrors.E7003, "artifact field is required").WithDetails(map[string]any{"field": field})
		}
	}
	if !sha256Pattern.MatchString(artifact.AssetSHA256) {
		return apperrors.New(apperrors.E7003, "artifact asset_sha256 must be lowercase hex")
	}
	if !artifactIDPattern.MatchString(artifact.ArtifactID) {
		return apperrors.New(apperrors.E7003, "artifact_id must be art- plus 16 lowercase hex")
	}
	if !safeAssetName(artifact.AssetName) {
		return apperrors.New(apperrors.E7003, "artifact asset_name must be a bare file name")
	}
	if !isAbsoluteLike(artifact.LocalPath) {
		return apperrors.New(apperrors.E7003, "artifact local_path must be absolute")
	}
	if err := validateWarnings(artifact.Warnings); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, artifact.DownloadedAt); err != nil {
		return apperrors.New(apperrors.E7003, "artifact downloaded_at must be RFC 3339").WithCause(err.Error())
	}
	if _, ok := data["verified"].(bool); !ok {
		return apperrors.New(apperrors.E7003, "artifact verified must be a boolean")
	}
	if !artifact.Verified {
		return apperrors.New(apperrors.E7003, "artifact verified must be true")
	}
	return nil
}

func isHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	// Loopback http is permitted for tests and self-hosted mirrors, matching
	// the download layer's URL policy.
	return parsed.Scheme == "http" && isLoopback(parsed.Hostname())
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func safeAssetName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return false
	}
	return true
}

func isAbsoluteLike(value string) bool {
	if strings.HasPrefix(value, "\\\\") || strings.HasPrefix(value, "//") {
		return true // UNC / Windows extended-length path
	}
	return strings.HasPrefix(value, "/") ||
		(len(value) >= 3 && value[1] == ':' && (value[2] == '/' || value[2] == '\\'))
}

func validateWarnings(warnings []model.Warning) error {
	for _, warning := range warnings {
		if !apperrors.IsKnown(warning.Code) {
			return apperrors.New(apperrors.E7003, "warning code is not registered").WithDetails(map[string]any{"code": warning.Code})
		}
	}
	return nil
}

func ValidateErrorRegistryDocs(path string, isKnown func(string) bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, match := range codePattern.FindAllString(string(raw), -1) {
		if seen[match] {
			continue
		}
		seen[match] = true
		if !isKnown(match) {
			return apperrors.New(apperrors.E9001, "documented code missing from registry").WithDetails(map[string]any{"code": match})
		}
	}
	return nil
}

func ValidateRegistryCoveredByDocs(path string, codes []string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	for _, code := range codes {
		if !strings.Contains(text, code) {
			return apperrors.New(apperrors.E9001, "registry code missing from documentation").WithDetails(map[string]any{"code": code})
		}
	}
	return nil
}
