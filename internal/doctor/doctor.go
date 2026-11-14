// Package doctor checks the managed installation without changing anything.
package doctor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appenv "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/env"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const defaultProbeTimeout = 5 * time.Second

type Options struct {
	DataRoot string
	Env      []string
	Runner   ffexec.Runner
}

// Run returns a doctor report. A failed check never becomes a command error.
func Run(ctx context.Context, opts Options) (model.DoctorData, error) {
	status := appenv.Status(appenv.Options{DataRoot: opts.DataRoot, Env: opts.Env})
	checks := make([]model.Check, 0, 5)

	checks = append(checks, model.Check{
		Name:   "data_root",
		OK:     dirExists(opts.DataRoot),
		Detail: opts.DataRoot,
	})
	checks = append(checks, model.Check{
		Name:   "current_exists",
		OK:     status.CurrentType != model.CurrentMissing,
		Detail: status.CurrentType,
	})

	trioOK, trioDetail, trioWarnings := checkTrio(ctx, opts, status)
	checks = append(checks, model.Check{Name: "trio_valid", OK: trioOK, Detail: trioDetail})
	checks = append(checks, model.Check{
		Name:   "path_contains_current",
		OK:     status.PathContainsCurrent,
		Detail: pathDetail(status),
	})

	pending, err := journal.Pending(filepath.Join(opts.DataRoot, "journal"))
	if err != nil {
		return model.DoctorData{}, err
	}
	pendingIDs := make([]string, 0, len(pending))
	for _, record := range pending {
		pendingIDs = append(pendingIDs, record.ID)
	}
	checks = append(checks, model.Check{
		Name:   "no_pending_journal",
		OK:     len(pending) == 0,
		Detail: fmt.Sprintf("%d pending journals", len(pending)),
	})

	healthy := true
	for _, check := range checks {
		if !check.OK {
			healthy = false
		}
	}
	warnings := append([]model.Warning{}, status.Warnings...)
	warnings = append(warnings, trioWarnings...)
	return model.DoctorData{
		Schema:         model.DoctorSchema,
		Healthy:        healthy,
		Checks:         checks,
		Warnings:       warnings,
		JournalPending: pendingIDs,
	}, nil
}

func checkTrio(ctx context.Context, opts Options, status model.EnvStatusData) (bool, string, []model.Warning) {
	if status.CurrentType == model.CurrentMissing {
		return false, "current is missing", nil
	}
	if status.CurrentType == model.CurrentPointerFile {
		return false, "current is a pointer file; run env set or use current_target/bin", nil
	}
	binDir := filepath.Join(opts.DataRoot, "current", "bin")
	runner := opts.Runner
	if runner.Timeout <= 0 {
		runner.Timeout = defaultProbeTimeout
	}
	missing := make([]string, 0)
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		path := findBinary(binDir, name)
		if path == "" {
			missing = append(missing, name)
			continue
		}
		result, err := runner.Run(ctx, path, "-version")
		if err != nil || result.ExitCode != 0 {
			missing = append(missing, name+"(unrunnable)")
		}
	}
	if len(missing) > 0 {
		return false, "unhealthy: " + strings.Join(missing, ", "), nil
	}
	healthy := []string{"ffmpeg", "ffprobe"}
	warnings := []model.Warning{}
	ffplayPath := findBinary(binDir, "ffplay")
	if ffplayPath == "" {
		warnings = append(warnings, model.Warning{Code: string(apperrors.W1001), Message: "ffplay is missing"})
	} else if result, err := runner.Run(ctx, ffplayPath, "-version"); err != nil || result.ExitCode != 0 {
		warnings = append(warnings, model.Warning{Code: string(apperrors.W1001), Message: "ffplay is present but not runnable"})
	} else {
		healthy = append(healthy, "ffplay")
	}
	return true, strings.Join(healthy, ", ") + " healthy", warnings
}

func pathDetail(status model.EnvStatusData) string {
	if status.PathContainsCurrent {
		return "current/bin found in PATH"
	}
	return "current/bin is not in PATH"
}

func findBinary(dir, name string) string {
	for _, candidate := range []string{name, name + ".exe"} {
		path := filepath.Join(dir, candidate)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
