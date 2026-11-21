package install

import (
	"os"
	"path/filepath"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

// InspectCurrent returns the current pointer kind and its target (if known).
func InspectCurrent(dataRoot string) (string, string) {
	current := filepath.Join(dataRoot, "current")
	if _, err := os.Lstat(current); err == nil && platform.IsLink(current) {
		if target, err := filepath.EvalSymlinks(current); err == nil {
			return linkKind(current), target
		}
	}
	if body, err := os.ReadFile(filepath.Join(dataRoot, "current.pointer")); err == nil {
		return model.CurrentPointerFile, strings.TrimSpace(string(body))
	}
	return currentType(dataRoot, ""), ""
}

func linkKind(current string) string {
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return model.CurrentSymlink
	}
	return model.CurrentJunction
}

func currentType(dataRoot, target string) string {
	current := filepath.Join(dataRoot, "current")
	pointer := filepath.Join(dataRoot, "current.pointer")
	info, err := os.Lstat(current)
	if err == nil {
		// Only a real link is a managed pointer. A plain file or directory that
		// happens to sit at <root>/current is NOT ours: EvalSymlinks succeeds
		// for those too, which used to make doctor/env report a healthy junction
		// and made rollback treat the foreign entry as removable.
		if platform.IsLink(current) {
			if resolved, evalErr := filepath.EvalSymlinks(current); evalErr == nil {
				if target != "" {
					expected, _ := filepath.EvalSymlinks(target)
					if expected == resolved {
						return linkKind(current)
					}
				}
				return linkKind(current)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return model.CurrentSymlink
			}
			return model.CurrentJunction
		}
		if _, pointerErr := os.Stat(pointer); pointerErr == nil {
			return model.CurrentPointerFile
		}
		return model.CurrentMissing
	}
	if _, err := os.Stat(pointer); err == nil {
		return model.CurrentPointerFile
	}
	return model.CurrentMissing
}

func switchCurrent(dataRoot, target string) (string, error) {
	current := filepath.Join(dataRoot, "current")
	if _, err := os.Lstat(current); err == nil && !platform.IsLink(current) {
		return "", apperrors.New(apperrors.E5004, "current exists and is not a managed link").WithDetails(map[string]any{"path": current})
	}
	linkKind, err := platform.ReplaceLink(current, target)
	if err == nil {
		_ = os.Remove(filepath.Join(dataRoot, "current.pointer"))
		// Map the platform-level link kind into the journal vocabulary at this
		// package boundary; platform no longer depends on model.
		kind := model.CurrentJunction
		if linkKind == platform.LinkKindSymlink {
			kind = model.CurrentSymlink
		}
		return kind, nil
	}
	// Last resort: remove any broken link, then use a pointer file.
	if removeErr := platform.RemoveLink(current); removeErr != nil {
		return "", apperrors.New(apperrors.E5004, "broken current pointer cannot be removed").WithCause(removeErr.Error())
	}
	if writeErr := os.WriteFile(filepath.Join(dataRoot, "current.pointer"), []byte(target+"\n"), 0o644); writeErr != nil {
		// v0.6.x: this used to report err (the ReplaceLink failure) instead of
		// the write failure, so the diagnostic pointed at the wrong operation.
		return "", apperrors.New(apperrors.E5004, "current pointer cannot be written").WithCause(writeErr.Error())
	}
	return model.CurrentPointerFile, nil
}
