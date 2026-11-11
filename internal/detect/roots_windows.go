//go:build windows

package detect

import "golang.org/x/sys/windows"

func defaultScanRoots(opts Options) []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil || mask == 0 {
		return []string{`C:\`}
	}
	roots := make([]string, 0, 26)
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		ptr, err := windows.UTF16PtrFromString(root)
		if err != nil {
			continue
		}
		kind := windows.GetDriveType(ptr)
		include := kind == windows.DRIVE_FIXED
		if opts.IncludeRemovable && kind == windows.DRIVE_REMOVABLE {
			include = true
		}
		if opts.IncludeNetwork && kind == windows.DRIVE_REMOTE {
			include = true
		}
		if include {
			roots = append(roots, root)
		}
	}
	if len(roots) == 0 {
		return []string{`C:\`}
	}
	return roots
}
