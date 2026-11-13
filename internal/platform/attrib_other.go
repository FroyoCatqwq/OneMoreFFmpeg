//go:build !windows

package platform

import "os"

// IsLink reports whether path is a symlink.
func IsLink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
