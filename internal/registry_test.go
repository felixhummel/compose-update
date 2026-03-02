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
