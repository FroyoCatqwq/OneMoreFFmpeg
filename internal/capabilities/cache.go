package capabilities

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/cache"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
)

func resolveCacheDir(explicit string) (string, error) {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		if !filepath.IsAbs(explicit) {
			return "", apperrors.New(apperrors.E7001, "--cache-dir must be an absolute path").WithDetails(map[string]any{"dir": explicit})
		}
		return explicit, nil
	}
	base, err := paths.CacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "capabilities"), nil
}

func readCache(path, fingerprint string, specs []commandSpec, topic, installationID string) (model.CapabilitySet, bool, []model.Warning) {
	var cached model.CapabilitySet
	// Shared storage layer owns JSON decoding, atomic-read locking and error
	// classification; this module only applies business validity rules below.
	if err := cache.ReadJSON(path, &cached); err != nil {
		if os.IsNotExist(err) {
			return model.CapabilitySet{}, false, nil
		}
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is unreadable or corrupt", map[string]any{"path": path, "cause": err.Error()})}
	}
	if cached.Schema != model.CapabilitiesSchema || cached.Fingerprint != fingerprint {
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is stale", map[string]any{"path": path})}
	}
	if cached.InstallationID != installationID {
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache targets another installation", map[string]any{"path": path})}
	}
	for _, spec := range specs {
		if _, ok := cached.Commands[spec.Key]; !ok {
			return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is incomplete", map[string]any{"path": path, "command": spec.Key})}
		}
	}
	if _, ok := cached.Help[topic]; !ok {
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is incomplete", map[string]any{"path": path, "help_topic": topic})}
	}
	return cached.Normalize(), true, nil
}

func writeCache(path string, set model.CapabilitySet) *model.Warning {
	if err := cache.WriteJSON(path, set, 0o644); err != nil {
		return warningPtr(apperrors.W1007, "cannot write capability cache", map[string]any{"path": path, "cause": err.Error()})
	}
	return nil
}

func stderrSummary(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 200 {
		return string(runes[:200]) + "..."
	}
	return string(runes)
}

func newWarning(code apperrors.Code, message string, details map[string]any) model.Warning {
	if details == nil {
		details = map[string]any{}
	}
	return model.Warning{Code: string(code), Message: message, Details: details}
}

func warningPtr(code apperrors.Code, message string, details map[string]any) *model.Warning {
	warning := newWarning(code, message, details)
	return &warning
}

func ensureWarning(warnings []model.Warning, warning model.Warning) []model.Warning {
	key := warningKey(warning)
	for _, existing := range warnings {
		if warningKey(existing) == key {
			return warnings
		}
	}
	return append(warnings, warning)
}

func warningKey(warning model.Warning) string {
	details, _ := json.Marshal(warning.Details)
	return warning.Code + "|" + warning.Message + "|" + string(details)
}

func mergeWarnings(base, extra []model.Warning) []model.Warning {
	out := append([]model.Warning{}, base...)
	for _, warning := range extra {
		out = ensureWarning(out, warning)
	}
	return out
}
