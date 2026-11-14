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
		apply     bool
		wantCode  apperrors.Code
	}{
		{"root_or_admin_direct", true, false, true, ""},
		{"elevated_and_consented", true, true, true, ""},
		// A dry-run may preview with consent alone.
		{"consented_not_elevated_dryrun", false, true, false, ""},
		// Consent is not elevation: an --apply without elevation would write
		// half a system scope and leave a failed journal, so it is refused up
		// front as the argument error E6003.
		{"consented_not_elevated_apply", false, true, true, apperrors.E6003},
		{"refused", false, false, false, apperrors.E6003},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := authorizeSystemScope(tc.elevated, tc.consented, tc.apply)
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