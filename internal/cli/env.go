package cli

import (
	"context"
	"fmt"
	"path/filepath"

	urfave "github.com/urfave/cli/v3"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/doctor"
	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/lock"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
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
		Flags: []urfave.Flag{
			&urfave.BoolFlag{Name: "strict", Usage: "exit non-zero (OMF-E6006) when the report is unhealthy"},
		},
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
			if cmd.Bool("strict") && !report.Healthy {
				return apperrors.New(apperrors.E6006, "doctor detected an unhealthy state").WithDetails(map[string]any{"report": report})
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
			&urfave.BoolFlag{Name: "system", Usage: "write system scope (HKLM / /usr/local/bin + /etc/profile.d); requires elevation"},
			&urfave.BoolFlag{Name: "authorize-system", Usage: "explicit user consent for system scope when not already elevated"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			root, err := paths.DataRoot()
			if err != nil {
				return err
			}
			system := cmd.Bool("system")
			if system {
				if err := platform.AuthorizeSystemScope(
					cmd.Bool("authorize-system"), cmd.Bool("apply")); err != nil {
					return err
				}
			}
			options := appenv.ApplyOptions{DataRoot: root, Apply: cmd.Bool("apply"), NoPath: cmd.Bool("no-path"), NoVariables: cmd.Bool("no-variables"), NoShellRC: cmd.Bool("no-shell-rc"), LAECompat: cmd.Bool("lae-compat"), System: system}
			if options.Apply {
				unlock, err := lock.DataRoot(root)
				if err != nil {
					return err
				}
				defer unlock()
			}
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
			root, err := paths.DataRoot()
			if err != nil {
				return err
			}
			record, err := journal.Read(filepath.Join(root, "journal"), id)
			if err != nil {
				return err
			}
			if record.Action != "env-set" && record.Action != "env-unset" {
				return apperrors.New(apperrors.E7001, "journal is not an env transaction").WithDetails(map[string]any{"action": record.Action})
			}
			if record.Status == journal.StatusRolledBack {
				return apperrors.New(apperrors.E7001, "journal was already rolled back")
			}
			if !cmd.Bool("apply") {
				preview := model.EnvSetData{Schema: model.EnvSetSchema, DryRun: true, Applied: false, Operations: []string{}, JournalID: id, RollbackSupported: true}
				return writeResult(cmd, "env.rollback", preview, "dry-run (use --apply to roll back)\n")
			}
			unlock, err := lock.DataRoot(root)
			if err != nil {
				return err
			}
			defer unlock()
			result, err := appenv.Rollback(ctx, root, id)
			if err != nil {
				return err
			}
			return writeResult(cmd, "env.rollback", result, "rollback applied\n")
		},
	}
}
