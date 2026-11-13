package provider

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

const (
	ChannelStable = "stable"
	ChannelDaily  = "daily"

	defaultChecksumsLimit = 4 << 20
	checksumsAssetName    = "checksums.sha256"
)

// ChecksumsFetcher downloads a small checksums file. Tests inject a fake.
type ChecksumsFetcher func(ctx context.Context, rawURL string) ([]byte, error)

// Resolver turns a Request into a fully pinned Release. It never downloads the
// asset itself; that is the download package's job.
type Resolver struct {
	Client       *github.Client
	Fetch        ChecksumsFetcher
	Now          func() time.Time
	Timeout      time.Duration
	MaxRedirects int
}

// DefaultTarget maps the host platform to the supported asset target.
func DefaultTarget() (string, error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "windows/amd64":
		return "win64", nil
	case "linux/amd64":
		return "linux64", nil
	default:
		return "", apperrors.New(apperrors.E3003, "unsupported platform for provider resolution").WithDetails(map[string]any{
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		})
	}
}

// NormalizeRequest applies the documented defaults (target/variant/linkage)
// and validates mutually exclusive options.
func NormalizeRequest(req Request) (Request, error) {
	if strings.TrimSpace(req.Provider) == "" {
		req.Provider = IDBtbn
	}
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = ChannelDaily
	}
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	if req.Channel != ChannelStable && req.Channel != ChannelDaily {
		return Request{}, apperrors.New(apperrors.E3004, "unsupported channel").WithDetails(map[string]any{"channel": req.Channel})
	}
	if strings.TrimSpace(req.Target) == "" {
		target, err := DefaultTarget()
		if err != nil {
			return Request{}, err
		}
		req.Target = target
	}
	req.Target = strings.ToLower(strings.TrimSpace(req.Target))
	if req.Target != "win64" && req.Target != "linux64" {
		return Request{}, apperrors.New(apperrors.E3003, "unsupported target").WithDetails(map[string]any{"target": req.Target})
	}
	req.Variant, req.Linkage = normalizeVariant(req.Variant, req.Linkage)
	switch req.Variant {
	case "gpl", "lgpl", "nonfree":
	default:
		return Request{}, apperrors.New(apperrors.E7001, "unsupported variant").WithDetails(map[string]any{"variant": req.Variant})
	}
	switch req.Linkage {
	case "static", "shared":
	default:
		return Request{}, apperrors.New(apperrors.E7001, "unsupported linkage").WithDetails(map[string]any{"linkage": req.Linkage})
	}
	req.ReleaseTag = strings.TrimSpace(req.ReleaseTag)
	req.ReleaseLine = strings.ToLower(strings.TrimSpace(req.ReleaseLine))
	if req.ReleaseTag != "" && req.ReleaseLine != "" {
		return Request{}, apperrors.New(apperrors.E7001, "--release-tag cannot be combined with --release-line")
	}
	if req.ReleaseLine != "" && req.ReleaseLine != "master" && !stableLinePattern.MatchString(req.ReleaseLine) {
		return Request{}, apperrors.New(apperrors.E7001, "--release-line must be master or nX.Y").WithDetails(map[string]any{"release_line": req.ReleaseLine})
	}
	if req.ReleaseLine == "master" && req.Channel == ChannelStable {
		return Request{}, apperrors.New(apperrors.E7001, "master line is only available on the daily channel")
	}
	if req.ReleaseLine != "" && req.ReleaseLine != "master" && req.Channel == ChannelDaily {
		return Request{}, apperrors.New(apperrors.E7001, "stable release lines are only available on the stable channel").WithDetails(map[string]any{"release_line": req.ReleaseLine})
	}
	if req.Channel == ChannelDaily && strings.EqualFold(req.ReleaseTag, "latest") {
		return Request{}, apperrors.New(apperrors.E7001, "daily channel requires a concrete release tag")
	}
	return req, nil
}

// Resolve resolves one release for req. Custom providers are looked up in the
// user registry first; built-ins use the GitHub client.
func (r *Resolver) Resolve(ctx context.Context, registry *Registry, req Request, custom []Custom) (model.Release, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized, err := NormalizeRequest(req)
	if err != nil {
		return model.Release{}, err
	}
	req = normalized

	var result model.Release
	if entry, ok := FindCustom(custom, req.Provider); ok {
		result, err = r.resolveCustom(ctx, req, entry)
	} else {
		if registry == nil {
			return model.Release{}, apperrors.New(apperrors.E3001, "provider registry is not configured").WithDetails(map[string]any{"provider": req.Provider})
		}
		item, lookupErr := registry.Get(req.Provider)
		if lookupErr != nil {
			return model.Release{}, lookupErr
		}
		result, err = r.resolveGitHub(ctx, req, item.Summary())
	}
	if err != nil {
		return model.Release{}, err
	}
	return result, nil
}

func (r *Resolver) resolveGitHub(ctx context.Context, req Request, summary model.ProviderSummary) (model.Release, error) {
	if !supportsVariant(summary, req.Variant, req.Linkage) {
		return model.Release{}, apperrors.New(apperrors.E7001, "provider does not offer the requested variant").WithDetails(map[string]any{
			"provider": summary.ID,
			"variant":  req.Variant,
			"linkage":  req.Linkage,
		})
	}
	if !contains(summary.Channels, req.Channel) {
		return model.Release{}, apperrors.New(apperrors.E3004, "provider does not support the requested channel").WithDetails(map[string]any{
			"provider": summary.ID,
			"channel":  req.Channel,
		})
	}
	if r.Client == nil {
		return model.Release{}, apperrors.New(apperrors.E3007, "github client is not configured")
	}
	owner, repo, ok := strings.Cut(summary.Repository, "/")
	if !ok || owner == "" || repo == "" {
		return model.Release{}, apperrors.New(apperrors.E3007, "provider repository is invalid").WithDetails(map[string]any{"repository": summary.Repository})
	}

	release, err := r.pickRelease(ctx, req, owner, repo)
	if err != nil {
		return model.Release{}, err
	}
	if release.PublishedAt.IsZero() {
		return model.Release{}, apperrors.New(apperrors.E3007, "github release has no published_at").WithDetails(map[string]any{"tag": release.TagName})
	}
	asset, _, err := selectAsset(release.Assets, assetQuery{
		Channel:     req.Channel,
		Target:      req.Target,
		Variant:     req.Variant,
		Linkage:     req.Linkage,
		ReleaseLine: req.ReleaseLine,
	})
	if err != nil {
		return model.Release{}, err
	}
	if !isStrictHTTPSURL(asset.BrowserDownloadURL) {
		return model.Release{}, apperrors.New(apperrors.E3007, "github asset url must be https").WithDetails(map[string]any{
			"asset": asset.Name,
			"url":   asset.BrowserDownloadURL,
		})
	}
	warnings, err := CheckNonfreeConsent(summary, req.Variant, req.AllowNonfree)
	if err != nil {
		return model.Release{}, err
	}

	assetSHA, checksumsURL, checksumsSHA, err := r.resolveChecksums(ctx, release, asset)
	if err != nil {
		return model.Release{}, err
	}

	pinnedAt := r.now().UTC().Format(time.RFC3339)
	licenseHint := licenseHintFor(summary.ID, req.Variant)
	return model.Release{
		Schema:          model.ReleaseSchema,
		Provider:        summary.ID,
		Repository:      summary.Repository,
		Channel:         req.Channel,
		ReleaseTag:      release.TagName,
		ReleaseID:       strconv.FormatInt(release.ID, 10),
		PublishedAt:     release.PublishedAt.UTC().Format(time.RFC3339),
		PinnedAt:        pinnedAt,
		AssetName:       asset.Name,
		AssetURL:        asset.BrowserDownloadURL,
		AssetSHA256:     assetSHA,
		ChecksumsURL:    checksumsURL,
		ChecksumsSHA256: checksumsSHA,
		Variant:         req.Variant,
		Linkage:         req.Linkage,
		Target:          req.Target,
		LicenseHint:     licenseHint,
		Redistributable: summary.Redistributable,
		Warnings:        warnings,
	}, nil
}

func (r *Resolver) pickRelease(ctx context.Context, req Request, owner, repo string) (github.Release, error) {
	if req.ReleaseTag != "" {
		return r.Client.ReleaseByTag(ctx, owner, repo, req.ReleaseTag)
	}
	if req.Channel == ChannelStable {
		return r.Client.LatestRelease(ctx, owner, repo)
	}
	releases, err := r.Client.ListReleases(ctx, owner, repo, 100)
	if err != nil {
		return github.Release{}, err
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		if strings.HasPrefix(strings.ToLower(release.TagName), "autobuild-") {
			return release, nil
		}
	}
	return github.Release{}, apperrors.New(apperrors.E3002, "no daily autobuild release found").WithDetails(map[string]any{"repository": owner + "/" + repo})
}

// resolveChecksums prefers the GitHub asset digest and falls back to the
// release's checksums.sha256. Returning no hash at all is OMF-E3005.
func (r *Resolver) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func licenseHintFor(providerID, variant string) string {
	if providerID == IDBnqdzjNonfree {
		return "nonfree"
	}
	switch variant {
	case "lgpl":
		return "lgpl"
	case "nonfree":
		return "nonfree"
	case "gpl":
		return "gpl"
	default:
		return "custom"
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func supportsVariant(summary model.ProviderSummary, variant, linkage string) bool {
	if contains(summary.Variants, variant) {
		return true
	}
	return linkage == "shared" && contains(summary.Variants, variant+"-shared")
}

// isStrictHTTPSURL enforces the https-only rule for GitHub-provided asset and
// checksums URLs. Custom providers are validated the same way at add time.
