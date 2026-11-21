package netpolicy_test

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/netpolicy"
)

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}

func TestValidateFamilies(t *testing.T) {
	cases := []struct {
		name   string
		policy netpolicy.Policy
		raw    string
		ok     bool
	}{
		{"strict https", netpolicy.StrictPolicy(), "https://example.com/a", true},
		{"strict loopback http", netpolicy.StrictPolicy(), "http://127.0.0.1:8080/a", true},
		{"strict localhost http", netpolicy.StrictPolicy(), "http://localhost/a", true},
		{"strict non-loopback http", netpolicy.StrictPolicy(), "http://example.com/a", false},
		{"strict userinfo", netpolicy.StrictPolicy(), "https://user@example.com/a", false},
		{"custom https", netpolicy.CustomPolicy(), "https://example.com/a", true},
		{"custom http any host", netpolicy.CustomPolicy(), "http://example.com/a", true},
		{"custom userinfo", netpolicy.CustomPolicy(), "http://user@example.com/a", false},
		{"custom ftp", netpolicy.CustomPolicy(), "ftp://example.com/a", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.policy.Validate(c.raw)
			if c.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestZeroPolicyFailsClosed(t *testing.T) {
	var zero netpolicy.Policy
	require.Error(t, zero.Validate("https://example.com"))
	_, err := zero.CheckRedirect(1, nil)
	require.Error(t, err)
}

func TestCheckRedirectParameterValidation(t *testing.T) {
	p := netpolicy.CustomPolicy()
	for _, max := range []int{0, -1, netpolicy.MaxRedirectsCeiling + 1} {
		fn, err := p.CheckRedirect(max, nil)
		require.Error(t, err, "max=%d", max)
		require.Nil(t, fn)
	}
	fn, err := p.CheckRedirect(netpolicy.MaxRedirectsCeiling, nil)
	require.NoError(t, err)
	require.NotNil(t, fn)
}

func TestCheckRedirectHopCount(t *testing.T) {
	p := netpolicy.CustomPolicy()
	t.Run("max 1 allows first hop", func(t *testing.T) {
		fn, err := p.CheckRedirect(1, nil)
		require.NoError(t, err)
		// via has the initial request only => 0 hops taken.
		require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/1")},
			[]*http.Request{{URL: mustURL(t, "http://example.com/start")}}))
	})
	t.Run("max 1 rejects second hop", func(t *testing.T) {
		fn, err := p.CheckRedirect(1, nil)
		require.NoError(t, err)
		via := []*http.Request{
			{URL: mustURL(t, "http://example.com/start")},
			{URL: mustURL(t, "http://example.com/1")},
		}
		require.Error(t, fn(&http.Request{URL: mustURL(t, "http://example.com/2")}, via))
	})
	t.Run("max 2 allows two hops", func(t *testing.T) {
		fn, err := p.CheckRedirect(2, nil)
		require.NoError(t, err)
		via := []*http.Request{
			{URL: mustURL(t, "http://example.com/start")},
			{URL: mustURL(t, "http://example.com/1")},
		}
		require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/2")}, via))
	})
	t.Run("empty via treated as zero hops", func(t *testing.T) {
		fn, err := p.CheckRedirect(1, nil)
		require.NoError(t, err)
		require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/1")}, nil))
	})
}

func TestCheckRedirectCeilingBoundary(t *testing.T) {
	p := netpolicy.CustomPolicy()
	fn, err := p.CheckRedirect(netpolicy.MaxRedirectsCeiling, nil)
	require.NoError(t, err)
	// Ceiling hops are allowed: via length = max (hops = max-1).
	via := make([]*http.Request, netpolicy.MaxRedirectsCeiling)
	for i := range via {
		via[i] = &http.Request{URL: mustURL(t, "http://example.com/hop")}
	}
	require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/final")}, via),
		"the Ceiling-th hop must be allowed")
	// One more request (hops = Ceiling) must be rejected with E4003.
	via = append(via, &http.Request{URL: mustURL(t, "http://example.com/hop")})
	err = fn(&http.Request{URL: mustURL(t, "http://example.com/too-far")}, via)
	require.Error(t, err)
	typed, ok := apperrors.As(err)
	require.True(t, ok)
	require.Equal(t, apperrors.E4003, typed.Code)
}

func TestCheckRedirectOtherCallbackErrorReturnedVerbatim(t *testing.T) {
	sentinel := errors.New("caller-specific stop")
	p := netpolicy.CustomPolicy()
	fn, err := p.CheckRedirect(2, func(_ *http.Request, _ []*http.Request) error { return sentinel })
	require.NoError(t, err)
	got := fn(&http.Request{URL: mustURL(t, "http://example.com/a")}, nil)
	require.True(t, errors.Is(got, sentinel), "caller error must not be wrapped")
}

func TestCheckRedirectNilRequestNoPanic(t *testing.T) {
	p := netpolicy.CustomPolicy()
	fn, err := p.CheckRedirect(1, nil)
	require.NoError(t, err)
	require.Error(t, fn(nil, nil))
	require.Error(t, fn(&http.Request{}, nil))
}

func TestCheckRedirectCallbackSecondValidation(t *testing.T) {
	p := netpolicy.CustomPolicy()
	t.Run("callback rewrites to invalid URL is rejected", func(t *testing.T) {
		fn, err := p.CheckRedirect(2, func(req *http.Request, _ []*http.Request) error {
			req.URL = mustURL(t, "ftp://example.com/bad")
			return nil
		})
		require.NoError(t, err)
		err = fn(&http.Request{URL: mustURL(t, "http://example.com/ok")}, nil)
		require.Error(t, err, "final URL must be re-validated after callback")
	})
	t.Run("callback sets URL nil is rejected without panic", func(t *testing.T) {
		fn, err := p.CheckRedirect(2, func(req *http.Request, _ []*http.Request) error {
			req.URL = nil
			return nil
		})
		require.NoError(t, err)
		require.Error(t, fn(&http.Request{URL: mustURL(t, "http://example.com/ok")}, nil))
	})
	t.Run("ErrUseLastResponse returned verbatim", func(t *testing.T) {
		fn, err := p.CheckRedirect(2, func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		})
		require.NoError(t, err)
		err = fn(&http.Request{URL: mustURL(t, "http://example.com/ok")}, nil)
		require.True(t, errors.Is(err, http.ErrUseLastResponse))
	})
	t.Run("callback rewrites to another valid URL is allowed", func(t *testing.T) {
		fn, err := p.CheckRedirect(2, func(req *http.Request, _ []*http.Request) error {
			req.URL = mustURL(t, "https://example.com/also-ok")
			return nil
		})
		require.NoError(t, err)
		require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/ok")}, nil))
	})
}

func TestStrictWholeChainDowngrade(t *testing.T) {
	p := netpolicy.StrictPolicy()
	t.Run("https to non-loopback http rejected", func(t *testing.T) {
		fn, err := p.CheckRedirect(3, nil)
		require.NoError(t, err)
		via := []*http.Request{{URL: mustURL(t, "https://example.com/start")}}
		require.Error(t, fn(&http.Request{URL: mustURL(t, "http://example.com/down")}, via))
	})
	t.Run("https to loopback http rejected", func(t *testing.T) {
		fn, err := p.CheckRedirect(3, nil)
		require.NoError(t, err)
		via := []*http.Request{{URL: mustURL(t, "https://example.com/start")}}
		require.Error(t, fn(&http.Request{URL: mustURL(t, "http://localhost/down")}, via))
	})
	t.Run("http to https to http rejected (whole chain)", func(t *testing.T) {
		fn, err := p.CheckRedirect(3, nil)
		require.NoError(t, err)
		via := []*http.Request{
			{URL: mustURL(t, "http://example.com/start")},
			{URL: mustURL(t, "https://example.com/middle")},
		}
		require.Error(t, fn(&http.Request{URL: mustURL(t, "http://example.com/down")}, via))
	})
	t.Run("callback cannot erase HTTPS history", func(t *testing.T) {
		fn, err := p.CheckRedirect(3, func(_ *http.Request, via []*http.Request) error {
			// Try to rewrite history so the final check sees no https hop.
			for _, r := range via {
				r.URL = mustURL(t, "http://example.com/rewritten")
			}
			return nil
		})
		require.NoError(t, err)
		via := []*http.Request{{URL: mustURL(t, "https://example.com/start")}}
		err = fn(&http.Request{URL: mustURL(t, "http://example.com/down")}, via)
		require.Error(t, err, "sawHTTPS snapshot must survive via mutation")
	})
	t.Run("callback cannot rewrite final URL to bypass downgrade", func(t *testing.T) {
		fn, err := p.CheckRedirect(3, func(req *http.Request, _ []*http.Request) error {
			req.URL = mustURL(t, "http://example.com/down")
			return nil
		})
		require.NoError(t, err)
		err = fn(&http.Request{URL: mustURL(t, "https://example.com/start")}, nil)
		require.Error(t, err, "final URL must be re-checked against chain rule")
	})
}

func TestCustomPolicyKeepsLegacySemantics(t *testing.T) {
	p := netpolicy.CustomPolicy()
	fn, err := p.CheckRedirect(3, nil)
	require.NoError(t, err)
	// Custom policy does not implement whole-chain downgrade rejection.
	via := []*http.Request{{URL: mustURL(t, "https://example.com/start")}}
	require.NoError(t, fn(&http.Request{URL: mustURL(t, "http://example.com/down")}, via))
}
