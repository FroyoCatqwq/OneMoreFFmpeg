package urlpolicy

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {
	require.NoError(t, ValidateURL("https://example.com/a"))
	require.NoError(t, ValidateURL("http://192.168.1.10:8080/a"))
	for _, bad := range []string{"ftp://x/a", "https://user:pass@x/a", "https:///a", "not-a-url"} {
		require.Error(t, ValidateURL(bad), bad)
	}
}

func TestNewClientTLS12(t *testing.T) {
	client, err := NewClient(nil, Policy{})
	require.NoError(t, err)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, uint16(0x0303), transport.TLSClientConfig.MinVersion) // TLS 1.2
	require.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestRedirectChainNoDowngrade(t *testing.T) {
	client, err := NewClient(nil, Policy{})
	require.NoError(t, err)
	// Simulate the CheckRedirect with a chain that already saw https.
	req, err := http.NewRequest("GET", "http://example.com/a", nil)
	require.NoError(t, err)
	via := []*http.Request{{URL: mustParse(t, "https://example.com/start")}}
	err = client.CheckRedirect(req, via)
	require.Error(t, err)
	require.Contains(t, err.Error(), "downgrade")
}

func TestCAPath(t *testing.T) {
	_, err := NewClient(nil, Policy{CAPath: "C:\\nonexistent\\ca.pem"})
	require.Error(t, err)
}

func mustParse(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	require.NoError(t, err)
	return u
}
