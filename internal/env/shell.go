package env

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/fsatomic"
)

// --- PATH handling -------------------------------------------------------

// pathEntryEqual compares PATH entries with case folding only on Windows:
// Unix filesystems are case-sensitive, so EqualFold there would treat
// /home/u/.LOCAL/bin and /home/u/.local/bin as the same entry.
func pathEntryEqual(left, right string) bool {
	left, right = filepath.Clean(left), filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func appendPath(previous, entry string) string {
	for _, part := range filepath.SplitList(previous) {
		if pathEntryEqual(part, entry) {
			return previous
		}
	}
	if previous == "" {
		return entry
	}
	return previous + string(os.PathListSeparator) + entry
}

func removePath(previous, entry string) (string, bool) {
	parts := filepath.SplitList(previous)
	kept := make([]string, 0, len(parts))
	removed := false
	for _, part := range parts {
		if pathEntryEqual(part, entry) {
			removed = true
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, string(os.PathListSeparator)), removed
}

// --- location resolution --------------------------------------------------

func shimDirectory(system bool) (string, error) {
	if system {
		return "/usr/local/bin", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "home directory cannot be determined").WithCause(err.Error())
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func shellProfile(system bool) (string, error) {
	if system {
		return "/etc/profile.d/onemoreffmpeg.sh", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "home directory cannot be determined").WithCause(err.Error())
	}
	return filepath.Join(home, ".profile"), nil
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'" }

// --- shell rc file handling ----------------------------------------------

func readShellRC(profile string) (bool, string, error) {
	body, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return false, "", apperrors.New(apperrors.E5009, "shell rc cannot be read").WithCause(err.Error())
	}
	return strings.Contains(string(body), shellRCMarkerStart), string(body), nil
}

func currentShellRCBlock(body string) string {
	start := strings.Index(body, shellRCMarkerStart)
	end := strings.Index(body, shellRCMarkerEnd)
	if start < 0 || end < start {
		return ""
	}
	return strings.Trim(body[start:end+len(shellRCMarkerEnd)], "\n")
}

func stripShellRCBlock(body string) string {
	start := strings.Index(body, shellRCMarkerStart)
	end := strings.Index(body, shellRCMarkerEnd)
	if start < 0 || end < start {
		return body
	}
	return body[:start] + body[end+len(shellRCMarkerEnd):]
}

func updateShellRC(profile, body, desired, journalDir string, details map[string]string) error {
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		return apperrors.New(apperrors.E5009, "shell rc backup directory cannot be created").WithCause(err.Error())
	}
	backup, err := os.CreateTemp(journalDir, "profile-*.bak")
	if err != nil {
		return apperrors.New(apperrors.E5009, "shell rc backup cannot be created").WithCause(err.Error())
	}
	backupPath := backup.Name()
	_, writeErr := backup.WriteString(body)
	closeErr := backup.Close()
	if writeErr != nil || closeErr != nil {
		return apperrors.New(apperrors.E5009, "shell rc backup cannot be written").WithCause(fmt.Sprint(writeErr, closeErr))
	}
	details["shell_rc_backup"] = backupPath
	stripped := stripShellRCBlock(body)
	updated := stripped
	if desired != "" {
		updated = strings.TrimRight(stripped, "\n") + desired
	}
	if err := writeFileAtomic(profile, []byte(updated), 0o644); err != nil {
		return apperrors.New(apperrors.E5009, "shell rc cannot be written").WithCause(err.Error())
	}
	return nil
}

func restoreShellRC(backup, profile string) error {
	content, err := os.ReadFile(backup)
	if err != nil {
		return apperrors.New(apperrors.E5009, "shell rc backup cannot be read").WithCause(err.Error())
	}
	return writeFileAtomic(profile, content, 0o644)
}

// --- shim files -----------------------------------------------------------

func writeShim(path, target string) error {
	body := "#!/bin/sh\n" + shimMarker + "\nexec " + shellQuote(target) + " \"$@\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperrors.New(apperrors.E6002, "shim directory cannot be created").WithCause(err.Error())
	}
	return writeFileAtomic(path, []byte(body), 0o755)
}

// validManagedShimBody reports whether body exactly matches the shim template
// this tool produces. A journal is untrusted input: writing its raw
// "shim_*_body" into /usr/local/bin during rollback would let a forged journal
// persist an arbitrary script as root. Only the fixed three-line template with
// a single quoted exec target is accepted.
func validManagedShimBody(body string) bool {
	prefix := "#!/bin/sh\n" + shimMarker + "\nexec "
	if !strings.HasPrefix(body, prefix) {
		return false
	}
	if !strings.HasSuffix(body, " \"$@\"\n") {
		return false
	}
	middle := strings.TrimSuffix(strings.TrimPrefix(body, prefix), " \"$@\"\n")
	if len(middle) < 2 || middle[0] != '\'' || middle[len(middle)-1] != '\'' {
		return false
	}
	if strings.Contains(middle, "\n") {
		return false
	}
	inner := middle[1 : len(middle)-1]
	for inner != "" {
		idx := strings.Index(inner, `'\''`)
		if idx < 0 {
			return !strings.Contains(inner, "'") && !strings.Contains(inner, "\x00")
		}
		inner = inner[idx+4:]
	}
	return true
}

func writeFileAtomic(path string, payload []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// fsatomic.WriteFile uses platform.RenamePath internally (with the Windows
	// sharing-violation retry) and applies the explicit mode before Sync.
	return fsatomic.WriteFile(path, payload, mode)
}

// --- small helpers --------------------------------------------------------

func copyDetails(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
