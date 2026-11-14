// Package download fetches one pinned release asset, verifies it against the
// SHA-256 pin and records an artifact.json next to the file.
package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/model"
)

const (
	defaultMaxBytes      = int64(512 << 20)
	defaultChecksumsSize = int64(4 << 20)
	defaultMaxRedirects  = 5
	defaultTimeout       = 30 * time.Second
	artifactFileName     = "artifact.json"
)

var (
	sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	chunkSize     = 64 * 1024
)

// Options controls one download run.
type Options struct {
	Dir          string
	MaxBytes     int64
	MaxRedirects int
	Timeout      time.Duration
	HTTPClient   *http.Client
	Now          func() time.Time

	// Warn receives non-fatal degradations. A release that ships no checksums
	// file still verifies against the pinned sha256, but the second, independent
	// source is missing -- the run must say so instead of silently downgrading.
	Warn func(string)
}

// Downloader is safe for reuse; each Fetch is independent.
type Downloader struct {
	options Options
}

// New validates options and applies safe defaults.
func New(options Options) (*Downloader, error) {
	if strings.TrimSpace(options.Dir) == "" {
		return nil, apperrors.New(apperrors.E4005, "download directory is required")
	}
	if !filepath.IsAbs(options.Dir) {
		return nil, apperrors.New(apperrors.E4005, "download directory must be absolute").WithDetails(map[string]any{"dir": options.Dir})
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = defaultMaxBytes
	}
	if options.MaxRedirects < 0 {
		options.MaxRedirects = defaultMaxRedirects
	}
	if options.Timeout <= 0 {
		options.Timeout = defaultTimeout
	}
	return &Downloader{options: options}, nil
}

// Fetch downloads, verifies and records one artifact.
func (d *Downloader) Fetch(ctx context.Context, release model.Release) (model.Artifact, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	sha := strings.ToLower(strings.TrimSpace(release.AssetSHA256))
	if !sha256Pattern.MatchString(sha) {
		return model.Artifact{}, apperrors.New(apperrors.E4002, "asset sha256 is invalid").WithDetails(map[string]any{"asset_sha256": release.AssetSHA256})
	}
	name := strings.TrimSpace(release.AssetName)
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return model.Artifact{}, apperrors.New(apperrors.E4005, "asset name is unsafe").WithDetails(map[string]any{"asset": release.AssetName})
	}
	if !isAllowedURL(release.AssetURL) {
		return model.Artifact{}, apperrors.New(apperrors.E4001, "asset url must be https").WithDetails(map[string]any{"url": release.AssetURL})
	}

	targetDir := filepath.Join(d.options.Dir, sha[:16])
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return model.Artifact{}, apperrors.New(apperrors.E4005, "cannot create download directory").WithCause(err.Error())
	}
	finalPath := filepath.Join(targetDir, name)

	// Verify the independent checksums source before touching the asset. This
	// makes checksums failures cheap, keeps reuse honest, and avoids deleting a
	// fully verified 500 MiB asset after the fact.
	if err := d.verifyChecksumsFile(ctx, release); err != nil {
		// Only the file this call wrote may be removed: targetDir is keyed by
		// sha[:16], so a concurrent run downloading the same asset shares it and
		// deleting the directory would pull the file out from under it.
		_ = os.Remove(finalPath)
		// Removing a non-empty directory fails harmlessly, so this only tidies
		// up the case where this call created the directory and nothing else is
		// in it -- a concurrent run keeps its files.
		_ = os.Remove(targetDir)
		return model.Artifact{}, err
	}

	downloaded := false
	if matches, err := fileMatches(ctx, finalPath, sha); err != nil {
		return model.Artifact{}, err
	} else if !matches {
		if err := d.download(ctx, release.AssetURL, finalPath, sha); err != nil {
			// Only the file this call wrote may be removed: targetDir is keyed by
		// sha[:16], so a concurrent run downloading the same asset shares it and
		// deleting the directory would pull the file out from under it.
		_ = os.Remove(finalPath)
		// Removing a non-empty directory fails harmlessly, so this only tidies
		// up the case where this call created the directory and nothing else is
		// in it -- a concurrent run keeps its files.
		_ = os.Remove(targetDir)
			return model.Artifact{}, err
		}
		downloaded = true
	}

	artifact, err := d.record(targetDir, release, finalPath, sha)
	if err != nil {
		if downloaded {
			_ = os.Remove(finalPath)
		// Removing a non-empty directory fails harmlessly, so this only tidies
		// up the case where this call created the directory and nothing else is
		// in it -- a concurrent run keeps its files.
		_ = os.Remove(targetDir)
			// Only the file this call wrote may be removed: targetDir is keyed by
		// sha[:16], so a concurrent run downloading the same asset shares it and
		// deleting the directory would pull the file out from under it.
		_ = os.Remove(finalPath)
		// Removing a non-empty directory fails harmlessly, so this only tidies
		// up the case where this call created the directory and nothing else is
		// in it -- a concurrent run keeps its files.
		_ = os.Remove(targetDir)
		}
		return model.Artifact{}, err
	}
	return artifact, nil
}

func (d *Downloader) record(targetDir string, release model.Release, finalPath, sha string) (model.Artifact, error) {
	now := time.Now
	if d.options.Now != nil {
		now = d.options.Now
	}
	artifact := model.Artifact{
		Schema:       model.ArtifactSchema,
		ArtifactID:   "art-" + sha[:16],
		Provider:     release.Provider,
		ReleaseTag:   release.ReleaseTag,
		AssetName:    filepath.Base(finalPath),
		AssetSHA256:  sha,
		LocalPath:    finalPath,
		Verified:     true,
		DownloadedAt: now().UTC().Format(time.RFC3339),
		Warnings:     release.Warnings,
	}
	if err := writeArtifact(filepath.Join(targetDir, artifactFileName), artifact); err != nil {
		return model.Artifact{}, err
	}
	return artifact, nil
}

func (d *Downloader) download(ctx context.Context, rawURL, finalPath, expected string) error {
	response, err := d.get(ctx, rawURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.ContentLength > d.options.MaxBytes {
		return apperrors.New(apperrors.E4004, "download exceeds size limit").WithDetails(map[string]any{
			"content_length": response.ContentLength,
			"limit":          d.options.MaxBytes,
		})
	}

	tmp, err := os.CreateTemp(filepath.Dir(finalPath), filepath.Base(finalPath)+".*.part")
	if err != nil {
		return apperrors.New(apperrors.E4005, "cannot create temporary file").WithCause(err.Error())
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	hasher := sha256.New()
	buffer := make([]byte, chunkSize)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			cleanup()
			return contextError(err)
		}
		read, readErr := response.Body.Read(buffer)
		if read > 0 {
			total += int64(read)
			if total > d.options.MaxBytes {
				cleanup()
				return apperrors.New(apperrors.E4004, "download exceeds size limit").WithDetails(map[string]any{"limit": d.options.MaxBytes})
			}
			if _, writeErr := tmp.Write(buffer[:read]); writeErr != nil {
				cleanup()
				return apperrors.New(apperrors.E4005, "cannot write temporary file").WithCause(writeErr.Error())
			}
			_, _ = hasher.Write(buffer[:read])
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			cleanup()
			if ctxErr := ctx.Err(); ctxErr != nil {
				return contextError(ctxErr)
			}
			return apperrors.New(apperrors.E4001, "download stream failed").WithCause(readErr.Error())
		}
	}

	actual := hex.EncodeToString(hasher.Sum(nil))
	if actual != expected {
		cleanup()
		return apperrors.New(apperrors.E4002, "asset sha256 mismatch").WithDetails(map[string]any{
			"expected": expected,
			"actual":   actual,
		})
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E4005, "cannot close temporary file").WithCause(err.Error())
	}
	_ = os.Chmod(tmpPath, 0o644)
	if err := commitFile(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E4005, "cannot commit downloaded file").WithCause(err.Error())
	}
	return nil
}

// verifyChecksumsFile performs the second, independent check required by
// PROVIDER_RESOLVER §6: the released checksums file must still name the same
// digest.
func (d *Downloader) verifyChecksumsFile(ctx context.Context, release model.Release) error {
	if strings.TrimSpace(release.ChecksumsURL) == "" {
		if d.options.Warn != nil {
			d.options.Warn("no independent checksums file for this release; " +
				"only the pinned sha256 was verified")
		}
		return nil
	}
	if !isAllowedURL(release.ChecksumsURL) {
		return apperrors.New(apperrors.E4001, "checksums url must be https").WithDetails(map[string]any{"url": release.ChecksumsURL})
	}
	response, err := d.get(ctx, release.ChecksumsURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, defaultChecksumsSize+1))
	if err != nil {
		return apperrors.New(apperrors.E4001, "checksums download failed").WithCause(err.Error())
	}
	if int64(len(body)) > defaultChecksumsSize {
		return apperrors.New(apperrors.E4004, "checksums file exceeds size limit").WithDetails(map[string]any{"limit": defaultChecksumsSize})
	}
	if release.ChecksumsSHA256 != "" {
		sum, _, err := checksum.StreamSHA256(ctx, bytes.NewReader(body))
		if err != nil {
			return err
		}
		if sum != strings.ToLower(release.ChecksumsSHA256) {
			return apperrors.New(apperrors.E4002, "checksums file sha256 mismatch").WithDetails(map[string]any{
				"expected": release.ChecksumsSHA256,
				"actual":   sum,
			})
		}
	}
	digest, err := checksum.ParseChecksums(bytes.NewReader(body), release.AssetName)
	if err != nil {
		return err
	}
	if digest != strings.ToLower(release.AssetSHA256) {
		return apperrors.New(apperrors.E4002, "checksums file and pinned sha256 disagree").WithDetails(map[string]any{
			"pinned":    release.AssetSHA256,
			"checksums": digest,
		})
	}
	return nil
}

func (d *Downloader) get(ctx context.Context, rawURL string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E4001, "download request could not be built").WithCause(err.Error())
	}
	client := d.options.HTTPClient
	if client == nil {
		maxRedirects := d.options.MaxRedirects
		timeout := d.options.Timeout
		client = &http.Client{
			// No overall Timeout: it would also cap the body read and abort a
			// large asset on a slow link. Bound the connection phases instead;
			// the request context still governs cancellation.
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext,
				TLSHandshakeTimeout:   timeout,
				ResponseHeaderTimeout: timeout,
				ExpectContinueTimeout: 1 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Never downgrade to plain http, even if the original URL was
				// https; loopback is the only exception (tests/mirrors).
				if !isAllowedURL(req.URL.String()) {
					return apperrors.New(apperrors.E4001, "redirect target must be https").WithDetails(map[string]any{"url": req.URL.String()})
				}
				if len(via) > maxRedirects {
					return apperrors.New(apperrors.E4003, "redirect limit exceeded")
				}
				return nil
			},
		}
	}
	response, err := client.Do(request)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, contextError(ctxErr)
		}
		if typed, ok := apperrors.As(err); ok {
			return nil, typed
		}
		return nil, apperrors.New(apperrors.E4001, "download request failed").WithCause(err.Error())
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, apperrors.New(apperrors.E4001, "download request failed").WithDetails(map[string]any{"status": response.StatusCode})
	}
	return response, nil
}

func writeArtifact(path string, artifact model.Artifact) error {
	payload, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return apperrors.New(apperrors.E4005, "cannot encode artifact").WithCause(err.Error())
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return apperrors.New(apperrors.E4005, "cannot create artifact temp file").WithCause(err.Error())
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E4005, "cannot write artifact").WithCause(err.Error())
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E4005, "cannot close artifact temp file").WithCause(err.Error())
	}
	_ = os.Chmod(tmpPath, 0o644)
	if err := commitFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return apperrors.New(apperrors.E4005, "cannot commit artifact").WithCause(err.Error())
	}
	return nil
}

// commitFile prefers an atomic rename. Windows cannot replace an existing
// file, so it renames the old target aside first and restores it on failure.
func commitFile(tmpPath, target string) error {
	if err := os.Rename(tmpPath, target); err == nil {
		return nil
	}
	backup := target + ".bak"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		if os.IsNotExist(err) {
			return os.Rename(tmpPath, target)
		}
		return err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(target)
		_ = os.Rename(backup, target)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func fileMatches(ctx context.Context, path, expected string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, apperrors.New(apperrors.E4005, "cannot read existing download").WithCause(err.Error())
	}
	defer file.Close()
	digest, _, err := checksum.StreamSHA256(ctx, file)
	if err != nil {
		return false, err
	}
	return digest == expected, nil
}

func contextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(apperrors.E4001, "download timed out").WithCause(err.Error())
	}
	return apperrors.New(apperrors.E4006, "download canceled").WithCause(err.Error())
}

func isAllowedURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	return parsed.Scheme == "http" && isLoopback(parsed.Hostname())
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
