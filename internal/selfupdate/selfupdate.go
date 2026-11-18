// Package selfupdate checks GitHub Releases for a newer OneMoreFFmpeg build
// and replaces the running executable atomically. It is deliberately split
// into Check (queries GitHub via internal/provider/github) and Apply
// (downloads + verifies + replaces) so CLI code never talks to GitHub or a
// download library directly.
package selfupdate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/logging"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/output"
	ggithub "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

const (
	defaultOwner     = "FroyoCatqwq"
	defaultRepo      = "OneMoreFFmpeg"
	defaultGitHubAPI = "https://api.github.com"
	maxUpdateBytes   = int64(256 << 20)
)

// Options controls a self-update run.
type Options struct {
	CurrentVersion string
	Owner          string
	Repo           string
	GOOS           string
	GOARCH         string
	Executable     string
	HTTPClient     *http.Client
	Client         *ggithub.Client
	Dir            string // staging directory for the new binary
	ProgressWriter io.Writer
	Now            func() time.Time
	FileOps        FileOps // injectable file operations; nil uses realFileOps
	GitHubAPI      string  // override API base (empty uses api.github.com)
	MaxAttempts    int     // GitHub API retry budget (0 uses default 3)
}

// Update describes an available update.
type Update struct {
	Version    string `json:"version"`
	AssetName  string `json:"asset_name"`
	AssetURL   string `json:"asset_url"`
	SHA256     string `json:"sha256"`
	ReleaseURL string `json:"release_url"`
}

func (o Options) normalized() Options {
	if o.Owner == "" {
		o.Owner = defaultOwner
	}
	if o.Repo == "" {
		o.Repo = defaultRepo
	}
	if o.GOOS == "" {
		o.GOOS = runtime.GOOS
	}
	if o.GOARCH == "" {
		o.GOARCH = runtime.GOARCH
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return o
}

func assetName(goos, goarch, version string) string {
	if goarch != "amd64" {
		return ""
	}
	switch goos {
	case "windows":
		return fmt.Sprintf("one-more-ffmpeg-v%s-windows-amd64.exe", version)
	case "linux":
		return fmt.Sprintf("one-more-ffmpeg-v%s-linux-amd64", version)
	default:
		return ""
	}
}

// Check returns an Update when a newer stable release exists, without
// downloading or writing files.
func (o Options) Check(ctx context.Context) (*Update, error) {
	o = o.normalized()
	var releases []releaseInfo
	var err error
	if o.Client != nil {
		// Test injection path: use the in-house client.
		raw, e := o.Client.ListReleases(ctx, o.Owner, o.Repo, 10)
		if e != nil {
			return nil, apperrors.New(apperrors.E3007, "releases cannot be fetched").WithCause(e.Error())
		}
		for _, r := range raw {
			info := releaseInfo{Tag: r.TagName, Draft: r.Draft, Prerelease: r.Prerelease}
			for _, a := range r.Assets {
				info.Assets = append(info.Assets, assetInfo{Name: a.Name, Digest: a.Digest, Download: a.BrowserDownloadURL})
			}
			releases = append(releases, info)
		}
	} else {
		// Production path: official go-github client.
		releases, err = o.listReleasesViaGoGitHub(ctx)
		if err != nil {
			return nil, err
		}
	}
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		version := strings.TrimPrefix(rel.Tag, "v")
		if !validVersion(version) || !isNewer(version, o.CurrentVersion) {
			continue
		}
		name := assetName(o.GOOS, o.GOARCH, version)
		if name == "" {
			return nil, apperrors.New(apperrors.E3007, "unsupported platform for self-update").
				WithDetails(map[string]any{"os": o.GOOS, "arch": o.GOARCH})
		}
		for _, asset := range rel.Assets {
			if asset.Name != name {
				continue
			}
			sum := strings.TrimPrefix(asset.Digest, "sha256:")
			if len(sum) != 64 || !isHex64(sum) {
				return nil, apperrors.New(apperrors.E3007, "release asset has no valid sha256 digest").
					WithDetails(map[string]any{"asset": name})
			}
			return &Update{
				Version: version, AssetName: name, AssetURL: asset.Download,
				SHA256:     strings.ToLower(sum),
				ReleaseURL: fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", o.Owner, o.Repo, version),
			}, nil
		}
		return nil, apperrors.New(apperrors.E3007, "release has no asset for this platform").
			WithDetails(map[string]any{"asset": name})
	}
	return nil, nil
}

// Apply downloads the update, verifies SHA-256, and replaces the executable.
// On Windows the running exe can be renamed but not overwritten, so the old
// binary is moved aside before the new one is put in place; on failure the old
// binary is restored.
func (o Options) Apply(ctx context.Context, u *Update) error {
	o = o.normalized()
	if u == nil {
		return apperrors.New(apperrors.E7001, "no update to apply")
	}
	if o.Executable == "" {
		return apperrors.New(apperrors.E7001, "executable path is required")
	}
	if !isHTTPSURL(u.AssetURL) {
		return apperrors.New(apperrors.E4001, "update asset url must be https")
	}
	dir := o.Dir
	if dir == "" {
		dir = filepath.Dir(o.Executable)
	}
	staging := filepath.Join(dir, u.AssetName)
	if samePath(staging, o.Executable) {
		return apperrors.New(apperrors.E7001, "staging path equals executable")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	httpClient := o.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	} else {
		clone := *httpClient
		httpClient = &clone
	}
	// Enforce the same URL policy as the rest of the tool even for injected
	// clients: no https->http downgrade, bounded redirects.
	originScheme, _ := urlScheme(u.AssetURL)
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > 10 {
			return fmt.Errorf("too many redirects")
		}
		scheme, host := urlScheme(req.URL.String())
		if scheme != "https" && !isLoopbackHost(host) {
			return fmt.Errorf("redirect target must be https")
		}
		if originScheme == "https" && scheme != "https" {
			return fmt.Errorf("redirect must not downgrade https")
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.AssetURL, nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return apperrors.New(apperrors.E4001, "update download failed").WithCause(err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperrors.New(apperrors.E4001, "update download returned an error").
			WithDetails(map[string]any{"status": resp.StatusCode})
	}
	if resp.ContentLength > maxUpdateBytes {
		return apperrors.New(apperrors.E4004, "update exceeds size limit")
	}

	tmp, err := os.CreateTemp(dir, u.AssetName+".*.part")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	var progress output.Progress
	if o.ProgressWriter != nil {
		progress = output.NewProgress(o.ProgressWriter, u.AssetName, resp.ContentLength)
		defer progress.Close()
	}
	failProgress := func(err error) {
		if progress == nil {
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			progress.Abort()
		} else {
			progress.Fail(err)
		}
	}
	buf := make([]byte, 128*1024)
	var total int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > maxUpdateBytes {
				err := apperrors.New(apperrors.E4004, "update exceeds size limit")
				failProgress(err)
				return err
			}
			if _, err := tmp.Write(buf[:n]); err != nil {
				failProgress(err)
				return err
			}
			_, _ = hasher.Write(buf[:n])
			if progress != nil {
				progress.Update(total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			failProgress(readErr)
			return readErr
		}
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); sum != u.SHA256 {
		err := apperrors.New(apperrors.E4002, "update sha256 mismatch").
			WithDetails(map[string]any{"expected": u.SHA256, "actual": sum})
		failProgress(err)
		return err
	}
	// Preserve the running executable's permission bits instead of assuming
	// 0755 (releases may be installed with stricter modes).
	targetMode := os.FileMode(0o755)
	if info, err := os.Stat(o.Executable); err == nil {
		targetMode = info.Mode().Perm()
	}
	if err := tmp.Chmod(targetMode); err != nil {
		failProgress(err)
		return err
	}
	if err := tmp.Close(); err != nil {
		failProgress(err)
		return err
	}
	ops := o.FileOps
	if ops == nil {
		ops = realFileOps{}
	}
	if err := ops.Rename(tmpPath, staging); err != nil {
		failProgress(err)
		return apperrors.New(apperrors.E5009, "cannot stage update").WithCause(err.Error())
	}
	ok = true

	// Stage journal: every transition is persisted atomically so a crash can
	// be recovered on next start (Options.Recover).
	state := updateState{Version: u.Version, Stage: "staged", Executable: o.Executable, Staging: staging}
	if err := writeState(ops, o.Executable, state); err != nil {
		failProgress(err)
		return err
	}
	if info, err := ops.Stat(staging); err != nil || info.Size() < 1 {
		failProgress(fmt.Errorf("staged update is empty"))
		return apperrors.New(apperrors.E4002, "staged update is empty")
	}
	state.Stage = "verified"
	if err := writeState(ops, o.Executable, state); err != nil {
		failProgress(err)
		return err
	}

	backup := o.Executable + ".old"
	// A leftover backup from a previous crashed run must be preserved as the
	// only recovery copy, not silently overwritten.
	if _, err := ops.Stat(backup); err == nil {
		failProgress(fmt.Errorf("backup already exists: %s", backup))
		return apperrors.New(apperrors.E5009, "previous update backup still exists; recover it first").
			WithDetails(map[string]any{"backup": backup})
	}
	if err := ops.Rename(o.Executable, backup); err != nil {
		failProgress(err)
		return apperrors.New(apperrors.E5009, "cannot move running executable aside").WithCause(err.Error())
	}
	state.Stage = "backup_created"
	state.Backup = backup
	if err := writeState(ops, o.Executable, state); err != nil {
		// Journal write failed after backup: roll back and surface both errors.
		rollbackErr := ops.Rename(backup, o.Executable)
		if rollbackErr != nil {
			return errors.Join(
				fmt.Errorf("write update state: %w", err),
				fmt.Errorf("restore previous executable: %w; backup retained at %s", rollbackErr, backup),
			)
		}
		failProgress(err)
		return fmt.Errorf("write update state: %w (previous executable restored)", err)
	}
	if err := ops.Rename(staging, o.Executable); err != nil {
		// Rollback must not be ignored; keep the backup if restoration also fails.
		rollbackErr := ops.Rename(backup, o.Executable)
		if rollbackErr != nil {
			failProgress(err)
			return errors.Join(
				fmt.Errorf("install updated executable: %w", err),
				fmt.Errorf("restore previous executable: %w; backup retained at %s", rollbackErr, backup),
			)
		}
		failProgress(err)
		return apperrors.New(apperrors.E5009, "update failed; previous executable restored").WithCause(err.Error())
	}
	state.Stage = "replaced"
	if err := writeState(ops, o.Executable, state); err != nil {
		// The new binary is in place; record failure but do not pretend success.
		logging.FromContext(ctx).Warn("update state write failed after replacement", "err", err)
	}
	state.Stage = "committed"
	if err := writeState(ops, o.Executable, state); err != nil {
		logging.FromContext(ctx).Warn("update commit state write failed", "err", err)
	}
	// Best-effort backup cleanup with diagnostics (never silent).
	if err := ops.Remove(backup); err != nil && !os.IsNotExist(err) {
		logging.FromContext(ctx).Warn("update backup cleanup failed", "backup", backup, "err", err)
	}
	if progress != nil {
		progress.Complete()
	}
	return nil
}

// parseStrictVersion accepts ONLY the project's stable X.Y.Z release format.
// It rejects missing/extra fields, empty fields, non-digits, leading zeros,
// negative values and integer overflow. Release tags with a leading "v" are
// normalized by Check before this call. Prerelease builds are never accepted.
func parseStrictVersion(v string) ([3]int, error) {
	var out [3]int
	if v == "" {
		return out, fmt.Errorf("empty version")
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version %q must be X.Y.Z", v)
	}
	for i, part := range parts {
		if part == "" {
			return out, fmt.Errorf("version %q has empty field", v)
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return out, fmt.Errorf("version %q has non-digit field", v)
			}
		}
		if len(part) > 1 && part[0] == '0' {
			return out, fmt.Errorf("version %q has leading zero", v)
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, fmt.Errorf("version %q overflow: %w", v, err)
		}
		out[i] = n
	}
	return out, nil
}

func validVersion(v string) bool {
	_, err := parseStrictVersion(v)
	return err == nil
}

func isNewer(candidate, current string) bool {
	c, err := parseStrictVersion(candidate)
	if err != nil {
		return false
	}
	cur, err := parseStrictVersion(current)
	if err != nil {
		// dev/unknown local versions are older than any valid release.
		return true
	}
	for i := 0; i < 3; i++ {
		if c[i] != cur[i] {
			return c[i] > cur[i]
		}
	}
	return false
}

func isHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, r := range s {
		digit := r >= '0' && r <= '9'
		lower := r >= 'a' && r <= 'f'
		upper := r >= 'A' && r <= 'F'
		if !digit && !lower && !upper {
			return false
		}
	}
	return true
}

func urlScheme(raw string) (string, string) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", ""
	}
	return strings.ToLower(parsed.Scheme), parsed.Hostname()
}

func isLoopbackHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// isHTTPSURL accepts https, plus loopback http for local tests. Credentials in
// the URL are always rejected (matching config/github-client/download policy).
func isHTTPSURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme == "https" || (scheme == "http" && isLoopbackHost(parsed.Hostname()))
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && aa == bb
}
