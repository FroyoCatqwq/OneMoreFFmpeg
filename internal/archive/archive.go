// Package archive extracts FFmpeg release archives without ever trusting
// entry names or links.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
)

const (
	maxEntries  = 100000
	maxFileSize = 2 << 30
	maxTotal    = 8 << 30
	// maxRatio caps uncompressed:compressed expansion to blunt zip/tar bombs.
	// FFmpeg archives expand well under 100x; 1000x is a generous safety margin.
	maxRatio = 1000
)

// FormatFor returns the archive format implied by a file name.
func FormatFor(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return "zip"
	case strings.HasSuffix(lower, ".tar.xz"):
		return "tar.xz"
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return "tar.gz"
	case strings.HasSuffix(lower, ".tar.zst"):
		return "tar.zst"
	default:
		return ""
	}
}

// Extract unpacks archivePath into destDir. destDir must exist and be empty.
func Extract(ctx context.Context, archivePath, destDir string) error {
	format := FormatFor(archivePath)
	if format == "" {
		return apperrors.New(apperrors.E5001, "unsupported archive format").WithDetails(map[string]any{"archive": filepath.Base(archivePath)})
	}
	if err := ensureEmptyDir(destDir); err != nil {
		return err
	}
	compressed := int64(0)
	if info, err := os.Stat(archivePath); err == nil {
		compressed = info.Size()
	}
	switch format {
	case "zip":
		return extractZip(ctx, archivePath, destDir, compressed)
	default:
		return extractTar(ctx, archivePath, destDir, format, compressed)
	}
}

// checkDiskSpace fails early when the volume clearly cannot hold the expected
// extraction. It is best-effort: an unavailable estimate never blocks extraction.
func checkDiskSpace(destDir string, need int64) error {
	if need <= 0 {
		return nil
	}
	available, err := platform.AvailableBytes(destDir)
	if err != nil {
		return nil
	}
	if available < uint64(need) {
		return apperrors.New(apperrors.E5009, "insufficient disk space for extraction").WithDetails(map[string]any{
			"available": available, "needed": need,
		})
	}
	return nil
}

// ratioExceeded reports whether actual expansion blows past the bomb ceiling.
func ratioExceeded(total, compressed int64) bool {
	return compressed > 0 && total > compressed*maxRatio
}

func ensureEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return apperrors.New(apperrors.E5001, "archive destination is not readable").WithCause(err.Error())
	}
	if len(entries) > 0 {
		return apperrors.New(apperrors.E5001, "archive destination must be empty").WithDetails(map[string]any{"dir": dir})
	}
	return nil
}

func extractZip(ctx context.Context, archivePath, destDir string, compressed int64) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return apperrors.New(apperrors.E5001, "archive cannot be opened").WithCause(err.Error())
	}
	defer reader.Close()
	if len(reader.File) > maxEntries {
		return apperrors.New(apperrors.E5002, "archive has too many entries").WithDetails(map[string]any{"entries": len(reader.File)})
	}
	var declared int64
	for _, entry := range reader.File {
		declared += int64(entry.UncompressedSize64)
	}
	if err := checkDiskSpace(destDir, declared); err != nil {
		return err
	}
	var total int64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil {
			return apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
		}
		if entry.FileInfo().IsDir() {
			if _, err := safePath(destDir, entry.Name); err != nil {
				return err
			}
			continue
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return apperrors.New(apperrors.E5002, "archive symlink is not allowed").WithDetails(map[string]any{"entry": entry.Name})
		}
		target, err := safePath(destDir, entry.Name)
		if err != nil {
			return err
		}
		if entry.UncompressedSize64 > maxFileSize || total+int64(entry.UncompressedSize64) > maxTotal {
			return apperrors.New(apperrors.E5002, "archive exceeds extraction size limit").WithDetails(map[string]any{"entry": entry.Name})
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return apperrors.New(apperrors.E5009, "archive directory cannot be created").WithCause(err.Error())
		}
		source, err := entry.Open()
		if err != nil {
			return apperrors.New(apperrors.E5001, "archive entry cannot be read").WithCause(err.Error())
		}
		written, copyErr := writeFile(target, source, entry.Mode().Perm())
		_ = source.Close()
		total += written
		if copyErr != nil {
			return copyErr
		}
		if ratioExceeded(total, compressed) {
			return apperrors.New(apperrors.E5002, "archive expansion ratio exceeds the safety limit").WithDetails(map[string]any{"entry": entry.Name})
		}
	}
	return nil
}

func extractTar(ctx context.Context, archivePath, destDir, format string, compressed int64) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return apperrors.New(apperrors.E5001, "archive cannot be opened").WithCause(err.Error())
	}
	defer file.Close()

	var reader io.Reader = file
	var closeReader func()
	switch format {
	case "tar.xz":
		reader, err = xz.NewReader(file)
	case "tar.gz":
		var gzipReader *gzip.Reader
		gzipReader, err = gzip.NewReader(file)
		if err == nil {
			reader = gzipReader
			closeReader = func() { _ = gzipReader.Close() }
		}
	case "tar.zst":
		var zstdReader *zstd.Decoder
		zstdReader, err = zstd.NewReader(file)
		if err == nil {
			reader = zstdReader
			closeReader = zstdReader.Close
		}
	}
	if err != nil {
		return apperrors.New(apperrors.E5001, "archive cannot be decompressed").WithCause(err.Error())
	}
	if closeReader != nil {
		defer closeReader()
	}

	tarReader := tar.NewReader(reader)
	var total int64
	entries := 0
	for {
		if err := ctx.Err(); err != nil {
			return apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
		}
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return apperrors.New(apperrors.E5001, "archive is corrupt").WithCause(err.Error())
		}
		entries++
		if entries > maxEntries {
			return apperrors.New(apperrors.E5002, "archive has too many entries").WithDetails(map[string]any{"entries": entries})
		}
		target, pathErr := safePath(destDir, header.Name)
		if pathErr != nil {
			return pathErr
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return apperrors.New(apperrors.E5009, "archive directory cannot be created").WithCause(err.Error())
			}
		case tar.TypeReg:
			if header.Size > maxFileSize || total+header.Size > maxTotal {
				return apperrors.New(apperrors.E5002, "archive exceeds extraction size limit").WithDetails(map[string]any{"entry": header.Name})
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return apperrors.New(apperrors.E5009, "archive directory cannot be created").WithCause(err.Error())
			}
			written, writeErr := writeFile(target, tarReader, os.FileMode(header.Mode).Perm())
			total += written
			if writeErr != nil {
				return writeErr
			}
			if ratioExceeded(total, compressed) {
				return apperrors.New(apperrors.E5002, "archive expansion ratio exceeds the safety limit").WithDetails(map[string]any{"entry": header.Name})
			}
		default:
			return apperrors.New(apperrors.E5002, "archive contains a non-regular entry").WithDetails(map[string]any{
				"entry": header.Name,
				"type":  fmt.Sprintf("%c", header.Typeflag),
			})
		}
	}
}

func writeFile(target string, source io.Reader, mode os.FileMode) (int64, error) {
	if mode == 0 {
		mode = 0o644
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return 0, apperrors.New(apperrors.E5009, "archive file cannot be written").WithCause(err.Error())
	}
	written, copyErr := io.Copy(file, io.LimitReader(source, maxFileSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(target)
		return written, apperrors.New(apperrors.E5009, "archive file cannot be written").WithCause(copyErr.Error())
	}
	if closeErr != nil {
		return written, apperrors.New(apperrors.E5009, "archive file cannot be closed").WithCause(closeErr.Error())
	}
	if written > maxFileSize {
		_ = os.Remove(target)
		return written, apperrors.New(apperrors.E5002, "archive entry exceeds size limit").WithDetails(map[string]any{"file": filepath.Base(target)})
	}
	return written, nil
}

// safePath joins name under destDir and rejects escapes, absolute paths and
// Windows-specific tricks (drive letters, NTFS alternate data streams, reserved
// device names, trailing dot/space aliases).
func safePath(destDir, name string) (string, error) {
	if name == "" {
		return "", apperrors.New(apperrors.E5002, "archive entry has an empty name")
	}
	cleaned := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(cleaned, "/") || strings.Contains(cleaned, "\x00") ||
		strings.Contains(cleaned, ":") || strings.HasPrefix(cleaned, "//") {
		return "", apperrors.New(apperrors.E5002, "archive entry has an unsafe path").WithDetails(map[string]any{"entry": name})
	}
	for _, part := range strings.Split(strings.TrimSuffix(cleaned, "/"), "/") {
		if part == "" || part == "." || part == ".." {
			return "", apperrors.New(apperrors.E5002, "archive entry escapes the destination").WithDetails(map[string]any{"entry": name})
		}
		if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || isWindowsReservedName(part) {
			return "", apperrors.New(apperrors.E5002, "archive entry has an unsafe path").WithDetails(map[string]any{"entry": name})
		}
	}
	relative := filepath.FromSlash(cleaned)
	target := filepath.Join(destDir, relative)
	root, err := filepath.Abs(destDir)
	if err != nil {
		return "", apperrors.New(apperrors.E5009, "archive destination is invalid").WithCause(err.Error())
	}
	absolute, err := filepath.Abs(target)
	if err != nil {
		return "", apperrors.New(apperrors.E5002, "archive entry has an unsafe path").WithDetails(map[string]any{"entry": name})
	}
	rel, err := filepath.Rel(root, absolute)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", apperrors.New(apperrors.E5002, "archive entry escapes the destination").WithDetails(map[string]any{"entry": name})
	}
	return target, nil
}

// isWindowsReservedName reports legacy DOS device names that Windows resolves
// specially regardless of extension (e.g. NUL, NUL.txt, COM1.log).
func isWindowsReservedName(part string) bool {
	base := strings.ToUpper(part)
	if dot := strings.IndexByte(base, '.'); dot >= 0 {
		base = base[:dot]
	}
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) {
		return base[3] >= '1' && base[3] <= '9'
	}
	return false
}
