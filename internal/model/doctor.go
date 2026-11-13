package model

// EnvStatusData is the data object of `env status`.
type EnvStatusData struct {
	Schema              string             `json:"schema"`
	CurrentType         string             `json:"current_type"`
	CurrentTarget       string             `json:"current_target"`
	PathContainsCurrent bool               `json:"path_contains_current"`
	ResolvedFFmpeg      string             `json:"resolved_ffmpeg"`
	Variables           map[string]*string `json:"variables"`
	Warnings            []Warning          `json:"warnings"`
}

// EnvSetData is the data object of `env set/unset/rollback`.
type EnvSetData struct {
	Schema            string   `json:"schema"`
	DryRun            bool     `json:"dry_run"`
	Applied           bool     `json:"applied"`
	Operations        []string `json:"operations"`
	JournalID         string   `json:"journal_id,omitempty"`
	RollbackSupported bool     `json:"rollback_supported"`
}

// DoctorData is the data object of `doctor`.
type DoctorData struct {
	Schema         string    `json:"schema"`
	Healthy        bool      `json:"healthy"`
	Checks         []Check   `json:"checks"`
	Warnings       []Warning `json:"warnings"`
	JournalPending []string  `json:"journal_pending"`
}

// UninstallData is the data object of `uninstall`.
type UninstallData struct {
	Schema    string `json:"schema"`
	RootDir   string `json:"root_dir"`
	DryRun    bool   `json:"dry_run"`
	Applied   bool   `json:"applied"`
	JournalID string `json:"journal_id,omitempty"`
}
