// Package netpolicy centralizes URL and redirect policy for OneMoreFFmpeg.
//
// It exposes two behavior families that must never be merged:
//   - StrictPolicy: control-plane URLs (only https; http loopback only);
//   - CustomPolicy: user-supplied static-hosting URLs (http + https).
//
// The Policy type is closed: all fields are unexported and the only valid
// constructors are StrictPolicy()/CustomPolicy(). The zero value fails closed.
//
// This package only validates URLs and builds redirect callbacks. It does not
// perform I/O, own an HTTP client, or depend on any feature package.
package netpolicy

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// MaxRedirectsCeiling is the hard upper bound for redirect hops. Configuration
// defaults belong to the config package; this is only the runtime ceiling.
const MaxRedirectsCeiling = 10

// policyKind is unexported: external packages cannot construct an arbitrary
// policy value. iota zero means invalid so the zero value fails closed.
type policyKind uint8

const (
	policyInvalid policyKind = iota
	policyStrict
	policyCustom
)

// Policy is a closed value type (no func/map/slice fields => comparable).
type Policy struct {
	kind policyKind
}

// StrictPolicy returns the strict (control-plane) URL policy.
func StrictPolicy() Policy { return Policy{kind: policyStrict} }

// CustomPolicy returns the custom (user-supplied static hosting) URL policy.
func CustomPolicy() Policy { return Policy{kind: policyCustom} }

// Validate checks one URL. It is a single-URL check: it has no knowledge of
// redirect history (see CheckRedirect for whole-chain rules).
func (p Policy) Validate(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return apperrors.New(apperrors.E3008, "url is empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil || parsed.Host == "" {
		return apperrors.New(apperrors.E3008, "url is invalid").WithDetails(map[string]any{"url": trimmed})
	}
	if parsed.User != nil {
		return apperrors.New(apperrors.E3008, "url must not contain credentials").WithDetails(map[string]any{"url": parsed.Redacted()})
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch p.kind {
	case policyStrict:
		if scheme == "https" {
			return nil
		}
		if scheme == "http" && IsLoopbackHost(parsed.Hostname()) {
			return nil
		}
		return apperrors.New(apperrors.E3008, "url scheme must be https (http only for loopback)").WithDetails(map[string]any{"url": trimmed})
	case policyCustom:
		if scheme == "http" || scheme == "https" {
			return nil
		}
		return apperrors.New(apperrors.E3008, "url scheme must be http or https").WithDetails(map[string]any{"url": trimmed})
	default:
		// Zero value / forged policy: fail closed.
		return apperrors.New(apperrors.E3008, "uninitialized url policy (fail closed)")
	}
}

// IsLoopbackHost reports whether host is a loopback address.
// It covers 127.0.0.0/8, ::1, expanded IPv6, and "localhost" (case-insensitive).
func IsLoopbackHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	h := strings.Trim(host, "[]")
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// CheckRedirect builds a callback for http.Client.CheckRedirect.
//
// maxRedirects is the maximum number of redirect HOPS (not the maximum number
// of requests). With via[0] being the initial request, len(via)-1 is the number
// of hops already taken; the limit check is len(via)-1 >= maxRedirects.
//
// Invalid maxRedirects (<=0 or >MaxRedirectsCeiling) is rejected before any
// callback is returned: no clamping, no implicit default.
//
// Callback order (must not be reordered):
//  0. req == nil or req.URL == nil -> typed error, never panic;
//  1. hop limit check -> E4003;
//  2. pre-Validate of the current target (Strict also snapshots chain HTTPS
//     state HERE, before original can mutate via);
//  3. call original (if non-nil); a non-nil return (including
//     http.ErrUseLastResponse) is returned verbatim, without wrapping;
//  4. if original returned nil, re-check req/URL and re-Validate the FINAL URL,
//     including the Strict whole-chain downgrade rule using the earlier
//     snapshot.
func (p Policy) CheckRedirect(maxRedirects int, original func(*http.Request, []*http.Request) error) (func(*http.Request, []*http.Request) error, error) {
	if p.kind != policyStrict && p.kind != policyCustom {
		return nil, apperrors.New(apperrors.E3008, "uninitialized url policy (fail closed)")
	}
	if maxRedirects <= 0 || maxRedirects > MaxRedirectsCeiling {
		return nil, apperrors.New(apperrors.E4003, "maxRedirects out of range").WithDetails(map[string]any{
			"max_redirects": maxRedirects, "ceiling": MaxRedirectsCeiling,
		})
	}

	return func(req *http.Request, via []*http.Request) error {
		if req == nil || req.URL == nil {
			return apperrors.New(apperrors.E3008, "redirect request has no URL")
		}
		hops := len(via) - 1
		if hops < 0 {
			hops = 0
		}
		if hops >= maxRedirects {
			return apperrors.New(apperrors.E4003, "too many redirects").WithDetails(map[string]any{
				"max_redirects": maxRedirects, "hops": hops,
			})
		}

		// Strict whole-chain state MUST be captured before original runs:
		// via is []*http.Request and original may mutate its elements.
		sawHTTPS := p.kind == policyStrict && strings.EqualFold(req.URL.Scheme, "https")
		if !sawHTTPS && p.kind == policyStrict {
			for _, previous := range via {
				if previous != nil && previous.URL != nil && strings.EqualFold(previous.URL.Scheme, "https") {
					sawHTTPS = true
					break
				}
			}
		}
		if err := p.validateWithChain(req.URL.String(), sawHTTPS); err != nil {
			return err
		}
		if original != nil {
			if err := original(req, via); err != nil {
				return err // verbatim: preserves http.ErrUseLastResponse semantics
			}
		}
		// original may have rewritten req.URL or set it to nil.
		if req == nil || req.URL == nil {
			return apperrors.New(apperrors.E3008, "redirect request has no URL after callback")
		}
		finalSawHTTPS := sawHTTPS || (p.kind == policyStrict && strings.EqualFold(req.URL.Scheme, "https"))
		return p.validateWithChain(req.URL.String(), finalSawHTTPS)
	}, nil
}

// validateWithChain applies the single-URL rule plus (for Strict) the
// whole-chain no-downgrade rule. sawHTTPS is a snapshot taken before any
// caller callback can mutate via.
func (p Policy) validateWithChain(raw string, sawHTTPS bool) error {
	if err := p.Validate(raw); err != nil {
		return err
	}
	if p.kind == policyStrict && sawHTTPS {
		parsed, err := url.Parse(raw)
		if err == nil && parsed != nil && strings.EqualFold(parsed.Scheme, "http") {
			return apperrors.New(apperrors.E3008, "redirect must not downgrade https")
		}
	}
	return nil
}
