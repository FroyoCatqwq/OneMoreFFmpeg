package cli

import (
	"context"
	"fmt"
	"strings"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func providerCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "provider",
		Usage: "inspect and register FFmpeg release providers",
		Commands: []*urfave.Command{
			providerListCommand(),
			providerAddCommand(),
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown provider subcommand").WithDetails(map[string]any{"args": cmd.Args().Slice()})
			}
			return urfave.DefaultShowSubcommandHelp(cmd)
		},
	}
}

func providerListCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "list",
		Usage: "list built-in and custom providers (offline)",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			list := provider.NewRegistry().List()
			custom, err := loadCustomProviders()
			if err != nil {
				return err
			}
			for _, entry := range custom {
				list = append(list, provider.CustomSummary(entry))
			}
			data := model.ProviderListData{
				Schema:    model.ProviderListSchema,
				Providers: list,
			}
			human := fmt.Sprintf("providers: %d\n", len(list))
			for _, item := range list {
				human += fmt.Sprintf("- %s (%s) channels=%v variants=%v linkages=%v\n",
					item.ID, item.Repository, item.Channels, item.Variants, item.Linkages)
			}
			return writeResult(cmd, "provider.list", data, human)
		},
	}
}

func providerAddCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "add",
		Usage: "register a custom direct-asset provider",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "id", Usage: "provider id (lowercase)"},
			&urfave.StringFlag{Name: "url", Usage: "https asset url"},
			&urfave.StringFlag{Name: "sha256", Usage: "asset sha256 pin (64 hex)"},
			&urfave.StringFlag{Name: "checksums-url", Usage: "optional https checksums url"},
			&urfave.StringFlag{Name: "license-hint", Usage: "gpl|lgpl|nonfree|custom"},
			&urfave.BoolFlag{Name: "dry-run", Usage: "validate without writing providers.toml"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			entry := provider.Custom{
				ID:           strings.TrimSpace(cmd.String("id")),
				URL:          strings.TrimSpace(cmd.String("url")),
				SHA256:       strings.ToLower(strings.TrimSpace(cmd.String("sha256"))),
				ChecksumsURL: strings.TrimSpace(cmd.String("checksums-url")),
				LicenseHint:  strings.ToLower(strings.TrimSpace(cmd.String("license-hint"))),
			}
			path, err := paths.ProvidersFile()
			if err != nil {
				return err
			}
			existing, err := provider.LoadCustomFile(path)
			if err != nil {
				return err
			}
			updated, err := provider.AddCustom(existing, entry)
			if err != nil {
				return err
			}
			status := "dry-run"
			if !cmd.Bool("dry-run") {
				if err := provider.SaveCustomFile(path, updated); err != nil {
					return err
				}
				status = "added"
			}
			data := model.ProviderAddData{
				Schema: model.ProviderAddSchema,
				ID:     entry.ID,
				URL:    entry.URL,
				SHA256: entry.SHA256,
				Status: status,
			}
			human := fmt.Sprintf("%s provider %s (%s)\n", status, entry.ID, entry.URL)
			return writeResult(cmd, "provider.add", data, human)
		},
	}
}

// loadCustomProviders returns registered custom providers. An unresolvable
// data root means "no registry" so offline builtin listing still works.
func loadCustomProviders() ([]provider.Custom, error) {
	path, err := paths.ProvidersFile()
	if err != nil {
		if typed, ok := apperrors.As(err); ok && typed.Code == apperrors.E9003 {
			return nil, nil
		}
		return nil, err
	}
	return provider.LoadCustomFile(path)
}

// loadCustomProvidersStrict is used when a custom provider must exist.
func loadCustomProvidersStrict() ([]provider.Custom, error) {
	path, err := paths.ProvidersFile()
	if err != nil {
		return nil, err
	}
	return provider.LoadCustomFile(path)
}
