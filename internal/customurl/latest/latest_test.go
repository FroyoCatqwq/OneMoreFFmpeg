package latest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"20261120", "20261120"},
		{"20261120\n", "20261120"},
		{"20261120\r\n", "20261120"},
		{"20261120T153000Z\n", "20261120T153000Z"},
	} {
		got, err := Parse([]byte(tc.in))
		require.NoError(t, err, tc.in)
		require.Equal(t, tc.want, got)
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{"", "latest", "2026-11-20", "20261120 extra", "20261120\n20261121"} {
		_, err := Parse([]byte(in))
		require.Error(t, err, in)
	}
}
