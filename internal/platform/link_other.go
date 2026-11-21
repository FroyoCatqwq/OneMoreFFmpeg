//go:build !windows

package platform

import (
	"os"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// CreateLink creates a relative or absolute symlink.
func CreateLink(link, target string) error {
	if err := os.Symlink(target, link); err != nil {
		return apperrors.New(apperrors.E5004, "symlink cannot be created").WithCause(err.Error())
	}
	return nil
}

// createOneLink creates a symlink at an exact (previously reserved) name.
// Unix symlinks carry no file/dir type bit and dangling targets are legal, so
// kind is validated by ReserveLink but does not change the creation path.
func createOneLink(link, target string, _ LinkTargetKind) error {
	if err := os.Symlink(target, link); err != nil {
		return apperrors.New(apperrors.E5004, "symlink cannot be created").WithCause(err.Error())
	}
	return nil
}

// RemoveLink removes a symlink or file without following it.
func RemoveLink(link string) error {
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		return apperrors.New(apperrors.E5004, "link cannot be removed").WithCause(err.Error())
	}
	return nil
}
