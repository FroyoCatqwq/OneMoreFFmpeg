package install

import (
	"os"
	"path/filepath"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

// Quarantine moves a managed installation aside so uninstall is reversible.
func Quarantine(dataRoot, root, journalID string) (string, error) {
	appsRoot := filepath.Join(dataRoot, "apps")
	if rel, err := filepath.Rel(appsRoot, root); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", apperrors.New(apperrors.E6007, "refusing to quarantine a path outside apps").WithDetails(map[string]any{"root": root})
	}
	quarantine := filepath.Join(dataRoot, "quarantine", journalID, filepath.Base(root))
	if err := os.MkdirAll(filepath.Dir(quarantine), 0o755); err != nil {
		return "", apperrors.New(apperrors.E5007, "quarantine directory cannot be created").WithCause(err.Error())
	}
	if err := platform.RenamePath(root, quarantine); err != nil {
		return "", apperrors.New(apperrors.E5007, "installation cannot be quarantined").WithCause(err.Error())
	}
	return quarantine, nil
}

// Rollback restores the state captured by an install/uninstall journal record.
func Rollback(dataRoot string, record journal.Record) (string, error) {
	target := record.Target
	if target == "" {
		return "", apperrors.New(apperrors.E6005, "journal has no target")
	}
	appsRoot := filepath.Join(dataRoot, "apps")
	if rel, err := filepath.Rel(appsRoot, target); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", apperrors.New(apperrors.E6007, "journal target is outside apps").WithDetails(map[string]any{"target": target})
	}
	quarantine := record.Details["quarantine"]
	currentTarget := record.Details["current_target"]

	switch record.Action {
	case "uninstall":
		if _, err := os.Stat(target); err == nil {
			return "target already exists; nothing to restore", nil
		}
		if quarantine == "" {
			return "", apperrors.New(apperrors.E6005, "journal has no quarantine path")
		}
		if _, err := os.Stat(quarantine); err != nil {
			return "", apperrors.New(apperrors.E6005, "quarantined installation is missing").WithCause(err.Error())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", apperrors.New(apperrors.E5005, "install parent cannot be restored").WithCause(err.Error())
		}
		if err := platform.RenamePath(quarantine, target); err != nil {
			return "", apperrors.New(apperrors.E5005, "installation cannot be restored").WithCause(err.Error())
		}
		if currentTarget != "" {
			if err := restoreCurrent(dataRoot, currentTarget, record.Details["current_kind"]); err != nil {
				return "", err
			}
		}
		return "installation restored from " + quarantine, nil
	case "install":
		restorePath := target
		if original := record.Details["quarantine_original"]; original != "" {
			restorePath = original
		}
		if _, err := os.Stat(target); err == nil {
			if err := os.RemoveAll(target); err != nil {
				return "", apperrors.New(apperrors.E5007, "installed directory cannot be removed").WithCause(err.Error())
			}
		}
		if quarantine != "" {
			if _, err := os.Stat(quarantine); err == nil {
				if err := os.MkdirAll(filepath.Dir(restorePath), 0o755); err != nil {
					return "", apperrors.New(apperrors.E5005, "previous install parent cannot be restored").WithCause(err.Error())
				}
				if err := platform.RenamePath(quarantine, restorePath); err != nil {
					return "", apperrors.New(apperrors.E5005, "previous installation cannot be restored").WithCause(err.Error())
				}
			}
		}
		switch {
		case record.Details["current_existed"] == "0":
			if err := restoreCurrentState(dataRoot, "", model.CurrentMissing); err != nil {
				return "", err
			}
		case currentTarget != "" && currentTarget == target && restorePath != target:
			if err := restoreCurrent(dataRoot, restorePath, record.Details["current_kind"]); err != nil {
				return "", err
			}
		case currentTarget != "" && currentTarget != target:
			if err := restoreCurrent(dataRoot, currentTarget, record.Details["current_kind"]); err != nil {
				return "", err
			}
		}
		return "install rolled back", nil
	default:
		return "", apperrors.New(apperrors.E7001, "journal action cannot be rolled back").WithDetails(map[string]any{"action": record.Action})
	}
}

// quarantineExistingInstalls moves the previous install for the same
// provider/tag aside. The versioned directory includes the asset hash, so a
// forced upgrade has a different target directory than the previous version.
func quarantineExistingInstalls(dataRoot, targetDir, journalID string, force bool) (string, string, error) {
	if info, err := os.Stat(targetDir); err == nil && info.IsDir() && !force {
		return "", "", apperrors.New(apperrors.E5003, "install directory already exists").WithDetails(map[string]any{"dir": targetDir})
	}
	if !force {
		return "", "", nil
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(targetDir), "*"))
	if err != nil {
		return "", "", apperrors.New(apperrors.E5007, "cannot inspect existing installs").WithCause(err.Error())
	}
	candidates := make([]string, 0, 1)
	if info, statErr := os.Stat(targetDir); statErr == nil && info.IsDir() {
		candidates = append(candidates, targetDir)
	}
	for _, match := range matches {
		if filepath.Clean(match) == filepath.Clean(targetDir) {
			continue
		}
		info, statErr := os.Stat(match)
		if statErr != nil || !info.IsDir() {
			continue
		}
		if _, recordErr := os.Stat(filepath.Join(match, "install.json")); recordErr == nil {
			candidates = append(candidates, match)
		}
	}
	if len(candidates) > 1 {
		return "", "", apperrors.New(apperrors.E5003, "multiple existing installs; remove them explicitly before --force").WithDetails(map[string]any{"dir": filepath.Dir(targetDir)})
	}
	if len(candidates) == 0 {
		return "", "", nil
	}
	original := candidates[0]
	quarantine := filepath.Join(dataRoot, "quarantine", journalID, filepath.Base(original))
	if err := os.MkdirAll(filepath.Dir(quarantine), 0o755); err != nil {
		return "", "", apperrors.New(apperrors.E5007, "quarantine directory cannot be created").WithCause(err.Error())
	}
	if err := platform.RenamePath(original, quarantine); err != nil {
		return "", "", apperrors.New(apperrors.E5007, "existing install cannot be quarantined").WithCause(err.Error())
	}
	return quarantine, original, nil
}

func restoreCurrent(dataRoot, target, kind string) error {
	if kind == model.CurrentMissing || target == "" {
		return removeCurrent(dataRoot)
	}
	current := filepath.Join(dataRoot, "current")
	if kind == model.CurrentPointerFile {
		_ = platform.RemoveLink(current)
		return os.WriteFile(filepath.Join(dataRoot, "current.pointer"), []byte(target+"\n"), 0o644)
	}
	if _, err := os.Lstat(current); err == nil && !platform.IsLink(current) {
		return apperrors.New(apperrors.E5004, "current exists and is not a managed link")
	}
	_ = os.Remove(filepath.Join(dataRoot, "current.pointer"))
	_, err := platform.ReplaceLink(current, target)
	return err
}

func restoreCurrentState(dataRoot, target, kind string) error {
	return restoreCurrent(dataRoot, target, kind)
}

func removeCurrent(dataRoot string) error {
	if err := platform.RemoveLink(filepath.Join(dataRoot, "current")); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dataRoot, "current.pointer")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
