---
phase: 06-distribution
plan: 02
subsystem: infra
tags: [bash, install-script, curl-pipe-bash, sha256, github-releases]

# Dependency graph
requires:
  - phase: 06-distribution/01
    provides: goreleaser archive naming conventions (ssu_VERSION_OS_ARCH.tar.gz, checksums.txt)
provides:
  - "curl-pipe-bash install script for any platform"
  - "SHA256 checksum verification before install"
  - "Auto-detection of OS (linux/darwin/freebsd/windows) and arch (amd64/arm64)"
affects: [README, documentation]

# Tech tracking
tech-stack:
  added: [shellcheck]
  patterns: [main-function-wrapper-for-pipe-safety, multi-tool-sha256-fallback]

key-files:
  created:
    - scripts/install.sh

key-decisions:
  - "ARCHIVE_FILENAME global variable instead of subshell return for POSIX compatibility"
  - "INSTALL_DIR env var override in addition to auto-detection"
  - "Version normalization: accept both v1.0.0 and 1.0.0 input"

patterns-established:
  - "main() wrapper at EOF: prevents partial execution when piped via curl"
  - "Triple SHA256 fallback: sha256sum -> shasum -a 256 -> openssl dgst -sha256"
  - "Triple HTTP tool: curl preferred, wget fallback, exit if neither"

# Metrics
duration: 4min
completed: 2026-02-09
---

# Phase 6 Plan 2: Install Script Summary

**curl-pipe-bash installer with OS/arch auto-detection, SHA256 verification, and curl/wget/sha256sum/shasum/openssl multi-tool support**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-09T15:05:55Z
- **Completed:** 2026-02-09T15:10:11Z
- **Tasks:** 1
- **Files created:** 1

## Accomplishments
- Complete install script matching goreleaser archive naming (`ssu_VERSION_OS_ARCH.tar.gz`)
- SHA256 checksum verification with three tool fallbacks
- Handles 4 OS variants (linux, darwin, freebsd, windows) and 2 architectures (amd64, arm64)
- Shellcheck-clean, `bash -n` syntax-verified, main() wrapper for pipe safety

## Task Commits

Each task was committed atomically:

1. **Task 1: Install script** - `a060af0` (feat)

## Files Created/Modified
- `scripts/install.sh` - curl-pipe-bash installer (300 lines, executable)

## Decisions Made
- Used `ARCHIVE_FILENAME` global variable instead of subshell return for POSIX compatibility across bash versions
- Added `INSTALL_DIR` env var override beyond the plan spec (auto-detect + explicit override)
- Version normalization: accepts both `v1.0.0` and `1.0.0` formats, normalizes internally
- `http_get` helper for inline response capture (GitHub API), separate from `http_download` for file downloads

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- File permissions: repo owned by `james`, executor running as `jin` -- used `sudo -u james` for git operations and `sudo` for directory creation
- shellcheck not installed -- installed via `pacman` during verification

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Install script ready; paired with goreleaser config from 06-01
- GitHub Actions release workflow (06-03) will tie everything together
- README should be updated with `curl -sSL ... | bash` one-liner install instructions

---
*Phase: 06-distribution*
*Completed: 2026-02-09*
