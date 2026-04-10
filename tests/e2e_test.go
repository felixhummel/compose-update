//go:build e2e

package e2e

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/felixhummel/compose-update/internal"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	os.Exit(m.Run())
}

func TestMainFixture(t *testing.T) {
	registry := internal.NewRegistryWithTimeout(5 * time.Second)
	updateChecker := internal.NewUpdateChecker("docker-compose.yml", registry)

	result, err := updateChecker.Check(true, true, true)
	assert.NoError(t, err)

	byImage := make(map[string]internal.UpdateInfo)
	for _, r := range result {
		byImage[r.ImageName] = r
	}

	assert.Equal(t, "14.5", byImage["postgres"].CurrentTag)
	assert.Equal(t, "3.16", byImage["alpine"].CurrentTag)
	assert.Equal(t, "14", byImage["data.forgejo.org/forgejo/forgejo"].CurrentTag)
}
