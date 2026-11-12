// Package github is a minimal, testable GitHub Releases API client.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	defaultMaxAttempts     = 3
	defaultMaxResponseSize = 4 << 20
	defaultRetryBase       = 500 * time.Millisecond
	defaultMaxRedirects    = 5
	defaultTimeout         = 30 * time.Second
	maxRetryAfter          = 60 * time.Second
	userAgent              = "OneMoreFFmpeg"
	apiVersion             = "2022-11-28"
)

// Asset is one downloadable file of a release.
type Asset struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Size               int64     `json:"size"`
	BrowserDownloadURL string    `json:"browser_download_url"`
	UpdatedAt          time.Time `json:"updated_at"`
	// Digest is "sha256:<hex>" when GitHub exposes an asset digest.
	Digest string `json:"digest"`
}

// Release is the subset of the GitHub release payload OneMoreFFmpeg needs.
type Release struct {
	ID          int64     `json:"id"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	Assets      []Asset   `json:"assets"`
}

// Client talks to the GitHub REST API. All fields are optional except BaseURL;
// tests substitute BaseURL and Sleep to stay offline and deterministic.
type Client struct {
	BaseURL          string
	HTTPClient       *http.Client
	Token            string
	MaxAttempts      int
	MaxResponseBytes int64
	RetryBase        time.Duration
	MaxRedirects     int
	Timeout          time.Duration
	Sleep            func(ctx context.Context, delay time.Duration) error
}

// NewClient validates baseURL and builds a client with safe defaults.
// Loopback http URLs are accepted so tests can use httptest.
func NewClient(baseURL, token string) (*Client, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" {
		return nil, apperrors.New(apperrors.E7005, "github api base url is invalid")
	}
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopback(parsed.Hostname())) {
		return nil, apperrors.New(apperrors.E7005, "github api base url must be https").WithDetails(map[string]any{"value": baseURL})
	}
	client := &Client{
		BaseURL:          strings.TrimRight(parsed.String(), "/"),
		Token:            strings.TrimSpace(token),
		MaxAttempts:      defaultMaxAttempts,
		MaxResponseBytes: defaultMaxResponseSize,
		RetryBase:        defaultRetryBase,
		MaxRedirects:     defaultMaxRedirects,
	}
	client.RebuildHTTPClient()
	return client, nil
}

// RebuildHTTPClient applies the current Timeout / MaxRedirects settings.
// Callers that mutate those fields must call this before the next request.
func (c *Client) RebuildHTTPClient() {
	c.HTTPClient = c.newHTTPClient()
}

func (c *Client) newHTTPClient() *http.Client {
	maxRedirects := c.MaxRedirects
	if maxRedirects < 0 {
		maxRedirects = defaultMaxRedirects
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// len(via) counts this request's redirect chain only; a client-wide
			// counter would poison later requests.
			if len(via) > maxRedirects {
				return apperrors.New(apperrors.E4003, "redirect limit exceeded")
			}
			// API calls must never be downgraded to plain http.
			if !isAllowedURL(req.URL.String()) {
				return apperrors.New(apperrors.E3007, "redirect target must be https").WithDetails(map[string]any{"url": req.URL.String()})
			}
			return nil
		},
	}
}

// LatestRelease returns the latest non-draft release of owner/repo.
func (c *Client) LatestRelease(ctx context.Context, owner, repo string) (Release, error) {
	var release Release
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases/latest"
	if err := c.getJSON(ctx, path, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

// ListReleases returns up to perPage releases, newest first.
func (c *Client) ListReleases(ctx context.Context, owner, repo string, perPage int) ([]Release, error) {
	if perPage <= 0 || perPage > 100 {
		perPage = 30
	}
	var releases []Release
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) +
		"/releases?per_page=" + strconv.Itoa(perPage)
	if err := c.getJSON(ctx, path, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

// ReleaseByTag returns one release by tag name.
func (c *Client) ReleaseByTag(ctx context.Context, owner, repo, tag string) (Release, error) {
	var release Release
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo) + "/releases/tags/" + url.PathEscape(tag)
	if err := c.getJSON(ctx, path, &release); err != nil {
		return Release{}, err
	}
	return release, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	attempts := c.MaxAttempts
	if attempts <= 0 {
		attempts = defaultMaxAttempts
	}
	maxBytes := c.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxResponseSize
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		status, body, header, err := c.do(ctx, path, maxBytes)
		if err != nil {
			lastErr = err
			if isRedirectLimit(err) || attempt == attempts {
				return err
			}
			if err := c.sleep(ctx, c.backoff(attempt)); err != nil {
				return err
			}
			continue
		}
		switch {
		case status == http.StatusOK:
			if err := json.Unmarshal(body, out); err != nil {
				return apperrors.New(apperrors.E3007, "github response is not valid JSON").WithCause(err.Error())
			}
			return nil
		case status == http.StatusNotFound:
			return apperrors.New(apperrors.E3002, "release or tag not found").WithDetails(map[string]any{"path": path})
		case isRateLimited(status, header):
			details := map[string]any{"status": status}
			if reset := header.Get("X-RateLimit-Reset"); reset != "" {
				details["reset"] = reset
			}
			lastErr = apperrors.New(apperrors.E3006, "github api rate limited").WithDetails(details)
			if retryAfter, ok := retryAfter(header); ok {
				// Never block the CLI for hours; fail fast and let the caller retry.
				if retryAfter > maxRetryAfter {
					details["retry_after"] = retryAfter.String()
					return lastErr
				}
				if attempt < attempts {
					if err := c.sleep(ctx, retryAfter); err != nil {
						return err
					}
					continue
				}
			}
			return lastErr
		case status >= 500 && status <= 599:
			lastErr = apperrors.New(apperrors.E3007, "github api server error").WithDetails(map[string]any{"status": status})
			if attempt == attempts {
				return lastErr
			}
			if err := c.sleep(ctx, c.backoff(attempt)); err != nil {
				return err
			}
		default:
			return apperrors.New(apperrors.E3007, "github api request failed").WithDetails(map[string]any{"status": status})
		}
	}
	if lastErr == nil {
		lastErr = apperrors.New(apperrors.E3007, "github api request failed")
	}
	return lastErr
}

func (c *Client) do(ctx context.Context, path string, maxBytes int64) (int, []byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return 0, nil, nil, apperrors.New(apperrors.E3007, "github request could not be built").WithCause(err.Error())
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	client := c.HTTPClient
	if client == nil {
		client = c.newHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, nil, nil, contextError(ctxErr)
		}
		if typed, ok := apperrors.As(err); ok {
			return 0, nil, nil, typed
		}
		return 0, nil, nil, apperrors.New(apperrors.E3007, "github request failed").WithCause(err.Error())
	}
	defer resp.Body.Close()

	// Read one byte past the limit so oversized responses are detected.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return 0, nil, nil, contextError(ctxErr)
		}
		return 0, nil, nil, apperrors.New(apperrors.E3007, "github response could not be read").WithCause(err.Error())
	}
	if int64(len(body)) > maxBytes {
		return 0, nil, nil, apperrors.New(apperrors.E3007, "github response exceeds size limit").WithDetails(map[string]any{"limit": maxBytes})
	}
	return resp.StatusCode, body, resp.Header, nil
}

func (c *Client) sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, delay time.Duration) error {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return contextError(ctx.Err())
			case <-timer.C:
				return nil
			}
		}
	}
	return sleep(ctx, delay)
}

func (c *Client) backoff(attempt int) time.Duration {
	base := c.RetryBase
	if base <= 0 {
		base = defaultRetryBase
	}
	delay := base << (attempt - 1)
	if delay > 10*time.Second {
		return 10 * time.Second
	}
	return delay
}

func contextError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(apperrors.E3007, "github request timed out").WithCause(err.Error())
	}
	return apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
}

func isRateLimited(status int, header http.Header) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	if status != http.StatusForbidden {
		return false
	}
	// Primary limit exposes X-RateLimit-Remaining; secondary limits only send
	// Retry-After.
	return header.Get("X-RateLimit-Remaining") == "0" || header.Get("Retry-After") != ""
}

func retryAfter(header http.Header) (time.Duration, bool) {
	raw := strings.TrimSpace(header.Get("Retry-After"))
	if raw == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	if when, err := http.ParseTime(raw); err == nil {
		delay := time.Until(when)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}
	return 0, false
}

func isRedirectLimit(err error) bool {
	typed, ok := apperrors.As(err)
	return ok && typed.Code == apperrors.E4003
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isAllowedURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return parsed.Scheme == "http" && isLoopback(parsed.Hostname())
}
