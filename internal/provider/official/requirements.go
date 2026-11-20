package official

import (
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// requirementsWarning points the user to the upstream README for OS/system
// requirements. We deliberately do NOT parse the README: upstream may reword
// or restructure it at any time, and a parser would silently break. A direct
// link is always correct.
func (r *Resolver) requirementsWarning() []model.Warning {
	rules := r.rules()
	readmeURL := "https://github.com/" + rules.Owner + "/" + rules.Repo + "#readme"
	return []model.Warning{{
		Code:    string(apperrors.W1003),
		Message: "check the upstream README for OS/system requirements (Windows version, UCRT, glibc/kernel) before installing: " + readmeURL,
		Details: map[string]any{"readme_url": readmeURL},
	}}
}
