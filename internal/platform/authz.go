package platform

import apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"

// AuthorizeSystemScope gates privileged (system-scope) operations. A process
// already running elevated — root on Linux, Administrator on Windows — is
// pre-authorized and may operate directly. Any other caller must have obtained
// explicit user consent (consented). The tool never elevates itself: a
// non-elevated caller is told to re-run under sudo/Administrator.
func AuthorizeSystemScope(consented, apply bool) error {
	return authorizeSystemScope(IsElevated(), consented, apply)
}

func authorizeSystemScope(elevated, consented, apply bool) error {
	if elevated {
		return nil
	}
	if !apply {
		// A dry-run only previews; consent is enough to show what would happen.
		if consented {
			return nil
		}
		return apperrors.New(apperrors.E6003,
			"system scope requires elevation; run as Administrator/root (e.g. sudo), or pass --authorize-system to consent").
			WithDetails(map[string]any{"elevated": false, "authorized": false})
	}
	// v0.6.x: consent is not elevation. Writing HKLM (Windows) or /usr/local/bin
	// plus /etc/profile.d (Linux) needs an elevated process; letting a consented
	// but unelevated --apply through produced a half-written system scope that
	// failed later with exit 1 and a failed journal entry, instead of the
	// argument error E6003 it actually is.
	return apperrors.New(apperrors.E6003,
		"system scope still requires an elevated process; --authorize-system records consent but cannot grant it").
		WithDetails(map[string]any{"elevated": false, "authorized": consented})
}