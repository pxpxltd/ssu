package git

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed semantic version (major.minor.patch with optional prerelease).
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // e.g. "beta.1" (empty if none)
}

// ParseSemVer parses a version string like "v1.2.3" or "v1.2.3-beta.1".
// Returns nil if the string is not a valid semver.
func ParseSemVer(s string) *SemVer {
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return nil
	}

	// Split off prerelease suffix.
	var prerelease string
	if idx := strings.Index(s, "-"); idx >= 0 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return nil
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil || minor < 0 {
		return nil
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil || patch < 0 {
		return nil
	}

	return &SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}
}

// IncrementPatch returns a new SemVer with the patch version bumped and prerelease cleared.
func (v SemVer) IncrementPatch() SemVer {
	return SemVer{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch + 1,
	}
}

// String returns the v-prefixed version string (e.g. "v1.2.4").
func (v SemVer) String() string {
	base := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		return base + "-" + v.Prerelease
	}
	return base
}

// Less reports whether v is less than other using semver ordering.
// Prerelease versions are less than the same version without prerelease.
func (v SemVer) Less(other SemVer) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch < other.Patch
	}
	// No prerelease > prerelease (1.0.0 > 1.0.0-beta)
	if v.Prerelease == "" && other.Prerelease != "" {
		return false
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return true
	}
	return v.Prerelease < other.Prerelease
}

// LatestSemVer finds the highest semver tag from a list of tag strings.
// Non-semver tags are ignored. Returns nil if no valid semver tags found.
func LatestSemVer(tags []string) *SemVer {
	var best *SemVer
	for _, tag := range tags {
		sv := ParseSemVer(tag)
		if sv == nil {
			continue
		}
		if best == nil || best.Less(*sv) {
			best = sv
		}
	}
	return best
}
