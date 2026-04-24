package internal

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func CheckImage(image string, registry *Registry) ([]UpdateInfo, error) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) != 2 || parts[1] == "" {
		return nil, fmt.Errorf("image must be in name:tag format, got %q", image)
	}
	name, tag := parts[0], parts[1]

	current, err := semver.NewVersion(tag)
	if err != nil {
		return nil, fmt.Errorf("invalid semver tag %q: %w", tag, err)
	}

	info := UpdateInfo{
		FullImageName: image,
		ImageName:     name,
		CurrentTag:    tag,
	}

	slog.Debug("Checking image", "image", image)
	tags, err := registry.FetchImageTags(image)
	if err != nil {
		return nil, fmt.Errorf("failed fetching tags for %s: %w", name, err)
	}

	latestVersion := FindLatestVersion(current, tags, true, true, true)
	if latestVersion != "" {
		slog.Info("update/available", "image", image, "latest", latestVersion)
		info.LatestTag = latestVersion
	} else {
		slog.Info("update/current", "image", image)
	}

	return []UpdateInfo{info}, nil
}
