package model

// InstallBinaries lists the hashes recorded for an installed trio.
type InstallBinaries struct {
	FFmpegSHA256  string `json:"ffmpeg_sha256"`
	FFprobeSHA256 string `json:"ffprobe_sha256"`
	FFplaySHA256  string `json:"ffplay_sha256,omitempty"`
}

// InstallData is the data object of `install`.
type InstallData struct {
	Schema          string          `json:"schema"`
	InstallationID  string          `json:"installation_id"`
	RootDir         string          `json:"root_dir"`
	CurrentPointer  string          `json:"current_pointer"`
	Provider        string          `json:"provider"`
	Repository      string          `json:"repository"`
	ReleaseTag      string          `json:"release_tag"`
	AssetSHA256     string          `json:"asset_sha256"`
	Binaries        InstallBinaries `json:"binaries"`
	LicenseHint     string          `json:"license_hint"`
	Redistributable bool            `json:"redistributable"`
	LockPath        string          `json:"lock_path,omitempty"`
	JournalID       string          `json:"journal_id,omitempty"`
	DryRun          bool            `json:"dry_run"`
}

// Current pointer kinds.
const (
	CurrentJunction    = "directory_junction"
	CurrentSymlink     = "symlink"
	CurrentPointerFile = "pointer_file"
	CurrentMissing     = "missing"
)
