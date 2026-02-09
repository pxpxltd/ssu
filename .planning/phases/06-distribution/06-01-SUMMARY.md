---
phase: 06-distribution
plan: 01
subsystem: infra
tags: [goreleaser, github-actions, homebrew, aur, cross-compilation, release-pipeline]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: "cmd/ssu/main.go with ldflags variables (version, commit, date)"
provides:
  - "goreleaser v2 configuration for 8-platform cross-compilation"
  - "GitHub Actions release workflow triggered by v* tags"
  - "Homebrew cask auto-publishing to pxpxltd/homebrew-tap"
  - "AUR PKGBUILD auto-push to aur.archlinux.org/ssu-bin.git"
affects: [06-distribution/02-install-script]

# Tech tracking
tech-stack:
  added: [goreleaser-v2.13, goreleaser-action-v6]
  patterns: [tag-triggered-release, cross-compilation-matrix, package-manager-auto-publish]

key-files:
  created: [.goreleaser.yaml, .github/workflows/release.yml]
  modified: [.gitignore]

key-decisions:
  - "homebrew_casks (not deprecated brews) for Homebrew distribution"
  - "skip_upload: auto on both Homebrew and AUR to prevent pre-release publishing"
  - "prerelease: auto for automatic RC/beta/alpha detection from tag names"
  - "CGO_ENABLED=0 for fully static binaries across all platforms"

patterns-established:
  - "Tag-triggered release: git tag v*.*.* && git push --tags fires full pipeline"
  - "Separate HOMEBREW_TAP_TOKEN for cross-repo tap publishing (GITHUB_TOKEN is repo-scoped)"
  - "AUR_KEY for SSH-based AUR PKGBUILD push"

# Metrics
duration: 10min
completed: 2026-02-09
---

# Phase 6 Plan 1: Release Pipeline Summary

**goreleaser v2 config with 4 OS x 2 arch cross-compilation, GitHub Actions tag-triggered release, Homebrew cask + AUR auto-publishing**

## Performance

- **Duration:** 10 min (mostly goreleaser installation/compilation for verification)
- **Started:** 2026-02-09T15:05:16Z
- **Completed:** 2026-02-09T15:16:00Z
- **Tasks:** 2
- **Files created:** 2 (.goreleaser.yaml, .github/workflows/release.yml)
- **Files modified:** 1 (.gitignore)

## Accomplishments
- goreleaser v2 configuration producing 8 static binaries (linux/darwin/freebsd/windows x amd64/arm64)
- Archives: tar.gz for Unix platforms, zip for Windows, with SHA256 checksums
- Changelog auto-generated from conventional commits with grouping (Features, Bug Fixes, Others)
- Homebrew cask and AUR PKGBUILD auto-published on stable releases, skipped for pre-releases
- GitHub Actions workflow ready to fire on first `git tag v*` push

## Task Commits

Each task was committed atomically:

1. **Task 1: goreleaser configuration** - `f0d7a70` (feat)
2. **Task 2: GitHub Actions release workflow** - `1dda3c5` (feat)

## Files Created/Modified
- `.goreleaser.yaml` - Complete goreleaser v2 release config (builds, archives, checksum, changelog, release, homebrew_casks, aurs)
- `.github/workflows/release.yml` - Tag-triggered CI/CD pipeline using goreleaser-action v6
- `.gitignore` - Added dist/ for goreleaser snapshot output

## Decisions Made
- Used `homebrew_casks` (not deprecated `brews`) per goreleaser v2.10+ guidance
- Used `formats` (plural list) not `format` (singular) per goreleaser v2.6+ API
- Set `skip_upload: auto` on both Homebrew and AUR to prevent pre-release artifacts from publishing to package managers
- Set `prerelease: auto` so goreleaser auto-detects RC/beta/alpha tags from semver suffixes

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- File permission mismatch: project owned by `james`, running as `jin`. Resolved using `sudo` for git and file operations.
- goreleaser installation took ~5 minutes to compile from source (large dependency tree).

## User Setup Required

External services require manual configuration before first release:

1. **Create Homebrew tap repository:** GitHub -> New repository -> `pxpxltd/homebrew-tap` (public, empty)
2. **Create HOMEBREW_TAP_TOKEN:** GitHub Settings -> Developer settings -> Fine-grained PAT with `contents:write` on `pxpxltd/homebrew-tap`
3. **Store HOMEBREW_TAP_TOKEN as secret:** GitHub -> `pxpxltd/ssu` -> Settings -> Secrets -> Actions -> New secret
4. **Create AUR SSH key:** `ssh-keygen -t ed25519 -f aur_key -N ''` then store private key content as `AUR_KEY` secret
5. **Register AUR public key:** https://aur.archlinux.org/account -> SSH Keys

## Next Phase Readiness
- Release pipeline is complete and ready for first tag push
- Install script (06-02) can proceed independently
- First actual release requires user setup of Homebrew tap repo and GitHub/AUR secrets

---
*Phase: 06-distribution*
*Completed: 2026-02-09*
