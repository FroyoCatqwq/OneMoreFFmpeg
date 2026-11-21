package install

import (
	"encoding/json"
	"os"
	"path/filepath"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/fsatomic"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const recordName = "install.json"

// record is the on-disk manifest, richer than the CLI data object.
type record struct {
	Schema       string            `json:"schema"`
	InstalledAt  string            `json:"installed_at"`
	AssetName    string            `json:"asset_name"`
	Installation model.InstallData `json:"installation"`
}

func writeRecord(dir string, value record) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E5008, "install record cannot be encoded").WithCause(err.Error())
	}
	return writeAtomic(filepath.Join(dir, recordName), payload)
}

func readRecord(dir string) (record, bool) {
	payload, err := os.ReadFile(filepath.Join(dir, recordName))
	if err != nil {
		return record{}, false
	}
	var value record
	if err := json.Unmarshal(payload, &value); err != nil {
		return record{}, false
	}
	return value, true
}

func writeLock(dir string, data model.InstallData) error {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E5009, "trio lock cannot be encoded").WithCause(err.Error())
	}
	return writeAtomic(filepath.Join(dir, "ffmpeg-trio.lock"), payload)
}

func writeAtomic(path string, payload []byte) error {
	// mode=0 keeps os.CreateTemp's historical 0600 permission (the old
	// implementation never Chmod'ed the temp file). fsatomic maps all failures
	// to E5009, matching this caller's contract.
	if err := fsatomic.WriteFile(path, payload, 0); err != nil {
		return apperrors.New(apperrors.E5009, "install metadata cannot be written").WithCause(err.Error())
	}
	return nil
}
