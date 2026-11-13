// Package install extracts a downloaded FFmpeg archive into a versioned
// directory, records a manifest and switches the `current` pointer.
package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/archive"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/journal"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const recordName = "install.json"

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

type record struct {
	Schema       string            `json:"schema"`
	InstalledAt  string            `json:"installed_at"`
	AssetName    string            `json:"asset_name"`
	Installation model.InstallData `json:"installation"`
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
			if opts.Apply && !opts.NoCurrent {
				pointer, switchErr := switchCurrent(opts.DataRoot, targetDir)
				if switchErr != nil {
					return model.InstallData{}, switchErr
				}
				data.CurrentPointer = pointer
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
	fail := func(cause error) (model.InstallData, error) {
		recordEntry.Status = journal.StatusFailed
		recordEntry.Details["error"] = cause.Error()
		_ = journal.Write(journalDir, recordEntry)
		return model.InstallData{}, cause
	}

	tempDir := filepath.Join(opts.DataRoot, "tmp", journalID)
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return fail(apperrors.New(apperrors.E5005, "temporary install directory cannot be created").WithCause(err.Error()))
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
		return fail(apperrors.New(apperrors.E5005, "staging directory cannot be created").WithCause(err.Error()))
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
		return fail(apperrors.New(apperrors.E5005, "install parent directory cannot be created").WithCause(err.Error()))
	}
	quarantine := ""
	if _, err := os.Stat(targetDir); err == nil {
		if !opts.Force {
			return fail(apperrors.New(apperrors.E5003, "install directory already exists").WithDetails(map[string]any{"dir": targetDir}))
		}
		quarantine = targetDir + ".old-" + now().UTC().Format("20060102T150405")
		if err := os.Rename(targetDir, quarantine); err != nil {
			return fail(apperrors.New(apperrors.E5007, "existing install cannot be quarantined").WithCause(err.Error()))
		}
		recordEntry.Previous = quarantine
		_ = journal.Write(journalDir, recordEntry)
	}
	if err := os.Rename(stage, targetDir); err != nil {
		if quarantine != "" {
			_ = os.Rename(quarantine, targetDir)
		}
		return fail(apperrors.New(apperrors.E5005, "install directory cannot be committed").WithCause(err.Error()))
	}

	installRecord.CurrentPointer = currentType(opts.DataRoot, targetDir)
	if !opts.NoCurrent {
		pointer, switchErr := switchCurrent(opts.DataRoot, targetDir)
		if switchErr != nil {
			_ = os.RemoveAll(targetDir)
			if quarantine != "" {
				_ = os.Rename(quarantine, targetDir)
			}
			return fail(switchErr)
		}
		installRecord.CurrentPointer = pointer
	}
	// Refresh the record with the final pointer type.
	_ = writeRecord(targetDir, record{Schema: "onemoreffmpeg.install-record/v1", InstalledAt: now().UTC().Format(time.RFC3339), AssetName: opts.Release.AssetName, Installation: installRecord})
	if quarantine != "" {
		_ = os.RemoveAll(quarantine)
	}
	recordEntry.Status = journal.StatusCommitted
	recordEntry.Details["installation_id"] = installRecord.InstallationID
	_ = journal.Write(journalDir, recordEntry)
	return installRecord, nil
}

func targetDir(dataRoot string, release model.Release) string {
	return filepath.Join(dataRoot, "apps", safeComponent(release.Provider), safeComponent(release.ReleaseTag), shortSHA(release.AssetSHA256))
}

func shortSHA(sum string) string {
	if len(sum) < 16 {
		return safeComponent(sum)
	}
	return strings.ToLower(sum[:16])
}

func safeComponent(value string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			return r
		default:
			return '-'
		}
	}, strings.TrimSpace(value))
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return "unknown"
	}
	return cleaned
}

func installationIDFor(binaries model.InstallBinaries) string {
	sum := checksumOfString(binaries.FFmpegSHA256 + "|" + binaries.FFprobeSHA256 + "|" + binaries.FFplaySHA256)
	return "inst-" + sum[:16]
}

func checksumOfString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func checksumFile(ctx context.Context, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", apperrors.New(apperrors.E5005, "installed binary cannot be read").WithCause(err.Error())
	}
	defer file.Close()
	sum, _, err := checksum.StreamSHA256(ctx, file)
	if err != nil {
		return "", err
	}
	return sum, nil
}

func findBinDir(root string) (string, error) {
	var fallback string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		if name == "ffmpeg" || name == "ffmpeg.exe" {
			dir := filepath.Dir(path)
			if findBinary(dir, "ffprobe") != "" {
				fallback = dir
				return filepath.SkipAll
			}
			if fallback == "" {
				fallback = dir
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", apperrors.New(apperrors.E5008, "archive cannot be inspected").WithCause(err.Error())
	}
	if fallback == "" {
		return "", apperrors.New(apperrors.E5008, "archive does not contain ffmpeg/ffprobe")
	}
	return fallback, nil
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

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return apperrors.New(apperrors.E5005, "archive binary cannot be read").WithCause(err.Error())
	}
	defer in.Close()
	info, _ := in.Stat()
	mode := os.FileMode(0o755)
	if info != nil {
		mode = info.Mode().Perm() | 0o111
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return apperrors.New(apperrors.E5005, "archive binary cannot be written").WithCause(err.Error())
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return apperrors.New(apperrors.E5005, "archive binary cannot be written").WithCause(copyErr.Error())
	}
	if closeErr != nil {
		return apperrors.New(apperrors.E5005, "archive binary cannot be closed").WithCause(closeErr.Error())
	}
	return nil
}

func validateBinaries(ctx context.Context, runner ffexec.Runner, stage string) error {
	if runner.Timeout <= 0 {
		runner.Timeout = 10 * time.Second
	}
	for _, name := range []string{"ffmpeg", "ffprobe"} {
		path := findBinary(filepath.Join(stage, "bin"), name)
		if path == "" {
			return apperrors.New(apperrors.E5008, "installed binary is missing").WithDetails(map[string]any{"binary": name})
		}
		result, err := runner.Run(ctx, path, "-version")
		if err != nil {
			return apperrors.New(apperrors.E5008, "installed binary cannot run").WithDetails(map[string]any{"binary": name}).WithCause(err.Error())
		}
		if result.ExitCode != 0 || strings.TrimSpace(result.Stdout) == "" {
			return apperrors.New(apperrors.E5008, "installed binary failed validation").WithDetails(map[string]any{"binary": name, "exit_code": result.ExitCode})
		}
	}
	return nil
}

func writeRecord(dir string, value record) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E5008, "install record cannot be encoded").WithCause(err.Error())
	}
	if err := os.WriteFile(filepath.Join(dir, recordName), payload, 0o644); err != nil {
		return apperrors.New(apperrors.E5005, "install record cannot be written").WithCause(err.Error())
	}
	return nil
}

func readRecord(dir string) (record, bool) {
	payload, err := os.ReadFile(filepath.Join(dir, recordName))
	if err != nil {
		return record{}, false
	}
	var value record
	if err := json.Unmarshal(payload, &value); err != nil {
		return record{}, false
	}
	return value, true
}

func writeLock(dir string, data model.InstallData) error {
	payload, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E5005, "trio lock cannot be encoded").WithCause(err.Error())
	}
	if err := os.WriteFile(filepath.Join(dir, "ffmpeg-trio.lock"), payload, 0o644); err != nil {
		return apperrors.New(apperrors.E5005, "trio lock cannot be written").WithCause(err.Error())
	}
	return nil
}

func currentType(dataRoot, target string) string {
	current := filepath.Join(dataRoot, "current")
	info, err := os.Lstat(current)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			if resolved, err := filepath.EvalSymlinks(current); err == nil {
				expected, _ := filepath.EvalSymlinks(target)
				if expected == resolved {
					return model.CurrentSymlink
				}
			}
			return model.CurrentSymlink
		}
		if info.IsDir() {
			return model.CurrentJunction
		}
	}
	if _, err := os.Stat(filepath.Join(dataRoot, "current.pointer")); err == nil {
		return model.CurrentPointerFile
	}
	return model.CurrentMissing
}

func switchCurrent(dataRoot, target string) (string, error) {
	current := filepath.Join(dataRoot, "current")
	if info, err := os.Lstat(current); err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return "", apperrors.New(apperrors.E5004, "current exists as a real directory").WithDetails(map[string]any{"path": current})
	}
	relative, err := filepath.Rel(dataRoot, target)
	if err != nil {
		return "", apperrors.New(apperrors.E5004, "current target cannot be computed").WithCause(err.Error())
	}
	_ = os.Remove(current)
	if err := os.Symlink(relative, current); err == nil {
		return model.CurrentSymlink, nil
	}
	// Windows without symlink privilege: keep a pointer file instead.
	pointer := filepath.Join(dataRoot, "current.pointer")
	if err := os.WriteFile(pointer, []byte(target+"\n"), 0o644); err != nil {
		return "", apperrors.New(apperrors.E5004, "current pointer cannot be written").WithCause(err.Error())
	}
	return model.CurrentPointerFile, nil
}
