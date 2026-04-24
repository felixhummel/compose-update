package internal

import (
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func FindLatestVersion(current *semver.Version, tags []string, level UpdateLevel) string {
	major := level.IncludeMajor()
	minor := level.IncludeMinor()
	patch := level.IncludePatch()

	type VersionTag struct {
		Version *semver.Version
		Tag     string
	}
	var versionTags []VersionTag

	// Collect valid semantic versions
	for _, tag := range tags {
		// Filter out invalid semantic versions
		if !isValidSemver(tag) {
			continue
		}

		// Normalize X.Y to X.Y.0 for comparison
		normalized := normalizeSemver(tag)

		// Attempt to parse the tag as a semantic version to compare it later easily
		v, err := semver.NewVersion(normalized)
		if err != nil {
			continue
		}
		versionTags = append(versionTags, VersionTag{Version: v, Tag: tag})
	}

	if len(versionTags) == 0 {
		return ""
	}

	// Sort versions in descending order
	// This is necessary to find the latest version
	sort.Slice(versionTags, func(i, j int) bool {
		return versionTags[i].Version.GreaterThan(versionTags[j].Version)
	})

	for _, vt := range versionTags {
		v := vt.Version
		tag := vt.Tag

		// Skip versions not newer than current
		if v.LessThanEqual(current) {
			continue
		}
		if v.Prerelease() != current.Prerelease() {
			continue
		}

		accept := false
		if major && v.Major() > current.Major() {
			accept = true
		} else if minor && isEqualMajor(v, current) && v.Minor() > current.Minor() {
			accept = true
		} else if patch && isEqualMajor(v, current) && isEqualMinor(v, current) && v.Patch() > current.Patch() {
			accept = true
		}

		if accept {
			return tag
		}
	}

	return ""
}

// SortTagsBySemver returns the subset of tags that are valid semver, sorted descending.
func SortTagsBySemver(tags []string) []string {
	type vt struct {
		v   *semver.Version
		tag string
	}
	var parsed []vt
	for _, tag := range tags {
		if !isValidSemver(tag) {
			continue
		}
		v, err := semver.NewVersion(normalizeSemver(tag))
		if err != nil {
			continue
		}
		parsed = append(parsed, vt{v, tag})
	}
	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].v.GreaterThan(parsed[j].v)
	})
	result := make([]string, len(parsed))
	for i, p := range parsed {
		result[i] = p.tag
	}
	return result
}

func isValidSemver(tag string) bool {
	tag = strings.TrimPrefix(tag, "v")
	// Accept both X.Y and X.Y.Z formats (X.Y is normalized to X.Y.0)
	regex := regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)(?:\.(?P<patch>0|[1-9]\d*))?(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	matches := regex.FindStringSubmatch(tag)
	return len(matches) > 0
}

// normalizeSemver converts X.Y to X.Y.0 for comparison purposes.
// Preserves prerelease suffix (e.g., 14.22-trixie -> 14.22.0-trixie).
func normalizeSemver(tag string) string {
	tag = strings.TrimPrefix(tag, "v")
	dotParts := strings.Split(tag, ".")
	if len(dotParts) == 2 {
		// X.Y or X.Y-prerelease -> X.Y.0 or X.Y.0-prerelease
		return dotParts[0] + "." + dotParts[1] + ".0"
	}
	return tag
}

func isEqualMajor(current, tag *semver.Version) bool {
	return current.Major() == tag.Major()
}

func isEqualMinor(current, tag *semver.Version) bool {
	return current.Minor() == tag.Minor()
}
