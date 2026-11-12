package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func newClient(t *testing.T, server *httptest.Server) *github.Client {
	t.Helper()
	client, err := github.NewClient(server.URL, "")
	require.NoError(t, err)
	client.Sleep = func(context.Context, time.Duration) error { return nil }
	return client
}

func errCode(t *testing.T, err error) apperrors.Code {
	t.Helper()
	typed, ok := apperrors.As(err)
	require.True(t, ok, "expected typed error, got %v", err)
	return typed.Code
}

func TestLatestReleaseParsesAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/FFmpeg-Builds/FFmpeg-Builds/releases/latest", r.URL.Path)
		require.Equal(t, "OneMoreFFmpeg", r.Header.Get("User-Agent"))
		require.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))
		require.Empty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{
			"id": 123,
			"tag_name": "autobuild-2026-11-11-12-39",
			"name": "Auto-Build 2026-11-11 12:39",
			"published_at": "2026-11-11T12:39:00Z",
			"assets": [
				{"id": 1, "name": "checksums.sha256", "size": 5296, "browser_download_url": "https://example.com/checksums.sha256", "updated_at": "2026-11-11T12:39:00Z"},
				{"id": 2, "name": "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip", "size": 123456, "browser_download_url": "https://example.com/ffmpeg.zip", "updated_at": "2026-11-11T12:39:00Z"}
			]
		}`))
	}))
	defer server.Close()

	release, err := newClient(t, server).LatestRelease(context.Background(), "FFmpeg-Builds", "FFmpeg-Builds")
	require.NoError(t, err)
	require.EqualValues(t, 123, release.ID)
	require.Equal(t, "autobuild-2026-11-11-12-39", release.TagName)
	require.Len(t, release.Assets, 2)
	require.Equal(t, int64(123456), release.Assets[1].Size)
	require.Equal(t, "https://example.com/ffmpeg.zip", release.Assets[1].BrowserDownloadURL)
}

func TestReleaseByTagEscapesTag(t *testing.T) {
	var seen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.EscapedPath()
		_, _ = w.Write([]byte(`{"id":1,"tag_name":"latest","assets":[]}`))
	}))
	defer server.Close()

	_, err := newClient(t, server).ReleaseByTag(context.Background(), "o", "r", "n9.0 / test")
	require.NoError(t, err)
	require.Equal(t, "/repos/o/r/releases/tags/n9.0%20%2F%20test", seen)
}

func TestReleaseNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	_, err := newClient(t, server).LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E3002, errCode(t, err))
}

func TestRateLimitedWithoutRetryAfter(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1763000000")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := newClient(t, server).LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E3006, errCode(t, err))
	require.EqualValues(t, 1, atomic.LoadInt32(&hits), "rate limit must not be retried without Retry-After")
}

func TestRateLimitedRejectsHugeRetryAfter(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := newClient(t, server).LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E3006, errCode(t, err))
	require.EqualValues(t, 1, atomic.LoadInt32(&hits), "huge Retry-After must fail fast")
}

func TestRateLimitedHonorsRetryAfter(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"tag_name":"latest","assets":[]}`))
	}))
	defer server.Close()

	client := newClient(t, server)
	var slept time.Duration
	client.Sleep = func(_ context.Context, delay time.Duration) error {
		slept = delay
		return nil
	}
	_, err := client.LatestRelease(context.Background(), "o", "r")
	require.NoError(t, err)
	require.Equal(t, 2*time.Second, slept)
	require.EqualValues(t, 2, atomic.LoadInt32(&hits))
}

func TestServerErrorsAreRetried(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newClient(t, server)
	client.RetryBase = time.Millisecond
	_, err := client.LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E3007, errCode(t, err))
	require.EqualValues(t, 3, atomic.LoadInt32(&hits))
}

func TestResponseSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	defer server.Close()

	client := newClient(t, server)
	client.MaxResponseBytes = 1024
	_, err := client.LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E3007, errCode(t, err))
}

func TestRedirectLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/loop", http.StatusFound)
	}))
	defer server.Close()

	_, err := newClient(t, server).LatestRelease(context.Background(), "o", "r")
	require.Equal(t, apperrors.E4003, errCode(t, err))
}

func TestRedirectBudgetIsPerRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`{"id":1,"tag_name":"latest","assets":[]}`))
	}))
	defer server.Close()

	client := newClient(t, server)
	// Point both calls at the redirecting path by using a tag name that maps
	// to /redirect is not possible, so exercise the client's redirect budget
	// through two requests to the same redirecting endpoint.
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/redirect", nil)
		require.NoError(t, err)
		resp, err := client.HTTPClient.Do(req)
		require.NoError(t, err, "redirect budget must reset per request")
		resp.Body.Close()
	}
}

func TestCanceledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newClient(t, server).LatestRelease(ctx, "o", "r")
	require.Equal(t, apperrors.E9004, errCode(t, err))
}

func TestTokenIsNotLeaked(t *testing.T) {
	const token = "secret-token-value"
	var authorized bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorized = r.Header.Get("Authorization") == "Bearer "+token
		w.WriteHeader(http.StatusTeapot)
	}))
	defer server.Close()

	client, err := github.NewClient(server.URL, token)
	require.NoError(t, err)
	client.Sleep = func(context.Context, time.Duration) error { return nil }
	_, err = client.LatestRelease(context.Background(), "o", "r")
	require.True(t, authorized)
	require.Equal(t, apperrors.E3007, errCode(t, err))
	typed, _ := apperrors.As(err)
	combined := err.Error() + typed.Cause
	details, _ := json.Marshal(typed.Details)
	require.NotContains(t, combined, token)
	require.NotContains(t, string(details), token)
}

func TestNewClientRejectsInsecureBaseURL(t *testing.T) {
	_, err := github.NewClient("http://example.com", "")
	require.Equal(t, apperrors.E7005, errCode(t, err))

	_, err = github.NewClient("ftp://example.com", "")
	require.Equal(t, apperrors.E7005, errCode(t, err))

	client, err := github.NewClient("http://127.0.0.1:9", "")
	require.NoError(t, err)
	require.NotNil(t, client)
}
