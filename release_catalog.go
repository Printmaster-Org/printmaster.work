package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultGitHubAPIURL = "https://api.github.com"
	releaseRepoOwner    = "printmaster-org"
	releaseRepoName     = "printmaster"
	releasesPerChannel  = 5
	maxReleasePages     = 10
	maxWebhookBodyBytes = 4096
)

var releaseVersionPattern = regexp.MustCompile(`^([0-9]+)\.([0-9]+)\.([0-9]+)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

type productRelease struct {
	Component   string         `json:"component"`
	Version     string         `json:"version"`
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	HTMLURL     string         `json:"html_url"`
	Prerelease  bool           `json:"prerelease"`
	PublishedAt time.Time      `json:"published_at"`
	Assets      []releaseAsset `json:"assets"`
}

type releaseChannels struct {
	Stable []productRelease `json:"stable"`
	Beta   []productRelease `json:"beta"`
}

type releaseCatalog struct {
	FetchedAt time.Time       `json:"fetched_at"`
	Stale     bool            `json:"stale"`
	Agent     releaseChannels `json:"agent"`
	Server    releaseChannels `json:"server"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type releaseCatalogService struct {
	apiURL        string
	token         string
	webhookSecret string
	client        *http.Client
	now           func() time.Time
	mu            sync.Mutex
	cached        *releaseCatalog
}

func newReleaseCatalogService(apiURL, token, webhookSecret string, client *http.Client) *releaseCatalogService {
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL == "" {
		apiURL = defaultGitHubAPIURL
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	return &releaseCatalogService{
		apiURL:        strings.TrimRight(apiURL, "/"),
		token:         strings.TrimSpace(token),
		webhookSecret: strings.TrimSpace(webhookSecret),
		client:        client,
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (s *releaseCatalogService) handleHTTP(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.get(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "release list temporarily unavailable"})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	if err := json.NewEncoder(w).Encode(catalog); err != nil {
		return
	}
}

func (s *releaseCatalogService) handleRefreshHTTP(w http.ResponseWriter, r *http.Request) {
	if s.webhookSecret == "" {
		http.Error(w, "release refresh webhook is not configured", http.StatusServiceUnavailable)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBodyBytes+1))
	if err != nil || len(body) > maxWebhookBodyBytes {
		http.Error(w, "invalid release notification", http.StatusBadRequest)
		return
	}
	if !verifyReleaseSignature(s.webhookSecret, body, r.Header.Get("X-PrintMaster-Signature")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var notification struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &notification); err != nil {
		http.Error(w, "invalid release notification", http.StatusBadRequest)
		return
	}
	if _, _, ok := parseReleaseTag(notification.TagName); !ok {
		http.Error(w, "unsupported release tag", http.StatusBadRequest)
		return
	}
	if err := s.Refresh(r.Context(), notification.TagName); err != nil {
		http.Error(w, "release metadata refresh failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func verifyReleaseSignature(secret string, body []byte, signature string) bool {
	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil || len(provided) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func (s *releaseCatalogService) get(ctx context.Context) (releaseCatalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return *s.cached, nil
	}
	catalog, err := s.fetch(ctx)
	if err != nil {
		return releaseCatalog{}, err
	}
	catalog.FetchedAt = s.now()
	s.cached = &catalog
	return catalog, nil
}

// Refresh replaces in-memory release metadata. Empty tag is used for startup/manual refresh.
func (s *releaseCatalogService) Refresh(ctx context.Context, tagName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tagName != "" && s.cached != nil && catalogContainsTag(s.cached, tagName) {
		return nil
	}
	catalog, err := s.fetch(ctx)
	if err != nil {
		return err
	}
	catalog.FetchedAt = s.now()
	s.cached = &catalog
	return nil
}

func catalogContainsTag(catalog *releaseCatalog, tagName string) bool {
	for _, channels := range []releaseChannels{catalog.Agent, catalog.Server} {
		for _, releases := range [][]productRelease{channels.Stable, channels.Beta} {
			for _, release := range releases {
				if release.TagName == tagName {
					return true
				}
			}
		}
	}
	return false
}

func (s *releaseCatalogService) fetch(ctx context.Context) (releaseCatalog, error) {
	catalog := releaseCatalog{
		Agent:  releaseChannels{Stable: []productRelease{}, Beta: []productRelease{}},
		Server: releaseChannels{Stable: []productRelease{}, Beta: []productRelease{}},
	}
	for page := 1; page <= maxReleasePages; page++ {
		endpoint, err := url.Parse(s.apiURL + "/repos/" + releaseRepoOwner + "/" + releaseRepoName + "/releases")
		if err != nil {
			return releaseCatalog{}, err
		}
		query := endpoint.Query()
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return releaseCatalog{}, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "PrintMaster-Website-ReleaseCatalog")
		if s.token != "" {
			req.Header.Set("Authorization", "Bearer "+s.token)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return releaseCatalog{}, err
		}
		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
			_ = resp.Body.Close()
			return releaseCatalog{}, fmt.Errorf("GitHub releases API returned %s", resp.Status)
		}

		var batch []githubRelease
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&batch)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return releaseCatalog{}, decodeErr
		}
		for _, raw := range batch {
			if raw.Draft {
				continue
			}
			release, ok := normalizeRelease(raw)
			if !ok {
				continue
			}
			channels := channelsFor(&catalog, release.Component)
			if isBetaRelease(raw.Prerelease, release.Version) {
				channels.Beta = append(channels.Beta, release)
			} else {
				channels.Stable = append(channels.Stable, release)
			}
		}

		if len(batch) < 100 || catalogHasEnough(&catalog) {
			break
		}
	}

	for _, channels := range []*releaseChannels{&catalog.Agent, &catalog.Server} {
		sort.SliceStable(channels.Stable, func(i, j int) bool {
			if compareVersions(channels.Stable[i].Version, channels.Stable[j].Version) == 0 {
				return channels.Stable[i].PublishedAt.After(channels.Stable[j].PublishedAt)
			}
			return compareVersions(channels.Stable[i].Version, channels.Stable[j].Version) > 0
		})
		sort.SliceStable(channels.Beta, func(i, j int) bool {
			return channels.Beta[i].PublishedAt.After(channels.Beta[j].PublishedAt)
		})
		if len(channels.Stable) > releasesPerChannel {
			channels.Stable = channels.Stable[:releasesPerChannel]
		}
		if len(channels.Beta) > releasesPerChannel {
			channels.Beta = channels.Beta[:releasesPerChannel]
		}
	}
	return catalog, nil
}

func normalizeRelease(raw githubRelease) (productRelease, bool) {
	component, version, ok := parseReleaseTag(raw.TagName)
	if !ok {
		return productRelease{}, false
	}
	if raw.HTMLURL == "" || !trustedGitHubURL(raw.HTMLURL) {
		return productRelease{}, false
	}

	assets := make([]releaseAsset, 0, len(raw.Assets))
	for _, asset := range raw.Assets {
		if !matchesReleaseAsset(component, version, asset.Name) || !trustedGitHubURL(asset.BrowserDownloadURL) {
			continue
		}
		assets = append(assets, releaseAsset{Name: asset.Name, URL: asset.BrowserDownloadURL, Size: asset.Size})
	}
	if len(assets) == 0 {
		return productRelease{}, false
	}
	name := strings.TrimSpace(raw.Name)
	if name == "" {
		name = raw.TagName
	}
	return productRelease{
		Component:   component,
		Version:     version,
		TagName:     raw.TagName,
		Name:        name,
		HTMLURL:     raw.HTMLURL,
		Prerelease:  raw.Prerelease,
		PublishedAt: raw.PublishedAt,
		Assets:      assets,
	}, true
}

func parseReleaseTag(tag string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(tag), "-", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	component := strings.ToLower(parts[0])
	if component != "agent" && component != "server" {
		return "", "", false
	}
	version := strings.TrimPrefix(parts[1], "v")
	if !releaseVersionPattern.MatchString(version) {
		return "", "", false
	}
	return component, version, true
}

func matchesReleaseAsset(component, version, name string) bool {
	if strings.HasSuffix(strings.ToLower(name), ".rpm") {
		return strings.HasPrefix(name, fmt.Sprintf("printmaster-%s-%s-", component, version))
	}
	return strings.HasPrefix(name, fmt.Sprintf("printmaster-%s-v%s-", component, version)) ||
		strings.HasPrefix(name, fmt.Sprintf("printmaster-%s_%s_", component, version))
}

func trustedGitHubURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && strings.EqualFold(parsed.Hostname(), "github.com")
}

func isBetaRelease(prerelease bool, version string) bool {
	return prerelease || strings.Contains(version, "-")
}

func channelsFor(catalog *releaseCatalog, component string) *releaseChannels {
	if component == "agent" {
		return &catalog.Agent
	}
	return &catalog.Server
}

func catalogHasEnough(catalog *releaseCatalog) bool {
	return len(catalog.Agent.Stable) >= releasesPerChannel && len(catalog.Server.Stable) >= releasesPerChannel
}

func compareVersions(a, b string) int {
	ma, mb := releaseVersionPattern.FindStringSubmatch(a), releaseVersionPattern.FindStringSubmatch(b)
	if ma == nil || mb == nil {
		return strings.Compare(a, b)
	}
	for i := 1; i <= 3; i++ {
		va, _ := strconv.ParseUint(ma[i], 10, 64)
		vb, _ := strconv.ParseUint(mb[i], 10, 64)
		if va > vb {
			return 1
		}
		if va < vb {
			return -1
		}
	}
	if ma[4] == mb[4] {
		return strings.Compare(ma[5], mb[5])
	}
	if ma[4] == "" {
		return 1
	}
	if mb[4] == "" {
		return -1
	}
	return strings.Compare(ma[4], mb[4])
}
