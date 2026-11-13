//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
)

// IsLink reports whether path is a symlink or junction (reparse point).
func IsLink(path string) bool {
	pointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attributes, err := windows.GetFileAttributes(pointer)
	if err != nil {
		return false
	}
	return attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
