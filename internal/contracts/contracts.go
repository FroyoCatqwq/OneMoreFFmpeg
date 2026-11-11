// Package contracts validates CLI JSON examples and the error-code registry.
package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	appcaps "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/capabilities"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var codePattern = regexp.MustCompile(`OMF-[EW][0-9]{4}`)

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
		if dataSchema == model.CapabilitiesSchema {
			commands, ok := data["commands"].(map[string]any)
			if !ok {
				return apperrors.New(apperrors.E7003, "capabilities.commands is required")
			}
			if len(commands) != len(appcaps.CommandKeys("full")) {
				return apperrors.New(apperrors.E7003, "capabilities.commands must contain all 13 commands")
			}
			for _, key := range appcaps.CommandKeys("full") {
				if _, ok := commands[key]; !ok {
					return apperrors.New(apperrors.E7003, "capabilities command missing").WithDetails(map[string]any{"command": key})
				}
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
		if !ok || message == "" || len(message) > 200 {
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
