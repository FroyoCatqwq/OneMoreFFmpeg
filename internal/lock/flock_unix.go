//go:build !windows

package lock

import (
	"os"

	"golang.org/x/sys/unix"
)

// tryOSLock takes an advisory exclusive lock on f. It returns (false, nil)
// when another process holds the lock; the kernel releases it automatically
// when the holder's file descriptor closes (including on process crash).
func tryOSLock(f *os.File) (bool, error) {
	err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == unix.EWOULDBLOCK || err == unix.EAGAIN {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func releaseOSLock(f *os.File) {
	_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
