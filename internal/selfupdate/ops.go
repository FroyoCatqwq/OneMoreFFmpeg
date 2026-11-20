package selfupdate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

// FileOps abstracts the filesystem operations used by a self-update. It lets
// tests run the exact replace/restore transaction on temp files (including on
// Windows) without touching the process' own running executable.
type FileOps interface {
	Stat(path string) (os.FileInfo, error)
	Rename(from, to string) error
	Remove(path string) error
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadFile(path string) ([]byte, error)
}

type realFileOps struct{}

func (realFileOps) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }
func (realFileOps) Rename(from, to string) error          { return platform.RenamePath(from, to) }
func (realFileOps) Remove(path string) error              { return os.Remove(path) }
func (realFileOps) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}
func (realFileOps) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// updateState is a small journal of the self-update transaction. It is written
// atomically (temp + rename) so a crash leaves the previous state readable.
type updateState struct {
	Version    string `json:"version"`
	Stage      string `json:"stage"` // staged|verified|backup_created|replaced|committed
	Executable string `json:"executable"`
	Backup     string `json:"backup,omitempty"`
	Staging    string `json:"staging,omitempty"`
}

func statePathFor(executable string) string {
	return filepath.Join(filepath.Dir(executable), ".one-more-ffmpeg-update.json")
}

func writeState(ops FileOps, executable string, state updateState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	path := statePathFor(executable)
	tmp := path + ".tmp"
	if err := ops.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write update state: %w", err)
	}
	if err := ops.Rename(tmp, path); err != nil {
		_ = ops.Remove(tmp)
		return fmt.Errorf("commit update state: %w", err)
	}
	return nil
}

func readState(ops FileOps, executable string) (*updateState, error) {
	data, err := ops.ReadFile(statePathFor(executable))
	if err != nil {
		return nil, err
	}
	var state updateState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode update state: %w", err)
	}
	return &state, nil
}

// Recover inspects a leftover update state after a crash. A new binary that
// passes its staged SHA-256 check is committed; otherwise the old binary is
// restored from backup. It returns a human-readable disposition.
func (o Options) Recover() (string, error) {
	o = o.normalized()
	ops := o.FileOps
	if ops == nil {
		ops = realFileOps{}
	}
	state, err := readState(ops, o.Executable)
	if err != nil {
		if os.IsNotExist(err) {
			return "none", nil
		}
		return "", err
	}
	switch state.Stage {
	case "committed":
		_ = ops.Remove(statePathFor(o.Executable))
		return "already-committed", nil
	case "replaced":
		// The verified new binary is already in place; a crash here happened
		// after the irreversible rename. Commit it: drop the old backup and
		// the state file. Rolling back here would discard a complete update.
		if state.Backup != "" {
			if err := ops.Remove(state.Backup); err != nil && !os.IsNotExist(err) {
				return "", fmt.Errorf("remove stale update backup %s: %w", state.Backup, err)
			}
		}
		_ = ops.Remove(statePathFor(o.Executable))
		return "committed-replacement", nil
	case "backup_created":
		// Old exe was moved aside but the new one was never installed: restore.
		if state.Backup != "" {
			if err := ops.Rename(state.Backup, o.Executable); err != nil {
				return "", fmt.Errorf("recover previous executable from %s: %w", state.Backup, err)
			}
		}
		_ = ops.Remove(statePathFor(o.Executable))
		return "restored-previous", nil
	default:
		// staged/verified never touched the running executable; clean up.
		if state.Staging != "" {
			_ = ops.Remove(state.Staging)
		}
		_ = ops.Remove(statePathFor(o.Executable))
		return "cleaned-staging", nil
	}
}
