// Package official resolves the FFmpeg-Builds "latest" release into a pinned
// asset. All naming rules live in Rules (data, not code) so a future change to
// the upstream repository's asset names requires only a rules update.
package official

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	gh "github.com/google/go-github/v93/github"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	DefaultOwner = "FFmpeg-Builds"
	DefaultRepo  = "FFmpeg-Builds"
	DefaultTag   = "latest"
	ProviderID   = "official"
)

// Rules is the configurable description of an upstream GitHub release layout.
// Defaults() mirrors FFmpeg-Builds; callers can override any field.
type Rules struct {
	Owner   string
	Repo    string
	Tag     string
	APIBase string
	// AssetTemplate renders an asset base name from the variables below.
	// Default follows "ffmpeg-<line>-latest-<target>-<variant><suffix><ext>".
	AssetTemplate string
	// TargetDir maps a logical target to the token used in asset names.
	TargetDir map[string]string
	// Extensions maps a target to its archive extension.
	Extensions map[string]string
	// SnapshotLine is the line token for the floating master snapshot.
	SnapshotLine string
	// StableLinePattern matches stable release lines (e.g. "n9.0"). Stable
	// lines are intentionally rejected until their naming is finalized.
	StableLinePattern string
}

// DefaultRules returns the FFmpeg-Builds rules. It is the single place where
// upstream naming conventions are encoded.
func DefaultRules() Rules {
	return Rules{
		Owner: DefaultOwner, Repo: DefaultRepo, Tag: DefaultTag,
		APIBase:           "https://api.github.com",
		AssetTemplate:     `ffmpeg-{{.Line}}-latest-{{.Target}}-{{.Variant}}{{.LinkageSuffix}}{{.Ext}}`,
		TargetDir:         map[string]string{"win64": "win64", "linux64": "linux64"},
		Extensions:        map[string]string{"win64": ".zip", "linux64": ".tar.xz"},
		SnapshotLine:      "master",
		StableLinePattern: `^n[0-9]+\.[0-9]+$`,
	}
}

// Request selects one asset from the official release.
type Request struct {
	Target  string // win64 | linux64
	Variant string // gpl | lgpl | nonfree
	Linkage string // static | shared
	Line    string // "" => snapshot (master); nX.Y => stable (rejected)
}

// Normalize validates and applies defaults (gpl + static + snapshot).
func (r Request) Normalize() (Request, error) {
	if r.Variant == "" {
		r.Variant = "gpl"
	}
	if r.Linkage == "" {
		r.Linkage = "static"
	}
	// Default line is applied in Resolve using rules.SnapshotLine (configurable).
	if r.Target != "win64" && r.Target != "linux64" {
		return Request{}, apperrors.New(apperrors.E3003, "official provider supports x86_64 only (win64|linux64)").
			WithDetails(map[string]any{"target": r.Target})
	}
	switch r.Variant {
	case "gpl", "lgpl", "nonfree":
	default:
		return Request{}, apperrors.New(apperrors.E7001, "unsupported variant").WithDetails(map[string]any{"variant": r.Variant})
	}
	switch r.Linkage {
	case "static", "shared":
	default:
		return Request{}, apperrors.New(apperrors.E7001, "unsupported linkage").WithDetails(map[string]any{"linkage": r.Linkage})
	}
	return r, nil
}

// Resolver queries GitHub and selects an asset from a release.
type Resolver struct {
	Rules      Rules
	HTTPClient *http.Client
	Client     *gh.Client
	Now        func() time.Time
}

// rules returns the effective rules: every non-zero field of the caller's
// Rules overrides the default. This makes partial overrides (e.g. only
// APIBase) work, which the previous all-or-nothing Owner check broke.
func (r *Resolver) rules() Rules {
	out := DefaultRules()
	if r == nil {
		return out
	}
	c := r.Rules
	if c.Owner != "" {
		out.Owner = c.Owner
	}
	if c.Repo != "" {
		out.Repo = c.Repo
	}
	if c.Tag != "" {
		out.Tag = c.Tag
	}
	if c.APIBase != "" {
		out.APIBase = c.APIBase
	}
	if c.AssetTemplate != "" {
		out.AssetTemplate = c.AssetTemplate
	}
	if len(c.TargetDir) > 0 {
		out.TargetDir = c.TargetDir
	}
	if len(c.Extensions) > 0 {
		out.Extensions = c.Extensions
	}
	if c.SnapshotLine != "" {
		out.SnapshotLine = c.SnapshotLine
	}
	if c.StableLinePattern != "" {
		out.StableLinePattern = c.StableLinePattern
	}
	return out
}

func (r *Resolver) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// assetName renders the expected asset name from the rules template.
func (r *Resolver) assetName(req Request) (string, error) {
	rules := r.rules()
	tmpl, err := template.New("asset").Parse(rules.AssetTemplate)
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "invalid asset naming rule").WithCause(err.Error())
	}
	linkageSuffix := ""
	if req.Linkage == "shared" {
		linkageSuffix = "-shared"
	}
	target, ok := rules.TargetDir[req.Target]
	if !ok {
		return "", apperrors.New(apperrors.E3003, "target not allowed by rules").WithDetails(map[string]any{"target": req.Target})
	}
	ext, ok := rules.Extensions[req.Target]
	if !ok {
		return "", apperrors.New(apperrors.E3003, "extension not configured for target").WithDetails(map[string]any{"target": req.Target})
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Line": req.Line, "Target": target, "Variant": req.Variant,
		"LinkageSuffix": linkageSuffix, "Ext": ext,
	}); err != nil {
		return "", apperrors.New(apperrors.E9003, "cannot render asset rule").WithCause(err.Error())
	}
	return buf.String(), nil
}

// enterpriseBase strips a trailing /api/v3 so go-github's WithEnterpriseURLs
// does not append it twice (users commonly configure the full REST base).
func enterpriseBase(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	b = strings.TrimSuffix(b, "/api/v3")
	return strings.TrimRight(b, "/")
}

// Resolve queries the upstream latest release and returns a pinned model.Release.
func (r *Resolver) Resolve(ctx context.Context, req Request) (model.Release, error) {
	req, err := req.Normalize()
	if err != nil {
		return model.Release{}, err
	}
	rules := r.rules()
	if req.Line == "" {
		req.Line = rules.SnapshotLine // configurable default
	}
	// Stable lines are intentionally unsupported (see user requirement).
	if req.Line != rules.SnapshotLine {
		stable, compileErr := regexp.Compile(rules.StableLinePattern)
		if compileErr != nil {
			return model.Release{}, apperrors.New(apperrors.E7005,
				"invalid stable-line rule in provider configuration").WithCause(compileErr.Error())
		}
		if stable.MatchString(req.Line) {
			return model.Release{}, apperrors.New(apperrors.E3004,
				"stable lines are not supported yet (n* asset naming is under review)").
				WithDetails(map[string]any{"line": req.Line})
		}
		return model.Release{}, apperrors.New(apperrors.E7001, "unsupported release line").WithDetails(map[string]any{"line": req.Line})
	}

	client := r.Client
	if client == nil {
		hc := r.HTTPClient
		if hc == nil {
			hc = &http.Client{Timeout: 30 * time.Second}
		}
		opts := []gh.ClientOptionsFunc{gh.WithHTTPClient(hc)}
		if rules.APIBase != "" && rules.APIBase != "https://api.github.com" {
			base := enterpriseBase(rules.APIBase)
			opts = append(opts, gh.WithEnterpriseURLs(base, base))
		}
		client, err = gh.NewClient(opts...)
		if err != nil {
			return model.Release{}, err
		}
	}
	release, _, err := client.Repositories.GetReleaseByTag(ctx, rules.Owner, rules.Repo, rules.Tag)
	if err != nil {
		return model.Release{}, apperrors.New(apperrors.E3007, "upstream release cannot be fetched").WithCause(err.Error())
	}
	if release.GetDraft() || release.GetPrerelease() {
		return model.Release{}, apperrors.New(apperrors.E3007, "upstream release is not stable")
	}

	name, err := r.assetName(req)
	if err != nil {
		return model.Release{}, err
	}
	var chosen *gh.ReleaseAsset
	for _, a := range release.Assets {
		if a != nil && a.GetName() == name {
			chosen = a
			break
		}
	}
	if chosen == nil {
		return model.Release{}, apperrors.New(apperrors.E3008, "asset not found in upstream release").
			WithDetails(map[string]any{"asset": name, "release": rules.Tag})
	}
	digest := strings.TrimPrefix(chosen.GetDigest(), "sha256:")
	if !checksum.IsLowerHex64(digest) {
		return model.Release{}, apperrors.New(apperrors.E3008, "upstream asset has no valid sha256 digest").
			WithDetails(map[string]any{"asset": name})
	}

	out := model.Release{
		Schema: model.ReleaseSchema, Provider: ProviderID,
		Repository: rules.Owner + "/" + rules.Repo, Channel: "daily",
		ReleaseTag: release.GetTagName(), ReleaseID: fmt.Sprintf("%d", release.GetID()),
		PublishedAt: release.GetPublishedAt().UTC().Format(time.RFC3339), PinnedAt: r.now().UTC().Format(time.RFC3339),
		AssetName: name, AssetURL: chosen.GetBrowserDownloadURL(), AssetSHA256: strings.ToLower(digest),
		Variant: req.Variant, Linkage: req.Linkage, Target: req.Target,
		LicenseHint: strings.ToLower(req.Variant), Redistributable: req.Variant != "nonfree",
	}
	// System requirements are derived from the upstream README, not hardcoded.
	out.Warnings = r.requirementsWarning()
	return out, nil
}
