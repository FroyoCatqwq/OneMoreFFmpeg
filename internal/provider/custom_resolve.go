package provider

import (
	"bytes"
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

func (r *Resolver) resolveCustom(ctx context.Context, req Request, entry Custom) (model.Release, error) {
	if req.ReleaseTag != "" || req.ReleaseLine != "" {
		return model.Release{}, apperrors.New(apperrors.E7001, "custom providers do not accept --release-tag or --release-line").WithDetails(map[string]any{"provider": entry.ID})
	}
	licenseHint := entry.LicenseHint
	if licenseHint == "" {
		licenseHint = "custom"
	}
	summary := CustomSummary(entry)
	if !contains(summary.Channels, req.Channel) {
		return model.Release{}, apperrors.New(apperrors.E3004, "provider does not support the requested channel").WithDetails(map[string]any{
			"provider": entry.ID,
			"channel":  req.Channel,
		})
	}
	needsConsent := IsNonfreeVariant(req.Variant) || licenseHint == "nonfree"
	var warnings []model.Warning
	if needsConsent {
		var err error
		warnings, err = CheckNonfreeConsent(summary, "nonfree", req.AllowNonfree)
		if err != nil {
			return model.Release{}, err
		}
	}

	if !isStrictHTTPSURL(entry.URL) {
		return model.Release{}, apperrors.New(apperrors.E3008, "custom provider url is invalid").WithDetails(map[string]any{"url": entry.URL})
	}
	assetName := assetNameFromURL(entry.URL)
	if assetName == "" {
		return model.Release{}, apperrors.New(apperrors.E3008, "custom provider url has no usable file name").WithDetails(map[string]any{"url": entry.URL})
	}
	if entry.ChecksumsURL != "" && !isStrictHTTPSURL(entry.ChecksumsURL) {
		return model.Release{}, apperrors.New(apperrors.E3008, "custom checksums url is invalid").WithDetails(map[string]any{"url": entry.ChecksumsURL})
	}
	pinnedAt := r.now().UTC().Format(time.RFC3339)
	sha := strings.ToLower(strings.TrimSpace(entry.SHA256))
	if !sha256Hex.MatchString(sha) {
		return model.Release{}, apperrors.New(apperrors.E3009, "custom provider requires a sha256 pin").WithDetails(map[string]any{"id": entry.ID})
	}

	checksumsSHA := ""
	if entry.ChecksumsURL != "" {
		body, err := r.fetch(ctx, entry.ChecksumsURL)
		if err != nil {
			return model.Release{}, err
		}
		sum, _, err := checksum.StreamSHA256(ctx, bytes.NewReader(body))
		if err != nil {
			return model.Release{}, err
		}
		digest, err := checksum.ParseChecksums(bytes.NewReader(body), assetName)
		if err != nil {
			return model.Release{}, err
		}
		if digest != sha {
			return model.Release{}, apperrors.New(apperrors.E4002, "checksums file and custom pin disagree").WithDetails(map[string]any{
				"pinned":    sha,
				"checksums": digest,
			})
		}
		checksumsSHA = sum
	}

	return model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        entry.ID,
		Repository:      "custom",
		Channel:         req.Channel,
		ReleaseTag:      "direct-" + sha[:12],
		PublishedAt:     pinnedAt,
		PinnedAt:        pinnedAt,
		AssetName:       assetName,
		AssetURL:        entry.URL,
		AssetSHA256:     sha,
		ChecksumsURL:    entry.ChecksumsURL,
		ChecksumsSHA256: checksumsSHA,
		Variant:         req.Variant,
		Linkage:         req.Linkage,
		Target:          req.Target,
		LicenseHint:     licenseHint,
		Redistributable: false,
		Warnings:        warnings,
	}, nil
}

func isStrictHTTPSURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	return parsed.Scheme == "https"
}
