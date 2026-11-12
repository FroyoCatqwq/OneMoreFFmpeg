package detect

import (
	"context"
	stderrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

var errScanLimit = stderrors.New("scan limit exceeded")

type emitError func(model.DetectError)
type addCandidate func(dir, source string)

func scanRoots(ctx context.Context, opts Options, add addCandidate, errs *[]model.DetectError) {
	roots := opts.Roots
	if len(roots) == 0 {
		roots = defaultScanRoots(opts)
	}
	seenErrors := map[string]bool{}
	emit := func(err model.DetectError) {
		key := err.Code + "|" + err.Message + "|" + strings.TrimSpace(joinDetails(err.Details))
		if seenErrors[key] {
			return
		}
		seenErrors[key] = true
		*errs = append(*errs, err)
	}

	scannedRoots := make([]string, 0, len(roots))
	// MaxFiles is a global budget; previously it restarted for every root,
	// so a multi-drive scan visited roots × MaxFiles entries.
	files := 0
	for _, root := range roots {
		if ctx.Err() != nil {
			// Detect reports the overall timeout once; per-root duplicates
			// only made errors[] noisy.
			return
		}
		canon := canonicalRoot(root)
		if isUnderAny(scannedRoots, canon) {
			continue
		}
		scannedRoots = append(scannedRoots, canon)
		err := scanOne(ctx, root, opts, add, emit, &files)
		if err != nil {
			switch {
			case stderrors.Is(err, errScanLimit):
				emit(newDetectError(apperrors.E1008, "scan limit exceeded", map[string]any{"root": root, "max_files": opts.MaxFiles}))
			case ctx.Err() != nil:
				return
			case os.IsPermission(err):
				emit(newDetectError(apperrors.E1005, "scan permission denied", map[string]any{"root": root}))
			default:
				emit(newDetectError(apperrors.E1009, "scan root invalid or unreadable", map[string]any{"root": root, "cause": err.Error()}))
			}
		}
	}
}

func scanOne(ctx context.Context, root string, opts Options, add addCandidate, emit emitError, files *int) error {
	abs := canonicalRoot(root)
	fi, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return os.ErrInvalid
	}
	visitedDirs := map[string]bool{}
	return filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			if os.IsPermission(walkErr) {
				emit(newDetectError(apperrors.E1005, "scan permission denied", map[string]any{"path": path}))
				return nil
			}
			emit(newDetectError(apperrors.E1009, "scan path unreadable", map[string]any{"path": path, "cause": walkErr.Error()}))
			return nil
		}
		if entry.IsDir() {
			if isExcludedDir(abs, path) || entry.Type()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			if opts.MaxDepth >= 0 && relativeDepth(abs, path) > opts.MaxDepth {
				return filepath.SkipDir
			}
			// WalkDir never follows symlinks, so a cheap lexical key is enough;
			// EvalSymlinks per directory dominated full-disk scan time.
			key := cheapKey(path)
			if visitedDirs[key] {
				return filepath.SkipDir
			}
			visitedDirs[key] = true
			return nil
		}
		*files++
		if opts.MaxFiles > 0 && *files > opts.MaxFiles {
			return errScanLimit
		}
		if isFFmpegBinaryName(entry.Name()) {
			add(filepath.Dir(path), sourceScan)
		}
		return nil
	})
}

func cheapKey(path string) string {
	key := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(key)
	}
	return key
}

func isFFmpegBinaryName(name string) bool {
	lower := strings.ToLower(name)
	base := strings.TrimSuffix(lower, ".exe")
	switch base {
	case "ffmpeg", "ffprobe", "ffplay":
		return true
	default:
		return false
	}
}

func relativeDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return 0
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(os.PathSeparator)))
}

func isExcludedDir(root, path string) bool {
	if filepath.Clean(root) == filepath.Clean(path) {
		return false
	}
	base := strings.ToLower(filepath.Base(path))
	switch base {
	case ".git", "node_modules", "$recycle.bin", "system volume information", "winsxs":
		return true
	}
	normalized := strings.ReplaceAll(strings.ToLower(filepath.Clean(path)), `\`, "/")
	for _, prefix := range []string{"/proc", "/sys", "/dev", "/run", "/tmp", "/var/lib/docker"} {
		if normalized == prefix || strings.HasPrefix(normalized, prefix+"/") {
			return true
		}
	}
	return false
}

func isUnderAny(roots []string, candidate string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return true
		}
	}
	return false
}

func joinDetails(details map[string]any) string {
	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.TrimSpace(fmt.Sprint(details[key])))
	}
	return strings.Join(parts, ",")
}
