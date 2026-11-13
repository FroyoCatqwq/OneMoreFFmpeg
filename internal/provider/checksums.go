package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/checksum"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func (r *Resolver) resolveChecksums(ctx context.Context, release github.Release, asset github.Asset) (string, string, string, error) {
	digest := ""
	if value, ok := strings.CutPrefix(asset.Digest, "sha256:"); ok {
		digest = strings.ToLower(strings.TrimSpace(value))
	}
	var checksumsAsset *github.Asset
	for index := range release.Assets {
		if release.Assets[index].Name == checksumsAssetName {
			checksumsAsset = &release.Assets[index]
			break
		}
	}
	if checksumsAsset == nil {
		if digest == "" {
			return "", "", "", apperrors.New(apperrors.E3005, "no trusted checksum is available for the asset").WithDetails(map[string]any{
				"asset": asset.Name,
			})
		}
		return digest, "", "", nil
	}

	// The asset digest is authoritative. A transient checksums failure must not
	// block resolving when the digest is already present; it only skips the
	// independent second source.
	body, fetchErr := r.fetch(ctx, checksumsAsset.BrowserDownloadURL)
	if fetchErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", "", "", fetchErr
		}
		if digest == "" {
			return "", "", "", fetchErr
		}
		return digest, "", "", nil
	}
	sum, _, err := checksum.StreamSHA256(ctx, bytes.NewReader(body))
	if err != nil {
		if digest == "" {
			return "", "", "", err
		}
		return digest, "", "", nil
	}
	checksumsDigest, err := checksum.ParseChecksums(bytes.NewReader(body), asset.Name)
	if err != nil {
		if digest == "" {
			return "", "", "", err
		}
		return digest, "", "", nil
	}
	if digest != "" && digest != checksumsDigest {
		return "", "", "", apperrors.New(apperrors.E4002, "asset digest and checksums file disagree").WithDetails(map[string]any{
			"asset":     asset.Name,
			"digest":    digest,
			"checksums": checksumsDigest,
		})
	}
	if digest == "" {
		digest = checksumsDigest
	}
	return digest, checksumsAsset.BrowserDownloadURL, sum, nil
}

func (r *Resolver) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if r != nil && r.Fetch != nil {
		return r.Fetch(ctx, rawURL)
	}
	if !isStrictHTTPSURL(rawURL) {
		return nil, apperrors.New(apperrors.E4001, "checksums url must be https").WithDetails(map[string]any{"url": rawURL})
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, apperrors.New(apperrors.E4001, "checksums request could not be built").WithCause(err.Error())
	}
	timeout := 30 * time.Second
	maxRedirects := 5
	if r != nil {
		if r.Timeout > 0 {
			timeout = r.Timeout
		}
		if r.MaxRedirects >= 0 {
			maxRedirects = r.MaxRedirects
		}
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return apperrors.New(apperrors.E4003, "redirect limit exceeded")
			}
			if !isStrictHTTPSURL(req.URL.String()) {
				return apperrors.New(apperrors.E4001, "checksums redirect must be https").WithDetails(map[string]any{"url": req.URL.String()})
			}
			return nil
		},
	}
	response, err := client.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, checksumsContextError(ctxErr)
		}
		if typed, ok := apperrors.As(err); ok {
			return nil, typed
		}
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithCause(err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithDetails(map[string]any{"status": response.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, defaultChecksumsLimit+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, checksumsContextError(ctxErr)
		}
		return nil, apperrors.New(apperrors.E4001, "checksums download failed").WithCause(err.Error())
	}
	if int64(len(body)) > defaultChecksumsLimit {
		return nil, apperrors.New(apperrors.E4004, "checksums file exceeds size limit").WithDetails(map[string]any{"limit": defaultChecksumsLimit})
	}
	return body, nil
}

func checksumsContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return apperrors.New(apperrors.E3007, "checksums request timed out").WithCause(err.Error())
	}
	return apperrors.New(apperrors.E9004, "context canceled").WithCause(err.Error())
}
