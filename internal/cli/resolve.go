package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func providerFlags() []urfave.Flag {
	return []urfave.Flag{
		&urfave.StringFlag{Name: "provider", Usage: "custom provider id (add one with `provider add`)"},
		&urfave.StringFlag{Name: "target", Usage: "win64|linux64 (default: current platform)"},
		&urfave.StringFlag{Name: "variant", Value: "gpl", Usage: "gpl|lgpl|nonfree"},
		&urfave.StringFlag{Name: "linkage", Value: "static", Usage: "static|shared"},
		&urfave.BoolFlag{Name: "allow-nonfree", Usage: "accept nonfree binaries (local use only)"},
	}
}

func requestFromFlags(cmd *urfave.Command) provider.Request {
	return provider.Request{
		Provider:     cmd.String("provider"),
		Channel:      provider.ChannelDaily,
		Target:       cmd.String("target"),
		Variant:      cmd.String("variant"),
		Linkage:      cmd.String("linkage"),
		AllowNonfree: cmd.Bool("allow-nonfree"),
	}
}

func resolveCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "resolve",
		Usage: "resolve one provider release into a pinned asset",
		Flags: providerFlags(),
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			release, err := resolveRelease(ctx, cmd)
			if err != nil {
				return err
			}
			human := fmt.Sprintf("resolved %s/%s\nasset: %s\nsha256: %s\nchecksums: %s\nvariant: %s linkage: %s target: %s\n",
				release.Provider, release.ReleaseTag, release.AssetName, release.AssetSHA256,
				emptyAs(release.ChecksumsURL, "-"), release.Variant, release.Linkage, release.Target)
			human += formatWarnings(release)
			return writeResult(cmd, "resolve", release, human)
		},
	}
}

func resolveRelease(ctx context.Context, cmd *urfave.Command) (model.Release, error) {
	request, err := provider.NormalizeRequest(requestFromFlags(cmd))
	if err != nil {
		return model.Release{}, err
	}
	custom, err := loadCustomProvidersStrict()
	if err != nil {
		return model.Release{}, err
	}
	if _, found := provider.FindCustom(custom, request.Provider); !found {
		return model.Release{}, apperrors.New(apperrors.E3001, "provider id is not registered").
			WithDetails(map[string]any{"provider": request.Provider})
	}
	cfg := configFrom(ctx)
	resolver := &provider.Resolver{Now: time.Now}
	if timeout, err := time.ParseDuration(cfg.Network.Timeout); err == nil {
		resolver.Timeout = timeout
	}
	resolver.MaxRedirects = cfg.Download.MaxRedirects
	return resolver.Resolve(ctx, nil, request, custom)
}

func formatWarnings(release model.Release) string {
	if len(release.Warnings) == 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, warning := range release.Warnings {
		fmt.Fprintf(&builder, "warning: %s: %s\n", warning.Code, warning.Message)
	}
	return builder.String()
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
