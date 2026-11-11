//go:build !windows

package detect

func defaultScanRoots(opts Options) []string {
	return []string{"/", "/home", "/opt", "/usr/local"}
}
