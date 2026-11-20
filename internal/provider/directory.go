package provider

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/customurl/checksum"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/customurl/latest"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/customurl/naming"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/customurl/urlpolicy"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// DirectoryOptions configures directory-mode resolution.
type DirectoryOptions struct {
	BaseClient   *http.Client
	MaxRedirects int
	CAPath       string
	Timeout      time.Duration
	Now          func() time.Time
}

// ResolveDirectory resolves a directory-mode Custom provider into a pinned
// release. It reads the optional snapshots/latest.txt, builds the artifact
// name from the naming convention, and derives checksum.sha256 in the same
// directory (explicit ChecksumsURL wins).
func ResolveDirectory(ctx context.Context, entry Custom, req Request, rawURL string, opts DirectoryOptions) (model.Release, error) {
	if err := urlpolicy.ValidateURL(rawURL); err != nil {
		return model.Release{}, err
	}
	client, err := urlpolicy.NewClient(opts.BaseClient, urlpolicy.Policy{MaxRedirects: opts.MaxRedirects, CAPath: opts.CAPath})
	if err != nil {
		return model.Release{}, err
	}
	if opts.Timeout > 0 {
		client.Timeout = opts.Timeout
	}

	base, err := url.Parse(strings.TrimRight(rawURL, "/") + "/")
	if err != nil {
		return model.Release{}, err
	}
	// Determine the build identifier: explicit --release-line, or latest.txt.
	line := strings.TrimSpace(req.ReleaseLine)
	if line == "" {
		latestURL := base.JoinPath("snapshots", "latest.txt").String()
		line, err = latest.Read(ctx, client, latestURL)
		if err != nil {
			return model.Release{}, apperrors.New(apperrors.E3007,
				"cannot read snapshots/latest.txt; pass --release-line explicitly").WithCause(err.Error())
		}
	}
	releaseType := "snapshot"
	if naming.IsStable(line) {
		releaseType = "stable"
	}
	license := entry.LicenseHint
	if license == "" || license == "custom" {
		license = strings.ToLower(req.Variant)
	}
	// Nonfree always requires explicit consent and emits W1006 at resolve,
	// matching direct-mode behavior (D5).
	var warnings []model.Warning
	if IsNonfreeVariant(req.Variant) || license == "nonfree" {
		warnings, err = CheckNonfreeConsent(
			model.ProviderSummary{ID: entry.ID, Repository: "custom", Channels: []string{"daily"},
				Variants: []string{"gpl", "lgpl", "nonfree"}, Linkages: []string{"static", "shared"}},
			"nonfree", req.AllowNonfree)
		if err != nil {
			return model.Release{}, err
		}
	}
	artifact := naming.Artifact{
		Name: entry.Name, Version: line, License: license, ReleaseType: releaseType,
		OS: targetOS(req.Target), Arch: targetArch(req.Target), Linkage: req.Linkage,
		Format: "tar.xz", // adjusted below per target
	}
	if req.Target == "win64" {
		artifact.Format = "zip"
	}
	name, err := naming.Build(artifact)
	if err != nil {
		return model.Release{}, err
	}

	dir := "snapshots"
	if releaseType == "stable" {
		dir = "stable"
	}
	assetURL := base.JoinPath(dir, line, name).String()
	checksumsURL := entry.ChecksumsURL
	if checksumsURL == "" {
		checksumsURL, err = checksum.ResolveURL(assetURL)
		if err != nil {
			return model.Release{}, err
		}
	}
	body, err := checksum.Fetch(ctx, client, checksumsURL)
	if err != nil {
		return model.Release{}, err
	}
	digest, err := checksum.Parse(body, name)
	if err != nil {
		return model.Release{}, err
	}
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}
	return model.Release{
		Schema: model.ReleaseSchema, Provider: entry.ID, Repository: "custom", Channel: "daily",
		ReleaseTag: line, PublishedAt: now.Format(time.RFC3339), PinnedAt: now.Format(time.RFC3339),
		AssetName: name, AssetURL: assetURL, AssetSHA256: digest,
		ChecksumsURL: checksumsURL, Variant: req.Variant, Linkage: req.Linkage, Target: req.Target,
		LicenseHint: license, Redistributable: license != "nonfree", Warnings: warnings,
	}, nil
}

// targetOS/targetArch map the CLI target token to naming fields.
func targetOS(target string) string {
	if strings.HasPrefix(target, "win") {
		return "windows"
	}
	if strings.HasPrefix(target, "linux") {
		return "linux"
	}
	if strings.HasPrefix(target, "darwin") {
		return "darwin"
	}
	return strings.ToLower(target)
}

func targetArch(target string) string {
	if strings.Contains(target, "64") {
		return "amd64"
	}
	return "amd64" // only x86_64 is supported for now
}
