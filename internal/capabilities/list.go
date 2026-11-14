package capabilities

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var capabilityKinds = map[string]bool{
	"muxers": true, "demuxers": true, "devices": true,
	"decoders": true, "encoders": true, "filters": true,
	"pixel_formats": true, "channel_layouts": true, "sample_formats": true,
}

// IsCapabilityKind reports whether kind is a supported single-list query.
func IsCapabilityKind(kind string) bool { return capabilityKinds[kind] }

// CapabilityKinds returns the supported --only values in a stable order.
func CapabilityKinds() []string {
	return []string{
		"muxers", "demuxers", "devices", "decoders", "encoders", "filters",
		"pixel_formats", "channel_layouts", "sample_formats",
	}
}

// CollectList runs only the FFmpeg command needed for one capability kind.
func (c *Collector) CollectList(ctx context.Context, opts Options, kind string) (model.CapabilityListData, error) {
	if !capabilityKinds[kind] {
		return model.CapabilityListData{}, apperrors.New(apperrors.E7005, "unsupported capability kind").WithDetails(map[string]any{"kind": kind, "allowed": CapabilityKinds()})
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Installation.FFmpeg == "" {
		return model.CapabilityListData{}, apperrors.New(apperrors.E1002, "ffmpeg executable is required")
	}
	info, err := os.Stat(opts.Installation.FFmpeg)
	if err != nil || info.IsDir() {
		return model.CapabilityListData{}, apperrors.New(apperrors.E1002, "ffmpeg executable not found")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return model.CapabilityListData{}, apperrors.New(apperrors.E1005, "ffmpeg is not executable")
	}

	var spec commandSpec
	for _, candidate := range commandSpecs("full") {
		if candidate.Kind == kind {
			spec = candidate
			break
		}
	}
	if spec.Key == "" {
		return model.CapabilityListData{}, apperrors.New(apperrors.E9001, "capability spec is missing").WithDetails(map[string]any{"kind": kind})
	}
	fingerprint := c.capabilityFingerprint(opts.Installation, kind, []commandSpec{spec}, opts.CommandTimeout)
	cachePath := ""
	warnings := []model.Warning{}
	if !opts.NoCache {
		cacheDir, cacheErr := resolveCacheDir(opts.CacheDir)
		if cacheErr != nil {
			return model.CapabilityListData{}, cacheErr
		}
		cachePath = filepath.Join(cacheDir, "list-"+kind+"-"+fingerprint+".json")
		if !opts.Refresh {
			if cached, ok, cacheWarnings := readListCache(cachePath, kind, fingerprint); ok {
				return cached, nil
			} else {
				warnings = append(warnings, cacheWarnings...)
			}
		}
	}

	commands, raw, runWarnings, runErr := c.runCommands(ctx, opts.Installation, []commandSpec{spec}, opts.CommandTimeout)
	warnings = append(warnings, runWarnings...)
	if runErr != nil {
		return model.CapabilityListData{}, runErr
	}
	if err := ctx.Err(); err != nil {
		if stderrors.Is(err, context.DeadlineExceeded) {
			return model.CapabilityListData{}, apperrors.New(apperrors.E1006, "capability collection timed out").WithCause(err.Error())
		}
		return model.CapabilityListData{}, apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
	}
	command := commands[spec.Key]
	if !command.OK {
		return model.CapabilityListData{}, apperrors.New(apperrors.E2001, "capability command failed").WithDetails(map[string]any{
			"command": spec.Key,
			"error":   command.Error,
		})
	}
	entries, unknown := parseList(kind, raw[spec.Key].Stdout)
	if unknown > 0 {
		warnings = append(warnings, newWarning(apperrors.W1002, "some output lines could not be parsed", map[string]any{"kind": kind, "lines": unknown}))
	}
	if opts.Installation.FFplay == "" {
		warnings = append(warnings, newWarning(apperrors.W1001, "ffplay is missing", nil))
	}
	result := model.CapabilityListData{
		Schema:         model.CapabilityListSchema,
		InstallationID: opts.Installation.ID,
		Fingerprint:    fingerprint,
		Kind:           kind,
		Entries:        entries,
		Command:        strings.Join(spec.Args, " "),
		Warnings:       warnings,
	}
	if cachePath != "" {
		if warning := writeListCache(cachePath, result); warning != nil {
			result.Warnings = ensureWarning(result.Warnings, *warning)
		}
	}
	return result, nil
}

func readListCache(path, kind, fingerprint string) (model.CapabilityListData, bool, []model.Warning) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.CapabilityListData{}, false, nil
		}
		return model.CapabilityListData{}, false, []model.Warning{newWarning(apperrors.W1003, "capability list cache is unreadable", map[string]any{"path": path, "cause": err.Error()})}
	}
	var cached model.CapabilityListData
	if err := json.Unmarshal(raw, &cached); err != nil || cached.Schema != model.CapabilityListSchema || cached.Kind != kind || cached.Fingerprint != fingerprint {
		return model.CapabilityListData{}, false, []model.Warning{newWarning(apperrors.W1003, "capability list cache is stale or corrupt", map[string]any{"path": path})}
	}
	return cached, true, nil
}

func writeListCache(path string, value model.CapabilityListData) *model.Warning {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return warningPtr(apperrors.W1007, "cannot create capability cache directory", map[string]any{"path": dir, "cause": err.Error()})
	}
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return warningPtr(apperrors.W1007, "cannot encode capability list cache", map[string]any{"path": path, "cause": err.Error()})
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return warningPtr(apperrors.W1007, "cannot create capability list temp file", map[string]any{"path": dir, "cause": err.Error()})
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(payload)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		return warningPtr(apperrors.W1007, "cannot write capability list cache", map[string]any{"path": tmpPath, "cause": fmt.Sprint(writeErr, closeErr)})
	}
	_ = os.Chmod(tmpPath, 0o644)
	if err := commitCacheFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return warningPtr(apperrors.W1007, "cannot commit capability list cache", map[string]any{"path": path, "cause": err.Error()})
	}
	return nil
}
