// Package model contains internal JSON contract models.
package model

import "encoding/json"

const (
	ResultSchema         = "onemoreffmpeg.result/v1"
	ErrorSchema          = "onemoreffmpeg.error/v1"
	VersionSchema        = "onemoreffmpeg.version/v1"
	DetectSchema         = "onemoreffmpeg.detect/v1"
	ValidateSchema       = "onemoreffmpeg.validate/v1"
	CapabilitiesSchema   = "onemoreffmpeg.capabilities/v1"
	ProviderListSchema   = "onemoreffmpeg.provider-list/v1"
	ProviderAddSchema    = "onemoreffmpeg.provider-add/v1"
	ReleaseSchema        = "onemoreffmpeg.release/v1"
	ArtifactSchema       = "onemoreffmpeg.artifact/v1"
	InstallSchema        = "onemoreffmpeg.install/v1"
	EnvStatusSchema      = "onemoreffmpeg.env-status/v1"
	EnvSetSchema         = "onemoreffmpeg.env-set/v1"
	DoctorSchema         = "onemoreffmpeg.doctor/v1"
	UninstallSchema      = "onemoreffmpeg.uninstall/v1"
	JournalListSchema    = "onemoreffmpeg.journal-list/v1"
	JournalActionSchema  = "onemoreffmpeg.journal-action/v1"
	CapabilityListSchema = "onemoreffmpeg.capability-list/v1"
	ReleaseCatalogSchema = "onemoreffmpeg.release-catalog/v1"
)

var KnownDataSchemas = map[string]struct{}{
	VersionSchema:        {},
	DetectSchema:         {},
	ValidateSchema:       {},
	CapabilitiesSchema:   {},
	ProviderListSchema:   {},
	ProviderAddSchema:    {},
	ReleaseSchema:        {},
	ArtifactSchema:       {},
	InstallSchema:        {},
	EnvStatusSchema:      {},
	EnvSetSchema:         {},
	DoctorSchema:         {},
	UninstallSchema:      {},
	JournalListSchema:    {},
	JournalActionSchema:  {},
	CapabilityListSchema: {},
	ReleaseCatalogSchema: {},
}

type Result struct {
	Schema  string `json:"schema"`
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Data    any    `json:"data"`
}

type ErrorObject struct {
	Schema  string         `json:"schema"`
	OK      bool           `json:"ok"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
	Cause   string         `json:"cause,omitempty"`
}

type VersionData struct {
	Schema    string `json:"schema"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

type Installation struct {
	ID                  string            `json:"id"`
	RootDir             string            `json:"root_dir"`
	FFmpeg              string            `json:"ffmpeg"`
	FFprobe             string            `json:"ffprobe"`
	FFplay              string            `json:"ffplay"`
	Sources             []string          `json:"sources"`
	Status              string            `json:"status"`
	Version             string            `json:"version,omitempty"`
	Fingerprint         string            `json:"fingerprint,omitempty"`
	VersionStdoutSHA256 string            `json:"version_stdout_sha256,omitempty"`
	LicenseStdoutSHA256 string            `json:"license_stdout_sha256,omitempty"`
	Warnings            []Warning         `json:"warnings"`
	BinarySHA           map[string]string `json:"binary_sha256,omitempty"`
}

type Warning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type DetectError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type DetectData struct {
	Schema        string         `json:"schema"`
	Placeholder   bool           `json:"placeholder,omitempty"`
	Installations []Installation `json:"installations"`
	Errors        []DetectError  `json:"errors"`
}

type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type ValidateData struct {
	Schema         string    `json:"schema"`
	InstallationID string    `json:"installation_id"`
	Status         string    `json:"status"`
	Checks         []Check   `json:"checks"`
	Warnings       []Warning `json:"warnings"`
}

func (v ValidateData) MarshalJSON() ([]byte, error) {
	if v.Checks == nil {
		v.Checks = []Check{}
	}
	if v.Warnings == nil {
		v.Warnings = []Warning{}
	}
	type alias ValidateData
	return json.Marshal(alias(v))
}

func (i Installation) MarshalJSON() ([]byte, error) {
	if i.Sources == nil {
		i.Sources = []string{}
	}
	if i.Warnings == nil {
		i.Warnings = []Warning{}
	}
	type alias Installation
	return json.Marshal(alias(i))
}

func (d DetectData) MarshalJSON() ([]byte, error) {
	if d.Installations == nil {
		d.Installations = []Installation{}
	}
	if d.Errors == nil {
		d.Errors = []DetectError{}
	}
	type alias DetectData
	return json.Marshal(alias(d))
}

type CapabilityCommand struct {
	OK            bool   `json:"ok"`
	ExitCode      int    `json:"exit_code"`
	StdoutSHA256  string `json:"stdout_sha256"`
	StderrSHA256  string `json:"stderr_sha256"`
	DurationMs    int64  `json:"duration_ms"`
	StdoutBytes   int64  `json:"stdout_bytes,omitempty"`
	StderrBytes   int64  `json:"stderr_bytes,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
	StderrSummary string `json:"stderr_summary,omitempty"`
	Error         string `json:"error,omitempty"`
}

type CapabilityEntry struct {
	Name        string `json:"name"`
	Flags       string `json:"flags,omitempty"`
	Description string `json:"description,omitempty"`
	Raw         string `json:"raw"`
}

type VersionInfo struct {
	Raw           string `json:"raw"`
	Version       string `json:"version,omitempty"`
	Configuration string `json:"configuration,omitempty"`
}

type LicenseInfo struct {
	Raw     string `json:"raw"`
	Summary string `json:"summary,omitempty"`
}

type HelpInfo struct {
	Topic        string `json:"topic,omitempty"`
	Raw          string `json:"raw"`
	StdoutSHA256 string `json:"stdout_sha256,omitempty"`
	Found        bool   `json:"found"`
}

type CapabilitySet struct {
	Schema                  string                       `json:"schema"`
	InstallationID          string                       `json:"installation_id"`
	Fingerprint             string                       `json:"fingerprint"`
	InstallationFingerprint string                       `json:"installation_fingerprint,omitempty"`
	CreatedAt               string                       `json:"created_at,omitempty"`
	CacheHit                bool                         `json:"cache_hit,omitempty"`
	Commands                map[string]CapabilityCommand `json:"commands"`
	Version                 VersionInfo                  `json:"version"`
	License                 LicenseInfo                  `json:"license"`
	Muxers                  []CapabilityEntry            `json:"muxers"`
	Demuxers                []CapabilityEntry            `json:"demuxers"`
	Devices                 []CapabilityEntry            `json:"devices"`
	Decoders                []CapabilityEntry            `json:"decoders"`
	Encoders                []CapabilityEntry            `json:"encoders"`
	Filters                 []CapabilityEntry            `json:"filters"`
	PixelFormats            []CapabilityEntry            `json:"pixel_formats"`
	ChannelLayouts          []CapabilityEntry            `json:"channel_layouts"`
	SampleFormats           []CapabilityEntry            `json:"sample_formats"`
	Help                    map[string]HelpInfo          `json:"help"`
	Warnings                []Warning                    `json:"warnings"`
}

// Normalize replaces nil collections with empty ones so JSON output is
// stable (`[]` / `{}` instead of `null`).
func (c CapabilitySet) Normalize() CapabilitySet {
	if c.Commands == nil {
		c.Commands = map[string]CapabilityCommand{}
	}
	if c.Help == nil {
		c.Help = map[string]HelpInfo{}
	}
	if c.Muxers == nil {
		c.Muxers = []CapabilityEntry{}
	}
	if c.Demuxers == nil {
		c.Demuxers = []CapabilityEntry{}
	}
	if c.Devices == nil {
		c.Devices = []CapabilityEntry{}
	}
	if c.Decoders == nil {
		c.Decoders = []CapabilityEntry{}
	}
	if c.Encoders == nil {
		c.Encoders = []CapabilityEntry{}
	}
	if c.Filters == nil {
		c.Filters = []CapabilityEntry{}
	}
	if c.PixelFormats == nil {
		c.PixelFormats = []CapabilityEntry{}
	}
	if c.ChannelLayouts == nil {
		c.ChannelLayouts = []CapabilityEntry{}
	}
	if c.SampleFormats == nil {
		c.SampleFormats = []CapabilityEntry{}
	}
	if c.Warnings == nil {
		c.Warnings = []Warning{}
	}
	return c
}

func (c CapabilitySet) MarshalJSON() ([]byte, error) {
	type alias CapabilitySet
	return json.Marshal(alias(c.Normalize()))
}
