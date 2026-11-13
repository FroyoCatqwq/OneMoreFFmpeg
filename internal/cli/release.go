package cli

import (
	"context"
	"fmt"
	"strings"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
	appgithub "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func catalogFlags() []urfave.Flag {
	return []urfave.Flag{
		&urfave.StringFlag{Name: "provider", Value: provider.IDBtbn, Usage: "provider id"},
		&urfave.StringFlag{Name: "channel", Value: provider.ChannelDaily, Usage: "stable|daily"},
		&urfave.StringFlag{Name: "target", Usage: "win64|linux64 (default: current platform)"},
		&urfave.StringFlag{Name: "variant", Value: "gpl", Usage: "gpl|lgpl|nonfree|nonfree-shared"},
		&urfave.StringFlag{Name: "linkage", Value: "static", Usage: "static|shared"},
		&urfave.IntFlag{Name: "days", Value: provider.DefaultSnapshotDays, Usage: "daily snapshot lookback in days"},
		&urfave.IntFlag{Name: "count", Value: 0, Usage: "limit daily snapshots (0 = all within --days)"},
		&urfave.StringFlag{Name: "releases-url", Usage: "override the FFmpeg official releases page (tests)"},
		&urfave.BoolFlag{Name: "allow-nonfree", Usage: "accept nonfree binaries"},
	}
}

func releaseCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "release",
		Usage: "discover and install FFmpeg releases and snapshots",
		Commands: []*urfave.Command{
			releaseListCommand(),
			releaseInstallCommand(),
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown release subcommand")
			}
			return urfave.DefaultShowSubcommandHelp(cmd)
		},
	}
}

func releaseListCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "list",
		Usage: "list stable versions or recent daily snapshots",
		Flags: catalogFlags(),
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			client, options, err := catalogOptions(ctx, cmd)
			if err != nil {
				return err
			}
			entries, err := provider.ListCatalog(ctx, provider.CatalogOptions{
				Provider: options.Provider, Channel: options.Channel, Target: options.Target,
				Variant: options.Variant, Linkage: options.Linkage, Days: options.Days,
				Count: options.Count, Client: client, ReleasesURL: options.ReleasesURL,
			})
			if err != nil {
				return err
			}
			modelEntries := make([]model.ReleaseCatalogEntry, 0, len(entries))
			for _, entry := range entries {
				modelEntries = append(modelEntries, model.ReleaseCatalogEntry{
					Channel: entry.Channel, Version: entry.Version, Line: entry.Line,
					Snapshot: entry.Snapshot, ReleaseTag: entry.ReleaseTag, AssetName: entry.AssetName,
					AssetURL: entry.AssetURL, PublishedAt: entry.PublishedAt, Provider: entry.Provider,
				})
			}
			source := provider.OfficialReleasesURL
			if options.Channel == provider.ChannelDaily {
				source = client.BaseURL
			}
			effectiveDays := options.Days
			if options.Channel == provider.ChannelStable {
				effectiveDays = 0
			}
			data := model.ReleaseCatalogData{
				Schema: model.ReleaseCatalogSchema, Provider: options.Provider,
				Channel: options.Channel, Days: effectiveDays, Count: options.Count,
				Source: source, Entries: modelEntries,
			}
			human := fmt.Sprintf("versions: %d (channel=%s days=%d count=%d)\n", len(modelEntries), options.Channel, effectiveDays, options.Count)
			for _, entry := range modelEntries {
				human += fmt.Sprintf("- %s tag=%s asset=%s\n", entry.Version, entry.ReleaseTag, entry.AssetName)
			}
			return writeResult(cmd, "release.list", data, human)
		},
	}
}

func releaseInstallCommand() *urfave.Command {
	flags := catalogFlags()
	flags = append(flags,
		&urfave.StringFlag{Name: "version", Usage: "catalog version to install (default: newest)"},
		&urfave.StringFlag{Name: "snapshot", Usage: "snapshot tag to install"},
		&urfave.StringFlag{Name: "artifact", Usage: "downloaded archive path (absolute)"},
		&urfave.BoolFlag{Name: "apply", Usage: "actually install (default is dry-run)"},
		&urfave.BoolFlag{Name: "force", Usage: "replace an existing install directory"},
		&urfave.BoolFlag{Name: "no-current", Usage: "do not update the current pointer"},
		&urfave.BoolFlag{Name: "emit-trio-lock", Usage: "write ffmpeg-trio.lock inside the install directory"},
	)
	return &urfave.Command{
		Name:  "install",
		Usage: "install the newest version or one selected version/snapshot",
		Flags: flags,
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			if strings.TrimSpace(cmd.String("version")) != "" && strings.TrimSpace(cmd.String("snapshot")) != "" {
				return apperrors.New(apperrors.E7001, "--version and --snapshot are mutually exclusive")
			}
			release, err := resolveCatalogRelease(ctx, cmd)
			if err != nil {
				return err
			}
			return installRelease(ctx, cmd, release)
		},
	}
}

type catalogFlagsValues struct {
	Provider    string
	Channel     string
	Target      string
	Variant     string
	Linkage     string
	Days        int
	Count       int
	ReleasesURL string
}

func catalogOptions(ctx context.Context, cmd *urfave.Command) (*appgithub.Client, catalogFlagsValues, error) {
	client, err := newGitHubClient(configFrom(ctx))
	if err != nil {
		return nil, catalogFlagsValues{}, err
	}
	target := cmd.String("target")
	if target == "" {
		target, err = provider.DefaultTarget()
		if err != nil {
			return nil, catalogFlagsValues{}, err
		}
	}
	values := catalogFlagsValues{
		Provider: cmd.String("provider"), Channel: cmd.String("channel"), Target: target,
		Variant: cmd.String("variant"), Linkage: cmd.String("linkage"),
		Days: cmd.Int("days"), Count: cmd.Int("count"), ReleasesURL: cmd.String("releases-url"),
	}
	if values.Days <= 0 {
		values.Days = provider.DefaultSnapshotDays
	}
	if !cmd.IsSet("variant") && strings.EqualFold(strings.TrimSpace(values.Provider), provider.IDBnqdzjNonfree) {
		values.Variant = "nonfree"
	}
	return client, values, nil
}

func resolveCatalogRelease(ctx context.Context, cmd *urfave.Command) (model.Release, error) {
	client, options, err := catalogOptions(ctx, cmd)
	if err != nil {
		return model.Release{}, err
	}
	explicit := strings.TrimSpace(cmd.String("version")) != "" || strings.TrimSpace(cmd.String("snapshot")) != ""
	days := options.Days
	if explicit && days < 3650 {
		days = 3650
	}
	entries, err := provider.ListCatalog(ctx, provider.CatalogOptions{
		Provider: options.Provider, Channel: options.Channel, Target: options.Target,
		Variant: options.Variant, Linkage: options.Linkage, Days: days,
		Count: 0, Client: client, ReleasesURL: options.ReleasesURL,
	})
	if err != nil {
		return model.Release{}, err
	}
	selected := provider.CatalogEntry{}
	for _, entry := range entries {
		if snapshot := strings.TrimSpace(cmd.String("snapshot")); snapshot != "" {
			if entry.Snapshot == snapshot || entry.ReleaseTag == snapshot {
				selected = entry
				break
			}
			continue
		}
		if version := strings.TrimSpace(cmd.String("version")); version != "" {
			if entry.Version == version || entry.Line == version {
				selected = entry
				break
			}
			continue
		}
		selected = entry
		break
	}
	if selected.ReleaseTag == "" {
		return model.Release{}, apperrors.New(apperrors.E3002, "catalog version not found")
	}
	request := provider.Request{
		Provider: options.Provider, Channel: options.Channel, Target: options.Target,
		Variant: options.Variant, Linkage: options.Linkage,
		AllowNonfree: cmd.Bool("allow-nonfree"),
	}
	if selected.Snapshot != "" {
		request.ReleaseTag = selected.Snapshot
	} else {
		request.ReleaseLine = selected.Line
	}
	resolver := &provider.Resolver{Client: client}
	return resolver.Resolve(ctx, provider.NewRegistry(), request, nil)
}
