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
)

// fetchURL downloads a small checksums file with strict HTTPS-only redirects
// and a bounded body. It replaces the old GitHub-checksums fetcher, which was
// tied to built-in providers.
func fetchURL(ctx context.Context, rawURL string, maxRedirects int) ([]byte, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Host == "" || parsed.User != nil {
		return nil, apperrors.New(apperrors.E3008, "checksums url must be https").WithDetails(map[string]any{"url": rawURL})
	}
	if maxRedirects <= 0 {
		maxRedirects = 5
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return apperrors.New(apperrors.E3007, "too many redirects")
			}
			if !strings.EqualFold(req.URL.Scheme, "https") {
				return apperrors.New(apperrors.E3008, "redirect must stay https")
			}
			return nil
		},
	}
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
