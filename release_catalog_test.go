package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseCatalogFetchAndCache(t *testing.T) {
	var apiHits int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHits++
		if r.URL.Path != "/repos/printmaster-org/printmaster/releases" {
			t.Errorf("unexpected GitHub API path %q", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != "1" {
			t.Errorf("unexpected GitHub API pagination query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(releaseFixtureJSON()))
	}))
	defer api.Close()

	service := newReleaseCatalogService(api.URL, "", "", api.Client())
	catalog, err := service.get(context.Background())
	if err != nil {
		t.Fatalf("get release catalog: %v", err)
	}
	if len(catalog.Agent.Stable) != 2 || catalog.Agent.Stable[0].Version != "1.10.0" {
		t.Fatalf("Agent stable releases = %#v; want semver-descending releases starting at 1.10.0", catalog.Agent.Stable)
	}
	if len(catalog.Agent.Beta) != 1 || catalog.Agent.Beta[0].Version != "1.11.0-beta.1" {
		t.Fatalf("Agent beta releases = %#v", catalog.Agent.Beta)
	}
	if len(catalog.Agent.Beta[0].Assets) != 3 {
		t.Fatalf("Beta binary/DEB/RPM assets = %#v", catalog.Agent.Beta[0].Assets)
	}
	if len(catalog.Agent.Dev) != 1 || catalog.Agent.Dev[0].Version != "1.11.0-dev.abc123" {
		t.Fatalf("Agent Dev releases = %#v", catalog.Agent.Dev)
	}
	if !catalogContainsTag(&catalog, catalog.Agent.Dev[0].TagName) {
		t.Fatal("Dev tag must participate in duplicate webhook detection")
	}
	if len(catalog.Server.Stable) != 1 || catalog.Server.Stable[0].Version != "2.0.0" {
		t.Fatalf("Server stable releases = %#v", catalog.Server.Stable)
	}
	if len(catalog.Server.Beta) != 1 || catalog.Server.Beta[0].Version != "2.1.0" {
		t.Fatalf("Server beta releases = %#v", catalog.Server.Beta)
	}
	if got := catalog.Agent.Stable[0].Assets[0].URL; !strings.HasPrefix(got, "https://github.com/") {
		t.Fatalf("asset URL = %q, want direct GitHub URL", got)
	}
	if apiHits != 1 {
		t.Fatalf("GitHub API hits = %d, want 1", apiHits)
	}

	if _, err := service.get(context.Background()); err != nil {
		t.Fatalf("get cached release catalog: %v", err)
	}
	if apiHits != 1 {
		t.Fatalf("cached read triggered another GitHub scan; hits = %d", apiHits)
	}
}

func TestReleaseWebhookRefreshesMetadataOnly(t *testing.T) {
	secret := "test-webhook-secret"
	var apiHits int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHits++
		w.Header().Set("Content-Type", "application/json")
		if apiHits == 1 {
			_, _ = w.Write([]byte(releaseFixtureJSON()))
			return
		}
		_, _ = w.Write([]byte(releaseFixtureWithNewAgentJSON()))
	}))
	defer api.Close()

	service := newReleaseCatalogService(api.URL, "", secret, api.Client())
	if err := service.Refresh(context.Background(), ""); err != nil {
		t.Fatalf("initial refresh: %v", err)
	}

	handler, err := newHandlerWithReleaseService(service)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	payload := []byte(`{"tag_name":"agent-v1.12.0"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/releases/refresh", strings.NewReader(string(payload)))
	req.Header.Set("X-PrintMaster-Signature", releaseSignature(secret, payload))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("valid webhook status = %d, want 204: %s", res.Code, res.Body.String())
	}
	if apiHits != 2 {
		t.Fatalf("GitHub API hits after release notification = %d, want 2", apiHits)
	}

	res = httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/releases", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("release catalog status = %d", res.Code)
	}
	var got releaseCatalog
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}
	if got.Agent.Stable[0].Version != "1.12.0" {
		t.Fatalf("latest Agent stable version = %q, want 1.12.0", got.Agent.Stable[0].Version)
	}
	if apiHits != 2 {
		t.Fatalf("reading cached catalog triggered GitHub request; hits = %d", apiHits)
	}

	// Duplicate delivery for a release already in the snapshot is idempotent.
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, signedWebhookRequest(secret, payload))
	if res.Code != http.StatusNoContent || apiHits != 2 {
		t.Fatalf("duplicate delivery status=%d API hits=%d; want 204 and no extra API call", res.Code, apiHits)
	}
}

func TestReleaseWebhookRejectsInvalidRequests(t *testing.T) {
	apiHits := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		apiHits++
		_, _ = w.Write([]byte("[]"))
	}))
	defer api.Close()

	tests := []struct {
		name    string
		secret  string
		payload string
		sig     string
		want    int
	}{
		{name: "missing configured secret", payload: `{"tag_name":"agent-v1.0.0"}`, want: http.StatusServiceUnavailable},
		{name: "invalid signature", secret: "configured", payload: `{"tag_name":"agent-v1.0.0"}`, sig: "sha256=00", want: http.StatusUnauthorized},
		{name: "unsupported tag", secret: "configured", payload: `{"tag_name":"other-v1.0.0"}`, want: http.StatusBadRequest},
		{name: "malformed JSON", secret: "configured", payload: `{`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := newReleaseCatalogService(api.URL, "", tt.secret, api.Client())
			handler, err := newHandlerWithReleaseService(service)
			if err != nil {
				t.Fatalf("new handler: %v", err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/releases/refresh", strings.NewReader(tt.payload))
			if tt.sig != "" {
				req.Header.Set("X-PrintMaster-Signature", tt.sig)
			} else if tt.secret != "" {
				req.Header.Set("X-PrintMaster-Signature", releaseSignature(tt.secret, []byte(tt.payload)))
			}
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != tt.want {
				t.Errorf("status = %d, want %d; body=%s", res.Code, tt.want, res.Body.String())
			}
		})
	}
	if apiHits != 0 {
		t.Fatalf("invalid notification caused %d GitHub API calls", apiHits)
	}
}

func TestDownloadsPageServed(t *testing.T) {
	service := newReleaseCatalogService("", "", "", nil)
	handler, err := newHandlerWithReleaseService(service)
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/downloads", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("downloads page status = %d", res.Code)
	}
	for _, expected := range []string{"Official PrintMaster downloads", "id=\"tab-agent\"", "id=\"tab-server\"", "id=\"dev-releases\"", "Beta has no MSI", "/assets/downloads.js", "/assets/downloads-readable.css"} {
		if !strings.Contains(res.Body.String(), expected) {
			t.Errorf("downloads page missing %q", expected)
		}
	}
}

func releaseFixtureJSON() string {
	return `[
		{"tag_name":"agent-v1.9.0","name":"Agent 1.9","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v1.9.0","published_at":"2026-08-01T00:00:00Z","assets":[{"name":"printmaster-agent-v1.9.0-windows-amd64.msi","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.9.0/printmaster-agent-v1.9.0-windows-amd64.msi","size":100}]},
		{"tag_name":"agent-v1.10.0","name":"Agent 1.10","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v1.10.0","published_at":"2026-08-02T00:00:00Z","assets":[{"name":"printmaster-agent-v1.10.0-windows-amd64.msi","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.10.0/printmaster-agent-v1.10.0-windows-amd64.msi","size":100},{"name":"setup.exe","browser_download_url":"https://example.com/setup.exe","size":20}]},
		{"tag_name":"agent-v1.11.0-dev.abc123","name":"Agent Dev","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v1.11.0-dev.abc123","prerelease":true,"published_at":"2026-08-03T00:00:00Z","assets":[{"name":"printmaster-agent-v1.11.0-dev.abc123-windows-amd64.exe","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.11.0-dev.abc123/printmaster-agent-v1.11.0-dev.abc123-windows-amd64.exe","size":100}]},
		{"tag_name":"agent-v1.11.0-beta.1","name":"Agent Beta","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v1.11.0-beta.1","prerelease":true,"published_at":"2026-08-03T00:00:00Z","assets":[{"name":"printmaster-agent-v1.11.0-beta.1-windows-amd64.exe","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.11.0-beta.1/printmaster-agent-v1.11.0-beta.1-windows-amd64.exe","size":100},{"name":"printmaster-agent_1.11.0-beta.1_amd64.deb","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.11.0-beta.1/printmaster-agent_1.11.0-beta.1_amd64.deb","size":100},{"name":"printmaster-agent-1.11.0-beta.1-1.fc44.x86_64.rpm","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.11.0-beta.1/printmaster-agent-1.11.0-beta.1-1.fc44.x86_64.rpm","size":100}]},
		{"tag_name":"server-v2.0.0","name":"Server 2.0","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/server-v2.0.0","published_at":"2026-08-04T00:00:00Z","assets":[{"name":"printmaster-server-v2.0.0-linux-amd64","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/server-v2.0.0/printmaster-server-v2.0.0-linux-amd64","size":100}]},
		{"tag_name":"server-v2.1.0","name":"Server beta","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/server-v2.1.0","prerelease":true,"published_at":"2026-08-05T00:00:00Z","assets":[{"name":"printmaster-server-v2.1.0-dev.abc123-linux-amd64","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/server-v2.1.0/printmaster-server-v2.1.0-dev.abc123-linux-amd64","size":100}]},
		{"tag_name":"other-v8.0.0","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/other-v8.0.0","assets":[{"name":"other-v8.0.0.exe","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/other-v8.0.0/other-v8.0.0.exe","size":100}]},
		{"tag_name":"agent-v9.0.0","draft":true,"html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v9.0.0","assets":[{"name":"printmaster-agent-v9.0.0-linux-amd64","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v9.0.0/printmaster-agent-v9.0.0-linux-amd64","size":100}]}
	]`
}

func releaseFixtureWithNewAgentJSON() string {
	fixture := releaseFixtureJSON()
	insert := `{"tag_name":"agent-v1.12.0","name":"Agent 1.12","html_url":"https://github.com/Printmaster-Org/printmaster/releases/tag/agent-v1.12.0","published_at":"2026-09-01T00:00:00Z","assets":[{"name":"printmaster-agent-v1.12.0-windows-amd64.msi","browser_download_url":"https://github.com/Printmaster-Org/printmaster/releases/download/agent-v1.12.0/printmaster-agent-v1.12.0-windows-amd64.msi","size":100}]},`
	return strings.Replace(fixture, "[", "["+insert, 1)
}

func releaseSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func signedWebhookRequest(secret string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/releases/refresh", strings.NewReader(string(body)))
	req.Header.Set("X-PrintMaster-Signature", releaseSignature(secret, body))
	return req
}
