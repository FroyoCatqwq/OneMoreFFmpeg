// Package model contains internal JSON contract models.
package model

import "encoding/json"

const (
	ResultSchema       = "onemoreffmpeg.result/v1"
	ErrorSchema        = "onemoreffmpeg.error/v1"
	VersionSchema      = "onemoreffmpeg.version/v1"
	DetectSchema       = "onemoreffmpeg.detect/v1"
	ValidateSchema     = "onemoreffmpeg.validate/v1"
	CapabilitiesSchema = "onemoreffmpeg.capabilities/v1"
	ProviderListSchema = "onemoreffmpeg.provider-list/v1"
	ProviderAddSchema  = "onemoreffmpeg.provider-add/v1"
	ReleaseSchema      = "onemoreffmpeg.release/v1"
	ArtifactSchema     = "onemoreffmpeg.artifact/v1"
	InstallSchema      = "onemoreffmpeg.install/v1"
	EnvStatusSchema    = "onemoreffmpeg.env-status/v1"
	EnvSetSchema       = "onemoreffmpeg.env-set/v1"
	DoctorSchema       = "onemoreffmpeg.doctor/v1"
)

var KnownDataSchemas = map[string]struct{}{
	VersionSchema:      {},
	DetectSchema:       {},
	ValidateSchema:     {},
	CapabilitiesSchema: {},
	ProviderListSchema: {},
	ProviderAddSchema:  {},
	ReleaseSchema:      {},
	ArtifactSchema:     {},
	InstallSchema:      {},
	EnvStatusSchema:    {},
	EnvSetSchema:       {},
	DoctorSchema:       {},
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
	ID          string            `json:"id"`
	RootDir     string            `json:"root_dir"`
	FFmpeg      string            `json:"ffmpeg"`
	FFprobe     string            `json:"ffprobe"`
	FFplay      string            `json:"ffplay"`
	Sources     []string          `json:"sources"`
	Status      string            `json:"status"`
	Version     string            `json:"version,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Warnings    []Warning         `json:"warnings"`
	BinarySHA   map[string]string `json:"binary_sha256,omitempty"`
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
