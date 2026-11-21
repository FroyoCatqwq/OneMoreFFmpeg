package platform

import (
	"fmt"
	"os"
	"path/filepath"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// LinkKind identifies the kind of pointer produced by CreateLink on this
// platform. The string values match the journal vocabulary; callers map them
// to their own DTO constants at the package boundary.
type LinkKind string

const (
	LinkKindJunction LinkKind = "directory_junction"
	LinkKindSymlink  LinkKind = "symlink"
)

// ReplaceLink creates the new link first, then swaps it in, keeping a backup
// so a failed swap restores the previous pointer. Temporary names are random
// so concurrent processes cannot delete each other's staging links.
func ReplaceLink(link, target string) (LinkKind, error) {
	directory := filepath.Dir(link)
	base := filepath.Base(link)
	temp, err := reservePath(directory, base+".new-*")
	if err != nil {
		return "", err
	}
	backup, err := reservePath(directory, base+".old-*")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = RemoveLink(temp)
		_ = RemoveLink(backup)
	}()
	if err := CreateLink(temp, target); err != nil {
		return "", err
	}
	kind := createdKind(temp)
	existed := false
	if _, err := os.Lstat(link); err == nil {
		if err := RenamePath(link, backup); err != nil {
			return "", apperrors.New(apperrors.E5004, "current pointer cannot be moved aside").WithCause(err.Error())
		}
		existed = true
	}
	if err := RenamePath(temp, link); err != nil {
		if existed {
			_ = RenamePath(backup, link)
		}
		return "", apperrors.New(apperrors.E5004, "current pointer cannot be committed").WithCause(err.Error())
	}
	return kind, nil
}

// createdKind reports the pointer kind actually produced for a link path.
func createdKind(path string) LinkKind {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return LinkKindSymlink
	}
	return LinkKindJunction
}

func reservePath(directory, pattern string) (string, error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", apperrors.New(apperrors.E5004, "temporary link path cannot be reserved").WithCause(err.Error())
	}
	path := file.Name()
	closeErr := file.Close()
	removeErr := os.Remove(path)
	if closeErr != nil || removeErr != nil {
		return "", apperrors.New(apperrors.E5004, "temporary link path cannot be released").WithCause(fmt.Sprint(closeErr, removeErr))
	}
	return path, nil
}
