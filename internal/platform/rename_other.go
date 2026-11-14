//go:build !windows

package platform

import "os"

// RenamePath renames a file or directory.
func RenamePath(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}