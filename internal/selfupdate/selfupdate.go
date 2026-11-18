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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/output"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/platform"
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
	client := o.Client
	if client == nil {
		var err error
		client, err = ggithub.NewClient(defaultGitHubAPI, "")
		if err != nil {
			return nil, err
		}
	}
	releases, err := client.ListReleases(ctx, o.Owner, o.Repo, 10)
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "releases cannot be fetched").WithCause(err.Error())
	}
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		version := strings.TrimPrefix(rel.TagName, "v")
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
			if len(sum) != 64 {
				return nil, apperrors.New(apperrors.E3007, "release asset has no sha256 digest").
					WithDetails(map[string]any{"asset": name})
			}
			return &Update{
				Version: version, AssetName: name, AssetURL: asset.BrowserDownloadURL,
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
	var progress *output.Progress
	if o.ProgressWriter != nil {
		progress = output.NewProgress(o.ProgressWriter, u.AssetName, resp.ContentLength)
	}
	buf := make([]byte, 128*1024)
	var total int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > maxUpdateBytes {
				if progress != nil {
					progress.Finish()
				}
				return apperrors.New(apperrors.E4004, "update exceeds size limit")
			}
			if _, err := tmp.Write(buf[:n]); err != nil {
				if progress != nil {
					progress.Finish()
				}
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
			if progress != nil {
				progress.Finish()
			}
			return readErr
		}
	}
	if progress != nil {
		progress.Finish()
	}
	if sum := hex.EncodeToString(hasher.Sum(nil)); sum != u.SHA256 {
		return apperrors.New(apperrors.E4002, "update sha256 mismatch").
			WithDetails(map[string]any{"expected": u.SHA256, "actual": sum})
	}
	if err := tmp.Chmod(0o755); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := platform.RenamePath(tmpPath, staging); err != nil {
		return err
	}
	ok = true

	backup := o.Executable + ".old"
	_ = os.Remove(backup)
	if err := platform.RenamePath(o.Executable, backup); err != nil {
		return apperrors.New(apperrors.E5009, "cannot move running executable aside").WithCause(err.Error())
	}
	if err := platform.RenamePath(staging, o.Executable); err != nil {
		_ = platform.RenamePath(backup, o.Executable)
		return apperrors.New(apperrors.E5009, "cannot install new executable").WithCause(err.Error())
	}
	_ = os.Remove(backup)
	return nil
}

func validVersion(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

func isNewer(candidate, current string) bool {
	c := parseVersion(candidate)
	cur := parseVersion(current)
	for i := 0; i < 3; i++ {
		if c[i] != cur[i] {
			return c[i] > cur[i]
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	var out [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

func isHTTPSURL(raw string) bool {
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "https://") {
		return true
	}
	// Loopback http is allowed for local tests only.
	return strings.HasPrefix(lower, "http://127.0.0.1:") ||
		strings.HasPrefix(lower, "http://localhost:") ||
		strings.HasPrefix(lower, "http://[::1]:")
}

func samePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	return err1 == nil && err2 == nil && aa == bb
}
