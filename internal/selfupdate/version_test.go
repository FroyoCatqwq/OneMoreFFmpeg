package selfupdate

import (
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStrictVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
		ok    bool
	}{
		{"1.1.0", [3]int{1, 1, 0}, true},
		{"1.10.0", [3]int{1, 10, 0}, true},
		{"0.0.0", [3]int{0, 0, 0}, true},
		{"1.2", [3]int{}, false},
		{"1.2.3.4", [3]int{}, false},
		{"1..3", [3]int{}, false},
		{"-1.2.3", [3]int{}, false},
		{"1.2.-3", [3]int{}, false},
		{"01.2.3", [3]int{}, false},
		{"1.1.0-rc.1", [3]int{}, false},
		{"v1.1.0", [3]int{}, false}, // tag prefix normalized by Check
		{"", [3]int{}, false},
		{strconv.Itoa(math.MaxInt) + ".0.0x", [3]int{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseStrictVersion(tc.input)
			if tc.ok {
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestIsNewerNumeric(t *testing.T) {
	require.True(t, isNewer("1.10.0", "1.9.0"))
	require.True(t, isNewer("2.0.0", "1.99.99"))
	require.True(t, isNewer("1.1.1", "dev"), "dev local build is older than any release")
	require.False(t, isNewer("1.1.0", "1.1.0"))
	require.False(t, isNewer("1.0.9", "1.1.0"))
	require.False(t, isNewer("bad", "1.0.0"), "invalid candidate must never be newer")
}
