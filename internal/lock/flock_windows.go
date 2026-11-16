//go:build windows

package lock

import (
	"os"

	"golang.org/x/sys/windows"
)

// tryOSLock takes an exclusive byte-range lock on the whole file via LockFileEx.
// The kernel releases it when the handle closes (including on process crash).
func tryOSLock(f *os.File) (bool, error) {
	handle := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	err := windows.LockFileEx(
		handle,
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol,
	)
	if err == windows.ERROR_LOCK_VIOLATION || err == windows.ERROR_IO_PENDING {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func releaseOSLock(f *os.File) {
	handle := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	_ = windows.UnlockFileEx(handle, 0, 1, 0, ol)
}
