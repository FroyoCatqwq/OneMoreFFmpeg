package model

import "encoding/json"

// ProviderSummary is one row of `provider list`.
type ProviderSummary struct {
	ID              string   `json:"id"`
	Repository      string   `json:"repository"`
	Channels        []string `json:"channels"`
	Variants        []string `json:"variants"`
	Linkages        []string `json:"linkages"`
	Redistributable bool     `json:"redistributable"`
}

// ProviderListData is the data object of `provider list`.
type ProviderListData struct {
	Schema    string            `json:"schema"`
	Providers []ProviderSummary `json:"providers"`
}

// ProviderAddData is the data object of `provider add`.
type ProviderAddData struct {
	Schema string `json:"schema"`
	ID     string `json:"id"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
	Status string `json:"status"`
}

// Release is the pinned result of `resolve`. It never carries a floating
// `latest` tag unless the provider records an immutable release id.
type Release struct {
	Schema          string    `json:"schema"`
	Provider        string    `json:"provider"`
	Repository      string    `json:"repository"`
	Channel         string    `json:"channel"`
	ReleaseTag      string    `json:"release_tag"`
	ReleaseID       string    `json:"release_id,omitempty"`
	PublishedAt     string    `json:"published_at"`
	PinnedAt        string    `json:"pinned_at,omitempty"`
	AssetName       string    `json:"asset_name"`
	AssetURL        string    `json:"asset_url"`
	AssetSHA256     string    `json:"asset_sha256"`
	ChecksumsURL    string    `json:"checksums_url,omitempty"`
	ChecksumsSHA256 string    `json:"checksums_sha256,omitempty"`
	Variant         string    `json:"variant"`
	Linkage         string    `json:"linkage"`
	Target          string    `json:"target"`
	LicenseHint     string    `json:"license_hint"`
	Redistributable bool      `json:"redistributable"`
	Warnings        []Warning `json:"warnings,omitempty"`
}

// Artifact is the local record of a verified download.
type Artifact struct {
	Schema       string    `json:"schema"`
	ArtifactID   string    `json:"artifact_id"`
	Provider     string    `json:"provider"`
	ReleaseTag   string    `json:"release_tag"`
	AssetName    string    `json:"asset_name"`
	AssetSHA256  string    `json:"asset_sha256"`
	LocalPath    string    `json:"local_path"`
	Verified     bool      `json:"verified"`
	DownloadedAt string    `json:"downloaded_at"`
	Warnings     []Warning `json:"warnings,omitempty"`
}

// Normalize replaces nil slices with empty ones so JSON output is stable.
func (p ProviderSummary) Normalize() ProviderSummary {
	if p.Channels == nil {
		p.Channels = []string{}
	}
	if p.Variants == nil {
		p.Variants = []string{}
	}
	if p.Linkages == nil {
		p.Linkages = []string{}
	}
	return p
}

// Normalize replaces nil slices with empty ones so JSON output is stable.
// The input is not modified; callers may keep sharing it safely.
func (p ProviderListData) Normalize() ProviderListData {
	out := p
	if out.Providers == nil {
		out.Providers = []ProviderSummary{}
		return out
	}
	providers := make([]ProviderSummary, len(out.Providers))
	for index, provider := range out.Providers {
		providers[index] = provider.Normalize()
	}
	out.Providers = providers
	return out
}

func (p ProviderSummary) MarshalJSON() ([]byte, error) {
	type alias ProviderSummary
	return json.Marshal(alias(p.Normalize()))
}

func (p ProviderListData) MarshalJSON() ([]byte, error) {
	type alias ProviderListData
	return json.Marshal(alias(p.Normalize()))
}
