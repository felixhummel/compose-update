package internal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockRegistryServer returns a test server that handles:
//   - token requests  → {"token": "test"}
//   - Docker Hub web API with name=v filter → {"results": [{"name": "v1.20.0"}], "next": null}
//   - OCI tags/list → {"tags": ["1.18.0", "1.19.0", "1.20.0"]}
func mockRegistryServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "service=") && strings.Contains(r.URL.RawQuery, "scope="):
			// auth.docker.io token endpoint
			w.Write([]byte(`{"token": "test"}`))
		case strings.HasPrefix(r.URL.Path, "/v2/repositories/") && strings.Contains(r.URL.RawQuery, "name=v"):
			// Docker Hub web API: name=v filter
			w.Write([]byte(`{"results": [{"name": "v999.0.0"}], "next": null}`))
		default:
			// OCI tags/list
			w.Write([]byte(`{"tags": ["1.18.0", "1.19.0", "999.0.0"]}`))
		}
	}))
}

func TestFetchImageTags_NoVPrefix(t *testing.T) {
	server := mockRegistryServer()
	defer server.Close()

	registry := NewRegistryForTest(server.URL)
	tags, err := registry.FetchImageTags("library/ubuntu:18.04")

	assert.NoError(t, err)
	// No v-prefix on current tag → OCI path
	assert.Equal(t, []string{"1.18.0", "1.19.0", "999.0.0"}, tags)
}

func TestFetchImageTags_VPrefix(t *testing.T) {
	server := mockRegistryServer()
	defer server.Close()

	registry := NewRegistryForTest(server.URL)
	tags, err := registry.FetchImageTags("prom/prometheus:v1.19.0")

	assert.NoError(t, err)
	// v-prefix on current tag → Docker Hub web API name=v path
	assert.Equal(t, []string{"v999.0.0"}, tags)
}

// TestFetchImageTags_DualAuthHeader tests registries that return both Bearer and
// Basic schemes in their WWW-Authenticate header (e.g. data.forgejo.org).
// The bug: parseBearer iterated all key=value pairs globally, so the Basic realm
// overwrote the Bearer realm, causing token requests to go to the wrong URL.
func TestFetchImageTags_DualAuthHeader(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/token":
			w.Write([]byte(`{"token": "test"}`))
		case strings.Contains(r.URL.Path, "/tags/list") && r.Header.Get("Authorization") == "Bearer test":
			w.Write([]byte(`{"tags": ["14.0.0", "13.0.0"]}`))
		case strings.Contains(r.URL.Path, "/tags/list"):
			// Dual Bearer+Basic WWW-Authenticate, as returned by data.forgejo.org
			realm := server.URL + "/v2/token"
			w.Header().Set("WWW-Authenticate",
				`Bearer realm="`+realm+`",service="container_registry",scope="*",Basic realm="`+server.URL+`/v2",service="container_registry",scope="*"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":""}]}`))
		}
	}))
	defer server.Close()


	registry := NewRegistryForTest(server.URL)
	tags, err := registry.FetchImageTags("data.forgejo.org/forgejo/forgejo:14")

	assert.NoError(t, err)
	assert.Equal(t, []string{"14.0.0", "13.0.0"}, tags)
}

func TestFetchImageTags_GitHubReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GitHub releases API endpoint
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			w.Write([]byte(`{"tag_name": "v2.0.0"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	registry := NewRegistryForTest(server.URL)
	tags, err := registry.FetchImageTags("ghcr.io/owner/repo:v1.0.0")

	assert.NoError(t, err)
	assert.Equal(t, []string{"v2.0.0"}, tags)
}
