package cli

import (
	"context"
	"fmt"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/doctor"
	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
)

func envCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "env",
		Usage: "inspect or update managed environment and PATH",
		Commands: []*urfave.Command{
			envStatusCommand(),
			envSetCommand(),
			envUnsetCommand(),
			envRollbackCommand(),
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown env subcommand").WithDetails(map[string]any{"args": cmd.Args().Slice()})
			}
			return urfave.DefaultShowSubcommandHelp(cmd)
		},
	}
}

func envStatusCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "status",
		Usage: "show current pointer, PATH and key variables",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			dataRoot, err := paths.DataRoot()
			if err != nil {
				return err
			}
			status := appenv.Status(appenv.Options{DataRoot: dataRoot})
			human := fmt.Sprintf("current: %s (%s)\npath: %v\nffmpeg: %s\n",
				status.CurrentType, emptyAs(status.CurrentTarget, "-"),
				status.PathContainsCurrent, emptyAs(status.ResolvedFFmpeg, "-"))
			return writeResult(cmd, "env.status", status, human)
		},
	}
}

func doctorCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "doctor",
		Usage: "check current pointer, trio, PATH and journals (read-only)",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			dataRoot, err := paths.DataRoot()
			if err != nil {
				return err
			}
			report, err := doctor.Run(ctx, doctor.Options{DataRoot: dataRoot, Runner: newInstallRunner()})
			if err != nil {
				return err
			}
			human := fmt.Sprintf("healthy: %v\n", report.Healthy)
			for _, check := range report.Checks {
				human += fmt.Sprintf("- %s: %v (%s)\n", check.Name, check.OK, check.Detail)
			}
			for _, warning := range report.Warnings {
				human += fmt.Sprintf("warning: %s: %s\n", warning.Code, warning.Message)
			}
			return writeResult(cmd, "doctor", report, human)
		},
	}
}

func envSetCommand() *urfave.Command {
	return envWriteCommand("set", "add current/bin to user PATH and set FFmpeg variables")
}

func envUnsetCommand() *urfave.Command {
	return envWriteCommand("unset", "remove managed PATH entries and variables")
}

func envWriteCommand(name, usage string) *urfave.Command {
	return &urfave.Command{
		Name: name, Usage: usage,
		Flags: []urfave.Flag{
			&urfave.BoolFlag{Name: "apply", Usage: "actually write (default is dry-run)"},
			&urfave.BoolFlag{Name: "no-path", Usage: "do not touch PATH"},
			&urfave.BoolFlag{Name: "no-variables", Usage: "do not touch FFmpeg variables"},
			&urfave.BoolFlag{Name: "lae-compat", Usage: "also set LAE_FFMPEG_DIR"},
			&urfave.BoolFlag{Name: "no-shell-rc", Usage: "do not touch ~/.profile (Linux)"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			root, err := paths.DataRoot()
			if err != nil {
				return err
			}
			options := appenv.ApplyOptions{DataRoot: root, Apply: cmd.Bool("apply"), NoPath: cmd.Bool("no-path"), NoVariables: cmd.Bool("no-variables"), NoShellRC: cmd.Bool("no-shell-rc"), LAECompat: cmd.Bool("lae-compat")}
			var result model.EnvSetData
			if name == "set" {
				result, err = appenv.Apply(ctx, options)
			} else {
				result, err = appenv.Unset(ctx, options)
			}
			if err != nil {
				return err
			}
			human := fmt.Sprintf("%s: applied=%v journal=%s\n", name, result.Applied, emptyAs(result.JournalID, "-"))
			for _, op := range result.Operations {
				human += "- " + op + "\n"
			}
			return writeResult(cmd, "env."+name, result, human)
		},
	}
}

func envRollbackCommand() *urfave.Command {
	return &urfave.Command{
		Name: "rollback", Usage: "restore the previous env state from a journal id",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "journal", Usage: "journal id"},
			&urfave.BoolFlag{Name: "apply", Usage: "actually roll back (default is dry-run)"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			id := cmd.String("journal")
			if id == "" {
				return apperrors.New(apperrors.E7001, "--journal is required")
			}
			if !cmd.Bool("apply") {
				preview := model.EnvSetData{Schema: model.EnvSetSchema, DryRun: true, Applied: false, Operations: []string{}, JournalID: id, RollbackSupported: true}
				return writeResult(cmd, "env.rollback", preview, "dry-run (use --apply to roll back)\n")
			}
			root, err := paths.DataRoot()
			if err != nil {
				return err
			}
			result, err := appenv.Rollback(ctx, root, id)
			if err != nil {
				return err
			}
			return writeResult(cmd, "env.rollback", result, "rollback applied\n")
		},
	}
}
