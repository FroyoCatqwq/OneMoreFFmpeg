package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider"
	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/provider/github"
)

func catalogServer(t *testing.T, now time.Time) *httptest.Server {
	t.Helper()
	html := `<a href="releases/ffmpeg-9.0.2.tar.xz">9.0.2</a>
<a href="releases/ffmpeg-8.1.3.tar.xz">8.1.3</a>
<a href="releases/ffmpeg-7.1.5.tar.xz">7.1.5</a>`
	stable := map[string]any{
		"id": 1, "tag_name": "latest", "published_at": now.Add(-time.Hour).Format(time.RFC3339),
		"assets": []map[string]any{
			{"name": "ffmpeg-n9.0-latest-win64-gpl-9.0.zip", "browser_download_url": "https://example.invalid/9.0.zip", "digest": "sha256:" + strings.Repeat("a", 64)},
			{"name": "ffmpeg-n8.1-latest-win64-gpl-8.1.zip", "browser_download_url": "https://example.invalid/8.1.zip", "digest": "sha256:" + strings.Repeat("b", 64)},
		},
	}
	daily := []map[string]any{
		{"id": 2, "tag_name": "autobuild-2026-11-12-12-00", "published_at": now.Add(-24 * time.Hour).Format(time.RFC3339),
			"assets": []map[string]any{{"name": "ffmpeg-N-128790-g9bee1e603c-win64-gpl.zip", "browser_download_url": "https://example.invalid/new.zip", "digest": "sha256:" + strings.Repeat("c", 64)}}},
		{"id": 3, "tag_name": "autobuild-2026-10-01-12-00", "published_at": now.Add(-45 * 24 * time.Hour).Format(time.RFC3339),
			"assets": []map[string]any{{"name": "ffmpeg-N-120000-gabcdef-win64-gpl.zip", "browser_download_url": "https://example.invalid/old.zip", "digest": "sha256:" + strings.Repeat("d", 64)}}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.URL.Path, "/releases/latest"):
			_ = json.NewEncoder(writer).Encode(stable)
		case strings.Contains(request.URL.Path, "/releases"):
			_ = json.NewEncoder(writer).Encode(daily)
		default:
			writer.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(writer, html)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func catalogClient(t *testing.T, server *httptest.Server) *github.Client {
	t.Helper()
	client, err := github.NewClient(server.URL, "")
	require.NoError(t, err)
	client.MaxAttempts = 1
	return client
}

func TestStableCatalogUsesOfficialLatestPatch(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	server := catalogServer(t, now)
	entries, err := provider.ListCatalog(context.Background(), provider.CatalogOptions{
		Provider: provider.IDBtbn, Channel: provider.ChannelStable, Target: "win64",
		Variant: "gpl", Linkage: "static", Client: catalogClient(t, server),
		ReleasesURL: server.URL, Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "9.0.2", entries[0].Version)
	require.Equal(t, "n9.0", entries[0].Line)
	require.Equal(t, "8.1.3", entries[1].Version)
	require.Equal(t, "n8.1", entries[1].Line)
}

func TestStableCatalogHonorsCount(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	server := catalogServer(t, now)
	entries, err := provider.ListCatalog(context.Background(), provider.CatalogOptions{
		Provider: provider.IDBtbn, Channel: provider.ChannelStable, Target: "win64",
		Variant: "gpl", Linkage: "static", Count: 1, Client: catalogClient(t, server),
		ReleasesURL: server.URL, Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "9.0.2", entries[0].Version)
}

func TestDailyCatalogVersionFallsBackToSnapshotTag(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	release := map[string]any{
		"id": 9, "tag_name": "autobuild-2026-11-12-12-00",
		"published_at": now.Add(-time.Hour).Format(time.RFC3339),
		"assets": []map[string]any{{
			"name":                 "ffmpeg-master-latest-win64-gpl.zip",
			"browser_download_url": "https://example.invalid/master.zip",
			"digest":               "sha256:" + strings.Repeat("e", 64),
		}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode([]map[string]any{release})
	}))
	t.Cleanup(server.Close)

	entries, err := provider.ListCatalog(context.Background(), provider.CatalogOptions{
		Provider: provider.IDBtbn, Channel: provider.ChannelDaily, Target: "win64",
		Variant: "gpl", Linkage: "static", Client: catalogClient(t, server),
		Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "autobuild-2026-11-12-12-00", entries[0].Version)
}

func TestDailyCatalogDefaultsToSevenDays(t *testing.T) {
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	server := catalogServer(t, now)
	entries, err := provider.ListCatalog(context.Background(), provider.CatalogOptions{
		Provider: provider.IDBtbn, Channel: provider.ChannelDaily, Target: "win64",
		Variant: "gpl", Linkage: "static", Client: catalogClient(t, server),
		Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "autobuild-2026-11-12-12-00", entries[0].Snapshot)
	require.Equal(t, "N-128790", entries[0].Version)

	capped, err := provider.ListCatalog(context.Background(), provider.CatalogOptions{
		Provider: provider.IDBtbn, Channel: provider.ChannelDaily, Target: "win64",
		Variant: "gpl", Linkage: "static", Days: 60, Count: 1,
		Client: catalogClient(t, server), Now: func() time.Time { return now },
	})
	require.NoError(t, err)
	require.Len(t, capped, 1)
}
