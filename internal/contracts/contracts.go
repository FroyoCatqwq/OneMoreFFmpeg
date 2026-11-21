// Package contracts validates CLI JSON examples and the error-code registry.
package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"unicode/utf8"

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
