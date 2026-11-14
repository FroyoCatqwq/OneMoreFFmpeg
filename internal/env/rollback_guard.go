package env

import (
	"os"
	"path/filepath"
	"strings"
)

// safeRollbackPath reports whether a path recorded in a journal may be touched
// during a rollback.
//
// A journal is an ordinary file under the data root, so anyone who can write one
// can put anything into Details. Trusting those strings let `--apply` restore a
// "shim" into an arbitrary directory and rewrite an arbitrary file as a shell
// profile. Both are now rejected unless the target is where the plan itself
// would have put it:
//
//   - anything inside the data root (user-scope shims live in <root>/bin);
//   - the two shell profiles the plan can legitimately record
//     (~/.profile for user scope, /etc/profile.d/onemoreffmpeg.sh for system).
func safeRollbackPath(path, dataRoot string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return false
	}
	if dataRoot != "" {
		root := filepath.Clean(dataRoot)
		if rel, err := filepath.Rel(root, clean); err == nil {
			if rel == "." {
				return true
			}
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return true
			}
		}
	}
	return safeShellProfile(clean)
}

// safeShellProfile is the narrower check for callers that only know the profile
// path (the uninstall rollback path has no data root in scope).
func safeShellProfile(path string) bool {
	clean := filepath.Clean(path)
	if clean == "" || !filepath.IsAbs(clean) {
		return false
	}
	if samePath(clean, "/etc/profile.d/onemoreffmpeg.sh") {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil {
		if samePath(clean, filepath.Join(home, ".profile")) {
			return true
		}
	}
	return false
}

