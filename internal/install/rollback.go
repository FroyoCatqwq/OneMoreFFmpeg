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

// containedIn reports whether candidate is strictly inside root (never the
// root itself) after lexical cleaning. It rejects ".." prefix tricks without
// rejecting names like "..stable" that merely start with two dots.
func containedIn(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil || rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

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
	if !containedIn(appsRoot, target) {
		return "", apperrors.New(apperrors.E6007, "journal target is outside apps").WithDetails(map[string]any{"target": target})
	}
	quarantine := record.Details["quarantine"]
	currentTarget := record.Details["current_target"]
	original := record.Details["quarantine_original"]

	// Every path that comes from a journal must be confined to the data root.
	// Journals are ordinary files; trusting their strings let a tampered record
	// move arbitrary directories into apps/ or point `current` anywhere.
	quarantineRoot := filepath.Join(dataRoot, "quarantine")
	if quarantine != "" && !containedIn(quarantineRoot, quarantine) {
		return "", apperrors.New(apperrors.E6007, "journal quarantine path is outside the data root").WithDetails(map[string]any{"quarantine": quarantine})
	}
	if original != "" && !containedIn(appsRoot, original) {
		return "", apperrors.New(apperrors.E6007, "journal quarantine_original is outside apps").WithDetails(map[string]any{"original": original})
	}
	if currentTarget != "" && !containedIn(dataRoot, currentTarget) {
		return "", apperrors.New(apperrors.E6007, "journal current_target is outside the data root").WithDetails(map[string]any{"current_target": currentTarget})
	}

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
		if original != "" {
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
	current := filepath.Join(dataRoot, "current")
	if kind == model.CurrentMissing {
		return removeCurrent(dataRoot)
	}
	if target == "" {
		// The pointer existed but had no resolvable target (an empty pointer
		// file or a dangling link). Remove any link the failed switch left and
		// restore an empty pointer file, so `current` never points at an install
		// directory that no longer exists. Never delete a foreign entry here.
		if _, err := os.Lstat(current); err == nil {
			if !platform.IsLink(current) {
				return apperrors.New(apperrors.E5004, "current exists and is not a managed link")
			}
			if err := platform.RemoveLink(current); err != nil {
				return err
			}
		}
		return os.WriteFile(filepath.Join(dataRoot, "current.pointer"), []byte("\n"), 0o644)
	}
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
	current := filepath.Join(dataRoot, "current")
	// A plain file or directory at <root>/current was not created by us.
	// platform.RemoveLink would os.Remove it; never destroy foreign data while
	// rolling back a transaction that could not even take over `current`.
	if _, err := os.Lstat(current); err == nil {
		if !platform.IsLink(current) {
			return apperrors.New(apperrors.E5004, "current exists and is not a managed link").WithDetails(map[string]any{"path": current})
		}
		if err := platform.RemoveLink(current); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(filepath.Join(dataRoot, "current.pointer")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
