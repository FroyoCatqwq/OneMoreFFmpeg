package provider

import (
	"context"
	"runtime"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	ChannelStable = "stable"
	ChannelDaily  = "daily"

	defaultChecksumsLimit = 4 << 20
)

// ChecksumsFetcher downloads a small checksums file. Tests inject a fake.
type ChecksumsFetcher func(ctx context.Context, rawURL string) ([]byte, error)

// Resolver turns a Request into a fully pinned Release from a custom provider.
// It never downloads the asset itself; that is the download package's job.
// OneMoreFFmpeg has no built-in download sources: the only supported provider
// is a user-supplied URL + SHA-256 added with `provider add`.
type Resolver struct {
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

// NormalizeRequest applies defaults and validates mutually exclusive options.
// Custom providers only support the daily channel and reject release-line/tag
// selectors (enforced in resolveCustom).
// NormalizeRequestDirectory is NormalizeRequest for directory-mode providers,
// which accept --release-line (a stable version or snapshot build id).
func NormalizeRequestDirectory(req Request) (Request, error) {
	allowLine := req.ReleaseLine
	req.ReleaseLine = ""
	normalized, err := NormalizeRequest(req)
	if err != nil {
		return Request{}, err
	}
	// Preserve case: snapshot timestamps use uppercase T/Z (YYYYMMDDTHHMMSSZ),
	// which naming.Parse/Build require. Lowercasing here broke the CLI entry.
	normalized.ReleaseLine = strings.TrimSpace(allowLine)
	return normalized, nil
}

func NormalizeRequest(req Request) (Request, error) {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	if req.Provider == "" {
		return Request{}, apperrors.New(apperrors.E7001, "provider id is required; add one with `provider add`")
	}
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = ChannelDaily
	}
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	if req.Channel != ChannelDaily {
		return Request{}, apperrors.New(apperrors.E3004, "custom providers only support the daily channel").
			WithDetails(map[string]any{"channel": req.Channel})
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
	req.Variant = strings.ToLower(strings.TrimSpace(req.Variant))
	if req.Variant == "" {
		req.Variant = "gpl"
	}
	req.Linkage = strings.ToLower(strings.TrimSpace(req.Linkage))
	if req.Linkage == "" {
		req.Linkage = "static"
	}
	if req.Variant != "gpl" && req.Variant != "lgpl" && req.Variant != "nonfree" {
		return Request{}, apperrors.New(apperrors.E7001, "unsupported variant").WithDetails(map[string]any{"variant": req.Variant})
	}
	if req.Linkage != "static" && req.Linkage != "shared" {
		return Request{}, apperrors.New(apperrors.E7001, "unsupported linkage").WithDetails(map[string]any{"linkage": req.Linkage})
	}
	req.ReleaseTag = strings.TrimSpace(req.ReleaseTag)
	req.ReleaseLine = strings.ToLower(strings.TrimSpace(req.ReleaseLine))
	if req.ReleaseTag != "" || req.ReleaseLine != "" {
		return Request{}, apperrors.New(apperrors.E7001, "custom providers do not accept --release-tag or --release-line")
	}
	return req, nil
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
	body, err := fetchURL(ctx, rawURL, r.maxRedirects(), r.timeout())
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, apperrors.New(apperrors.E9004, "context canceled").WithCause(ctxErr.Error())
		}
	}
	return body, err
}

func (r *Resolver) maxRedirects() int {
	if r != nil && r.MaxRedirects > 0 {
		return r.MaxRedirects
	}
	return 5
}

func (r *Resolver) timeout() time.Duration {
	if r != nil && r.Timeout > 0 {
		return r.Timeout
	}
	return 30 * time.Second
}

// Resolve resolves one release from a registered custom provider. The
// registry argument is retained for API compatibility but is unused: there
// are no built-in providers.
func (r *Resolver) Resolve(ctx context.Context, _ *Registry, req Request, custom []Custom) (model.Release, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized, err := NormalizeRequest(req)
	if err != nil {
		return model.Release{}, err
	}
	entry, ok := FindCustom(custom, normalized.Provider)
	if !ok {
		return model.Release{}, apperrors.New(apperrors.E3001, "provider id is not registered").
			WithDetails(map[string]any{"provider": normalized.Provider})
	}
	return r.resolveCustom(ctx, normalized, entry)
}
