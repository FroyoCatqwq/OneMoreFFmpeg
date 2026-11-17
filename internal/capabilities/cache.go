package capabilities

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.CapabilitySet{}, false, nil
		}
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is unreadable", map[string]any{"path": path, "cause": err.Error()})}
	}
	var cached model.CapabilitySet
	if err := json.Unmarshal(raw, &cached); err != nil {
		return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is corrupt", map[string]any{"path": path, "cause": err.Error()})}
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
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return warningPtr(apperrors.W1007, "cannot create capability cache directory", map[string]any{"path": dir, "cause": err.Error()})
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return warningPtr(apperrors.W1007, "cannot encode capability cache", map[string]any{"path": path, "cause": err.Error()})
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return warningPtr(apperrors.W1007, "cannot create capability cache temp file", map[string]any{"path": dir, "cause": err.Error()})
	}
	tmpPath := tmp.Name()
	discard := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := tmp.Write(data); err != nil {
		discard()
		return warningPtr(apperrors.W1007, "cannot write capability cache", map[string]any{"path": tmpPath, "cause": err.Error()})
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return warningPtr(apperrors.W1007, "cannot close capability cache temp file", map[string]any{"path": tmpPath, "cause": err.Error()})
	}
	_ = os.Chmod(tmpPath, 0o644)
	if err := commitCacheFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return warningPtr(apperrors.W1007, "cannot commit capability cache", map[string]any{"path": path, "cause": err.Error()})
	}
	return nil
}

// commitCacheFile relies on os.Rename replacing the target atomically on
// both POSIX and Windows. If the target is locked by another process the
// commit fails and the previous cache file is left untouched, which is
// preferable to deleting the only good copy.
func commitCacheFile(tmpPath, path string) error {
	return os.Rename(tmpPath, path)
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
