package platform

import (
	"testing"

	"github.com/stretchr/testify/require"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

func TestAuthorizeSystemScope(t *testing.T) {
	cases := []struct {
		name      string
		elevated  bool
		consented bool
		wantCode  apperrors.Code
	}{
		{"root_or_admin_direct", true, false, ""},
		{"elevated_and_consented", true, true, ""},
		{"consented_not_elevated", false, true, ""},
		{"refused", false, false, apperrors.E6003},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := authorizeSystemScope(tc.elevated, tc.consented)
			if tc.wantCode == "" {
				require.NoError(t, err)
				return
			}
			typed, ok := apperrors.As(err)
			require.True(t, ok)
			require.Equal(t, tc.wantCode, typed.Code)
			require.Equal(t, 2, typed.ExitCode())
		})
	}
}