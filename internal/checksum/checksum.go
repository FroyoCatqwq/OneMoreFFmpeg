// Package checksum streams SHA-256 digests and parses `checksums.sha256`
// files without loading artifacts into memory.
package checksum

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"io"
	"path/filepath"
	"strings"
	"unicode"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

const (
	chunkSize    = 32 * 1024
	maxLineBytes = 1 << 20
)

// StreamSHA256 hashes r and returns the lowercase hex digest and byte count.
func StreamSHA256(ctx context.Context, r io.Reader) (string, int64, error) {
	return StreamSHA256Limit(ctx, r, 0)
}

// StreamSHA256Limit hashes r and fails with OMF-E4004 once more than limit
// bytes have been read. A limit <= 0 means unlimited.
func StreamSHA256Limit(ctx context.Context, r io.Reader, limit int64) (string, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	hasher := sha256.New()
	buffer := make([]byte, chunkSize)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				return "", total, apperrors.New(apperrors.E4001, "download timed out").WithCause(err.Error())
			}
			return "", total, apperrors.New(apperrors.E4006, "download canceled").WithCause(err.Error())
		}
		n, readErr := r.Read(buffer)
		if n > 0 {
			total += int64(n)
			if limit > 0 && total > limit {
				return "", total, apperrors.New(apperrors.E4004, "download exceeds size limit").WithDetails(map[string]any{"limit": limit})
			}
			if _, err := hasher.Write(buffer[:n]); err != nil {
				return "", total, apperrors.New(apperrors.E4001, "download stream failed").WithCause(err.Error())
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return "", total, apperrors.New(apperrors.E4001, "download stream failed").WithCause(readErr.Error())
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), total, nil
}

// ParseChecksums returns the SHA-256 for assetName from a GNU-style
// `checksums.sha256` file. Lines that do not contain a 64-bit hex digest and a
// bare file name are ignored; a missing entry is OMF-E4002.
func ParseChecksums(r io.Reader, assetName string) (string, error) {
	name := strings.TrimSpace(assetName)
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return "", apperrors.New(apperrors.E4002, "invalid asset name").WithDetails(map[string]any{"asset": name})
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLineBytes)
	found := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		separator := strings.IndexFunc(line, unicode.IsSpace)
		if separator <= 0 {
			continue
		}
		digest := strings.ToLower(line[:separator])
		if len(digest) != 64 || !isLowerHex(digest) {
			continue
		}
		file := strings.TrimLeft(line[separator:], " \t")
		file = strings.TrimPrefix(file, "*")
		if !isRelativeChecksumPath(file) || filepath.Base(file) != name {
			continue
		}
		if found != "" && found != digest {
			return "", apperrors.New(apperrors.E4002, "checksums file has conflicting entries for asset").WithDetails(map[string]any{"asset": name})
		}
		found = digest
	}
	if err := scanner.Err(); err != nil {
		return "", apperrors.New(apperrors.E4001, "checksums file could not be read").WithCause(err.Error())
	}
	if found == "" {
		return "", apperrors.New(apperrors.E4002, "checksums file has no entry for asset").WithDetails(map[string]any{"asset": name})
	}
	return found, nil
}

// isRelativeChecksumPath accepts "name", "./name" and "dist/name" but rejects
// absolute paths and parent traversal.
func isRelativeChecksumPath(file string) bool {
	if file == "" {
		return false
	}
	cleaned := filepath.Clean(file)
	if filepath.IsAbs(cleaned) {
		return false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
