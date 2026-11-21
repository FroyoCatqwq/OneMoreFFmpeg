// Package httpfetch performs bounded, text-only HTTP fetches.
//
// It is intentionally narrow: no archive downloads, no progress reporting,
// no filesystem writes, and no policy of its own. URL and redirect safety is
// injected explicitly via netpolicy.Policy, so every fetch must state which
// family (Strict or Custom) it uses.
package httpfetch

import (
	"context"
	"io"
	"net/http"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/netpolicy"
)

// MaxBytesCeiling is the hard upper bound for a single in-memory text fetch.
// It is owned by this package because FetchLimited returns []byte: 8 MiB is
// generous for checksums (~4 MiB), latest.txt (a timestamp) and release JSON,
// while preventing unbounded allocation. Larger content needs a separate
// streaming-to-file API, not a bigger []byte limit.
const MaxBytesCeiling = 8 << 20 // 8 MiB

// FetchLimited fetches a bounded text response with explicit policy injection.
//
// Parameters are ALL required; there are no implicit defaults and no
// convenience wrappers. All invalid inputs return typed errors before any I/O:
//   - ctx == nil                   -> E7005
//   - client == nil                -> E7005 (the caller owns TLS/proxy/rate
//     configuration; the helper must not create a client behind its back)
//   - blank rawURL                 -> E3008
//   - maxBytes <= 0 or > ceiling   -> E4004 (also guards maxBytes+1 overflow)
//   - maxRedirects out of [1,10]   -> E4003
//   - zero/foreign policy          -> E3008 (fail closed)
//
// The caller's client is never mutated: a shallow clone is used and only its
// CheckRedirect is replaced. The injected policy validates the initial URL and
// every redirect target; a caller-supplied CheckRedirect (if any) is chained
// AFTER the project policy and cannot bypass it. http.ErrUseLastResponse and
// other callback errors are returned verbatim (net/http semantics).
//
// Behavior: User-Agent "OneMoreFFmpeg"; non-2xx -> E3007; Response.Body is
// always closed; reading more than maxBytes -> E4004 (never a truncated
// success).
func FetchLimited(ctx context.Context, client *http.Client, rawURL string,
	maxBytes int64, maxRedirects int, policy netpolicy.Policy) ([]byte, error) {
	if ctx == nil {
		return nil, apperrors.New(apperrors.E7005, "nil context")
	}
	if client == nil {
		return nil, apperrors.New(apperrors.E7005, "nil http client")
	}
	if strings.TrimSpace(rawURL) == "" {
		return nil, apperrors.New(apperrors.E3008, "url is empty")
	}
	if maxBytes <= 0 || maxBytes > MaxBytesCeiling {
		return nil, apperrors.New(apperrors.E4004, "maxBytes out of range").WithDetails(map[string]any{
			"max_bytes": maxBytes, "ceiling": MaxBytesCeiling,
		})
	}
	// Validate the initial URL before building the client. CheckRedirect only
	// sees redirects; the initial request must be checked explicitly.
	if err := policy.Validate(rawURL); err != nil {
		return nil, err
	}

	original := client.CheckRedirect
	check, err := policy.CheckRedirect(maxRedirects, original)
	if err != nil {
		return nil, err
	}
	clone := *client
	clone.CheckRedirect = check

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E3008, "request could not be built").WithCause(err.Error())
	}
	req.Header.Set("User-Agent", "OneMoreFFmpeg")

	resp, err := clone.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apperrors.New(apperrors.E3007, "fetch failed").WithDetails(map[string]any{
			"status": resp.StatusCode, "url": resp.Request.URL.Redacted(),
		})
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, apperrors.New(apperrors.E4001, "response body read failed").WithCause(err.Error())
	}
	if int64(len(body)) > maxBytes {
		return nil, apperrors.New(apperrors.E4004, "response exceeds size limit").WithDetails(map[string]any{
			"max_bytes": maxBytes,
		})
	}
	return body, nil
}
