package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateUpdateInfos(t *testing.T) {
	tests := []struct {
		name     string
		fileData string
		expected []UpdateInfo
	}{
		{
			name: "Single image",
			fileData: `
image: library/ubuntu:18.04.0
`,
			expected: []UpdateInfo{
				{
					RawLine:       "image: library/ubuntu:18.04.0",
					FullImageName: "library/ubuntu:18.04.0",
					ImageName:     "library/ubuntu",
					CurrentTag:    "18.04.0",
				},
			},
		},
		{
			name: "Multiple images",
			fileData: `
image: library/ubuntu:18.04.0
image: library/nginx:1.19.0
`,
			expected: []UpdateInfo{
				{
					RawLine:       "image: library/ubuntu:18.04.0",
					FullImageName: "library/ubuntu:18.04.0",
					ImageName:     "library/ubuntu",
					CurrentTag:    "18.04.0",
				},
				{
					RawLine:       "image: library/nginx:1.19.0",
					FullImageName: "library/nginx:1.19.0",
					ImageName:     "library/nginx",
					CurrentTag:    "1.19.0",
				},
			},
		},
		{
			name: "Duplicate images",
			fileData: `
image: library/ubuntu:18.04.0
image: library/ubuntu:18.04.0
`,
			expected: []UpdateInfo{
				{
					RawLine:       "image: library/ubuntu:18.04.0",
					FullImageName: "library/ubuntu:18.04.0",
					ImageName:     "library/ubuntu",
					CurrentTag:    "18.04.0",
				},
			},
		},
		{
			name: "No tag",
			fileData: `
image: library/ubuntu
`,
			expected: []UpdateInfo{
				{
					RawLine:       "image: library/ubuntu",
					FullImageName: "library/ubuntu",
					ImageName:     "library/ubuntu",
					CurrentTag:    "",
				},
			},
		},
		{
			name: "IMG build arg",
			fileData: `
services:
  caddy:
    build:
      context: caddy/
      args:
        IMG: caddy:2.10.2
  prometheus:
    build:
      context: prometheus/
      args:
        IMG: prom/prometheus:v3.7.2
`,
			expected: []UpdateInfo{
				{
					RawLine:       "        IMG: caddy:2.10.2",
					FullImageName: "caddy:2.10.2",
					ImageName:     "caddy",
					CurrentTag:    "2.10.2",
				},
				{
					RawLine:       "        IMG: prom/prometheus:v3.7.2",
					FullImageName: "prom/prometheus:v3.7.2",
					ImageName:     "prom/prometheus",
					CurrentTag:    "v3.7.2",
				},
			},
		},
		{
			name: "Mixed image and IMG build arg",
			fileData: `
services:
  app:
    image: library/nginx:1.19.0
  caddy:
    build:
      args:
        IMG: caddy:2.10.2
`,
			expected: []UpdateInfo{
				{
					RawLine:       "    image: library/nginx:1.19.0",
					FullImageName: "library/nginx:1.19.0",
					ImageName:     "library/nginx",
					CurrentTag:    "1.19.0",
				},
				{
					RawLine:       "        IMG: caddy:2.10.2",
					FullImageName: "caddy:2.10.2",
					ImageName:     "caddy",
					CurrentTag:    "2.10.2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.CreateTemp("", "testfile.yaml")
			assert.NoError(t, err)
			defer os.Remove(file.Name())

			_, err = file.WriteString(tt.fileData)
			assert.NoError(t, err)
			file.Close()

			for i := range tt.expected {
				tt.expected[i].FilePath = file.Name()
			}

			updateChecker := NewUpdateChecker(file.Name(), NewRegistryForTest(""))
			updateInfos, err := updateChecker.createUpdateInfos()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, updateInfos)
		})
	}
}

func TestUpdateCheckerCheck(t *testing.T) {
	tests := []struct {
		name     string
		fileData string
		expected []UpdateInfo
	}{
		{
			name: "Single image",
			fileData: `
image: library/myimage:1.19.0
`,
			expected: []UpdateInfo{
				{
					RawLine:       "image: library/myimage:1.19.0",
					FullImageName: "library/myimage:1.19.0",
					ImageName:     "library/myimage",
					CurrentTag:    "1.19.0",
					LatestTag:     "999.0.0",
				},
			},
		},
		{
			name: "IMG build arg update",
			fileData: `
services:
  caddy:
    build:
      context: caddy/
      args:
        IMG: caddy:1.19.0
`,
			expected: []UpdateInfo{
				{
					RawLine:       "        IMG: caddy:1.19.0",
					FullImageName: "caddy:1.19.0",
					ImageName:     "caddy",
					CurrentTag:    "1.19.0",
					LatestTag:     "999.0.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := os.CreateTemp("", "testfile.yaml")
			assert.NoError(t, err)
			defer os.Remove(file.Name())

			_, err = file.WriteString(tt.fileData)
			assert.NoError(t, err)
			file.Close()

			for i := range tt.expected {
				tt.expected[i].FilePath = file.Name()
			}

			server := mockRegistryServer()
			defer server.Close()

			registry := NewRegistryForTest(server.URL)
			updateChecker := NewUpdateChecker(file.Name(), registry)

			result, err := updateChecker.Check(MajorLevel)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// newMockRegistry serves canned tag lists for every registry code path:
//   - Docker Hub web API with name=v (v-prefixed tags) → v999.0.0
//   - GitHub releases API (ghcr.io images)              → v999.0.0
//   - OCI tags/list (everything else)                   → 999.0.0
func newMockRegistry(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.RawQuery, "service=") && strings.Contains(r.URL.RawQuery, "scope="):
			w.Write([]byte(`{"token": "test"}`))
		case strings.HasPrefix(r.URL.Path, "/v2/repositories/") && strings.Contains(r.URL.RawQuery, "name=v"):
			w.Write([]byte(`{"results": [{"name": "v999.0.0"}], "next": null}`))
		case strings.HasPrefix(r.URL.Path, "/repos/"):
			w.Write([]byte(`{"tag_name": "v999.0.0"}`))
		default:
			w.Write([]byte(`{"tags": ["999.0.0"]}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// assertUpdatedCompose copies <fixtureDir>/docker-compose.yml to a temp dir, runs
// Check + Update against the mock registry and compares the result with
// <fixtureDir>/expected.yml. Regenerate with `go test ./internal -update`.
func assertUpdatedCompose(t *testing.T, fixtureDir string) {
	t.Helper()
	original, err := os.ReadFile(filepath.Join(fixtureDir, "docker-compose.yml"))
	assert.NoError(t, err)
	composePath := filepath.Join(t.TempDir(), "docker-compose.yml")
	assert.NoError(t, os.WriteFile(composePath, original, 0644))

	registry := NewRegistryForTest(newMockRegistry(t).URL)
	updateChecker := NewUpdateChecker(composePath, registry)

	result, err := updateChecker.Check(MajorLevel)
	assert.NoError(t, err)
	for _, r := range result {
		if r.HasNewVersion() {
			assert.NoError(t, r.Update())
		}
	}

	updated, err := os.ReadFile(composePath)
	assert.NoError(t, err)
	g := goldie.New(t, goldie.WithFixtureDir(fixtureDir), goldie.WithNameSuffix(".yml"))
	g.Assert(t, "expected", updated)
}

func TestUpdateCheckerCheckBuildArgImgFixture(t *testing.T) {
	// Mixes v-prefixed (prom/prometheus:v3.7.2, prom/node-exporter:v1.10.2) and
	// non-prefixed (caddy:1.19.0, authelia/authelia:4.39, grafana/grafana:12.3.4) images.
	assertUpdatedCompose(t, "../tests/build-arg-img")
}

func TestUpdateCheckerCheckMainFixture(t *testing.T) {
	// Covers :latest (skipped), Docker Hub, a third-party OCI registry (forgejo) and ghcr.io.
	assertUpdatedCompose(t, "../tests")
}
