package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReleaseChannels(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version    string
		prerelease bool
		channel    string
	}{
		{"0.32.0", false, "stable"},
		{"0.32.0+build-with-hyphen", false, "stable"},
		{"0.32.0", true, "beta"},
		{"0.32.0-beta.1", false, "beta"},
		{"0.32.0-rc.1", true, "beta"},
		{"0.32.0-dev.abc123", true, "dev"},
		{"0.32.0-dev.abc123+build", false, "dev"},
		{"0.32.0-dev", false, "dev"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			if got := releaseChannel(tc.prerelease, tc.version); got != tc.channel {
				t.Fatalf("channel = %q, want %q", got, tc.channel)
			}
		})
	}
}

func TestBetaSemanticOrdering(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"0.32.0-beta.10", "0.32.0-beta.9", 1},
		{"0.32.0", "0.32.0-beta.10", 1},
		{"0.32.0-beta.1+one", "0.32.0-beta.1+two", 0},
		{"0.32.0-beta.1", "0.32.0-beta.2", -1},
		{"0.32.0-beta", "0.32.0-beta.1", -1},
		{"0.32.0-1", "0.32.0-beta", -1},
	} {
		if got := compareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCatalogFindsBetaBeyondFrequentDevReleases(t *testing.T) {
	t.Parallel()
	makeRelease := func(component, version string, date time.Time) githubRelease {
		tag := component + "-v" + version
		filename := fmt.Sprintf("printmaster-%s-v%s-linux-amd64", component, version)
		return githubRelease{
			TagName: tag, HTMLURL: "https://github.com/Printmaster-Org/printmaster/releases/tag/" + tag,
			PublishedAt: date, Prerelease: strings.Contains(version, "-"),
			Assets: []githubAsset{{Name: filename, BrowserDownloadURL: "https://github.com/Printmaster-Org/printmaster/releases/download/" + tag + "/" + filename}},
		}
	}
	first := make([]githubRelease, 0, 100)
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	for _, component := range []string{"agent", "server"} {
		for i := 0; i < 5; i++ {
			first = append(first, makeRelease(component, fmt.Sprintf("0.31.%d", i), now))
		}
		for i := 0; i < 45; i++ {
			first = append(first, makeRelease(component, fmt.Sprintf("0.32.0-dev.sha%d", i), now.Add(time.Duration(i)*time.Hour)))
		}
	}
	second := []githubRelease{
		makeRelease("agent", "0.32.0-beta.9", now.Add(2*time.Hour)),
		makeRelease("agent", "0.32.0-beta.10", now),
		makeRelease("server", "0.32.0-beta.1", now),
	}
	hits := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		batch := first
		if r.URL.Query().Get("page") == "2" {
			batch = second
		}
		_ = json.NewEncoder(w).Encode(batch)
	}))
	defer api.Close()
	service := newReleaseCatalogService(api.URL, "", "fixture-secret", api.Client())
	catalog, err := service.get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 2 || len(catalog.Agent.Beta) != 2 || catalog.Agent.Beta[0].Version != "0.32.0-beta.10" {
		t.Fatalf("Beta pagination/order failed: hits=%d, beta=%+v", hits, catalog.Agent.Beta)
	}
	for _, channels := range []releaseChannels{catalog.Agent, catalog.Server} {
		if len(channels.Stable) != 5 || len(channels.Dev) != 5 || channels.Dev[0].Version != "0.32.0-dev.sha44" {
			t.Fatalf("channel caps/order failed: %+v", channels)
		}
	}
	payload := []byte(`{"tag_name":"agent-v0.32.0-dev.sha44"}`)
	res := httptest.NewRecorder()
	service.handleRefreshHTTP(res, signedWebhookRequest("fixture-secret", payload))
	if res.Code != http.StatusNoContent || hits != 2 {
		t.Fatalf("duplicate Dev webhook = %d, hits=%d", res.Code, hits)
	}
}
