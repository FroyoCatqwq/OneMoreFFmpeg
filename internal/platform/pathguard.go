package platform

import (
	"os"
	"path/filepath"
	"strings"
)

// LexicallyInside reports whether candidate is strictly below root after
// cleaning, without following symlinks. It rejects the root itself, ".."
// escapes and names that merely begin with two dots (e.g. "..stable").
func LexicallyInside(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil || rel == "." || rel == ".." {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

// ResolvedInside resolves every existing component of candidate (following
// symlinks) and requires the result to stay strictly inside the resolved root.
// Use it before destructive operations on paths that came from untrusted input
// such as journal files: a purely lexical check is bypassed by a symlink placed
// under the data root.
//
// Missing trailing components are allowed (the operation often creates them);
// only components that exist are resolved. If root itself cannot be resolved
// (it does not exist), the caller must create it first.
func ResolvedInside(root, candidate string) bool {
	if !LexicallyInside(root, candidate) {
		return false
	}
	resolvedRoot, err := filepath.EvalSymlinks(filepath.Clean(root))
	if err != nil {
		// Root missing: fall back to the lexical result for the common case,
		// but only if no component of candidate is a symlink.
		return noSymlinkComponents(root, candidate)
	}
	resolvedCandidate, err := evalExistingPrefix(filepath.Clean(candidate))
	if err != nil {
		return false
	}
	return LexicallyInside(resolvedRoot, resolvedCandidate)
}

// NoSymlinkComponents reports whether the path from root down to candidate
// contains any symbolic link. It is used before creating/writing a path so a
// pre-planted symlink directory cannot redirect the write outside the root.
func NoSymlinkComponents(root, candidate string) bool {
	return noSymlinkComponents(root, candidate)
}

func noSymlinkComponents(root, candidate string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	current := filepath.Clean(root)
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if info, err := os.Lstat(current); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return false
			}
		} else if !os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// evalExistingPrefix resolves symlinks on the longest existing prefix of path.
// Missing trailing components are appended lexically.
func evalExistingPrefix(path string) (string, error) {
	clean := filepath.Clean(path)
	current := clean
	var missing []string
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}
