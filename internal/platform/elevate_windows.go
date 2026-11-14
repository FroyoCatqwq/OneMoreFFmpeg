//go:build windows

package platform

import "golang.org/x/sys/windows"

// IsElevated reports whether the process token is elevated (running as
// Administrator). The built-in Administrator account runs elevated by default.
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}