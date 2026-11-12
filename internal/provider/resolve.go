package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
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
func (r *Resolver) resolveChecksums(ctx context.Context, release github.Release, asset github.Asset) (string, string, string, error) {
	digest := ""
	if value, ok := strings.CutPrefix(asset.Digest, "sha256:"); ok {
		digest = strings.ToLower(strings.TrimSpace(value))
	}
	var checksumsAsset *github.Asset
	for index := range release.Assets {
		if release.Assets[index].Name == checksumsAssetName {
			checksumsAsset = &release.Assets[index]
			break
		}
	}
	if checksumsAsset == nil {
		if digest == "" {
			return "", "", "", apperrors.New(apperrors.E3005, "no trusted checksum is available for the asset").WithDetails(map[string]any{
				"asset": asset.Name,
			})
		}
		return digest, "", "", nil
	}

	// The asset digest is authoritative. A transient checksums failure must not
	// block resolving when the digest is already present; it only skips the
	// independent second source.
	body, fetchErr := r.fetch(ctx, checksumsAsset.BrowserDownloadURL)
	if fetchErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", "", "", fetchErr
		}
		if digest == "" {
			return "", "", "", fetchErr
		}
		return digest, "", "", nil
	}
	sum, _, err := checksum.StreamSHA256(ctx, bytes.NewReader(body))
	if err != nil {
		if digest == "" {
			return "", "", "", err
		}
		return digest, "", "", nil
	}
	checksumsDigest, err := checksum.ParseChecksums(bytes.NewReader(body), asset.Name)
	if err != nil {
		if digest == "" {
			return "", "", "", err
		}
		return digest, "", "", nil
	}
	if digest != "" && digest != checksumsDigest {
		return "", "", "", apperrors.New(apperrors.E4002, "asset digest and checksums file disagree").WithDetails(map[string]any{
			"asset":     asset.Name,
			"digest":    digest,
			"checksums": checksumsDigest,
		})
	}
	if digest == "" {
		digest = checksumsDigest
	}
	return digest, checksumsAsset.BrowserDownloadURL, sum, nil
}

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

func (r *Resolver) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (r *Resolver) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if r != nil && r.Fetch != nil {
		return r.Fetch(ctx, rawURL)
	}
	if !isStrictHTTPSURL(rawURL) {
		return nil, apperrors.New(apperrors.E4001, "checksums url must be https").WithDetails(map[string]any{"url": rawURL})
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E4001, "checksums request could not be built").WithCause(err.Error())
	}
	timeout := 30 * time.Second
	maxRedirects := 5
	if r != nil {
		if r.Timeout > 0 {
			timeout = r.Timeout
		}
		if r.MaxRedirects >= 0 {
			maxRedirects = r.MaxRedirects
		}
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return apperrors.New(apperrors.E4003, "redirect limit exceeded")
			}
			if !isStrictHTTPSURL(req.URL.String()) {
				return apperrors.New(apperrors.E4001, "checksums redirect must be https").WithDetails(map[string]any{"url": req.URL.String()})
			}
			return nil
		},
	}
	response, err := client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, checksumsContextError(ctxErr)
		}
		if typed, ok := apperrors.As(err); ok {
			return nil, typed
		}
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithCause(err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithDetails(map[string]any{"status": response.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, defaultChecksumsLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, checksumsContextError(ctxErr)
		}
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithCause(err.Error())
	}
	if int64(len(body)) > defaultChecksumsLimit {
		return nil, apperrors.New(apperrors.E4004, "checksums file exceeds size limit").WithDetails(map[string]any{"limit": defaultChecksumsLimit})
	}
	return body, nil
}

func checksumsContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(apperrors.E3007, "checksums request timed out").WithCause(err.Error())
	}
	return apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
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
func isStrictHTTPSURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	return parsed.Scheme == "https"
}
