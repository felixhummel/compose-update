package internal

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
)

type UpdateChecker struct {
	path     string
	registry *Registry
}

func NewUpdateChecker(path string, registry *Registry) *UpdateChecker {
	if registry == nil {
		registry = NewRegistryWithTimeout(5 * time.Second)
	}
	return &UpdateChecker{path: path, registry: registry}
}

func (u *UpdateChecker) Check() ([]UpdateInfo, error) {
	updateInfos, err := u.createUpdateInfos()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	for i, updateInfo := range updateInfos {
		version, err := semver.NewVersion(updateInfo.CurrentTag)
		if err != nil {
			slog.Warn(fmt.Sprintf("Skipping (invalid semver) \t Image: %s \t Path: %s", updateInfo.ImageName, updateInfo.FilePath))
			continue
		}

		wg.Add(1)
		go func(i int, updateInfo UpdateInfo, version *semver.Version) {
			defer wg.Done()
			slog.Debug("Checking image", "image", updateInfo.FullImageName)
			tags, err := u.registry.FetchImageTags(updateInfo.FullImageName)
			if err != nil {
				slog.Error(fmt.Sprintf("Skipping (failed fetching tags) \t Image: %s \t Path: %s", updateInfo.ImageName, updateInfo.FilePath))
				return
			}

			latestVersion := FindLatestVersion(version, tags, true, true, true)
			if latestVersion != "" {
				slog.Info("update/available", "image", updateInfo.FullImageName, "latest", latestVersion)
				updateInfos[i].LatestTag = latestVersion
			} else {
				slog.Info("update/current", "image", updateInfo.FullImageName)
			}
		}(i, updateInfo, version)
	}
	wg.Wait()

	return updateInfos, nil
}

func (u *UpdateChecker) createUpdateInfos() ([]UpdateInfo, error) {
	var updateInfos []UpdateInfo
	uniqueImages := make(map[string]struct{})

	file, err := os.Open(u.path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	imageNamePattern := regexp.MustCompile(`^\s*image:\s*(\S+)\s*$`)
	imgArgPattern := regexp.MustCompile(`^\s*IMG:\s*(\S+)\s*$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		var imageName string
		if matches := imageNamePattern.FindStringSubmatch(line); len(matches) > 1 {
			imageName = matches[1]
		} else if matches := imgArgPattern.FindStringSubmatch(line); len(matches) > 1 {
			imageName = matches[1]
		} else {
			continue
		}

		name, tag := u.getNameAndTag(imageName)
		imageKey := name + ":" + tag

		if _, exists := uniqueImages[imageKey]; !exists {
			uniqueImages[imageKey] = struct{}{}
			updateInfos = append(updateInfos, UpdateInfo{
				FilePath:      u.path,
				RawLine:       line,
				FullImageName: imageName,
				ImageName:     name,
				CurrentTag:    tag,
			})
		}
	}

	return updateInfos, nil
}

func (u *UpdateChecker) getNameAndTag(imageName string) (string, string) {
	parts := strings.Split(imageName, ":")
	if len(parts) < 2 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
