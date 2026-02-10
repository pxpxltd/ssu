---
phase: quick
plan: 004
subsystem: distribution
tags: [documentation, release, goreleaser, homebrew, aur]
completed: 2026-02-10
duration: 2min
tech-stack:
  patterns:
    - goreleaser v2 release pipeline
    - curl-pipe-bash installer
    - conventional commit changelog
key-files:
  created:
    - RELEASING.md
  modified:
    - README.md
---

# Quick Task 004: Release Guide and Installation Docs Summary

Complete release guide (RELEASING.md) with first-release setup, routine checklist, and publishing matrix; README Installation section expanded from 2 to 6 install methods with go install caveat.

## What Was Done

### Task 1: Create RELEASING.md

Created a 181-line release guide at project root covering:
- **Prerequisites**: goreleaser v2, GitHub access, three required secrets
- **First Release Setup**: Step-by-step for Homebrew tap creation (PAT with contents:write scope), AUR SSH key registration, local goreleaser validation, RC tagging, and stable release
- **Routine Release Checklist**: 7-step quick reference for subsequent releases
- **Version Strategy**: Semantic versioning, pre-release auto-detection, changelog filtering
- **Publishing Matrix**: Table showing what each release type triggers across 6 channels
- **Troubleshooting**: 6 common issues with solutions

All references match actual config:
- `skip_upload: auto` behavior explained (matches `.goreleaser.yaml` lines 73 and 89)
- Secret names match `.github/workflows/release.yml` (GITHUB_TOKEN, HOMEBREW_TAP_TOKEN, AUR_KEY)
- Tag pattern `v*` matches workflow trigger
- `prerelease: auto` behavior documented

### Task 2: Update README Installation Section

Expanded Installation from 2 methods (From Source, Go Install) to 6 methods:
1. **Install Script** (recommended) - curl-pipe-bash with SHA256 verification
2. **Homebrew** - `brew install pxpxltd/tap/ssu`
3. **AUR** - `yay -S ssu-bin`
4. **Go Install** - with caveat about requiring a published release
5. **From Source** - kept existing content
6. **Pre-built Binaries** - link to GitHub Releases with 8-platform list

Added `go install @latest` caveat as a blockquote note explaining the published-tag requirement.

## Commits

| # | Hash | Message | Files |
|---|------|---------|-------|
| 1 | eee727f | docs(quick-004): create RELEASING.md release guide | RELEASING.md |
| 2 | f613be5 | docs(quick-004): update README Installation with all methods | README.md |

## Deviations from Plan

None - plan executed exactly as written.

## Decisions Made

No architectural decisions required. All content derived from existing config files.

## Verification

- RELEASING.md: 181 lines, all 6 sections present, all secret/config references verified
- README.md: All 6 install methods present, go install caveat present, rest of README unchanged
- Cross-references verified: RELEASING.md references .goreleaser.yaml and .github/workflows/release.yml correctly
