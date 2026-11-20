// Package naming parses and builds artifact file names for the custom-URL
// convention:
//
//	{name}-{version}-{license}-{release_type}-{os}-{arch}-{linkage}.{archive_format}
//
// All field enums live here (data, not scattered regex). The format is not
// JSON/manifest based: a plain static file server is sufficient.
package naming

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// Artifact is one parsed file name.
type Artifact struct {
	Name        string // software name, e.g. "ffmpeg"
	Version     string // stable "8.1.0" or snapshot "20261120" / "20261120T153000Z"
	License     string // gpl | lgpl | nonfree | custom
	ReleaseType string // stable | snapshot
	OS          string // windows | linux | darwin | ...
	Arch        string // amd64 | arm64 | ...
	Linkage     string // static | shared
	Format      string // zip | tar.gz | tar.xz | tar.zst
	FileName    string // original full file name
}

var (
	stableVersion = regexp.MustCompile(`^[0-9]+(\.[0-9]+){2,3}$`)
	snapshotDay   = regexp.MustCompile(`^[0-9]{8}$`)
	snapshotStamp = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z$`)
	fieldChars    = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

var (
	licenses    = []string{"gpl", "lgpl", "nonfree", "custom"}
	releaseTyps = []string{"stable", "snapshot"}
	oses        = []string{"windows", "linux", "darwin", "freebsd", "openbsd", "netbsd", "android"}
	arches      = []string{"amd64", "arm64", "386", "arm", "ppc64le", "riscv64", "s390x"}
	linkages    = []string{"static", "shared"}
	formats     = []string{"zip", "tar.gz", "tar.xz", "tar.zst"}
)

func contains(values []string, v string) bool {
	for _, item := range values {
		if item == v {
			return true
		}
	}
	return false
}

func sortedCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

// FormatOf returns the archive format of a file name (longest suffix wins),
// or "" for unsupported formats.
func FormatOf(name string) string {
	lower := strings.ToLower(name)
	for _, f := range formats { // zip first, then tar.* (longer checked first)
		if strings.HasSuffix(lower, "."+f) {
			return f
		}
	}
	return ""
}

// splitName separates the archive format from the dash-separated fields.
func splitName(fileName string) (base, format string, err error) {
	format = FormatOf(fileName)
	if format == "" {
		return "", "", apperrors.New(apperrors.E4002, "unsupported archive format").
			WithDetails(map[string]any{"file": fileName})
	}
	base = strings.TrimSuffix(fileName, "."+format)
	return base, format, nil
}

// Parse parses and fully validates a file name.
func Parse(fileName string) (Artifact, error) {
	fileName = strings.TrimSpace(fileName)
	base, format, err := splitName(fileName)
	if err != nil {
		return Artifact{}, err
	}
	parts := strings.Split(base, "-")
	if len(parts) != 7 {
		return Artifact{}, apperrors.New(apperrors.E4002, "artifact name must have 7 fields").
			WithDetails(map[string]any{"file": fileName, "fields": len(parts)})
	}
	a := Artifact{
		Name: parts[0], Version: parts[1], License: parts[2], ReleaseType: parts[3],
		OS: parts[4], Arch: parts[5], Linkage: parts[6], Format: format, FileName: fileName,
	}
	if err := a.Validate(); err != nil {
		return Artifact{}, err
	}
	return a, nil
}

// Validate checks every field's format and enum (not just the regex).
func (a Artifact) Validate() error {
	if a.FileName != "" {
		if strings.ContainsAny(a.FileName, "/\\ \t\r\n") {
			return apperrors.New(apperrors.E4002, "artifact name must not contain separators")
		}
		// Only the T and Z of a snapshot timestamp may be uppercase.
		lowered := strings.ToLower(a.FileName)
		if lowered != a.FileName && strings.ReplaceAll(strings.ReplaceAll(a.FileName, "T", "t"), "Z", "z") != lowered {
			return apperrors.New(apperrors.E4002, "artifact name must be lowercase ASCII (except T/Z in timestamps)")
		}
	}
	if !fieldChars.MatchString(a.Name) {
		return apperrors.New(apperrors.E4002, "invalid software name").WithDetails(map[string]any{"name": a.Name})
	}
	switch a.ReleaseType {
	case "stable":
		if !stableVersion.MatchString(a.Version) {
			return apperrors.New(apperrors.E4002, "stable version must be dotted numeric (no v prefix)")
		}
	case "snapshot":
		if !snapshotDay.MatchString(a.Version) && !snapshotStamp.MatchString(a.Version) {
			return apperrors.New(apperrors.E4002, "snapshot version must be YYYYMMDD or YYYYMMDDTHHMMSSZ")
		}
	default:
		return apperrors.New(apperrors.E4002, "release_type must be stable or snapshot")
	}
	if !contains(licenses, a.License) {
		return apperrors.New(apperrors.E4002, "unsupported license").WithDetails(map[string]any{"license": a.License})
	}
	if !contains(oses, a.OS) {
		return apperrors.New(apperrors.E4002, "unsupported os").WithDetails(map[string]any{"os": a.OS})
	}
	if !contains(arches, a.Arch) {
		return apperrors.New(apperrors.E4002, "unsupported arch").WithDetails(map[string]any{"arch": a.Arch})
	}
	if !contains(linkages, a.Linkage) {
		return apperrors.New(apperrors.E4002, "unsupported linkage").WithDetails(map[string]any{"linkage": a.Linkage})
	}
	if !contains(formats, a.Format) {
		return apperrors.New(apperrors.E4002, "unsupported archive format").WithDetails(map[string]any{"format": a.Format})
	}
	return nil
}

// Build renders an Artifact into its canonical file name.
func Build(a Artifact) (string, error) {
	if err := a.Validate(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s.%s",
		a.Name, a.Version, a.License, a.ReleaseType, a.OS, a.Arch, a.Linkage, a.Format), nil
}

// IsStable reports whether a version identifier follows the stable dotted
// numeric convention (as opposed to a snapshot date/timestamp).
func IsStable(version string) bool { return stableVersion.MatchString(version) }

// Supported returns the enum sets (useful for help/tests).
func Supported() map[string][]string {
	return map[string][]string{
		"license": sortedCopy(licenses), "release_type": sortedCopy(releaseTyps),
		"os": sortedCopy(oses), "arch": sortedCopy(arches),
		"linkage": sortedCopy(linkages), "format": sortedCopy(formats),
	}
}
