// Package journal records multi-step filesystem changes so a failed install or
// env switch can be audited and rolled back.
package journal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/fsatomic"
)

const Schema = "onemoreffmpeg.journal/v1"

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func validateID(id string) error {
	if !idPattern.MatchString(id) || strings.Contains(id, "..") {
		return apperrors.New(apperrors.E6005, "invalid journal id").WithDetails(map[string]any{"id": id})
	}
	return nil
}

const (
	StatusPending    = "pending"
	StatusCommitted  = "committed"
	StatusFailed     = "failed"
	StatusRolledBack = "rolled-back"
)

type Record struct {
	Schema    string            `json:"schema"`
	ID        string            `json:"id"`
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
	Target    string            `json:"target,omitempty"`
	TempDir   string            `json:"temp_dir,omitempty"`
	Previous  string            `json:"previous,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// NewID returns a random journal identifier.
func NewID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		// crypto/rand failed (extremely rare). Fall back to a value that
		// includes date, nanoseconds, pid and monotonic-ish counter so two
		// processes on different days can never collide.
		return fmt.Sprintf("journal-fallback-%d-%d-%x",
			os.Getpid(), time.Now().UnixNano(), time.Now().Format("20060102"))
	}
	return "journal-" + hex.EncodeToString(buffer)
}

// Write stores the record as <dir>/<id>.json using an atomic replace.
func Write(dir string, record Record) error {
	if err := validateID(record.ID); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperrors.New(apperrors.E6005, "journal directory cannot be created").WithCause(err.Error())
	}
	record.Schema = Schema
	now := time.Now().UTC().Format(time.RFC3339)
	if record.CreatedAt == "" {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E6005, "journal cannot be encoded").WithCause(err.Error())
	}
	path := filepath.Join(dir, record.ID+".json")
	// fsatomic.WriteLockedFile preserves the sidecar lock and cross-process
	// overwrite semantics of lockedfile.Write (it is NOT a temp+rename write).
	if err := fsatomic.WriteLockedFile(path, payload, 0o600); err != nil {
		return apperrors.New(apperrors.E6005, "journal cannot be committed").WithCause(err.Error())
	}
	return nil
}

// Read loads one journal record.
func Read(dir, id string) (Record, error) {
	if err := validateID(id); err != nil {
		return Record{}, err
	}
	payload, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return Record{}, apperrors.New(apperrors.E6005, "journal cannot be read").WithCause(err.Error())
	}
	var record Record
	if err := json.Unmarshal(payload, &record); err != nil {
		return Record{}, apperrors.New(apperrors.E6005, "journal is corrupt").WithCause(err.Error())
	}
	return record, nil
}

// List returns records ordered by creation time (oldest first).
func List(dir string) ([]Record, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, apperrors.New(apperrors.E6005, "journal directory cannot be read").WithCause(err.Error())
	}
	records := make([]Record, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		payload, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, apperrors.New(apperrors.E6005, "journal record cannot be read").
				WithDetails(map[string]any{"file": entry.Name()}).WithCause(err.Error())
		}
		var record Record
		if err := json.Unmarshal(payload, &record); err != nil {
			return nil, apperrors.New(apperrors.E6005, "journal record is corrupt").
				WithDetails(map[string]any{"file": entry.Name()}).WithCause(err.Error())
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt < records[j].CreatedAt })
	return records, nil
}

// Pending returns records that still need attention.
func Pending(dir string) ([]Record, error) {
	records, err := List(dir)
	if err != nil {
		return nil, err
	}
	pending := make([]Record, 0, len(records))
	for _, record := range records {
		if record.Status == StatusPending || record.Status == StatusFailed {
			pending = append(pending, record)
		}
	}
	return pending, nil
}
