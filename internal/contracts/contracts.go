// Package contracts validates CLI JSON examples and the error-code registry.
package contracts

import (
	"encoding/json"
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
	"version":          {},
	"detect":           {},
	"validate":         {},
	"capabilities":     {},
	"provider.list":    {},
	"provider.add":     {},
	"resolve":          {},
	"download":         {},
	"install":          {},
	"env.status":       {},
	"env.set":          {},
	"env.unset":        {},
	"env.rollback":     {},
	"doctor":           {},
	"uninstall":        {},
	"journal.list":     {},
	"journal.rollback": {},
	"release.list":     {},
	"update":           {},
}

// commandDataSchemas is the authoritative command -> allowed data.schema map.
// It is shared by example validation, freeze tests and runtime result output so
// a command can never emit a schema it is not contracted to emit.
var CommandDataSchemas = map[string][]string{
	"version":          {model.VersionSchema},
	"detect":           {model.DetectSchema},
	"validate":         {model.ValidateSchema},
	"capabilities":     {model.CapabilitiesSchema, model.CapabilityListSchema}, // --only emits a capability-list
	"provider.list":    {model.ProviderListSchema},
	"provider.add":     {model.ProviderAddSchema},
	"resolve":          {model.ReleaseSchema},
	"download":         {model.ArtifactSchema},
	"install":          {model.InstallSchema},
	"env.status":       {model.EnvStatusSchema},
	"env.set":          {model.EnvSetSchema},
	"env.unset":        {model.EnvSetSchema},
	"env.rollback":     {model.EnvSetSchema},
	"doctor":           {model.DoctorSchema},
	"uninstall":        {model.UninstallSchema},
	"journal.list":     {model.JournalListSchema},
	"journal.rollback": {model.JournalActionSchema},
	"release.list":     {model.ReleaseCatalogSchema},
	"update":           {model.UpdateSchema},
}

// ValidateCommand checks that a result command is part of the frozen CLI set.
func ValidateCommand(command string) error {
	if _, ok := knownCommands[command]; !ok {
		return apperrors.New(apperrors.E7003, "unknown result.command").WithDetails(map[string]any{"command": command})
	}
	return nil
}

// ValidateCommandData checks that (command, data.schema) is an allowed pair.
// It is the single source of truth used at runtime and for golden examples.
func ValidateCommandData(command string, data any) error {
	if err := ValidateCommand(command); err != nil {
		return err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return apperrors.New(apperrors.E7006, "data cannot be serialized").WithCause(err.Error())
	}
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		return apperrors.New(apperrors.E7003, "data is not a JSON object").WithCause(err.Error())
	}
	schema, _ := object["schema"].(string)
	allowed := CommandDataSchemas[command]
	for _, candidate := range allowed {
		if schema == candidate {
			return nil
		}
	}
	return apperrors.New(apperrors.E7003, "command emits an unexpected data.schema").
		WithDetails(map[string]any{"command": command, "schema": schema, "allowed": allowed})
}

// ErrorObjectFieldsAreValid enforces the published error-envelope field rules.
func ErrorObjectFieldsAreValid(code, message string, details map[string]any, cause string) error {
	if !apperrors.IsKnown(code) {
		return apperrors.New(apperrors.E7003, "unknown error.code").WithDetails(map[string]any{"code": code})
	}
	if message == "" || utf8.RuneCountInString(message) > 200 {
		return apperrors.New(apperrors.E7003, "error.message must be 1..200 characters")
	}
	if details == nil {
		return apperrors.New(apperrors.E7003, "error.details must be an object")
	}
	if cause != "" {
		if utf8.RuneCountInString(cause) > 500 {
			return apperrors.New(apperrors.E7003, "error.cause must be at most 500 characters")
		}
		if looksLikeCredential(cause) {
			return apperrors.New(apperrors.E7003, "error.cause must not contain credentials")
		}
	}
	return nil
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
		// The example must use a schema the command is actually allowed to emit.
		if err := ValidateCommandData(command, data); err != nil {
			return err
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
		message, ok := obj["message"].(string)
		if !ok {
			return apperrors.New(apperrors.E7003, "error.message is required")
		}
		details, _ := obj["details"].(map[string]any)
		cause, _ := obj["cause"].(string)
		if rawCause, present := obj["cause"]; present {
			if _, ok := rawCause.(string); !ok {
				return apperrors.New(apperrors.E7003, "error.cause must be a string")
			}
		}
		if err := ErrorObjectFieldsAreValid(code, message, details, cause); err != nil {
			return err
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
	case model.CapabilityListSchema:
		return validateCapabilityList(raw)
	case model.ReleaseCatalogSchema:
		return validateReleaseCatalog(raw)
	case model.VersionSchema:
		return validateVersionData(raw)
	case model.DetectSchema:
		return validateDetectData(raw)
	case model.ValidateSchema:
		return validateValidateData(raw)
	case model.EnvStatusSchema:
		return validateEnvStatus(raw)
	case model.EnvSetSchema:
		return validateEnvSet(raw)
	case model.DoctorSchema:
		return validateDoctorData(raw)
	case model.UninstallSchema:
		return validateUninstallData(raw)
	case model.JournalListSchema:
		return validateJournalList(raw)
	case model.JournalActionSchema:
		return validateJournalAction(raw)
	}
	return nil
}

func validateVersionData(raw []byte) error {
	var data model.VersionData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.VersionSchema || data.Name == "" || data.OS == "" || data.Arch == "" {
		return apperrors.New(apperrors.E7003, "version data is malformed")
	}
	return nil
}

func validateDetectData(raw []byte) error {
	var data model.DetectData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.DetectSchema {
		return apperrors.New(apperrors.E7003, "detect data is malformed")
	}
	for _, key := range []string{"installations", "errors"} {
		if _, ok := rawField(raw, key); !ok {
			return apperrors.New(apperrors.E7003, "detect field is required").WithDetails(map[string]any{"field": key})
		}
	}
	return nil
}

func validateValidateData(raw []byte) error {
	var data model.ValidateData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.ValidateSchema || data.Status == "" {
		return apperrors.New(apperrors.E7003, "validate data is malformed")
	}
	return validateWarnings(data.Warnings)
}

func validateEnvStatus(raw []byte) error {
	var data model.EnvStatusData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.EnvStatusSchema || data.CurrentType == "" || data.Variables == nil {
		return apperrors.New(apperrors.E7003, "env status data is malformed")
	}
	return validateWarnings(data.Warnings)
}

func validateEnvSet(raw []byte) error {
	var data model.EnvSetData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.EnvSetSchema {
		return apperrors.New(apperrors.E7003, "env set data is malformed")
	}
	if _, ok := rawField(raw, "dry_run"); !ok {
		return apperrors.New(apperrors.E7003, "env set dry_run is required")
	}
	if _, ok := rawField(raw, "operations"); !ok {
		return apperrors.New(apperrors.E7003, "env set operations are required")
	}
	return nil
}

func validateDoctorData(raw []byte) error {
	var data model.DoctorData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.DoctorSchema {
		return apperrors.New(apperrors.E7003, "doctor data is malformed")
	}
	for _, key := range []string{"healthy", "checks", "journal_pending"} {
		if _, ok := rawField(raw, key); !ok {
			return apperrors.New(apperrors.E7003, "doctor field is required").WithDetails(map[string]any{"field": key})
		}
	}
	return validateWarnings(data.Warnings)
}

func validateUninstallData(raw []byte) error {
	var data model.UninstallData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.UninstallSchema || data.RootDir == "" {
		return apperrors.New(apperrors.E7003, "uninstall data is malformed")
	}
	if _, ok := rawField(raw, "dry_run"); !ok {
		return apperrors.New(apperrors.E7003, "uninstall dry_run is required")
	}
	return nil
}

func validateJournalList(raw []byte) error {
	var data model.JournalListData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.JournalListSchema {
		return apperrors.New(apperrors.E7003, "journal list is malformed")
	}
	for _, key := range []string{"records", "pending"} {
		if _, ok := rawField(raw, key); !ok {
			return apperrors.New(apperrors.E7003, "journal list field is required").WithDetails(map[string]any{"field": key})
		}
	}
	return nil
}

func validateJournalAction(raw []byte) error {
	var data model.JournalActionData
	if err := json.Unmarshal(raw, &data); err != nil || data.Schema != model.JournalActionSchema || data.JournalID == "" || data.Action == "" {
		return apperrors.New(apperrors.E7003, "journal action is malformed")
	}
	return nil
}

func validateReleaseCatalog(raw []byte) error {
	var data model.ReleaseCatalogData
	if err := json.Unmarshal(raw, &data); err != nil {
		return apperrors.New(apperrors.E7003, "release catalog is malformed").WithCause(err.Error())
	}
	if data.Provider == "" || (data.Channel != "stable" && data.Channel != "daily") || data.Source == "" {
		return apperrors.New(apperrors.E7003, "release catalog field is required")
	}
	for _, entry := range data.Entries {
		if entry.Version == "" || entry.ReleaseTag == "" || entry.AssetName == "" || !isHTTPS(entry.AssetURL) {
			return apperrors.New(apperrors.E7003, "release catalog entry is malformed").WithDetails(map[string]any{"version": entry.Version})
		}
	}
	return nil
}

func validateCapabilityList(raw []byte) error {
	var data model.CapabilityListData
	if err := json.Unmarshal(raw, &data); err != nil {
		return apperrors.New(apperrors.E7003, "capability list is malformed").WithCause(err.Error())
	}
	if data.Kind == "" || data.InstallationID == "" || data.Fingerprint == "" || data.Command == "" {
		return apperrors.New(apperrors.E7003, "capability list field is required")
	}
	if !appcaps.IsCapabilityKind(data.Kind) {
		return apperrors.New(apperrors.E7003, "capability list kind is invalid").WithDetails(map[string]any{"kind": data.Kind})
	}
	return validateWarnings(data.Warnings)
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
	dryRun, found := rawField(raw, "dry_run")
	if !found {
		return apperrors.New(apperrors.E7003, "install dry_run is required")
	}
	isDryRun, isBool := dryRun.(bool)
	if !isBool {
		return apperrors.New(apperrors.E7003, "install dry_run must be a boolean")
	}
	// An applied install MUST name its transaction: a caller needs the id to
	// roll it back. A dry-run creates no journal, so the field is omitted.
	if !isDryRun && data.JournalID == "" {
		return apperrors.New(apperrors.E7003, "install journal_id is required when applied")
	}
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
	if fingerprint, ok := data["fingerprint"].(string); !ok || !sha256Pattern.MatchString(fingerprint) {
		return apperrors.New(apperrors.E7003, "capabilities.fingerprint must be lowercase hex sha256")
	}
	for _, field := range []string{"version", "license", "help"} {
		if _, ok := data[field].(map[string]any); !ok {
			return apperrors.New(apperrors.E7003, "capabilities field must be an object").WithDetails(map[string]any{"field": field})
		}
	}
	for _, field := range []string{
		"muxers", "demuxers", "devices", "decoders", "encoders", "filters",
		"pixel_formats", "channel_layouts", "sample_formats", "warnings",
	} {
		if _, ok := data[field].([]any); !ok {
			return apperrors.New(apperrors.E7003, "capabilities field must be an array").WithDetails(map[string]any{"field": field})
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
		if release.ReleaseID == "" || release.PublishedAt == "" || release.AssetSHA256 == "" {
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

// looksLikeCredential is a conservative guard for golden error examples: a
// documented cause must be a human-readable summary, never a token or header.
func looksLikeCredential(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"authorization:", "bearer ", "ghp_", "github_pat_", "token=", "api_key="} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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
