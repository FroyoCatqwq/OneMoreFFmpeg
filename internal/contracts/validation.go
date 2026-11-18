package contracts

import (
	"net/url"
	"path/filepath"
	"strings"
)

// isHTTPS accepts http and https for user-supplied custom URLs (self-hosted
// mirrors are often plain http), matching the download/provider URL policy.
// Credentials in the URL are always rejected so tokens cannot be smuggled into
// release files.
func isHTTPS(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	return parsed.Scheme == "https" || parsed.Scheme == "http"
}

// safeAssetName rejects path separators and relative-path components so a
// release asset name cannot escape its download directory.
func safeAssetName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return false
	}
	return true
}

// isAbsoluteLike detects absolute POSIX/Windows and UNC paths regardless of
// the host OS, so golden examples cannot encode a host-specific absolute path.
func isAbsoluteLike(value string) bool {
	if strings.HasPrefix(value, "\\\\") || strings.HasPrefix(value, "//") {
		return true
	}
	return strings.HasPrefix(value, "/") ||
		(len(value) >= 3 && value[1] == ':' && (value[2] == '/' || value[2] == '\\'))
}
