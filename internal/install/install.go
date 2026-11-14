// Package install extracts a downloaded FFmpeg archive into a versioned
// directory, records a manifest and switches the `current` pointer.
package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/archive"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

type Options struct {
	DataRoot    string
	Release     model.Release
	ArchivePath string
	Apply       bool
	Force       bool
	NoCurrent   bool
	EmitLock    bool
	Now         func() time.Time
	Runner      ffexec.Runner
}

// Run installs one archive. Without Apply it only validates and previews.
func Run(ctx context.Context, opts Options) (model.InstallData, error) {
	if opts.Release.Schema != model.ReleaseSchema {
		return model.InstallData{}, apperrors.New(apperrors.E7003, "release schema mismatch")
	}
	if !filepath.IsAbs(opts.DataRoot) {
		return model.InstallData{}, apperrors.New(apperrors.E7001, "data root must be absolute")
	}
	if opts.ArchivePath == "" || !filepath.IsAbs(opts.ArchivePath) {
		return model.InstallData{}, apperrors.New(apperrors.E7001, "archive path must be absolute")
	}
	if archive.FormatFor(opts.ArchivePath) == "" {
		return model.InstallData{}, apperrors.New(apperrors.E5001, "unsupported archive format").WithDetails(map[string]any{"archive": filepath.Base(opts.ArchivePath)})
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}

	targetDir := targetDir(opts.DataRoot, opts.Release)
	preview := model.InstallData{
		Schema:          model.InstallSchema,
		InstallationID:  "inst-" + shortSHA(opts.Release.AssetSHA256),
		RootDir:         targetDir,
		CurrentPointer:  currentType(opts.DataRoot, targetDir),
		Provider:        opts.Release.Provider,
		Repository:      opts.Release.Repository,
		ReleaseTag:      opts.Release.ReleaseTag,
		AssetSHA256:     opts.Release.AssetSHA256,
		LicenseHint:     opts.Release.LicenseHint,
		Redistributable: opts.Release.Redistributable,
		DryRun:          !opts.Apply,
	}

	if existing, ok := readRecord(targetDir); ok {
		if existing.Installation.AssetSHA256 != opts.Release.AssetSHA256 && !opts.Force {
			return model.InstallData{}, apperrors.New(apperrors.E5003, "install directory already exists with another asset").WithDetails(map[string]any{"dir": targetDir})
		}
		if existing.Installation.AssetSHA256 == opts.Release.AssetSHA256 {
			data := existing.Installation
			data.DryRun = !opts.Apply
			data.CurrentPointer = currentType(opts.DataRoot, targetDir)
			if opts.Apply {
				if _, err := os.Stat(opts.ArchivePath); err != nil {
					return model.InstallData{}, apperrors.New(apperrors.E7007, "archive file does not exist").WithDetails(map[string]any{"path": opts.ArchivePath})
				}
				archiveSum, err := checksumFile(ctx, opts.ArchivePath)
				if err != nil {
					return model.InstallData{}, err
				}
				if archiveSum != strings.ToLower(opts.Release.AssetSHA256) {
					return model.InstallData{}, apperrors.New(apperrors.E4002, "archive does not match the pinned sha256")
				}
				if err := validateBinaries(ctx, opts.Runner, targetDir); err != nil {
					return model.InstallData{}, err
				}
				if !opts.NoCurrent {
					pointer, switchErr := switchCurrent(opts.DataRoot, targetDir)
					if switchErr != nil {
						return model.InstallData{}, switchErr
					}
					data.CurrentPointer = pointer
				}
				if err := writeRecord(targetDir, record{Schema: "onemoreffmpeg.install-record/v1", InstalledAt: now().UTC().Format(time.RFC3339), AssetName: opts.Release.AssetName, Installation: data}); err != nil {
					return model.InstallData{}, err
				}
			}
			return data, nil
		}
	}
	if _, err := os.Stat(targetDir); err == nil && !opts.Force {
		return model.InstallData{}, apperrors.New(apperrors.E5003, "install directory already exists").WithDetails(map[string]any{"dir": targetDir})
	}

	if !opts.Apply {
		return preview, nil
	}

	stat, err := os.Stat(opts.ArchivePath)
	if err != nil || stat.IsDir() {
		return model.InstallData{}, apperrors.New(apperrors.E7007, "archive file does not exist").WithDetails(map[string]any{"path": opts.ArchivePath})
	}
	archiveSum, err := checksumFile(ctx, opts.ArchivePath)
	if err != nil {
		return model.InstallData{}, err
	}
	if archiveSum != strings.ToLower(opts.Release.AssetSHA256) {
		return model.InstallData{}, apperrors.New(apperrors.E4002, "archive does not match the pinned sha256").WithDetails(map[string]any{
			"expected": opts.Release.AssetSHA256,
			"actual":   archiveSum,
		})
	}
	journalID := journal.NewID()
	journalDir := filepath.Join(opts.DataRoot, "journal")
	recordEntry := journal.Record{ID: journalID, Action: "install", Status: journal.StatusPending, Target: targetDir,
		Details: map[string]string{"provider": opts.Release.Provider, "release_tag": opts.Release.ReleaseTag, "asset_sha256": opts.Release.AssetSHA256}}
	if err := journal.Write(journalDir, recordEntry); err != nil {
		return model.InstallData{}, err
	}
	committed := false
	stageCommitted := false
	restored := false
	var restoreOnFailure func()
	fail := func(cause error) (model.InstallData, error) {
		if restoreOnFailure != nil {
			restoreOnFailure()
		}
		recordEntry.Status = journal.StatusFailed
		recordEntry.Details["error"] = cause.Error()
		_ = journal.Write(journalDir, recordEntry)
		return model.InstallData{}, cause
	}

	previousKind, previousTarget := InspectCurrent(opts.DataRoot)
	recordEntry.Details["current_kind"] = previousKind
	recordEntry.Details["current_target"] = previousTarget
	if previousKind == model.CurrentMissing {
		recordEntry.Details["current_existed"] = "0"
	} else {
		recordEntry.Details["current_existed"] = "1"
	}
	quarantine, quarantineOriginal, quarantineErr := quarantineExistingInstalls(opts.DataRoot, targetDir, journalID, opts.Force)
	if quarantineErr != nil {
		return fail(quarantineErr)
	}
	restoreTarget := targetDir
	if quarantine != "" {
		recordEntry.Previous = quarantine
		recordEntry.Details["quarantine"] = quarantine
		recordEntry.Details["quarantine_original"] = quarantineOriginal
		if quarantineOriginal != "" {
			restoreTarget = quarantineOriginal
		}
	}
	if err := journal.Write(journalDir, recordEntry); err != nil {
		// v0.6.x: the quarantine has already moved the previous install away,
		// but the closure that puts it back is only assigned below. Without this
		// the old version sat in quarantine/ forever and `current` kept pointing
		// at a directory that no longer exists.
		if quarantine != "" && quarantineOriginal != "" {
			if _, statErr := os.Stat(quarantineOriginal); os.IsNotExist(statErr) {
				if _, qErr := os.Stat(quarantine); qErr == nil {
					_ = os.MkdirAll(filepath.Dir(quarantineOriginal), 0o755)
					_ = platform.RenamePath(quarantine, quarantineOriginal)
				}
			}
		}
		return fail(err)
	}
	restoreOnFailure = func() {
		// v0.6.x: fail() calls this, and the deferred call below runs it again.
		// The second pass used to RemoveAll(targetDir) *after* the first pass had
		// moved the quarantined previous install back there -- deleting the very
		// copy it had just restored, with a journal that still pointed at the
		// (now empty) quarantine. Both halves must run at most once.
		if committed || restored {
			return
		}
		restored = true
		if quarantine != "" || stageCommitted {
			_ = os.RemoveAll(targetDir)
		}
		if quarantine != "" {
			if _, err := os.Stat(restoreTarget); os.IsNotExist(err) {
				if _, qErr := os.Stat(quarantine); qErr == nil {
					_ = os.MkdirAll(filepath.Dir(restoreTarget), 0o755)
					_ = platform.RenamePath(quarantine, restoreTarget)
				}
			}
		}
		if recordEntry.Details["current_existed"] == "0" {
			_ = restoreCurrentState(opts.DataRoot, "", model.CurrentMissing)
		} else if previousTarget != "" {
			_ = restoreCurrent(opts.DataRoot, previousTarget, previousKind)
		}
	}
	defer func() { restoreOnFailure() }()

	tempDir := filepath.Join(opts.DataRoot, "tmp", journalID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fail(apperrors.New(apperrors.E5009, "temporary install directory cannot be created").WithCause(err.Error()))
	}
	recordEntry.TempDir = tempDir
	_ = journal.Write(journalDir, recordEntry)
	defer os.RemoveAll(tempDir)

	if err := archive.Extract(ctx, opts.ArchivePath, tempDir); err != nil {
		return fail(err)
	}
	binDir, err := findBinDir(tempDir)
	if err != nil {
		return fail(err)
	}

	stage := filepath.Join(tempDir, "stage")
	if err := os.MkdirAll(filepath.Join(stage, "bin"), 0o755); err != nil {
		return fail(apperrors.New(apperrors.E5009, "staging directory cannot be created").WithCause(err.Error()))
	}
	binaries := model.InstallBinaries{}
	for _, name := range []string{"ffmpeg", "ffprobe", "ffplay"} {
		source := findBinary(binDir, name)
		if source == "" {
			if name == "ffplay" {
				continue
			}
			return fail(apperrors.New(apperrors.E5008, "archive does not contain the required binaries").WithDetails(map[string]any{"missing": name}))
		}
		destination := filepath.Join(stage, "bin", filepath.Base(source))
		if err := copyFile(source, destination); err != nil {
			return fail(err)
		}
		sum, err := checksumFile(ctx, source)
		if err != nil {
			return fail(err)
		}
		switch name {
		case "ffmpeg":
			binaries.FFmpegSHA256 = sum
		case "ffprobe":
			binaries.FFprobeSHA256 = sum
		case "ffplay":
			binaries.FFplaySHA256 = sum
		}
	}

	copyDocs(tempDir, stage)
	installationID := installationIDFor(binaries)
	installRecord := model.InstallData{
		Schema:          model.InstallSchema,
		InstallationID:  installationID,
		RootDir:         targetDir,
		CurrentPointer:  model.CurrentMissing,
		Provider:        opts.Release.Provider,
		Repository:      opts.Release.Repository,
		ReleaseTag:      opts.Release.ReleaseTag,
		AssetSHA256:     opts.Release.AssetSHA256,
		Binaries:        binaries,
		LicenseHint:     opts.Release.LicenseHint,
		Redistributable: opts.Release.Redistributable,
		JournalID:       journalID,
	}
	if opts.EmitLock {
		installRecord.LockPath = filepath.Join(targetDir, "ffmpeg-trio.lock")
	}
	if err := validateBinaries(ctx, opts.Runner, stage); err != nil {
		return fail(err)
	}
	if err := writeRecord(stage, record{Schema: "onemoreffmpeg.install-record/v1", InstalledAt: now().UTC().Format(time.RFC3339), AssetName: opts.Release.AssetName, Installation: installRecord}); err != nil {
		return fail(err)
	}
	if opts.EmitLock {
		if err := writeLock(stage, installRecord); err != nil {
			return fail(err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return fail(apperrors.New(apperrors.E5009, "install parent directory cannot be created").WithCause(err.Error()))
	}
	if err := platform.RenamePath(stage, targetDir); err != nil {
		if quarantine != "" {
			_ = os.MkdirAll(filepath.Dir(restoreTarget), 0o755)
			_ = platform.RenamePath(quarantine, restoreTarget)
		}
		return fail(apperrors.New(apperrors.E5009, "install directory cannot be committed").WithCause(err.Error()))
	}
	stageCommitted = true

	installRecord.CurrentPointer = currentType(opts.DataRoot, targetDir)
	if !opts.NoCurrent {
		pointer, switchErr := switchCurrent(opts.DataRoot, targetDir)
		if switchErr != nil {
			_ = os.RemoveAll(targetDir)
			if quarantine != "" {
				_ = os.MkdirAll(filepath.Dir(restoreTarget), 0o755)
				_ = platform.RenamePath(quarantine, restoreTarget)
			}
			_ = restoreCurrentState(opts.DataRoot, previousTarget, previousKind)
			return fail(switchErr)
		}
		installRecord.CurrentPointer = pointer
	}
	// Refresh the record with the final pointer type.
	if err := writeRecord(targetDir, record{Schema: "onemoreffmpeg.install-record/v1", InstalledAt: now().UTC().Format(time.RFC3339), AssetName: opts.Release.AssetName, Installation: installRecord}); err != nil {
		return fail(err)
	}
	// Keep the quarantine so `journal rollback` can restore the previous install.
	recordEntry.Status = journal.StatusCommitted
	recordEntry.Details["installation_id"] = installRecord.InstallationID
	if err := journal.Write(journalDir, recordEntry); err != nil {
		return fail(err)
	}
	committed = true
	return installRecord, nil
}
