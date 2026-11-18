package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// InstallSource identifies who owns the running OneMoreFFmpeg executable.
type InstallSource int

const (
	InstallSourceUnknown InstallSource = iota
	InstallSourceManual
	InstallSourceScoop
	InstallSourceRPM
)

func (s InstallSource) String() string {
	switch s {
	case InstallSourceManual:
		return "manual"
	case InstallSourceScoop:
		return "scoop"
	case InstallSourceRPM:
		return "rpm"
	default:
		return "unknown"
	}
}

// SourceInfo is the machine-readable result of source detection, safe to put
// into a JSON error envelope's details map.
type SourceInfo struct {
	Source     string `json:"source"`
	Executable string `json:"executable"`
	Advice     string `json:"advice,omitempty"`
}

// RPMQuery is injectable for tests; nil uses `rpm -qf <path>` with an argument
// slice (never a shell string).
var rpmQuery = func(executable string) (string, error) {
	cmd := exec.Command("rpm", "-qf", executable)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// DetectSource resolves who manages the executable. It never assumes Manual
// for files in system directories: Scoop/RPM are detected from real evidence,
// user-owned locations are treated as manual, everything else is Unknown.
func DetectSource(executable string) (InstallSource, SourceInfo) {
	abs, err := filepath.Abs(executable)
	if err != nil {
		abs = executable
	}
	info := SourceInfo{Source: InstallSourceUnknown.String(), Executable: abs}
	if runtime.GOOS == "linux" {
		if _, err := rpmQuery(abs); err == nil {
			info.Source = InstallSourceRPM.String()
			info.Advice = "managed by RPM; use: sudo dnf upgrade one-more-ffmpeg"
			return InstallSourceRPM, info
		}
	}
	if isScoopPath(abs) {
		info.Source = InstallSourceScoop.String()
		info.Advice = "managed by Scoop; use: scoop update one-more-ffmpeg"
		return InstallSourceScoop, info
	}
	if isUserOwnedPath(abs) {
		info.Source = InstallSourceManual.String()
		return InstallSourceManual, info
	}
	info.Advice = "install source cannot be determined; refusing in-place replacement; " +
		"use your package manager or reinstall manually"
	return InstallSourceUnknown, info
}

// isScoopPath checks the resolved path against known Scoop app roots. It does
// NOT rely on the SCOOP env var alone (the tool may be installed elsewhere).
func isScoopPath(executable string) bool {
	clean := filepath.Clean(executable)
	home, _ := os.UserHomeDir()
	roots := []string{}
	if v := os.Getenv("SCOOP"); v != "" {
		roots = append(roots, filepath.Join(v, "apps"))
	}
	if home != "" {
		roots = append(roots, filepath.Join(home, "scoop", "apps"),
			filepath.Join(home, "AppData", "Local", "scoop", "apps"))
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(root), clean)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// isUserOwnedPath treats paths under the user's home/profile as manual
// installations. System-wide locations are deliberately NOT assumed manual.
func isUserOwnedPath(executable string) bool {
	clean := filepath.Clean(executable)
	home, _ := os.UserHomeDir()
	if home != "" {
		rel, err := filepath.Rel(filepath.Clean(home), clean)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	localAppData := os.Getenv("LOCALAPPDATA")
	if runtime.GOOS == "windows" && localAppData != "" {
		rel, err := filepath.Rel(filepath.Clean(localAppData), clean)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// CheckSource is called by update --check: it never blocks checking, only
// annotates the source so the caller can explain the situation.
func CheckSource(executable string) SourceInfo {
	_, info := DetectSource(executable)
	return info
}

// GuardApply enforces the source policy for --apply. Unknown and package-managed
// sources are rejected; an explicit override is allowed for Unknown only.
func GuardApply(executable string, allowUnknownOverride bool) error {
	source, info := DetectSource(executable)
	switch source {
	case InstallSourceManual:
		return nil
	case InstallSourceScoop, InstallSourceRPM:
		return apperrors.New(apperrors.E7008, "self-update blocked: executable is package-managed").
			WithDetails(map[string]any{"source": info.Source, "executable": info.Executable, "advice": info.Advice})
	case InstallSourceUnknown:
		if allowUnknownOverride {
			return nil
		}
		return apperrors.New(apperrors.E7008, "self-update blocked: install source unknown").
			WithDetails(map[string]any{"source": info.Source, "executable": info.Executable, "advice": info.Advice})
	}
	return fmt.Errorf("unreachable install source")
}
