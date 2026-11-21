package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/netpolicy"
)

// fetchURL downloads a small checksums file with a bounded body.
//
// URL and redirect validation now come from netpolicy.CustomPolicy (the
// user-supplied static-hosting family: http + https, no SSRF filtering). The
// original error-code contract is preserved: this function maps policy
// rejections back to its historical codes (E3008 for URL/scheme, E3007 for
// redirect count) so no caller or golden contract changes.
func fetchURL(ctx context.Context, rawURL string, maxRedirects int, timeout time.Duration) ([]byte, error) {
	// Validate the initial URL with the shared Custom policy. netpolicy reports
	// all URL problems as E3008, matching this function's historical contract.
	policy := netpolicy.CustomPolicy()
	if err := policy.Validate(rawURL); err != nil {
		return nil, apperrors.New(apperrors.E3008, "checksums url is invalid").WithDetails(map[string]any{"url": rawURL})
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, apperrors.New(apperrors.E3008, "checksums url is invalid").WithDetails(map[string]any{"url": rawURL})
	}
	scheme := strings.ToLower(parsed.Scheme)
	if maxRedirects <= 0 {
		maxRedirects = 5
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Build the redirect callback from the shared policy, but preserve the
	// historical "sticky initial scheme" behavior: CustomPolicy itself does
	// not enforce whole-chain downgrade, so add the initial-scheme check in
	// the chained callback (this is the exact pre-migration behavior).
	baseCheck, err := policy.CheckRedirect(maxRedirects, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "invalid redirect configuration").WithCause(err.Error())
	}
	checkRedirect := func(req *http.Request, via []*http.Request) error {
		if err := baseCheck(req, via); err != nil {
			// netpolicy uses E4003 for the hop limit; this function historically
			// reported E3007. Map only that code; other policy errors stay as-is.
			if typed, ok := apperrors.As(err); ok && typed.Code == apperrors.E4003 {
				return apperrors.New(apperrors.E3007, "too many redirects").WithCause(err.Error())
			}
			return err
		}
		// Never downgrade an https source to http (only the initial scheme was
		// inspected before migration; keep that exact behavior).
		if scheme == "https" && strings.EqualFold(req.URL.Scheme, "http") {
			return apperrors.New(apperrors.E3008, "redirect must not downgrade https")
		}
		return nil
	}

	client := &http.Client{Timeout: timeout, CheckRedirect: checkRedirect}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OneMoreFFmpeg")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apperrors.New(apperrors.E3007, "checksums fetch failed").
			WithDetails(map[string]any{"status": resp.StatusCode})
	}
	limited := io.LimitReader(resp.Body, defaultChecksumsLimit+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > defaultChecksumsLimit {
		return nil, apperrors.New(apperrors.E4004, "checksums file exceeds size limit").
			WithDetails(map[string]any{"limit": defaultChecksumsLimit})
	}
	return bytes.Clone(body), nil
}
