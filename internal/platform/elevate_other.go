//go:build !windows

package platform

import "os"

// IsElevated reports whether the process runs as root (euid 0).
func IsElevated() bool { return os.Geteuid() == 0 }