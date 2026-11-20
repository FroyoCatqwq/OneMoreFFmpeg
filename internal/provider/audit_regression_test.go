package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRequestDirectoryPreservesTimestampCase(t *testing.T) {
	// Regression: YYYYMMDDTHHMMSSZ must keep uppercase T/Z so naming can parse it.
	req, err := NormalizeRequestDirectory(Request{
		Provider: "p", Target: "linux64", ReleaseLine: "20261120T153000Z",
	})
	require.NoError(t, err)
	require.Equal(t, "20261120T153000Z", req.ReleaseLine)
}

func TestNormalizeRequestDirectoryStable(t *testing.T) {
	req, err := NormalizeRequestDirectory(Request{Provider: "p", Target: "win64", ReleaseLine: "8.1.0"})
	require.NoError(t, err)
	require.Equal(t, "8.1.0", req.ReleaseLine)
}
