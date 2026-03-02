package internal

import (
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type UpdateInfo struct {
	FilePath      string
	RawLine       string
	ImageName     string
	FullImageName string
	CurrentTag    string
	LatestTag     string
}

func (u *UpdateInfo) HasNewVersion() bool {
	if u.CurrentTag == "" || u.LatestTag == "" {
		return false
	}

	current, err := semver.NewVersion(u.CurrentTag)
	if err != nil {
		return false
	}

	latest, err := semver.NewVersion(u.LatestTag)
	if err != nil {
		return false
	}

	return latest.GreaterThan(current)
}

func (u *UpdateInfo) Update() error {
	input, err := os.ReadFile(u.FilePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.Contains(line, u.RawLine) {
			lines[i] = strings.Replace(line, u.CurrentTag, u.LatestTag, 1)
		}
	}

	return os.WriteFile(u.FilePath, []byte(strings.Join(lines, "\n")), 0644)
}
