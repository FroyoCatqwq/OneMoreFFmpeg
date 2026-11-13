package model

// JournalEntry is a compact view of a transaction record.
type JournalEntry struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Target    string `json:"target,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// JournalListData is the data object of `journal list`.
type JournalListData struct {
	Schema  string         `json:"schema"`
	Records []JournalEntry `json:"records"`
	Pending []string       `json:"pending"`
}

// JournalActionData is the data object of `journal rollback`.
type JournalActionData struct {
	Schema    string `json:"schema"`
	JournalID string `json:"journal_id"`
	Action    string `json:"action"`
	Applied   bool   `json:"applied"`
	Detail    string `json:"detail"`
}
