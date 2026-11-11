package detect

import (
	"context"
	"fmt"
	"sort"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	statusHealthy  = "healthy"
	statusDegraded = "degraded"
	statusUnusable = "unusable"
	statusUnknown  = "unknown"
)

func (d *Detector) buildInstallation(ctx context.Context, candidate *Candidate, opts Options, runner ffexec.Runner) (model.Installation, model.ValidateData) {
	root := candidate.Root
	ffmpegPath := findBinary(root, "ffmpeg")
	ffprobePath := findBinary(root, "ffprobe")
	ffplayPath := findBinary(root, "ffplay")

	hashes := map[string]string{}
	hashFail := false
	for name, path := range map[string]string{"ffmpeg": ffmpegPath, "ffprobe": ffprobePath, "ffplay": ffplayPath} {
		if path == "" {
			continue
		}
		sum, err := hashFile(path)
		if err != nil {
			hashFail = true
			continue
		}
		hashes[name] = sum
	}

	warnings := []model.Warning{}
	if ffplayPath == "" {
		warnings = append(warnings, newWarning(apperrors.W1001, "ffplay is missing; ffmpeg/ffprobe still usable"))
	}
	checks := []model.Check{}
	status := statusUnknown
	version := ""

	if opts.NoValidate {
		checks = append(checks, model.Check{Name: "validation_skipped", OK: true, Detail: "validation disabled by --no-validate"})
	} else {
		_, ffmpegVersion, ffmpegOK := runVersionCheck(ctx, runner, ffmpegPath)
		_, ffprobeVersion, ffprobeOK := runVersionCheck(ctx, runner, ffprobePath)
		_, ffplayVersion, ffplayOK := runVersionCheck(ctx, runner, ffplayPath)

		checks = append(checks,
			model.Check{Name: "ffmpeg_exists", OK: ffmpegPath != "", Detail: detailPath(ffmpegPath)},
			model.Check{Name: "ffprobe_exists", OK: ffprobePath != "", Detail: detailPath(ffprobePath)},
			model.Check{Name: "ffplay_exists", OK: ffplayPath != "", Detail: detailPath(ffplayPath)},
		)
		checks = append(checks, checkFromRun("ffmpeg_version", ffmpegOK, ffmpegVersion))
		checks = append(checks, checkFromRun("ffprobe_version", ffprobeOK, ffprobeVersion))
		if ffplayPath == "" {
			checks = append(checks, model.Check{Name: "ffplay_version", OK: true, Detail: "skipped: ffplay missing (warning)"})
		} else {
			checks = append(checks, checkFromRun("ffplay_version", ffplayOK, ffplayVersion))
		}

		licenseOK, licenseDetail := runLicenseCheck(ctx, runner, ffmpegPath)
		checks = append(checks, model.Check{Name: "ffmpeg_license", OK: licenseOK, Detail: licenseDetail})

		consistencyOK, consistencyDetail := versionsConsistent(ffmpegVersion, ffprobeVersion, ffplayVersion, ffplayPath != "")
		checks = append(checks, model.Check{Name: "version_consistency", OK: consistencyOK, Detail: consistencyDetail})
		checks = append(checks, model.Check{Name: "binary_sha256", OK: !hashFail && len(hashes) >= 2, Detail: fmt.Sprintf("%d binary hashes computed", len(hashes))})

		version = ffmpegVersion
		hardFail := ffmpegPath == "" || ffprobePath == "" || !ffmpegOK || !ffprobeOK || !licenseOK || !consistencyOK || hashFail
		if ffplayPath != "" && !ffplayOK {
			hardFail = true
		}
		switch {
		case hardFail:
			status = statusUnusable
		case version == "":
			status = statusUnknown
		case len(warnings) > 0:
			status = statusDegraded
		default:
			status = statusHealthy
		}
	}

	fingerprint := buildFingerprint(root, hashes, version)
	id := "inst-" + fingerprint[:16]
	installation := model.Installation{
		ID:          id,
		RootDir:     root,
		FFmpeg:      ffmpegPath,
		FFprobe:     ffprobePath,
		FFplay:      ffplayPath,
		Sources:     sourceList(candidate.Sources),
		Status:      status,
		Version:     version,
		Fingerprint: fingerprint,
		Warnings:    warnings,
		BinarySHA:   hashes,
	}
	report := model.ValidateData{
		Schema:         model.ValidateSchema,
		InstallationID: id,
		Status:         status,
		Checks:         checks,
		Warnings:       warnings,
	}
	return installation, report
}

func runVersionCheck(ctx context.Context, runner ffexec.Runner, path string) (ffexec.Result, string, bool) {
	if path == "" {
		return ffexec.Result{}, "", false
	}
	result, err := runner.Run(ctx, path, "-version")
	if err != nil {
		return result, "", false
	}
	return result, versionFromOutput(result.Stdout), result.ExitCode == 0
}

func runLicenseCheck(ctx context.Context, runner ffexec.Runner, path string) (bool, string) {
	if path == "" {
		return false, "ffmpeg missing"
	}
	result, err := runner.Run(ctx, path, "-L")
	if err != nil {
		return false, err.Error()
	}
	if result.ExitCode != 0 {
		return false, fmt.Sprintf("exit=%d", result.ExitCode)
	}
	return true, fmt.Sprintf("stdout=%d bytes", len(result.Stdout))
}

func checkFromRun(name string, ok bool, version string) model.Check {
	detail := "version=" + version
	if !ok {
		detail = "command failed or path missing"
	}
	return model.Check{Name: name, OK: ok, Detail: detail}
}

func versionsConsistent(ffmpegVersion, ffprobeVersion, ffplayVersion string, ffplayRequired bool) (bool, string) {
	if ffmpegVersion == "" || ffprobeVersion == "" {
		return false, "ffmpeg or ffprobe version is missing"
	}
	if ffmpegVersion != ffprobeVersion {
		return false, fmt.Sprintf("ffmpeg=%s ffprobe=%s", ffmpegVersion, ffprobeVersion)
	}
	if ffplayRequired {
		if ffplayVersion == "" {
			return false, "ffplay version is missing"
		}
		if ffmpegVersion != ffplayVersion {
			return false, fmt.Sprintf("ffmpeg=%s ffplay=%s", ffmpegVersion, ffplayVersion)
		}
	}
	return true, "version=" + ffmpegVersion
}

func detailPath(path string) string {
	if path == "" {
		return "missing"
	}
	return path
}

func buildFingerprint(root string, hashes map[string]string, version string) string {
	keys := make([]string, 0, len(hashes))
	for key := range hashes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := []string{rootKey(root)}
	for _, key := range keys {
		parts = append(parts, key+"="+hashes[key])
	}
	parts = append(parts, "version="+version)
	return sha256Hex([]byte(strings.Join(parts, "\n")))
}
