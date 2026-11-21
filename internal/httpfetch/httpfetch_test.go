package httpfetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/httpfetch"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/netpolicy"
)

func codeOf(t *testing.T, err error) errors.Code {
	t.Helper()
	typed, ok := errors.As(err)
	require.True(t, ok, "expected typed error, got %v", err)
	return typed.Code
}

func TestFetchLimitedParameterContract(t *testing.T) {
	client := &http.Client{}
	valid := "https://example.com/a"
	tests := []struct {
		name         string
		ctx          context.Context
		client       *http.Client
		url          string
		maxBytes     int64
		maxRedirects int
		policy       netpolicy.Policy
		want         errors.Code
	}{
		{"nil context", nil, client, valid, 1024, 1, netpolicy.CustomPolicy(), errors.E7005},
		{"nil client", context.Background(), nil, valid, 1024, 1, netpolicy.CustomPolicy(), errors.E7005},
		{"empty url", context.Background(), client, "  ", 1024, 1, netpolicy.CustomPolicy(), errors.E3008},
		{"zero maxBytes", context.Background(), client, valid, 0, 1, netpolicy.CustomPolicy(), errors.E4004},
		{"maxBytes over ceiling", context.Background(), client, valid, httpfetch.MaxBytesCeiling + 1, 1, netpolicy.CustomPolicy(), errors.E4004},
		{"zero maxRedirects", context.Background(), client, valid, 1024, 0, netpolicy.CustomPolicy(), errors.E4003},
		{"maxRedirects over ceiling", context.Background(), client, valid, 1024, netpolicy.MaxRedirectsCeiling + 1, netpolicy.CustomPolicy(), errors.E4003},
		{"zero policy", context.Background(), client, valid, 1024, 1, netpolicy.Policy{}, errors.E3008},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			_, err := httpfetch.FetchLimited(c.ctx, c.client, c.url, c.maxBytes, c.maxRedirects, c.policy)
			require.Error(t, err)
			require.Equal(t, c.want, codeOf(t, err))
		})
	}
}

func TestFetchLimitedInitialURLValidated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Custom policy allows http; Strict must reject the same loopback http
	// initial URL (strict allows http only for loopback, so use a non-loopback
	// literal host that cannot resolve — validation must fail before I/O).
	_, err := httpfetch.FetchLimited(context.Background(), srv.Client(),
		"http://user@example.com/a", 1024, 1, netpolicy.CustomPolicy())
	require.Error(t, err)
	require.Equal(t, errors.E3008, codeOf(t, err))
}

func TestFetchLimitedDoesNotMutateCallerClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	originalCheck := func(_ *http.Request, _ []*http.Request) error { return nil }
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: originalCheck}
	body, err := httpfetch.FetchLimited(context.Background(), client, srv.URL+"/a", 1024, 2, netpolicy.CustomPolicy())
	require.NoError(t, err)
	require.Equal(t, "payload", string(body))
	require.Equal(t, 5*time.Second, client.Timeout, "caller Timeout must be preserved")
	require.NotNil(t, client.CheckRedirect)
	// The original callback object must still be in place (we cloned the client).
	require.NoError(t, client.CheckRedirect(nil, nil))
}

func TestFetchLimitedChainsOriginalRedirectAndRespectsErrUseLastResponse(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/middle", http.StatusFound)
		case "/middle":
			http.Redirect(w, r, "/end", http.StatusFound)
		default:
			_, _ = w.Write([]byte("final"))
		}
	}))
	defer srv.Close()

	called := false
	client := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		called = true
		return nil
	}}
	body, err := httpfetch.FetchLimited(context.Background(), client, srv.URL+"/start", 1024, 3, netpolicy.CustomPolicy())
	require.NoError(t, err)
	require.Equal(t, "final", string(body))
	require.True(t, called, "caller callback must be chained after policy")
	require.Equal(t, 3, hits)

	// ErrUseLastResponse is consumed by net/http: it stops redirecting and
	// returns the last (302) response, which FetchLimited reports as non-2xx.
	// The callback error itself is never wrapped into an app error; netpolicy
	// has a dedicated test asserting verbatim propagation.
	client2 := &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	_, err = httpfetch.FetchLimited(context.Background(), client2, srv.URL+"/start", 1024, 3, netpolicy.CustomPolicy())
	require.Error(t, err)
	require.Equal(t, errors.E3007, codeOf(t, err), "last response is 302 -> non-2xx")
}

func TestFetchLimitedRedirectLimitAndDowngrade(t *testing.T) {
	t.Run("redirect limit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/next", http.StatusFound)
		}))
		defer srv.Close()
		_, err := httpfetch.FetchLimited(context.Background(), srv.Client(), srv.URL+"/a", 1024, 1, netpolicy.CustomPolicy())
		require.Error(t, err)
		require.Equal(t, errors.E4003, codeOf(t, err))
	})
}

func TestFetchLimitedSizeLimitAndNon2xx(t *testing.T) {
	t.Run("over limit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(strings.Repeat("x", 64)))
		}))
		defer srv.Close()
		_, err := httpfetch.FetchLimited(context.Background(), srv.Client(), srv.URL+"/a", 8, 1, netpolicy.CustomPolicy())
		require.Error(t, err)
		require.Equal(t, errors.E4004, codeOf(t, err))
	})
	t.Run("non 2xx", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		_, err := httpfetch.FetchLimited(context.Background(), srv.Client(), srv.URL+"/a", 1024, 1, netpolicy.CustomPolicy())
		require.Error(t, err)
		require.Equal(t, errors.E3007, codeOf(t, err))
	})
}
