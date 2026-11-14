package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

type ApplyOptions struct {
	DataRoot    string
	Apply       bool
	NoPath      bool
	NoVariables bool
	NoShellRC   bool
	LAECompat   bool
	Store       platform.EnvStore
	Now         func() time.Time
}

const (
	shellRCMarkerStart = "# >>> OneMoreFFmpeg >>>"
	shellRCMarkerEnd   = "# <<< OneMoreFFmpeg <<<"
	shimMarker         = "# OneMoreFFmpeg managed shim"
)

type envAction struct {
	description string
	apply       func() error
	undo        func() error
}

type envPlan struct {
	operations []string
	details    map[string]string
	actions    []envAction
	store      platform.EnvStore
}

// Apply manages the user PATH/variables (Windows) or shims (Linux).
func Apply(ctx context.Context, opts ApplyOptions) (model.EnvSetData, error) {
	return runEnvTransaction(ctx, opts, "env-set", true)
}

// Unset removes managed PATH entries, variables and shims.
func Unset(ctx context.Context, opts ApplyOptions) (model.EnvSetData, error) {
	return runEnvTransaction(ctx, opts, "env-unset", false)
}

func runEnvTransaction(ctx context.Context, opts ApplyOptions, action string, add bool) (model.EnvSetData, error) {
	_ = ctx
	if !filepath.IsAbs(opts.DataRoot) {
		return model.EnvSetData{}, apperrors.New(apperrors.E7001, "data root must be absolute")
	}
	plan, err := buildEnvPlan(opts, add)
	if err != nil {
		return model.EnvSetData{}, err
	}
	result := model.EnvSetData{
		Schema:            model.EnvSetSchema,
		DryRun:            !opts.Apply,
		Operations:        plan.operations,
		RollbackSupported: opts.Apply,
	}
	if !opts.Apply {
		return result, nil
	}

	journalDir := filepath.Join(opts.DataRoot, "journal")
	id := journal.NewID()
	record := journal.Record{
		ID: id, Action: action, Status: journal.StatusPending,
		Target: opts.DataRoot, Details: copyDetails(plan.details),
	}
	if err := journal.Write(journalDir, record); err != nil {
		return result, err
	}
	fail := func(cause error) (model.EnvSetData, error) {
		record.Status = journal.StatusFailed
		record.Details = copyDetails(plan.details)
		record.Details["error"] = cause.Error()
		_ = journal.Write(journalDir, record)
		return result, cause
	}

	for index, step := range plan.actions {
		if err := step.apply(); err != nil {
			_ = undoAppliedActions(plan, index)
			return fail(err)
		}
		record.Details = copyDetails(plan.details)
		if err := journal.Write(journalDir, record); err != nil {
			_ = undoAppliedActions(plan, index+1)
			return fail(err)
		}
	}
	if runtime.GOOS == "windows" {
		if err := platform.NotifyEnvChange(); err != nil {
			_ = undoEnvPlan(plan)
			return fail(err)
		}
	}
	record.Status = journal.StatusCommitted
	record.Details = copyDetails(plan.details)
	if err := journal.Write(journalDir, record); err != nil {
		_ = undoEnvPlan(plan)
		return fail(err)
	}
	result.Applied = true
	result.JournalID = id
	result.Operations = plan.operations
	return result, nil
}

func buildEnvPlan(opts ApplyOptions, add bool) (envPlan, error) {
	plan := envPlan{details: map[string]string{}, operations: []string{}}
	if runtime.GOOS == "windows" {
		store := opts.Store
		if store == nil {
			store = platform.NewEnvStore()
		}
		plan.store = store
		if !opts.NoPath {
			previous, exists, err := store.Get("PATH")
			if err != nil {
				return plan, err
			}
			plan.details["path_previous"] = previous
			plan.details["path_existed"] = boolString(exists)
			entry := filepath.Join(opts.DataRoot, "current", "bin")
			updated := appendPath(previous, entry)
			changed := updated != previous
			if !add {
				updated, changed = removePath(previous, entry)
			}
			plan.operations = append(plan.operations, fmt.Sprintf("PATH %s %s", map[bool]string{true: "+=", false: "-="}[add], entry))
			if changed {
				plan.actions = append(plan.actions, envAction{
					description: "write PATH",
					apply:       func() error { return store.Set("PATH", updated) },
					undo: func() error {
						if !exists {
							return store.Remove("PATH")
						}
						return store.Set("PATH", previous)
					},
				})
			}
		}
		if !opts.NoVariables {
			for _, name := range managedVariables(opts.LAECompat) {
				previous, exists, err := store.Get(name)
				if err != nil {
					return plan, err
				}
				key := strings.ToLower(name)
				plan.details[key+"_previous"] = previous
				plan.details[key+"_existed"] = boolString(exists)
				if !add && !exists {
					continue
				}
				value := filepath.Join(opts.DataRoot, "current")
				plan.operations = append(plan.operations, fmt.Sprintf("%s %s", name, map[bool]string{true: "=", false: "unset"}[add]))
				plan.actions = append(plan.actions, envAction{
					description: "write " + name,
					apply: func() error {
						if add {
							return store.Set(name, value)
						}
						return store.Remove(name)
					},
					undo: func() error {
						if !exists {
							return store.Remove(name)
						}
						return store.Set(name, previous)
					},
				})
			}
		}
		return plan, nil
	}

	shimDir, err := shimDirectory()
	if err != nil {
		return plan, err
	}
	plan.details["shim_dir"] = shimDir
	if !opts.NoPath {
		for _, name := range []string{"ffmpeg", "ffprobe", "ffplay"} {
			shim := filepath.Join(shimDir, name)
			body, readErr := os.ReadFile(shim)
			exists := readErr == nil
			managed := exists && strings.Contains(string(body), shimMarker)
			plan.details["shim_"+name+"_present"] = boolString(exists)
			plan.details["shim_"+name+"_managed"] = boolString(managed)
			plan.details["shim_"+name+"_body"] = string(body)
			target := filepath.Join(opts.DataRoot, "current", "bin", name)
			if exists && !managed {
				plan.operations = append(plan.operations, "skip shim "+shim+" (non-managed)")
				continue
			}
			if add {
				plan.operations = append(plan.operations, "shim "+shim+" -> "+target)
				plan.actions = append(plan.actions, envAction{
					description: "write shim " + name,
					apply:       func() error { return writeShim(shim, target) },
					undo: func() error {
						if exists {
							return writeFileAtomic(shim, body, 0o755)
						}
						if err := os.Remove(shim); err != nil && !os.IsNotExist(err) {
							return err
						}
						return nil
					},
				})
			} else if exists {
				plan.operations = append(plan.operations, "remove shim "+shim)
				plan.actions = append(plan.actions, envAction{
					description: "remove shim " + name,
					apply:       func() error { return os.Remove(shim) },
					undo:        func() error { return writeFileAtomic(shim, body, 0o755) },
				})
			}
		}
	}
	if !opts.NoShellRC && (!opts.NoPath || !opts.NoVariables) {
		profile, err := shellProfile()
		if err != nil {
			return plan, err
		}
		desired := shellRCBlock(opts)
		hasBlock, body, err := readShellRC(profile)
		if err != nil {
			return plan, err
		}
		plan.details["shell_rc_block_present"] = boolString(hasBlock)
		plan.details["shell_rc_previous"] = body
		if add && desired != "" && currentShellRCBlock(body) != strings.Trim(desired, "\n") {
			plan.operations = append(plan.operations, "shell rc += OneMoreFFmpeg block ("+profile+")")
			plan.actions = append(plan.actions, envAction{
				description: "write shell rc",
				apply:       func() error { return updateShellRC(profile, body, desired, journalDirFor(opts), plan.details) },
				undo:        func() error { return restoreShellRCBackup(plan.details) },
			})
		} else if !add && hasBlock {
			plan.operations = append(plan.operations, "shell rc -= OneMoreFFmpeg block ("+profile+")")
			plan.actions = append(plan.actions, envAction{
				description: "remove shell rc block",
				apply:       func() error { return updateShellRC(profile, body, "", journalDirFor(opts), plan.details) },
				undo:        func() error { return restoreShellRCBackup(plan.details) },
			})
		} else if !opts.Apply {
			plan.operations = append(plan.operations, "shell rc (dry-run): "+profile)
		}
	}
	return plan, nil
}

func undoEnvPlan(plan envPlan) error {
	return undoAppliedActions(plan, len(plan.actions))
}

func undoAppliedActions(plan envPlan, count int) error {
	var first error
	if count > len(plan.actions) {
		count = len(plan.actions)
	}
	for index := count - 1; index >= 0; index-- {
		undo := plan.actions[index].undo
		if undo == nil {
			continue
		}
		if err := undo(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func restoreShellRCBackup(details map[string]string) error {
	backup := details["shell_rc_backup"]
	if backup == "" {
		return nil
	}
	return restoreShellRC(backup)
}

// Rollback restores the values captured before an env-set/env-unset record.
func Rollback(ctx context.Context, dataRoot, journalID string) (model.EnvSetData, error) {
	return RollbackWithStore(ctx, dataRoot, journalID, platform.NewEnvStore())
}

// RollbackWithStore restores an env transaction using an injectable backend.
func RollbackWithStore(ctx context.Context, dataRoot, journalID string, store platform.EnvStore) (model.EnvSetData, error) {
	_ = ctx
	journalDir := filepath.Join(dataRoot, "journal")
	record, err := journal.Read(journalDir, journalID)
	if err != nil {
		return model.EnvSetData{}, err
	}
	if record.Action != "env-set" && record.Action != "env-unset" {
		return model.EnvSetData{}, apperrors.New(apperrors.E7001, "journal is not an env transaction").WithDetails(map[string]any{"action": record.Action})
	}
	if record.Status == journal.StatusRolledBack {
		return model.EnvSetData{}, apperrors.New(apperrors.E7001, "journal was already rolled back")
	}
	operations := []string{}
	if runtime.GOOS == "windows" {
		for _, name := range []string{"PATH", "ONE_MORE_FFMPEG_DIR", "FFMPEG_DIR", "LAE_FFMPEG_DIR"} {
			key := strings.ToLower(name)
			previous, ok := record.Details[key+"_previous"]
			if !ok {
				continue
			}
			if record.Details[key+"_existed"] == "0" {
				if err := store.Remove(name); err != nil {
					return model.EnvSetData{}, err
				}
				operations = append(operations, "unset "+name)
			} else {
				if err := store.Set(name, previous); err != nil {
					return model.EnvSetData{}, err
				}
				operations = append(operations, "restore "+name)
			}
		}
		if err := platform.NotifyEnvChange(); err != nil {
			return model.EnvSetData{}, err
		}
	} else {
		for _, name := range []string{"ffmpeg", "ffprobe", "ffplay"} {
			present, tracked := record.Details["shim_"+name+"_present"]
			if !tracked || (present == "1" && record.Details["shim_"+name+"_managed"] == "0") {
				continue
			}
			shim := filepath.Join(record.Details["shim_dir"], name)
			if present == "1" {
				if err := writeFileAtomic(shim, []byte(record.Details["shim_"+name+"_body"]), 0o755); err != nil {
					return model.EnvSetData{}, err
				}
				operations = append(operations, "restore "+shim)
			} else {
				if err := os.Remove(shim); err != nil && !os.IsNotExist(err) {
					return model.EnvSetData{}, err
				}
				operations = append(operations, "remove "+shim)
			}
		}
	}
	if backup := record.Details["shell_rc_backup"]; backup != "" {
		if err := restoreShellRC(backup); err != nil {
			return model.EnvSetData{}, err
		}
		operations = append(operations, "restore shell rc")
	}
	record.Status = journal.StatusRolledBack
	if err := journal.Write(journalDir, record); err != nil {
		return model.EnvSetData{}, err
	}
	return model.EnvSetData{Schema: model.EnvSetSchema, Applied: true, Operations: operations, JournalID: journalID, RollbackSupported: false}, nil
}

func managedVariables(lae bool) []string {
	names := []string{"ONE_MORE_FFMPEG_DIR", "FFMPEG_DIR"}
	if lae {
		names = append(names, "LAE_FFMPEG_DIR")
	}
	return names
}

func appendPath(previous, entry string) string {
	for _, part := range filepath.SplitList(previous) {
		if strings.EqualFold(filepath.Clean(part), filepath.Clean(entry)) {
			return previous
		}
	}
	if previous == "" {
		return entry
	}
	return previous + string(os.PathListSeparator) + entry
}

func removePath(previous, entry string) (string, bool) {
	parts := filepath.SplitList(previous)
	kept := make([]string, 0, len(parts))
	removed := false
	for _, part := range parts {
		if strings.EqualFold(filepath.Clean(part), filepath.Clean(entry)) {
			removed = true
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, string(os.PathListSeparator)), removed
}

func shimDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "home directory cannot be determined").WithCause(err.Error())
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func shellProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", apperrors.New(apperrors.E9003, "home directory cannot be determined").WithCause(err.Error())
	}
	return filepath.Join(home, ".profile"), nil
}

func journalDirFor(opts ApplyOptions) string { return filepath.Join(opts.DataRoot, "journal") }

func shellRCBlock(opts ApplyOptions) string {
	lines := []string{shellRCMarkerStart}
	if !opts.NoPath {
		lines = append(lines, `export PATH="$HOME/.local/bin:$PATH"`)
	}
	if !opts.NoVariables {
		current := shellQuote(filepath.Join(opts.DataRoot, "current"))
		lines = append(lines, `export ONE_MORE_FFMPEG_DIR=`+current)
		lines = append(lines, `export FFMPEG_DIR=`+current)
		if opts.LAECompat {
			lines = append(lines, `export LAE_FFMPEG_DIR=`+current)
		}
	}
	if len(lines) == 1 {
		return ""
	}
	lines = append(lines, shellRCMarkerEnd)
	return "\n" + strings.Join(lines, "\n") + "\n"
}

func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'" }

func readShellRC(profile string) (bool, string, error) {
	body, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return false, "", apperrors.New(apperrors.E6004, "shell rc cannot be read").WithCause(err.Error())
	}
	return strings.Contains(string(body), shellRCMarkerStart), string(body), nil
}

func currentShellRCBlock(body string) string {
	start := strings.Index(body, shellRCMarkerStart)
	end := strings.Index(body, shellRCMarkerEnd)
	if start < 0 || end < start {
		return ""
	}
	return strings.Trim(body[start:end+len(shellRCMarkerEnd)], "\n")
}

func stripShellRCBlock(body string) string {
	start := strings.Index(body, shellRCMarkerStart)
	end := strings.Index(body, shellRCMarkerEnd)
	if start < 0 || end < start {
		return body
	}
	return body[:start] + body[end+len(shellRCMarkerEnd):]
}

func updateShellRC(profile, body, desired, journalDir string, details map[string]string) error {
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		return apperrors.New(apperrors.E6004, "shell rc backup directory cannot be created").WithCause(err.Error())
	}
	backup, err := os.CreateTemp(journalDir, "profile-*.bak")
	if err != nil {
		return apperrors.New(apperrors.E6004, "shell rc backup cannot be created").WithCause(err.Error())
	}
	backupPath := backup.Name()
	_, writeErr := backup.WriteString(body)
	closeErr := backup.Close()
	if writeErr != nil || closeErr != nil {
		return apperrors.New(apperrors.E6004, "shell rc backup cannot be written").WithCause(fmt.Sprint(writeErr, closeErr))
	}
	details["shell_rc_backup"] = backupPath
	updated := body
	if desired == "" {
		updated = stripShellRCBlock(body)
	} else {
		updated = strings.TrimRight(stripShellRCBlock(body), "\n") + desired
	}
	if err := writeFileAtomic(profile, []byte(updated), 0o644); err != nil {
		return apperrors.New(apperrors.E6004, "shell rc cannot be written").WithCause(err.Error())
	}
	return nil
}

func restoreShellRC(backup string) error {
	content, err := os.ReadFile(backup)
	if err != nil {
		return apperrors.New(apperrors.E6004, "shell rc backup cannot be read").WithCause(err.Error())
	}
	profile, err := shellProfile()
	if err != nil {
		return err
	}
	return writeFileAtomic(profile, content, 0o644)
}

func writeShim(path, target string) error {
	body := "#!/bin/sh\n" + shimMarker + "\nexec " + shellQuote(target) + " \"$@\"\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return apperrors.New(apperrors.E6002, "shim directory cannot be created").WithCause(err.Error())
	}
	return writeFileAtomic(path, []byte(body), 0o755)
}

func writeFileAtomic(path string, payload []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(payload)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %v %v", writeErr, closeErr)
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func copyDetails(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
