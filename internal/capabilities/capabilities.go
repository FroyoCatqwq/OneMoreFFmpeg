// Package capabilities runs FFmpeg information commands and parses their output.
package capabilities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	shortTimeout = 5 * time.Second
	listTimeout  = 15 * time.Second
	maxOutput    = 8 << 20
)

var topicNamePattern = regexp.MustCompile(`^[A-Za-z0-9._+,=-]+$`)

type Options struct {
	Installation model.Installation
	HelpTopic    string
	Refresh      bool
	NoCache      bool
	CacheDir     string
}

type commandSpec struct {
	Key     string
	Args    []string
	Timeout time.Duration
	Kind    string
}

type Collector struct {
	Runner ffexec.Runner
}

func New() *Collector {
	return &Collector{
		Runner: ffexec.Runner{
			Timeout:        listTimeout,
			WaitDelay:      500 * time.Millisecond,
			MaxOutputBytes: maxOutput,
		},
	}
}

func ValidateHelpTopic(topic string) error {
	if topic == "full" {
		return nil
	}
	kind, name, ok := strings.Cut(topic, "=")
	if !ok || name == "" {
		return apperrors.New(apperrors.E2005, "invalid help topic")
	}
	switch kind {
	case "decoder", "encoder", "filter", "muxer", "demuxer":
	default:
		return apperrors.New(apperrors.E2005, "invalid help topic kind")
	}
	if !topicNamePattern.MatchString(name) {
		return apperrors.New(apperrors.E2005, "invalid help topic name")
	}
	return nil
}

func commandSpecs(topic string) []commandSpec {
	return []commandSpec{
		{Key: "-L", Args: []string{"-L"}, Timeout: shortTimeout, Kind: "license"},
		{Key: "-license", Args: []string{"-license"}, Timeout: shortTimeout, Kind: "license"},
		{Key: "-h " + topic, Args: []string{"-h", topic}, Timeout: shortTimeout, Kind: "help"},
		{Key: "-version", Args: []string{"-version"}, Timeout: shortTimeout, Kind: "version"},
		{Key: "-muxers", Args: []string{"-muxers"}, Timeout: listTimeout, Kind: "muxers"},
		{Key: "-demuxers", Args: []string{"-demuxers"}, Timeout: listTimeout, Kind: "demuxers"},
		{Key: "-devices", Args: []string{"-devices"}, Timeout: listTimeout, Kind: "devices"},
		{Key: "-decoders", Args: []string{"-decoders"}, Timeout: listTimeout, Kind: "decoders"},
		{Key: "-encoders", Args: []string{"-encoders"}, Timeout: listTimeout, Kind: "encoders"},
		{Key: "-filters", Args: []string{"-filters"}, Timeout: listTimeout, Kind: "filters"},
		{Key: "-pix_fmts", Args: []string{"-pix_fmts"}, Timeout: listTimeout, Kind: "pixel_formats"},
		{Key: "-layouts", Args: []string{"-layouts"}, Timeout: listTimeout, Kind: "channel_layouts"},
		{Key: "-sample_fmts", Args: []string{"-sample_fmts"}, Timeout: listTimeout, Kind: "sample_formats"},
	}
}

func CommandKeys(topic string) []string {
	specs := commandSpecs(topic)
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.Key)
	}
	return keys
}

func (c *Collector) Collect(ctx context.Context, opts Options) (model.CapabilitySet, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.HelpTopic == "" {
		opts.HelpTopic = "full"
	}
	if err := ValidateHelpTopic(opts.HelpTopic); err != nil {
		return model.CapabilitySet{}, err
	}
	if opts.Installation.FFmpeg == "" {
		return model.CapabilitySet{}, apperrors.New(apperrors.E1002, "ffmpeg executable is required")
	}
	fi, err := os.Stat(opts.Installation.FFmpeg)
	if err != nil || fi.IsDir() {
		return model.CapabilitySet{}, apperrors.New(apperrors.E1002, "ffmpeg executable not found")
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o111 == 0 {
		return model.CapabilitySet{}, apperrors.New(apperrors.E1005, "ffmpeg is not executable")
	}

	specs := commandSpecs(opts.HelpTopic)
	fingerprint := capabilityFingerprint(opts.Installation, opts.HelpTopic, specs)
	cacheDir, err := resolveCacheDir(opts.CacheDir)
	if err != nil {
		return model.CapabilitySet{}, err
	}
	cachePath := filepath.Join(cacheDir, fingerprint+".json")

	warnings := []model.Warning{}
	if opts.Installation.FFplay == "" {
		warnings = append(warnings, newWarning(apperrors.W1001, "ffplay is missing", nil))
	}

	if !opts.NoCache && !opts.Refresh {
		cached, ok, cacheWarnings := readCache(cachePath, fingerprint, specs)
		warnings = mergeWarnings(warnings, cacheWarnings)
		if ok {
			cached.CacheHit = true
			if opts.Installation.FFplay == "" {
				cached.Warnings = ensureWarning(cached.Warnings, newWarning(apperrors.W1001, "ffplay is missing", nil))
			}
			cached.Warnings = mergeWarnings(cached.Warnings, warnings)
			return cached, nil
		}
	}

	set := newCapabilitySet(opts.Installation, fingerprint)
	results, raw, runWarnings := c.runCommands(ctx, opts.Installation, specs)
	warnings = mergeWarnings(warnings, runWarnings)
	c.buildSet(&set, specs, results, raw, opts)
	set.Warnings = mergeWarnings(set.Warnings, warnings)

	if !opts.NoCache {
		if writeWarning := writeCache(cachePath, set); writeWarning != nil {
			set.Warnings = ensureWarning(set.Warnings, *writeWarning)
		}
	}
	return set, nil
}

func (c *Collector) runCommands(ctx context.Context, installation model.Installation, specs []commandSpec) (map[string]model.CapabilityCommand, map[string]ffexec.Result, []model.Warning) {
	results := map[string]model.CapabilityCommand{}
	raw := map[string]ffexec.Result{}
	warnings := []model.Warning{}
	var mu sync.Mutex
	var group errgroup.Group
	group.SetLimit(4)

	for _, spec := range specs {
		spec := spec
		group.Go(func() error {
			cmd, result, warn := c.runOne(ctx, installation, spec)
			mu.Lock()
			results[spec.Key] = cmd
			raw[spec.Key] = result
			for _, item := range warn {
				warnings = ensureWarning(warnings, item)
			}
			mu.Unlock()
			return nil
		})
	}
	_ = group.Wait()
	return results, raw, warnings
}

func (c *Collector) runOne(ctx context.Context, installation model.Installation, spec commandSpec) (model.CapabilityCommand, ffexec.Result, []model.Warning) {
	runner := c.Runner
	runner.Timeout = spec.Timeout
	if runner.WaitDelay <= 0 {
		runner.WaitDelay = 500 * time.Millisecond
	}
	if runner.MaxOutputBytes <= 0 {
		runner.MaxOutputBytes = maxOutput
	}
	result, runErr := runner.Run(ctx, installation.FFmpeg, spec.Args...)
	cmd := model.CapabilityCommand{
		OK:            runErr == nil && result.ExitCode == 0,
		ExitCode:      result.ExitCode,
		StdoutSHA256:  result.StdoutSHA256,
		StderrSHA256:  result.StderrSHA256,
		DurationMs:    result.DurationMs,
		StdoutBytes:   result.StdoutTotalBytes,
		StderrBytes:   result.StderrTotalBytes,
		Truncated:     result.Truncated,
		StderrSummary: stderrSummary(result.Stderr),
	}
	if runErr != nil {
		cmd.Error = runErr.Error()
	} else if result.ExitCode != 0 {
		cmd.Error = fmt.Sprintf("command exited with code %d", result.ExitCode)
	}
	warnings := []model.Warning{}
	if !cmd.OK {
		warnings = append(warnings, newWarning(apperrors.W1002, "capability command failed", map[string]any{"command": spec.Key, "error": cmd.Error}))
	}
	if cmd.Truncated {
		warnings = append(warnings, newWarning(apperrors.W1002, "capability stdout truncated", map[string]any{"command": spec.Key}))
	}
	return cmd, result, warnings
}

func (c *Collector) buildSet(set *model.CapabilitySet, specs []commandSpec, results map[string]model.CapabilityCommand, raw map[string]ffexec.Result, opts Options) {
	set.Commands = results
	set.Version = parseVersion(raw["-version"].Stdout)
	set.License = parseLicense(raw["-L"].Stdout, raw["-license"].Stdout)

	for _, spec := range specs {
		result := raw[spec.Key]
		cmd := results[spec.Key]
		switch spec.Kind {
		case "help":
			set.Help[opts.HelpTopic] = model.HelpInfo{
				Topic:        opts.HelpTopic,
				Raw:          result.Stdout,
				StdoutSHA256: result.StdoutSHA256,
			}
			continue
		case "license", "version":
			continue
		}
		entries, unknown := parseList(spec.Kind, result.Stdout)
		assignCapabilityList(set, spec.Kind, entries)
		if cmd.OK && unknown > 0 {
			set.Warnings = ensureWarning(set.Warnings, newWarning(apperrors.W1002, "unparsed capability lines", map[string]any{"command": spec.Key, "count": unknown}))
		}
	}
}

func assignCapabilityList(set *model.CapabilitySet, kind string, entries []model.CapabilityEntry) {
	switch kind {
	case "muxers":
		set.Muxers = entries
	case "demuxers":
		set.Demuxers = entries
	case "devices":
		set.Devices = entries
	case "decoders":
		set.Decoders = entries
	case "encoders":
		set.Encoders = entries
	case "filters":
		set.Filters = entries
	case "pixel_formats":
		set.PixelFormats = entries
	case "channel_layouts":
		set.ChannelLayouts = entries
	case "sample_formats":
		set.SampleFormats = entries
	}
}

func newCapabilitySet(installation model.Installation, fingerprint string) model.CapabilitySet {
	return model.CapabilitySet{
		Schema:                  model.CapabilitiesSchema,
		InstallationID:          installation.ID,
		Fingerprint:             fingerprint,
		InstallationFingerprint: installation.Fingerprint,
		CreatedAt:               time.Now().UTC().Format(time.RFC3339),
		Commands:                map[string]model.CapabilityCommand{},
		Help:                    map[string]model.HelpInfo{},
		Muxers:                  []model.CapabilityEntry{},
		Demuxers:                []model.CapabilityEntry{},
		Devices:                 []model.CapabilityEntry{},
		Decoders:                []model.CapabilityEntry{},
		Encoders:                []model.CapabilityEntry{},
		Filters:                 []model.CapabilityEntry{},
		PixelFormats:            []model.CapabilityEntry{},
		ChannelLayouts:          []model.CapabilityEntry{},
		SampleFormats:           []model.CapabilityEntry{},
		Warnings:                []model.Warning{},
	}
}

func capabilityFingerprint(installation model.Installation, topic string, specs []commandSpec) string {
	parts := []string{
		model.CapabilitiesSchema,
		"schema-version=capability-commands-v1",
		"os=" + runtime.GOOS,
		"arch=" + runtime.GOARCH,
		"installation_id=" + installation.ID,
		"installation_fingerprint=" + installation.Fingerprint,
		"version=" + installation.Version,
		"version_stdout_sha256=" + installation.VersionStdoutSHA256,
		"license_stdout_sha256=" + installation.LicenseStdoutSHA256,
		"help_topic=" + topic,
	}
	keys := make([]string, 0, len(installation.BinarySHA))
	for key := range installation.BinarySHA {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		parts = append(parts, "binary_sha256["+key+"]="+installation.BinarySHA[key])
	}
	specKeys := make([]string, 0, len(specs))
	for _, spec := range specs {
		specKeys = append(specKeys, spec.Key)
	}
	parts = append(parts, "commands="+strings.Join(specKeys, "|"))
	return sha256Hex([]byte(strings.Join(parts, "\n")))
}

func resolveCacheDir(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "cannot determine cache directory").WithCause(err.Error())
	}
	return filepath.Join(base, "OneMoreFFmpeg", "cache", "capabilities"), nil
}

func readCache(path, fingerprint string, specs []commandSpec) (model.CapabilitySet, bool, []model.Warning) {
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
	for _, spec := range specs {
		if _, ok := cached.Commands[spec.Key]; !ok {
			return model.CapabilitySet{}, false, []model.Warning{newWarning(apperrors.W1003, "capability cache is incomplete", map[string]any{"path": path, "command": spec.Key})}
		}
	}
	return normalizeCachedSet(cached), true, nil
}

func normalizeCachedSet(set model.CapabilitySet) model.CapabilitySet {
	if set.Commands == nil {
		set.Commands = map[string]model.CapabilityCommand{}
	}
	if set.Help == nil {
		set.Help = map[string]model.HelpInfo{}
	}
	if set.Muxers == nil {
		set.Muxers = []model.CapabilityEntry{}
	}
	if set.Demuxers == nil {
		set.Demuxers = []model.CapabilityEntry{}
	}
	if set.Devices == nil {
		set.Devices = []model.CapabilityEntry{}
	}
	if set.Decoders == nil {
		set.Decoders = []model.CapabilityEntry{}
	}
	if set.Encoders == nil {
		set.Encoders = []model.CapabilityEntry{}
	}
	if set.Filters == nil {
		set.Filters = []model.CapabilityEntry{}
	}
	if set.PixelFormats == nil {
		set.PixelFormats = []model.CapabilityEntry{}
	}
	if set.ChannelLayouts == nil {
		set.ChannelLayouts = []model.CapabilityEntry{}
	}
	if set.SampleFormats == nil {
		set.SampleFormats = []model.CapabilityEntry{}
	}
	if set.Warnings == nil {
		set.Warnings = []model.Warning{}
	}
	return set
}

func writeCache(path string, set model.CapabilitySet) *model.Warning {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return warningPtr(apperrors.W1003, "cannot create capability cache directory", map[string]any{"path": dir, "cause": err.Error()})
	}
	data, err := json.MarshalIndent(set, "", "  ")
	if err != nil {
		return warningPtr(apperrors.W1003, "cannot encode capability cache", map[string]any{"path": path, "cause": err.Error()})
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return warningPtr(apperrors.W1003, "cannot write capability cache", map[string]any{"path": tmp, "cause": err.Error()})
	}
	_ = os.Remove(path)
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return warningPtr(apperrors.W1003, "cannot commit capability cache", map[string]any{"path": path, "cause": err.Error()})
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

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
