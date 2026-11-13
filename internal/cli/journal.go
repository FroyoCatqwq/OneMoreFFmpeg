package cli

import (
	"context"
	"fmt"
	"path/filepath"

	urfave "github.com/urfave/cli/v3"

	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
)

func journalCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "journal",
		Usage: "inspect and roll back install/environment transactions",
		Commands: []*urfave.Command{
			journalListCommand(),
			journalRollbackCommand(),
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if cmd.NArg() > 0 {
				return apperrors.New(apperrors.E7001, "unknown journal subcommand")
			}
			return urfave.DefaultShowSubcommandHelp(cmd)
		},
	}
}

func journalListCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "list",
		Usage: "list transaction journals",
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			dataRoot, err := paths.DataRoot()
			if err != nil {
				return err
			}
			records, err := journal.List(filepath.Join(dataRoot, "journal"))
			if err != nil {
				return err
			}
			result := model.JournalListData{Schema: model.JournalListSchema, Records: []model.JournalEntry{}, Pending: []string{}}
			for _, record := range records {
				result.Records = append(result.Records, model.JournalEntry{
					ID: record.ID, Action: record.Action, Status: record.Status,
					Target: record.Target, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
				})
				if record.Status == journal.StatusPending || record.Status == journal.StatusFailed {
					result.Pending = append(result.Pending, record.ID)
				}
			}
			return writeResult(cmd, "journal.list", result, fmt.Sprintf("journals: %d pending=%d\n", len(result.Records), len(result.Pending)))
		},
	}
}

func journalRollbackCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "rollback",
		Usage: "roll back one install/uninstall/env transaction",
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
				preview := model.JournalActionData{Schema: model.JournalActionSchema, JournalID: id, Action: "rollback", Applied: false, Detail: "dry-run (use --apply to roll back)"}
				return writeResult(cmd, "journal.rollback", preview, preview.Detail+"\n")
			}
			dataRoot, err := paths.DataRoot()
			if err != nil {
				return err
			}
			journalDir := filepath.Join(dataRoot, "journal")
			record, err := journal.Read(journalDir, id)
			if err != nil {
				return err
			}
			if record.Status == journal.StatusRolledBack {
				return apperrors.New(apperrors.E7001, "journal was already rolled back")
			}
			detail := ""
			switch record.Action {
			case "env-set", "env-unset":
				result, err := appenv.Rollback(ctx, dataRoot, id)
				if err != nil {
					return err
				}
				detail = fmt.Sprintf("%d operations restored", len(result.Operations))
			case "install", "uninstall":
				detail, err = install.Rollback(dataRoot, record)
				if err != nil {
					return err
				}
				record.Status = journal.StatusRolledBack
				if err := journal.Write(journalDir, record); err != nil {
					return err
				}
			default:
				return apperrors.New(apperrors.E7001, "journal action cannot be rolled back").WithDetails(map[string]any{"action": record.Action})
			}
			result := model.JournalActionData{Schema: model.JournalActionSchema, JournalID: id, Action: record.Action, Applied: true, Detail: detail}
			return writeResult(cmd, "journal.rollback", result, detail+"\n")
		},
	}
}
