package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/ffexec"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

// TargetDir returns the versioned install directory for a release.
func TargetDir(dataRoot string, release model.Release) string {
	return targetDir(dataRoot, release)
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
		return "", apperrors.New(apperrors.E5005, "binary cannot be read").WithCause(err.Error())
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
	if err != nil && err != filepath.SkipAll {
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

// copyDocs keeps license/readme files from the archive root (best effort).
func copyDocs(sourceRoot, stage string) {
	copied := map[string]bool{}
	depthRoot := strings.Count(filepath.Clean(sourceRoot), string(os.PathSeparator))
	_ = filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if path == stage || strings.HasPrefix(path, stage+string(os.PathSeparator)) {
			return nil
		}
		if strings.Count(filepath.Clean(path), string(os.PathSeparator))-depthRoot > 2 {
			return nil
		}
		name := strings.ToUpper(entry.Name())
		matched := false
		for _, prefix := range []string{"LICENSE", "COPYING", "README", "NOTICE"} {
			if strings.HasPrefix(name, prefix) {
				matched = true
				break
			}
		}
		if !matched || copied[entry.Name()] {
			return nil
		}
		if copyFile(path, filepath.Join(stage, entry.Name())) == nil {
			copied[entry.Name()] = true
		}
		return nil
	})
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
