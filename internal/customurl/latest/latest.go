// Package latest reads the optional plain-text snapshots/latest.txt pointer.
// The upstream server only needs to serve a text file; no JSON, redirects,
// rewrites or scripts are involved.
package latest

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const maxBytes = 4096

var (
	dayStamp  = regexp.MustCompile(`^[0-9]{8}$`)
	fullStamp = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z$`)
)

// Read fetches rawURL and returns the single build identifier it contains.
func Read(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OneMoreFFmpeg")
	resp, err := client.Do(req)
	if err != nil {
		return "", apperrors.New(apperrors.E3007, "latest.txt cannot be fetched").WithCause(err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apperrors.New(apperrors.E3007, "latest.txt returned an error").
			WithDetails(map[string]any{"status": resp.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxBytes {
		return "", apperrors.New(apperrors.E4004, "latest.txt exceeds size limit")
	}
	return Parse(body)
}

// Parse validates the one-line content: YYYYMMDD or YYYYMMDDTHHMMSSZ,
// optional single trailing newline, no extra text.
func Parse(data []byte) (string, error) {
	line := string(bytes.TrimSuffix(data, []byte("\n")))
	line = strings.TrimSuffix(line, "\r")
	if line == "" || strings.ContainsAny(line, "\r\n \t") {
		return "", apperrors.New(apperrors.E7003, "latest.txt must contain one build identifier")
	}
	if !dayStamp.MatchString(line) && !fullStamp.MatchString(line) {
		return "", apperrors.New(apperrors.E7003, "latest.txt has invalid build identifier")
	}
	return line, nil
}
