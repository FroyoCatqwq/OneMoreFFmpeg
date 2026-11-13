package model

// CapabilityListData is the data object of `capabilities --only <kind>`.
type CapabilityListData struct {
	Schema         string            `json:"schema"`
	InstallationID string            `json:"installation_id"`
	Fingerprint    string            `json:"fingerprint"`
	Kind           string            `json:"kind"`
	Entries        []CapabilityEntry `json:"entries"`
	Command        string            `json:"command"`
	Warnings       []Warning         `json:"warnings"`
}
