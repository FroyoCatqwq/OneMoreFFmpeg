package selfupdate

import (
	"context"
	"net/http"
	"time"

	gh "github.com/google/go-github/v93/github"

	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

// releaseInfo is the provider-agnostic release view used by Check. Both the
// in-house github client (test injection) and go-github (production) map into
// it so the selection logic has a single implementation.
type releaseInfo struct {
	Tag        string
	Draft      bool
	Prerelease bool
	Assets     []assetInfo
}

type assetInfo struct {
	Name     string
	Digest   string
	Download string
}

// listReleasesViaGoGitHub queries GitHub Releases using the official
// github.com/google/go-github client. This is the production path and the
// foundation for future "fetch FFmpeg release assets from GitHub" features.
func (o Options) listReleasesViaGoGitHub(ctx context.Context) ([]releaseInfo, error) {
	base := o.GitHubAPI
	if base == "" {
		base = defaultGitHubAPI
	}
	httpClient := o.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Minute}
	} else {
		clone := *httpClient
		httpClient = &clone
	}
	// network.max_attempts must govern the GitHub API too.
	httpClient = withRetry(httpClient, o.MaxAttempts)
	opts := []gh.ClientOptionsFunc{gh.WithHTTPClient(httpClient)}
	// The public API is the go-github default. Only override for a custom base
	// (e.g. GitHub Enterprise), where WithEnterpriseURLs is the supported way.
	if base != defaultGitHubAPI {
		opts = append(opts, gh.WithEnterpriseURLs(base, base))
	}
	ghc, err := gh.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	listOpts := &gh.ListOptions{PerPage: 10}
	var out []releaseInfo
	for page := 0; page < 5; page++ {
		releases, resp, err := ghc.Repositories.ListReleases(ctx, o.Owner, o.Repo, listOpts)
		if err != nil {
			return nil, apperrors.New(apperrors.E3007, "releases cannot be fetched").WithCause(err.Error())
		}
		for _, r := range releases {
			if r == nil {
				continue
			}
			info := releaseInfo{
				Tag: r.GetTagName(), Draft: r.GetDraft(), Prerelease: r.GetPrerelease(),
			}
			for _, a := range r.GetAssets() {
				if a == nil {
					continue
				}
				info.Assets = append(info.Assets, assetInfo{
					Name: a.GetName(), Digest: a.GetDigest(), Download: a.GetBrowserDownloadURL(),
				})
			}
			out = append(out, info)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}
	return out, nil
}
