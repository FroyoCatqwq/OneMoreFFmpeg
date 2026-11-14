package platform

import apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"

// AuthorizeSystemScope gates privileged (system-scope) operations. A process
// already running elevated — root on Linux, Administrator on Windows — is
// pre-authorized and may operate directly. Any other caller must have obtained
// explicit user consent (consented). The tool never elevates itself: a
// non-elevated caller is told to re-run under sudo/Administrator.
func AuthorizeSystemScope(consented bool) error {
	return authorizeSystemScope(IsElevated(), consented)
}

func authorizeSystemScope(elevated, consented bool) error {
	if elevated || consented {
		return nil
	}
	return apperrors.New(apperrors.E6003,
		"system scope requires elevation; run as Administrator/root (e.g. sudo), or pass --authorize-system to consent").
		WithDetails(map[string]any{"elevated": elevated, "authorized": consented})
}