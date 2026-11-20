// Package checksum implements the custom-URL checksum.sha256 convention:
// GNU sha256sum format, one line per archive, exact file name match, strict
// duplicate detection, and checksum.sha256 resolved in the archive's own
// directory unless explicitly overridden.
package checksum

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const maxChecksumsBytes = 4 << 20

// ResolveURL returns the checksum.sha256 URL for an archive URL, by replacing
// the archive file name while keeping the same directory (never guessing other
// directories). Uses net/url, not string concatenation.
func ResolveURL(archiveURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(archiveURL))
	if err != nil || parsed.Host == "" {
		return "", apperrors.New(apperrors.E3008, "invalid archive url")
	}
	dir := path.Dir(parsed.Path)
	if dir == "." || dir == "/" {
		dir = ""
	}
	parsed.Path = path.Join(dir, "checksum.sha256")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

// Fetch downloads a checksum.sha256 file with size limit and context support.
func Fetch(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OneMoreFFmpeg")
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "checksum.sha256 cannot be fetched").WithCause(err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apperrors.New(apperrors.E3007, "checksum.sha256 returned an error").
			WithDetails(map[string]any{"status": resp.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChecksumsBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxChecksumsBytes {
		return nil, apperrors.New(apperrors.E4004, "checksum.sha256 exceeds size limit")
	}
	return body, nil
}

// Parse finds the exact digest for fileName in GNU sha256sum content.
// Rules: 64 lowercase hex, whitespace separator, exact bare file name (no
// paths, no .. or backslash), duplicate entries always error, CRLF allowed.
func Parse(data []byte, fileName string) (string, error) {
	name := strings.TrimSpace(fileName)
	if name == "" || name == "." || name == ".." || path.Base(name) != name || strings.ContainsAny(name, "\\/") {
		return "", apperrors.New(apperrors.E4002, "invalid archive file name")
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), maxLineBytes)
	found := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexFunc(line, unicode.IsSpace)
		if idx <= 0 {
			continue
		}
		digest := strings.ToLower(line[:idx])
		if len(digest) != 64 || !isLowerHex(digest) {
			continue
		}
		file := strings.TrimLeft(line[idx:], " \t")
		file = strings.TrimPrefix(file, "*")
		if file != name { // strict: exact bare name, no dist/ prefix
			continue
		}
		if found != "" {
			return "", apperrors.New(apperrors.E4002, "duplicate checksum entry").
				WithDetails(map[string]any{"file": name})
		}
		found = digest
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan checksums: %w", err)
	}
	if found == "" {
		return "", apperrors.New(apperrors.E4002, "checksum.sha256 has no entry for the asset").
			WithDetails(map[string]any{"file": name})
	}
	return found, nil
}

const maxLineBytes = 1 << 20

func isLowerHex(s string) bool {
	for _, r := range s {
		digit := r >= '0' && r <= '9'
		lower := r >= 'a' && r <= 'f'
		if !digit && !lower {
			return false
		}
	}
	return true
}
