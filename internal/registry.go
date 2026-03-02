package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

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

type IRegistry interface {
	FetchImageTags(image string) ([]string, error)
}

type Registry struct {
	client  *http.Client
	timeout time.Duration
	// testURL overrides the registry base URL for all images (used in tests)
	testURL string
}

func NewRegistry(url string) *Registry {
	return &Registry{client: &http.Client{Timeout: 5 * time.Second}, timeout: 5 * time.Second, testURL: url}
}

func NewRegistryWithTimeout(timeout time.Duration) *Registry {
	return &Registry{client: &http.Client{Timeout: timeout}, timeout: timeout}
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

func (r *Registry) FetchImageTags(image string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	repo := strings.Split(image, ":")[0]

	if r.testURL != "" {
		return r.fetchTags(ctx, r.testURL, repo, "")
	}

	registry, repo := parseImageRef(image)
	baseURL := "https://" + registry

	token, err := r.fetchToken(ctx, baseURL, registry, repo)
	if err != nil {
		return nil, err
	}

	return r.fetchTags(ctx, baseURL, repo, token)
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

func (r *Registry) fetchTags(ctx context.Context, baseURL, repo, token string) ([]string, error) {
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
			// Deadline exceeded mid-pagination: return what we have so far
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
		err = json.NewDecoder(resp.Body).Decode(&result)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		tags = append(tags, result.Tags...)

		url = parseLinkNext(resp.Header.Get("Link"), baseURL)
	}

	return tags, nil
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
