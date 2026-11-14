//go:build !windows

package platform

import "syscall"

// AvailableBytes returns the free bytes available to unprivileged users on the
// filesystem containing path. It is best-effort: callers must tolerate an error.
func AvailableBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}