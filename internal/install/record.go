package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
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
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return apperrors.New(apperrors.E5009, "install metadata temp file cannot be created").WithCause(err.Error())
	}
	tempPath := temp.Name()
	_, writeErr := temp.Write(payload)
	closeErr := temp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tempPath)
		return apperrors.New(apperrors.E5009, "install metadata cannot be written").WithCause(fmt.Sprint(writeErr, closeErr))
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return apperrors.New(apperrors.E5009, "install metadata cannot be committed").WithCause(err.Error())
	}
	return nil
}
