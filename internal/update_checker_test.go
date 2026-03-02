package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
			// Create a temporary file with the test data
			file, err := os.CreateTemp("", "testfile.yaml")
			assert.NoError(t, err)
			defer os.Remove(file.Name())

			_, err = file.WriteString(tt.fileData)
			assert.NoError(t, err)
			file.Close()

			// Update the expected FilePath to match the temporary file name
			for i := range tt.expected {
				tt.expected[i].FilePath = file.Name()
			}

			// Create an UpdateChecker instance
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"tags": ["1.18.0", "1.18.1", "1.19.0", "1.20.0"]}`))
			}))
			defer server.Close()

			registry := NewRegistry(server.URL)
			updateChecker := NewUpdateChecker(file.Name(), registry)

			// Call createUpdateInfos
			updateInfos, err := updateChecker.createUpdateInfos()
			assert.NoError(t, err)

			// Verify the results
			assert.Equal(t, tt.expected, updateInfos)
		})
	}
}

func TestUpdateCheckerCheck(t *testing.T) {
	mockTags := `{"tags": ["1.18.0", "1.18.1", "1.19.0", "1.20.0"]}`

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
					LatestTag:     "1.20.0",
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
					LatestTag:     "1.20.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file with the test data
			file, err := os.CreateTemp("", "testfile.yaml")
			assert.NoError(t, err)
			defer os.Remove(file.Name())

			_, err = file.WriteString(tt.fileData)
			assert.NoError(t, err)
			file.Close()

			// Update the expected FilePath to match the temporary file name
			for i := range tt.expected {
				tt.expected[i].FilePath = file.Name()
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockTags))
			}))
			defer server.Close()

			registry := NewRegistry(server.URL)
			updateChecker := NewUpdateChecker(file.Name(), registry)

			result, err := updateChecker.Check(true, true, true)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUpdateCheckerCheckBuildArgImgFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tags": ["1.18.0", "1.18.1", "1.19.0", "1.20.0"]}`))
	}))
	defer server.Close()

	registry := NewRegistry(server.URL)
	updateChecker := NewUpdateChecker("../tests/build-arg-img/docker-compose.yml", registry)

	result, err := updateChecker.Check(true, true, true)
	assert.NoError(t, err)

	// caddy:1.19.0 should be detected as updatable to 1.20.0
	var caddyInfo *UpdateInfo
	for i := range result {
		if result[i].ImageName == "caddy" {
			caddyInfo = &result[i]
			break
		}
	}
	assert.NotNil(t, caddyInfo, "caddy IMG entry not found")
	assert.Equal(t, "1.19.0", caddyInfo.CurrentTag)
	assert.Equal(t, "1.20.0", caddyInfo.LatestTag)
}
