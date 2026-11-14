package provider

import (
	"bytes"
	"errors"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/archive"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// CustomSchema is the on-disk schema of <data-root>/providers.toml.
const CustomSchema = "onemoreffmpeg.providers/v1"

var (
	customIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	sha256Hex       = regexp.MustCompile(`^[0-9a-f]{64}$`)
	licenseHints    = map[string]bool{"gpl": true, "lgpl": true, "nonfree": true, "custom": true}
)

// Custom is one user-registered direct asset provider.
type Custom struct {
	ID           string `toml:"id" json:"id"`
	URL          string `toml:"url" json:"url"`
	SHA256       string `toml:"sha256" json:"sha256"`
	ChecksumsURL string `toml:"checksums_url,omitempty" json:"checksums_url,omitempty"`
	LicenseHint  string `toml:"license_hint,omitempty" json:"license_hint,omitempty"`
}

type customFile struct {
	Schema    string   `toml:"schema"`
	Providers []Custom `toml:"provider"`
}

// LoadCustomFile reads providers.toml; a missing file yields no entries.
func LoadCustomFile(path string) ([]Custom, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Custom{}, nil
		}
		return nil, apperrors.New(apperrors.E7007, "providers file cannot be read").WithCause(err.Error())
	}
	var file customFile
	decoder := toml.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		var strictErr *toml.StrictMissingError
		if errors.As(err, &strictErr) {
			return nil, apperrors.New(apperrors.E7004, "unknown key in providers file").WithCause(err.Error())
		}
		return nil, apperrors.New(apperrors.E7002, "providers file is not valid TOML").WithCause(err.Error())
	}
	if file.Schema != CustomSchema {
		return nil, apperrors.New(apperrors.E7003, "providers file schema mismatch").WithDetails(map[string]any{"schema": file.Schema})
	}
	normalized := make([]Custom, 0, len(file.Providers))
	for _, entry := range file.Providers {
		entry = normalizeCustom(entry)
		if err := validateCustom(entry, normalized); err != nil {
			return nil, err
		}
		normalized = append(normalized, entry)
	}
	return normalized, nil
}

// SaveCustomFile writes providers.toml atomically.
func SaveCustomFile(path string, entries []Custom) error {
	unlock, err := acquireFileLock(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	return saveCustomFileNoLock(path, entries)
}

// AddAndSaveCustomFile locks the whole read-modify-write transaction.
func AddAndSaveCustomFile(path string, entry Custom) ([]Custom, error) {
	unlock, err := acquireFileLock(path + ".lock")
	if err != nil {
		return nil, err
	}
	defer unlock()
	existing, err := LoadCustomFile(path)
	if err != nil {
		return nil, err
	}
	updated, err := AddCustom(existing, entry)
	if err != nil {
		return nil, err
	}
	if err := saveCustomFileNoLock(path, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func saveCustomFileNoLock(path string, entries []Custom) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return apperrors.New(apperrors.E7007, "cannot create provider directory").WithCause(err.Error())
	}
	normalized := make([]Custom, 0, len(entries))
	for _, entry := range entries {
		entry = normalizeCustom(entry)
		if err := validateCustom(entry, normalized); err != nil {
			return err
		}
		normalized = append(normalized, entry)
	}
	payload, err := toml.Marshal(customFile{Schema: CustomSchema, Providers: normalized})
	if err != nil {
		return apperrors.New(apperrors.E7007, "cannot encode providers file").WithCause(err.Error())
	}
	tmp, err := os.CreateTemp(directory, filepath.Base(path)+".*.tmp")
	if err != nil {
		return apperrors.New(apperrors.E7007, "cannot create providers temp file").WithCause(err.Error())
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E7007, "cannot write providers file").WithCause(err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E7007, "cannot close providers temp file").WithCause(err.Error())
	}
	_ = os.Chmod(tmpPath, 0o644)
	if err := commitProvidersFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E7007, "cannot commit providers file").WithCause(err.Error())
	}
	return nil
}

// commitProvidersFile preserves the previous registry until the new one is in
// place. Windows needs a backup rename because Rename cannot replace a file.
func commitProvidersFile(tmpPath, path string) error {
	if err := os.Rename(tmpPath, path); err == nil {
		return nil
	}
	backup := path + ".bak"
	_ = os.Remove(backup)
	if err := os.Rename(path, backup); err != nil {
		if os.IsNotExist(err) {
			return os.Rename(tmpPath, path)
		}
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(path)
		_ = os.Rename(backup, path)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

// AddCustom appends one entry; duplicate ids are rejected so `provider add`
// never silently overwrites an existing pin.
func AddCustom(entries []Custom, entry Custom) ([]Custom, error) {
	entry = normalizeCustom(entry)
	if err := validateCustom(entry, entries); err != nil {
		return nil, err
	}
	updated := make([]Custom, 0, len(entries)+1)
	updated = append(updated, entries...)
	updated = append(updated, entry)
	return updated, nil
}

// FindCustom returns the registered custom provider with the given id.
func FindCustom(entries []Custom, id string) (Custom, bool) {
	wanted := strings.ToLower(strings.TrimSpace(id))
	for _, entry := range entries {
		if strings.EqualFold(entry.ID, wanted) {
			return entry, true
		}
	}
	return Custom{}, false
}

// CustomSummary exposes a conservative summary for `provider list`.
func CustomSummary(entry Custom) model.ProviderSummary {
	return model.ProviderSummary{
		ID:              entry.ID,
		Repository:      "custom",
		Channels:        []string{"daily"},
		Variants:        []string{"gpl", "lgpl", "nonfree", "nonfree-shared"},
		Linkages:        []string{"static", "shared"},
		Redistributable: false,
	}
}

func normalizeCustom(entry Custom) Custom {
	entry.ID = strings.ToLower(strings.TrimSpace(entry.ID))
	entry.URL = strings.TrimSpace(entry.URL)
	entry.SHA256 = strings.ToLower(strings.TrimSpace(entry.SHA256))
	entry.ChecksumsURL = strings.TrimSpace(entry.ChecksumsURL)
	entry.LicenseHint = strings.ToLower(strings.TrimSpace(entry.LicenseHint))
	return entry
}

func validateCustom(entry Custom, existing []Custom) error {
	id := entry.ID
	if !customIDPattern.MatchString(id) {
		return apperrors.New(apperrors.E7005, "provider id is invalid").WithDetails(map[string]any{"id": entry.ID})
	}
	switch id {
	case IDBtbn, IDBnqdzjNonfree, IDCustom:
		return apperrors.New(apperrors.E7005, "provider id is reserved").WithDetails(map[string]any{"id": id})
	}
	for _, current := range existing {
		if current.ID == id {
			return apperrors.New(apperrors.E7005, "provider id already exists").WithDetails(map[string]any{"id": id})
		}
	}
	if !isHTTPSURL(entry.URL) {
		return apperrors.New(apperrors.E3008, "custom provider url must be https").WithDetails(map[string]any{"url": entry.URL})
	}
	if !sha256Hex.MatchString(entry.SHA256) {
		return apperrors.New(apperrors.E3009, "custom provider requires a sha256 pin").WithDetails(map[string]any{"id": id})
	}
	if name := assetNameFromURL(entry.URL); name == "" {
		return apperrors.New(apperrors.E3008, "custom provider url has no usable file name").WithDetails(map[string]any{"url": entry.URL})
	} else if !hasArchiveExtension(name) {
		return apperrors.New(apperrors.E3008, "custom provider url must point to a zip/tar.xz/tar.gz/tar.zst archive").WithDetails(map[string]any{"url": entry.URL})
	}
	if entry.ChecksumsURL != "" && !isHTTPSURL(entry.ChecksumsURL) {
		return apperrors.New(apperrors.E3008, "custom checksums url must be https").WithDetails(map[string]any{"url": entry.ChecksumsURL})
	}
	if entry.LicenseHint != "" && !licenseHints[entry.LicenseHint] {
		return apperrors.New(apperrors.E7005, "custom license_hint is invalid").WithDetails(map[string]any{"license_hint": entry.LicenseHint})
	}
	return nil
}

func isHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil
}

// assetNameFromURL returns the safe file name of a direct asset URL, or "".
func assetNameFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	name := path.Base(parsed.Path)
	switch name {
	case "", ".", "/", "..", "\\":
		return ""
	}
	if strings.ContainsAny(name, "/\\") {
		return ""
	}
	return name
}

// acquireFileLock serializes providers.toml writers across processes.
func acquireFileLock(path string) (func(), error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !os.IsExist(err) {
			return nil, apperrors.New(apperrors.E7007, "cannot create provider lock").WithCause(err.Error())
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > 30*time.Second {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return nil, apperrors.New(apperrors.E7005, "providers file is locked by another process").WithDetails(map[string]any{"lock": path})
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func hasArchiveExtension(name string) bool {
	return archive.FormatFor(name) != ""
}
