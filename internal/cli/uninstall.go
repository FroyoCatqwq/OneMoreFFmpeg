package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	urfave "github.com/urfave/cli/v3"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/install"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/lock"
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
			// The three locator styles are mutually exclusive. Silently
			// preferring one could uninstall a different installation than the
			// caller intended.
			locators := 0
			if strings.TrimSpace(cmd.String("root")) != "" {
				locators++
			}
			if strings.TrimSpace(cmd.String("release")) != "" {
				locators++
			}
			if cmd.String("provider") != "" || cmd.String("release-tag") != "" {
				locators++
			}
			if locators > 1 {
				return apperrors.New(apperrors.E7001,
					"use exactly one of --root, --release, or --provider with --release-tag")
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
			if rel, err := filepath.Rel(appsRoot, root); err != nil || rel == "." || outsideApps(rel) {
				return apperrors.New(apperrors.E6007, "refusing to remove a path outside the managed apps directory").WithDetails(map[string]any{"root": root})
			}
			if resolved, err := filepath.EvalSymlinks(root); err == nil {
				resolvedApps, appsErr := filepath.EvalSymlinks(appsRoot)
				if appsErr == nil {
					if rel, err := filepath.Rel(resolvedApps, resolved); err != nil || rel == "." || outsideApps(rel) {
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
			unlock, err := lock.DataRoot(dataRoot)
			if err != nil {
				return err
			}
			defer unlock()
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
			// Persist the quarantine path before any destructive pointer change. If
			// anything below fails, the journal already has everything needed to
			// restore both the directory and `current`.
			journalDir := filepath.Join(dataRoot, "journal")
			if err := journal.Write(journalDir, record); err != nil {
				_ = platform.RenamePath(quarantine, root)
				return apperrors.New(apperrors.E5009, "uninstall journal cannot be written").WithCause(err.Error())
			}
			fail := func(msg string, cause error) error {
				record.Status = journal.StatusFailed
				record.Details["error"] = msg
				if writeErr := journal.Write(journalDir, record); writeErr != nil {
					return apperrors.New(apperrors.E5009,
						"uninstall failed and its journal could not be saved; run doctor and restore "+quarantine).
						WithCause(fmt.Sprintf("%s; journal: %v", msg, writeErr))
				}
				return apperrors.New(apperrors.E5004, msg).WithCause(cause.Error())
			}
			if currentTarget != "" && samePath(currentTarget, root) {
				if err := platform.RemoveLink(filepath.Join(dataRoot, "current")); err != nil {
					_ = platform.RenamePath(quarantine, root)
					return fail("current pointer cannot be removed", err)
				}
				if err := os.Remove(filepath.Join(dataRoot, "current.pointer")); err != nil && !os.IsNotExist(err) {
					// Restore the link we just removed so `current` is not left dangling.
					_ = install.RestoreCurrent(dataRoot, currentTarget, kind)
					_ = platform.RenamePath(quarantine, root)
					return fail("current pointer file cannot be removed", err)
				}
			}
			record.Status = journal.StatusCommitted
			if err := journal.Write(journalDir, record); err != nil {
				// The directory is quarantined and `current` may be gone. Everything
				// needed to restore is in the last persisted journal record (pending
				// with quarantine); surface a recovery hint rather than a bare error.
				return apperrors.New(apperrors.E5009,
					"uninstall committed changes but the final journal write failed; "+
						"run `journal rollback --journal "+journalID+" --apply` to restore").
					WithCause(err.Error())
			}
			data.Applied = true
			return writeResult(cmd, "uninstall", data, fmt.Sprintf("quarantined %s (rollback: journal rollback --journal %s)\n", root, journalID))
		},
	}
}

// outsideApps reports whether a relative path escapes its base directory.
// It avoids the too-broad strings.HasPrefix(rel, ".."), which also rejected
// legitimate names such as "..stable".
func outsideApps(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
