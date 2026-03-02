package internal

import (
	"os"
	"testing"
)

func TestHasNewVersion(t *testing.T) {
	tests := []struct {
		name       string
		currentTag string
		latestTag  string
		expected   bool
	}{
		{"No new version", "1.0.0", "1.0.0", false},
		{"New patch version", "1.0.0", "1.0.1", true},
		{"New minor version", "1.0.0", "1.1.0", true},
		{"New major version", "1.0.0", "2.0.0", true},
		{"With suffix", "1.0.0-rc1", "1.0.0-rc2", true},
		{"With suffix, no new version", "1.0.0-rc1", "1.0.0-rc1", false},
		{"Invalid current tag", "", "1.0.0", false},
		{"Invalid latest tag", "1.0.0", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UpdateInfo{
				CurrentTag: tt.currentTag,
				LatestTag:  tt.latestTag,
			}
			if got := u.HasNewVersion(); got != tt.expected {
				t.Errorf("HasNewVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := []byte("image: myapp:1.0.0")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	u := &UpdateInfo{
		FilePath:   tmpFile.Name(),
		RawLine:    "image: myapp:1.0.0",
		CurrentTag: "1.0.0",
		LatestTag:  "1.1.0",
	}

	if err := u.Update(); err != nil {
		t.Errorf("Update() error = %v", err)
	}

	updatedContent, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "image: myapp:1.1.0"
	if string(updatedContent) != expected {
		t.Errorf("Update() = %v, want %v", string(updatedContent), expected)
	}

	// No backup file should be created
	if _, err := os.Stat(tmpFile.Name() + ".ccu"); !os.IsNotExist(err) {
		t.Errorf("Update() should not create a backup file")
	}
}
