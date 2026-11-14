//go:build windows

package platform

import (
	"path/filepath"

	"golang.org/x/sys/windows"
)

// AvailableBytes returns the free bytes available to the caller on the volume
// containing path. It is best-effort: callers must tolerate an error.
func AvailableBytes(path string) (uint64, error) {
	root := filepath.VolumeName(path) + `\`
	pointer, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0, err
	}
	var freeAvailable uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &freeAvailable, nil, nil); err != nil {
		return 0, err
	}
	return freeAvailable, nil
}