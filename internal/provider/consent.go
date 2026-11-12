package provider

import (
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// IsNonfreeVariant reports whether a variant denotes a nonfree build.
func IsNonfreeVariant(variant string) bool {
	normalized := strings.ToLower(strings.TrimSpace(variant))
	return normalized == "nonfree" || strings.HasPrefix(normalized, "nonfree-")
}

// CheckNonfreeConsent enforces the D5 policy: nonfree variants require explicit
// consent (--allow-nonfree) and always emit OMF-W1006. Free variants return no
// warnings and no error.
func CheckNonfreeConsent(summary model.ProviderSummary, variant string, allowNonfree bool) ([]model.Warning, error) {
	// A nonfree-only provider requires consent regardless of the requested
	// variant; otherwise `--variant gpl` could silently bypass the policy.
	if summary.ID != IDBnqdzjNonfree && !IsNonfreeVariant(variant) {
		return nil, nil
	}
	details := map[string]any{
		"provider": summary.ID,
		"variant":  variant,
	}
	if !allowNonfree {
		return nil, apperrors.New(apperrors.E7001, "nonfree binaries require --allow-nonfree").WithDetails(details)
	}
	return []model.Warning{{
		Code:    string(apperrors.W1006),
		Message: "nonfree binary: local use only, must not be redistributed",
		Details: details,
	}}, nil
}
