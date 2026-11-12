package detect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	sourceExplicitDir = "explicit_dir"
	sourceEnv         = "env"
	sourcePATH        = "PATH"
	sourceCommonDir   = "common_dir"
	sourceScan        = "scan"
)

var sourcePriority = map[string]int{
	sourceExplicitDir: 0,
	sourceEnv:         1,
	sourcePATH:        2,
	sourceCommonDir:   3,
	sourceScan:        4,
}

var versionPattern = regexp.MustCompile(`(?im)^(?:ffmpeg|ffprobe|ffplay)\s+version\s+(\S+)`)

func canonicalRoot(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	return filepath.Clean(abs)
}

func rootKey(path string) string {
	key := canonicalRoot(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func resolveDirs(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return []string{filepath.Dir(abs)}, nil
	}
	dirs := []string{abs}
	binDir := filepath.Join(abs, "bin")
	if binInfo, err := os.Stat(binDir); err == nil && binInfo.IsDir() {
		dirs = append(dirs, binDir)
	}
	return dirs, nil
}

func binaryNames(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{base + ".exe", base}
	}
	return []string{base}
}

func findBinary(dir, base string) string {
	for _, name := range binaryNames(base) {
		path := filepath.Join(dir, name)
		fi, err := os.Stat(path)
		if err != nil || fi.IsDir() {
			continue
		}
		if isExecutable(path, fi) {
			return path
		}
	}
	return ""
}

func isExecutable(path string, fi os.FileInfo) bool {
	if fi.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		// Extensionless files on Windows are almost never PE binaries; a
		// stray file named "ffmpeg" must not shadow a real ffmpeg.exe.
		return ext == ".exe" || ext == ".com"
	}
	return fi.Mode().Perm()&0o111 != 0
}

// hashFile hashes a binary and stays responsive to cancellation so a stale
// context cannot leave detection blocked on a large or network-bound file.
func hashFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hasher := sha256.New()
	buffer := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := f.Read(buffer)
		if n > 0 {
			if _, err := hasher.Write(buffer[:n]); err != nil {
				return "", err
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// pathError maps filesystem failures to stable codes instead of reporting
// every unreadable path as "does not exist".
func pathError(err error) (apperrors.Code, string) {
	switch {
	case os.IsNotExist(err):
		return apperrors.E1001, "source path does not exist"
	case os.IsPermission(err):
		return apperrors.E1005, "source path is not readable"
	default:
		return apperrors.E1009, "source path is invalid or unreadable"
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func versionFromOutput(output string) string {
	match := versionPattern.FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func newWarning(code apperrors.Code, message string) model.Warning {
	return model.Warning{Code: string(code), Message: message, Details: map[string]any{}}
}

func newDetectError(code apperrors.Code, message string, details map[string]any) model.DetectError {
	return model.DetectError{Code: string(code), Message: message, Details: details}
}

func sourceList(sources map[string]bool) []string {
	ordered := []string{sourceExplicitDir, sourceEnv, sourcePATH, sourceCommonDir, sourceScan}
	out := make([]string, 0, len(sources))
	for _, source := range ordered {
		if sources[source] {
			out = append(out, source)
		}
	}
	return out
}

func commonDirs() []string {
	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		dirs := []string{
			`C:\ffmpeg\bin`,
			`C:\Program Files\ffmpeg\bin`,
		}
		if localAppData != "" {
			dirs = append(dirs, filepath.Join(localAppData, "Programs", "ffmpeg", "bin"))
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "bin"))
		}
		return dirs
	}
	dirs := []string{
		"/usr/bin",
		"/usr/local/bin",
		"/opt/ffmpeg/bin",
	}
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local", "bin"))
	}
	return dirs
}
