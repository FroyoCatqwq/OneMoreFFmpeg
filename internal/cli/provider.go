package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
)

func providerCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "provider",
		Usage: "inspect FFmpeg release providers",
		Commands: []*urfave.Command{
			providerListCommand(),
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
		Usage: "list built-in providers (offline)",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			list := provider.NewRegistry().List()
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
