package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/buildinfo"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/selfupdate"
)

// updateCommand exposes the self-update feature. Default is a dry-run check;
// --apply performs the download + verified replacement.
func updateCommand(info buildinfo.Info) *urfave.Command {
	return &urfave.Command{
		Name:  "update",
		Usage: "check for a newer OneMoreFFmpeg and optionally self-update (default: dry-run)",
		Flags: []urfave.Flag{
			&urfave.BoolFlag{Name: "check", Usage: "only check for updates without applying (default)"},
			&urfave.BoolFlag{Name: "apply", Usage: "actually download and install the update"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.Bool("apply") && cmd.Bool("check") {
				return apperrors.New(apperrors.E1001, "--apply and --check are mutually exclusive")
			}
			exe, err := os.Executable()
			if err != nil {
				return apperrors.New(apperrors.E9003, "cannot determine executable path").WithCause(err.Error())
			}
			cfg := configFrom(ctx)
			opts := selfupdate.Options{
				CurrentVersion: info.Version,
				Executable:     exe,
				GitHubAPI:      cfg.Network.GitHubAPI,
				MaxAttempts:    cfg.Network.MaxAttempts,
			}
			update, err := opts.Check(ctx)
			if err != nil {
				return err
			}
			data := model.UpdateData{
				Schema:          model.UpdateSchema,
				CurrentVersion:  info.Version,
				UpdateAvailable: update != nil,
			}
			if update != nil {
				data.LatestVersion = update.Version
				data.AssetName = update.AssetName
				data.ReleaseURL = update.ReleaseURL
			}
			human := fmt.Sprintf("current: %s\n", info.Version)
			if update == nil {
				human += "already up to date\n"
				data.Operations = []string{"check: up to date"}
				return writeResult(cmd, "update", data, human)
			}
			human += fmt.Sprintf("update available: %s (%s)\n", update.Version, update.ReleaseURL)
			if !cmd.Bool("apply") {
				human += "dry run: pass --apply to install\n"
				data.Operations = []string{"check: update available"}
				return writeResult(cmd, "update", data, human)
			}

			var progress io.Writer
			if !jsonEnabled(cmd) {
				progress = stderrFrom(ctx)
			}
			opts.ProgressWriter = progress
			if err := opts.Apply(ctx, update); err != nil {
				return err
			}
			data.Applied = true
			data.Operations = []string{"download", "verify sha256", "replace executable"}
			human += fmt.Sprintf("applied update to %s\n", update.Version)
			return writeResult(cmd, "update", data, human)
		},
	}
}
