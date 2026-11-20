// Package urlpolicy centralizes URL/TLS/redirect safety for custom-URL
// downloads. No SSRF host filtering is applied (by design: this is a local CLI
// with user-supplied URLs, not a server proxying arbitrary requests).
package urlpolicy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// Policy controls HTTP client behavior.
type Policy struct {
	MaxRedirects int
	CAPath       string // optional PEM bundle for a private CA
	MinTLS       uint16 // 0 => TLS 1.2
}

// ValidateURL accepts absolute http(s) URLs with a host and no userinfo.
func ValidateURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return apperrors.New(apperrors.E3008, "url is invalid or contains credentials")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return apperrors.New(apperrors.E3008, "url scheme must be http or https")
	}
	return nil
}

// NewClient builds an http.Client with TLS >=1.2, normal certificate
// verification and redirect-chain checks (no downgrade at any hop).
func NewClient(base *http.Client, p Policy) (*http.Client, error) {
	if p.MaxRedirects <= 0 {
		p.MaxRedirects = 10
	}
	if p.MinTLS == 0 {
		p.MinTLS = tls.VersionTLS12
	}
	client := &http.Client{}
	if base != nil {
		*client = *base
	}
	transport := &http.Transport{}
	if base != nil && base.Transport != nil {
		if t, ok := base.Transport.(*http.Transport); ok {
			transport = t.Clone()
		}
	}
	tlsConfig := &tls.Config{MinVersion: p.MinTLS} // InsecureSkipVerify stays false
	if p.CAPath != "" {
		pem, err := os.ReadFile(p.CAPath)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, apperrors.New(apperrors.E7007, "CA file contains no valid certificates")
		}
		tlsConfig.RootCAs = pool
	}
	transport.TLSClientConfig = tlsConfig
	client.Transport = transport

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= p.MaxRedirects {
			return apperrors.New(apperrors.E3007, "too many redirects")
		}
		if err := ValidateURL(req.URL.String()); err != nil {
			return err
		}
		// Inspect the whole chain (no shared mutable state across requests):
		// if any hop was https, an http target now is a downgrade, including
		// the http -> https -> http case.
		if strings.EqualFold(req.URL.Scheme, "http") {
			for _, previous := range via {
				if strings.EqualFold(previous.URL.Scheme, "https") {
					return apperrors.New(apperrors.E3008, "redirect must not downgrade https")
				}
			}
		}
		return nil
	}
	return client, nil
}
