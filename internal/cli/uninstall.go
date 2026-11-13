package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/paths"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

func uninstallCommand() *urfave.Command {
	return &urfave.Command{
		Name:  "uninstall",
		Usage: "remove one managed installation (dry-run by default)",
		Flags: []urfave.Flag{
			&urfave.StringFlag{Name: "root", Usage: "installation root under <data-root>/apps"},
			&urfave.StringFlag{Name: "release", Usage: "release JSON used to locate the install directory"},
			&urfave.StringFlag{Name: "provider", Usage: "provider id (with --release-tag)"},
			&urfave.StringFlag{Name: "release-tag", Usage: "release tag (with --provider)"},
			&urfave.BoolFlag{Name: "apply", Usage: "actually delete (default is dry-run)"},
		},
		Action: func(ctx context.Context, cmd *urfave.Command) error {
			if err := rejectExtraArgs(cmd); err != nil {
				return err
			}
			dataRoot, err := paths.DataRoot()
			if err != nil {
				return err
			}
			root := strings.TrimSpace(cmd.String("root"))
			if root == "" {
				switch {
				case strings.TrimSpace(cmd.String("release")) != "":
					release, err := loadReleaseFile(cmd.String("release"))
					if err != nil {
						return err
					}
					root = install.TargetDir(dataRoot, release)
				case cmd.String("provider") != "" && cmd.String("release-tag") != "":
					pattern := filepath.Join(dataRoot, "apps", cmd.String("provider"), cmd.String("release-tag"), "*")
					matches, err := filepath.Glob(pattern)
					if err != nil || len(matches) != 1 {
						return apperrors.New(apperrors.E7001, "cannot uniquely resolve the installation directory").WithDetails(map[string]any{"pattern": pattern, "matches": len(matches)})
					}
					root = matches[0]
				default:
					return apperrors.New(apperrors.E7001, "use --root, --release, or --provider with --release-tag")
				}
			}
			if !filepath.IsAbs(root) {
				return apperrors.New(apperrors.E7001, "installation root must be an absolute path")
			}
			root = filepath.Clean(root)
			appsRoot := filepath.Join(dataRoot, "apps")
			if rel, err := filepath.Rel(appsRoot, root); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
				return apperrors.New(apperrors.E6007, "refusing to remove a path outside the managed apps directory").WithDetails(map[string]any{"root": root})
			}
			if resolved, err := filepath.EvalSymlinks(root); err == nil {
				resolvedApps, appsErr := filepath.EvalSymlinks(appsRoot)
				if appsErr == nil {
					if rel, err := filepath.Rel(resolvedApps, resolved); err != nil || rel == "." || strings.HasPrefix(rel, "..") {
						return apperrors.New(apperrors.E6007, "installation path resolves outside the managed apps directory").WithDetails(map[string]any{"root": root, "resolved": resolved})
					}
				}
			}
			if _, err := os.Stat(filepath.Join(root, "install.json")); err != nil {
				return apperrors.New(apperrors.E6007, "refusing to remove a directory without install.json").WithDetails(map[string]any{"root": root})
			}
			data := model.UninstallData{Schema: model.UninstallSchema, RootDir: root, DryRun: !cmd.Bool("apply")}
			if !cmd.Bool("apply") {
				return writeResult(cmd, "uninstall", data, "dry-run (use --apply to delete)\nroot: "+root+"\n")
			}
			if _, err := os.Stat(root); err != nil {
				return apperrors.New(apperrors.E7007, "installation root does not exist").WithDetails(map[string]any{"root": root})
			}
			journalID := journal.NewID()
			data.JournalID = journalID
			kind, currentTarget := install.InspectCurrent(dataRoot)
			record := journal.Record{
				ID: journalID, Action: "uninstall", Status: journal.StatusPending, Target: root,
				Details: map[string]string{"current_kind": kind, "current_target": currentTarget},
			}
			if err := journal.Write(filepath.Join(dataRoot, "journal"), record); err != nil {
				return err
			}
			quarantine, err := install.Quarantine(dataRoot, root, journalID)
			if err != nil {
				record.Status = journal.StatusFailed
				record.Details["error"] = err.Error()
				_ = journal.Write(filepath.Join(dataRoot, "journal"), record)
				return err
			}
			record.Details["quarantine"] = quarantine
			if currentTarget != "" && filepath.Clean(currentTarget) == root {
				if err := platform.RemoveLink(filepath.Join(dataRoot, "current")); err != nil {
					_ = os.Rename(quarantine, root)
					record.Status = journal.StatusFailed
					record.Details["error"] = err.Error()
					_ = journal.Write(filepath.Join(dataRoot, "journal"), record)
					return apperrors.New(apperrors.E5004, "current pointer cannot be removed").WithCause(err.Error())
				}
				_ = os.Remove(filepath.Join(dataRoot, "current.pointer"))
			}
			record.Status = journal.StatusCommitted
			if err := journal.Write(filepath.Join(dataRoot, "journal"), record); err != nil {
				return err
			}
			data.Applied = true
			return writeResult(cmd, "uninstall", data, fmt.Sprintf("quarantined %s (rollback: journal rollback --journal %s)\n", root, journalID))
		},
	}
}
