package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveDirectoryLatestSnapshot(t *testing.T) {
	asset := "ffmpeg-20261120-gpl-snapshot-linux-amd64-static.tar.xz"
	payload := []byte("fake archive")
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ffmpeg/snapshots/latest.txt":
			_, _ = w.Write([]byte("20261120\n"))
		case "/ffmpeg/snapshots/20261120/checksum.sha256":
			_, _ = fmt.Fprintf(w, "%s  %s\n", digest, asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	entry := Custom{ID: "my-dir", URL: srv.URL + "/ffmpeg", Mode: "directory", Name: "ffmpeg", LicenseHint: "gpl"}
	req := Request{Provider: "my-dir", Channel: "daily", Target: "linux64", Variant: "gpl", Linkage: "static"}
	rel, err := ResolveDirectory(context.Background(), entry, req, entry.URL, DirectoryOptions{
		MaxRedirects: 5, Now: func() time.Time { return time.Unix(0, 0).UTC() },
	})
	require.NoError(t, err)
	require.Equal(t, asset, rel.AssetName)
	require.Equal(t, digest, rel.AssetSHA256)
	require.Equal(t, "20261120", rel.ReleaseTag)
	require.Equal(t, srv.URL+"/ffmpeg/snapshots/20261120/checksum.sha256", rel.ChecksumsURL)
}

func TestResolveDirectoryExplicitVersion(t *testing.T) {
	asset := "ffmpeg-8.1.0-gpl-stable-windows-amd64-static.zip"
	payload := []byte("fake")
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ffmpeg/stable/8.1.0/checksum.sha256" {
			_, _ = fmt.Fprintf(w, "%s  %s\n", digest, asset)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	entry := Custom{ID: "d", URL: srv.URL + "/ffmpeg", Mode: "directory", Name: "ffmpeg", LicenseHint: "gpl"}
	req := Request{Provider: "d", Channel: "daily", Target: "win64", Variant: "gpl", Linkage: "static", ReleaseLine: "8.1.0"}
	rel, err := ResolveDirectory(context.Background(), entry, req, entry.URL, DirectoryOptions{MaxRedirects: 5})
	require.NoError(t, err)
	require.Equal(t, asset, rel.AssetName)
	require.Equal(t, "8.1.0", rel.ReleaseTag)
}
