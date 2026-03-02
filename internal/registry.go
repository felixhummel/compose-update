package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

type IRegistry interface {
	FetchImageTags(image string) ([]string, error)
}

type Registry struct {
	client  *http.Client
	timeout time.Duration
}

func NewRegistryWithTimeout(timeout time.Duration) *Registry {
	return &Registry{client: &http.Client{Timeout: timeout}, timeout: timeout}
}

// NewRegistryForTest returns a Registry that redirects all HTTP requests to
// serverURL, preserving path and query. This lets tests route by URL path
// instead of bypassing registry logic entirely.
func NewRegistryForTest(serverURL string) *Registry {
	var transport http.RoundTripper
	if serverURL != "" {
		u, _ := url.Parse(serverURL)
		transport = &redirectTransport{scheme: u.Scheme, host: u.Host}
	}
	return &Registry{
		client:  &http.Client{Timeout: 5 * time.Second, Transport: transport},
		timeout: 5 * time.Second,
	}
}

// redirectTransport rewrites every request's host to a fixed test server,
// while preserving the original path and query string.
type redirectTransport struct{ scheme, host string }

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.scheme
	req.URL.Host = t.host
	return http.DefaultTransport.RoundTrip(req)
}

// parseImageRef returns the registry host and repository for an image name.
// Examples:
//
//	"caddy"                    -> "registry-1.docker.io", "library/caddy"
//	"grafana/loki"             -> "registry-1.docker.io", "grafana/loki"
//	"gcr.io/cadvisor/cadvisor" -> "gcr.io", "cadvisor/cadvisor"
func parseImageRef(image string) (registry, repo string) {
	image = strings.Split(image, ":")[0]
	parts := strings.SplitN(image, "/", 2)
	if len(parts) == 1 {
		return "registry-1.docker.io", "library/" + image
	}
	if strings.ContainsAny(parts[0], ".:") || parts[0] == "localhost" {
		return parts[0], parts[1]
	}
	return "registry-1.docker.io", image
}

func isDockerHub(registry string) bool {
	return registry == "registry-1.docker.io" || registry == "docker.io"
}

func hasSemver(tags []string) bool {
	for _, tag := range tags {
		if _, err := semver.NewVersion(tag); err == nil {
			return true
		}
	}
	return false
}

func (r *Registry) FetchImageTags(image string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	parts := strings.SplitN(image, ":", 2)
	currentTag := ""
	if len(parts) == 2 {
		currentTag = parts[1]
	}

	registry, repo := parseImageRef(image)
	baseURL := "https://" + registry

	// For Docker Hub with a v-prefixed current tag: try the web API name=v filter first
	// (fetches only v-prefixed tags, ordered by last_updated — usually 1-2 pages)
	if isDockerHub(registry) && strings.HasPrefix(currentTag, "v") {
		tags, err := r.fetchDockerHubVPrefix(ctx, repo)
		if err == nil && hasSemver(tags) {
			slog.Debug("Using v-prefix tags from Docker Hub web API", "repo", repo, "count", len(tags))
			return tags, nil
		}
	}

	// Fall back to OCI API with early stop when pages contain no semver tags
	token, err := r.fetchToken(ctx, baseURL, registry, repo)
	if err != nil {
		return nil, err
	}
	return r.fetchTagsOCI(ctx, baseURL, repo, token)
}

// fetchDockerHubVPrefix fetches tags from the Docker Hub web API filtered to name=v,
// ordered by last_updated descending. Returns all pages that contain semver tags.
func (r *Registry) fetchDockerHubVPrefix(ctx context.Context, repo string) ([]string, error) {
	var tags []string
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?name=v&page_size=100&ordering=-last_updated", repo)

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		resp, _, err := do(r.client, req)
		if err != nil {
			return nil, err
		}

		var result struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
			Next string `json:"next"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		var page []string
		for _, t := range result.Results {
			page = append(page, t.Name)
		}
		tags = append(tags, page...)

		// Stop when a page has no semver tags — we've left the semver range
		if !hasSemver(page) {
			break
		}
		url = result.Next
	}

	return tags, nil
}

// fetchToken gets a bearer token for the given registry and repository.
// For Docker Hub, uses the known auth endpoint directly.
// For other registries, probes to discover auth via WWW-Authenticate.
func (r *Registry) fetchToken(ctx context.Context, baseURL, registry, repo string) (string, error) {
	if isDockerHub(registry) {
		tokenURL := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:pull", repo)
		return r.doTokenRequest(ctx, tokenURL)
	}

	// For other registries: probe to discover auth endpoint
	probeURL := fmt.Sprintf("%s/v2/%s/tags/list", baseURL, repo)
	resp, _, err := get(ctx, r.client, probeURL)
	if err != nil {
		return "", err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "", nil
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("registry probe returned %d", resp.StatusCode)
	}

	wwwAuth := resp.Header.Get("WWW-Authenticate")
	realm, queryParams, err := parseBearer(wwwAuth)
	if err != nil {
		return "", fmt.Errorf("parsing WWW-Authenticate: %w", err)
	}

	return r.doTokenRequest(ctx, realm+"?"+queryParams)
}

func (r *Registry) doTokenRequest(ctx context.Context, tokenURL string) (string, error) {
	resp, _, err := get(ctx, r.client, tokenURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

// parseBearer parses a WWW-Authenticate: Bearer header and returns the realm
// and query parameters string (e.g. "service=...&scope=...").
func parseBearer(wwwAuth string) (realm, queryParams string, err error) {
	re := regexp.MustCompile(`(\w+)="([^"]*)"`)
	matches := re.FindAllStringSubmatch(wwwAuth, -1)

	params := make(map[string]string)
	for _, m := range matches {
		params[m[1]] = m[2]
	}

	realm = params["realm"]
	if realm == "" {
		return "", "", fmt.Errorf("no realm in WWW-Authenticate: %q", wwwAuth)
	}

	var parts []string
	for k, v := range params {
		if k != "realm" {
			parts = append(parts, k+"="+v)
		}
	}
	return realm, strings.Join(parts, "&"), nil
}

// fetchTagsOCI paginates the OCI tags/list endpoint and stops early when a page
// contains no semver tags (we've left the semver range) or the context deadline fires.
func (r *Registry) fetchTagsOCI(ctx context.Context, baseURL, repo, token string) ([]string, error) {
	var tags []string
	url := fmt.Sprintf("%s/v2/%s/tags/list?n=100", baseURL, repo)

	for url != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, _, err := do(r.client, req)
		if err != nil {
			if ctx.Err() != nil {
				slog.Debug("Stopping pagination (deadline exceeded)", "repo", repo, "tags_so_far", len(tags))
				return tags, nil
			}
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("registry returned %d fetching tags", resp.StatusCode)
		}

		var result struct {
			Tags []string `json:"tags"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		tags = append(tags, result.Tags...)

		// Stop when a page has no semver tags — we've left the semver range
		if !hasSemver(result.Tags) {
			slog.Debug("Stopping pagination (no semver on page)", "repo", repo, "tags_so_far", len(tags))
			break
		}

		url = parseLinkNext(resp.Header.Get("Link"), baseURL)
	}

	return tags, nil
}

func get(ctx context.Context, client *http.Client, url string) (*http.Response, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	return do(client, req)
}

func do(client *http.Client, req *http.Request) (*http.Response, time.Duration, error) {
	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	slog.Debug("GET", "url", req.URL, "elapsed", elapsed, "status", statusCode(resp), "err", err)
	return resp, elapsed, err
}

func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// parseLinkNext parses the Link header for the next page URL.
// Example: Link: </v2/repo/tags/list?last=tag&n=100>; rel="next"
func parseLinkNext(link, baseURL string) string {
	if link == "" {
		return ""
	}
	re := regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
	m := re.FindStringSubmatch(link)
	if m == nil {
		return ""
	}
	path := m[1]
	if strings.HasPrefix(path, "http") {
		return path
	}
	return baseURL + path
}
