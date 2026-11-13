package provider

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

const (
	OfficialReleasesURL = "https://ffmpeg.org/download.html#releases"
	DefaultSnapshotDays = 7
)

var (
	officialVersionPattern = regexp.MustCompile(`ffmpeg-(\d+\.\d+(?:\.\d+)?)\.tar\.xz`)
	assetLinePattern       = regexp.MustCompile(`(?:^|-)n(\d+\.\d+)(?:-latest)?-`)
	assetVersionPattern    = regexp.MustCompile(`(?i)^ffmpeg-(?:n(\d+\.\d+\.\d+(?:-\d+-g[0-9a-f]+)?)|n-(\d+)-g[0-9a-f]+)-`)
	autobuildTagPattern    = regexp.MustCompile(`^autobuild-(\d{4}-\d{2}-\d{2}-\d{2}-\d{2})$`)
)

// CatalogEntry is one installable/discoverable FFmpeg version.
type CatalogEntry struct {
	Channel     string `json:"channel"`
	Version     string `json:"version"`
	Line        string `json:"line,omitempty"`
	Snapshot    string `json:"snapshot,omitempty"`
	ReleaseTag  string `json:"release_tag"`
	AssetName   string `json:"asset_name"`
	AssetURL    string `json:"asset_url"`
	PublishedAt string `json:"published_at"`
	Provider    string `json:"provider"`
}

// CatalogOptions controls release discovery.
type CatalogOptions struct {
	Provider    string
	Channel     string
	Target      string
	Variant     string
	Linkage     string
	Days        int
	Count       int
	Client      *github.Client
	ReleasesURL string
	HTTPClient  *http.Client
	Now         func() time.Time
}

// ListCatalog builds the stable/daily version catalog dynamically.
func ListCatalog(ctx context.Context, opts CatalogOptions) ([]CatalogEntry, error) {
	if opts.Client == nil {
		return nil, apperrors.New(apperrors.E3007, "github client is required")
	}
	registry := NewRegistry()
	descriptor, err := registry.Get(normalizeProvider(opts.Provider))
	if err != nil {
		return nil, err
	}
	repository := descriptor.Summary().Repository
	owner, repo, err := splitRepository(repository)
	if err != nil {
		return nil, err
	}
	if opts.Days <= 0 {
		opts.Days = DefaultSnapshotDays
	}
	switch opts.Channel {
	case ChannelStable:
		return stableCatalog(ctx, opts, owner, repo)
	case ChannelDaily:
		return dailyCatalog(ctx, opts, owner, repo)
	default:
		return nil, apperrors.New(apperrors.E3004, "unsupported channel").WithDetails(map[string]any{"channel": opts.Channel})
	}
}

func normalizeProvider(value string) string {
	if strings.TrimSpace(value) == "" {
		return IDBtbn
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func splitRepository(repository string) (string, string, error) {
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", apperrors.New(apperrors.E3007, "provider repository is invalid").WithDetails(map[string]any{"repository": repository})
	}
	return parts[0], parts[1], nil
}

func stableCatalog(ctx context.Context, opts CatalogOptions, owner, repo string) ([]CatalogEntry, error) {
	if normalizeProvider(opts.Provider) == IDBnqdzjNonfree {
		return []CatalogEntry{}, nil
	}
	versions, err := officialVersions(ctx, opts)
	if err != nil {
		return nil, err
	}
	latest, err := opts.Client.LatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	lines := map[string]bool{}
	for _, asset := range latest.Assets {
		if line := assetLine(asset.Name); line != "" {
			lines[line] = true
		}
	}
	entries := make([]CatalogEntry, 0, len(lines))
	for rawLine, version := range versions {
		line := "n" + rawLine
		if !lines[line] {
			continue
		}
		asset, _, err := selectAsset(latest.Assets, assetQuery{
			Channel: ChannelStable, Target: opts.Target, Variant: opts.Variant,
			Linkage: opts.Linkage, ReleaseLine: line,
		})
		if err != nil {
			continue
		}
		entries = append(entries, CatalogEntry{
			Channel: ChannelStable, Version: version, Line: line, ReleaseTag: latest.TagName,
			AssetName: asset.Name, AssetURL: asset.BrowserDownloadURL,
			PublishedAt: latest.PublishedAt.UTC().Format(time.RFC3339),
			Provider:    normalizeProvider(opts.Provider),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return compareVersion(entries[i].Version, entries[j].Version) > 0 })
	if opts.Count > 0 && len(entries) > opts.Count {
		entries = entries[:opts.Count]
	}
	return entries, nil
}

func dailyCatalog(ctx context.Context, opts CatalogOptions, owner, repo string) ([]CatalogEntry, error) {
	releases, err := opts.Client.ListReleases(ctx, owner, repo, 100)
	if err != nil {
		return nil, err
	}
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	cutoff := now().UTC().Add(-time.Duration(opts.Days) * 24 * time.Hour)
	entries := make([]CatalogEntry, 0, len(releases))
	for _, release := range releases {
		if !autobuildTagPattern.MatchString(release.TagName) || release.PublishedAt.Before(cutoff) {
			continue
		}
		asset, _, err := selectAsset(release.Assets, assetQuery{
			Channel: ChannelDaily, Target: opts.Target, Variant: opts.Variant, Linkage: opts.Linkage,
		})
		if err != nil {
			continue
		}
		version := versionFromAsset(asset.Name)
		if version == "" {
			version = release.TagName
		}
		entries = append(entries, CatalogEntry{
			Channel: ChannelDaily, Version: version, Snapshot: release.TagName,
			ReleaseTag: release.TagName, AssetName: asset.Name, AssetURL: asset.BrowserDownloadURL,
			PublishedAt: release.PublishedAt.UTC().Format(time.RFC3339),
			Provider:    normalizeProvider(opts.Provider),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].PublishedAt > entries[j].PublishedAt })
	if opts.Count > 0 && len(entries) > opts.Count {
		entries = entries[:opts.Count]
	}
	return entries, nil
}

func officialVersions(ctx context.Context, opts CatalogOptions) (map[string]string, error) {
	url := opts.ReleasesURL
	if url == "" {
		url = OfficialReleasesURL
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "ffmpeg release page request is invalid").WithCause(err.Error())
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "ffmpeg release page cannot be fetched").WithCause(err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, apperrors.New(apperrors.E3007, "ffmpeg release page returned an error").WithDetails(map[string]any{"status": response.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return nil, apperrors.New(apperrors.E3007, "ffmpeg release page cannot be read").WithCause(err.Error())
	}
	result := map[string]string{}
	for _, match := range officialVersionPattern.FindAllStringSubmatch(string(body), -1) {
		version := match[1]
		parts := strings.Split(version, ".")
		line := parts[0] + "." + parts[1]
		if current := result[line]; current == "" || compareVersion(version, current) > 0 {
			result[line] = version
		}
	}
	if len(result) == 0 {
		return nil, apperrors.New(apperrors.E3007, "ffmpeg release page has no versions")
	}
	return result, nil
}

func assetLine(name string) string {
	match := assetLinePattern.FindStringSubmatch(strings.ToLower(name))
	if len(match) < 2 {
		return ""
	}
	return "n" + match[1]
}

func versionFromAsset(name string) string {
	match := assetVersionPattern.FindStringSubmatch(strings.ToLower(name))
	if len(match) >= 2 && match[1] != "" {
		return match[1]
	}
	if len(match) >= 3 && match[2] != "" {
		return "N-" + match[2]
	}
	return ""
}

func compareVersion(left, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for index := 0; index < 3; index++ {
		l, r := 0, 0
		if index < len(leftParts) {
			l, _ = strconv.Atoi(strings.TrimLeft(leftParts[index], "nN"))
		}
		if index < len(rightParts) {
			r, _ = strconv.Atoi(strings.TrimLeft(rightParts[index], "nN"))
		}
		if l != r {
			return l - r
		}
	}
	return 0
}
