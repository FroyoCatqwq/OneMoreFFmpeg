package model

// ReleaseCatalogEntry is one discovered FFmpeg version.
type ReleaseCatalogEntry struct {
	Channel     string `json:"channel"`
	Version     string `json:"version"`
	Line        string `json:"line,omitempty"`
	Snapshot    string `json:"snapshot,omitempty"`
	ReleaseTag  string `json:"release_tag"`
	AssetName   string `json:"asset_name"`
	AssetURL    string `json:"asset_url"`
	PublishedAt string `json:"published_at"`
	Provider    string `json:"provider"`
}

// ReleaseCatalogData is the data object of `release list`.
type ReleaseCatalogData struct {
	Schema   string                `json:"schema"`
	Provider string                `json:"provider"`
	Channel  string                `json:"channel"`
	Days     int                   `json:"days"`
	Count    int                   `json:"count"`
	Source   string                `json:"source"`
	Entries  []ReleaseCatalogEntry `json:"entries"`
}
