package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfigDir(t *testing.T) {
	// Test with explicit XDG_CONFIG_HOME
	os.Setenv("XDG_CONFIG_HOME", "/custom/config")
	assert.Equal(t, "/custom/config/compose-update", filepath.Join(getConfigDir(), "compose-update"))
	os.Unsetenv("XDG_CONFIG_HOME")

	// Test with default (will use home dir)
	home, err := os.UserHomeDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".config/compose-update"), filepath.Join(getConfigDir(), "compose-update"))
}