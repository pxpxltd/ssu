---
phase: quick
plan: 004
type: execute
wave: 1
depends_on: []
files_modified:
  - RELEASING.md
  - README.md
autonomous: true

must_haves:
  truths:
    - "Maintainer can follow RELEASING.md step-by-step to publish the first release"
    - "Maintainer can follow RELEASING.md for subsequent routine releases"
    - "README Installation section lists all methods with availability caveats"
  artifacts:
    - path: "RELEASING.md"
      provides: "Complete release guide with first-release setup and routine release checklist"
      min_lines: 80
    - path: "README.md"
      provides: "Updated Installation section with all methods and go install caveat"
      contains: "curl -sSL"
  key_links:
    - from: "RELEASING.md"
      to: ".goreleaser.yaml"
      via: "references goreleaser config and explains skip_upload: auto behavior"
      pattern: "goreleaser|skip_upload"
    - from: "RELEASING.md"
      to: ".github/workflows/release.yml"
      via: "references workflow and required secrets"
      pattern: "HOMEBREW_TAP_TOKEN|AUR_KEY|GITHUB_TOKEN"
    - from: "README.md"
      to: "scripts/install.sh"
      via: "curl pipe bash install command"
      pattern: "curl.*install\\.sh"
---

<objective>
Create a comprehensive release guide (RELEASING.md) and update the README Installation section to document all available install methods with proper caveats.

Purpose: The project has goreleaser config, GitHub Actions workflow, and an install script, but no documentation tying them together. A maintainer needs a clear step-by-step guide for first-time setup (external services) and routine releases. Users need to know which install methods are available and that `go install` only works after the first release is published.

Output: RELEASING.md at project root, updated Installation section in README.md
</objective>

<execution_context>
@/home/james/.claude/get-shit-done/workflows/execute-plan.md
@/home/james/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.goreleaser.yaml
@.github/workflows/release.yml
@scripts/install.sh
@Makefile
@README.md
@go.mod
@.planning/phases/06-distribution/06-VERIFICATION.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create RELEASING.md</name>
  <files>RELEASING.md</files>
  <action>
Create `RELEASING.md` in the project root with these sections:

**Prerequisites section:**
- goreleaser v2 installed locally (for snapshot testing): `go install github.com/goreleaser/goreleaser/v2@latest`
- GitHub repository push access
- List the three required GitHub Actions secrets: GITHUB_TOKEN (automatic), HOMEBREW_TAP_TOKEN (manual), AUR_KEY (manual)

**First Release Setup section (one-time):**
Step-by-step with exact commands/URLs:

1. Create Homebrew tap repository:
   - Create `pxpxltd/homebrew-tap` on GitHub (public, empty, with README)
   - Generate a fine-grained personal access token scoped to homebrew-tap with contents:write
   - Add as `HOMEBREW_TAP_TOKEN` secret in pxpxltd/ssu repo Settings > Secrets > Actions

2. AUR setup (optional, Arch Linux):
   - Generate SSH key: `ssh-keygen -t ed25519 -f ~/.ssh/aur -C "aur"`
   - Register at https://aur.archlinux.org, add public key to account
   - Add private key content as `AUR_KEY` secret in pxpxltd/ssu repo

3. Validate goreleaser config locally:
   ```
   goreleaser check
   goreleaser release --snapshot --clean
   ```
   Verify the `dist/` directory contains 8 binaries (4 OS x 2 arch), archives, and checksums.txt

4. First release candidate:
   ```
   git tag -a v1.0.0-rc1 -m "Release v1.0.0-rc1"
   git push origin v1.0.0-rc1
   ```
   - Explain that `prerelease: auto` in goreleaser marks rc/beta/alpha tags as pre-releases
   - Explain that `skip_upload: auto` means Homebrew and AUR are NOT updated for pre-releases
   - Verify: GitHub Actions > release workflow runs green, GitHub release page shows binaries

5. First stable release (after RC validation):
   ```
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
   - This triggers full distribution: binaries + Homebrew tap update + AUR update
   - Verify all three distribution channels

**Routine Release Checklist section:**
Quick checklist for subsequent releases:
1. Ensure all changes on master/main
2. Run tests: `make test`
3. Run snapshot: `goreleaser release --snapshot --clean` (optional sanity check)
4. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
5. Push tag: `git push origin vX.Y.Z`
6. Monitor: GitHub Actions > release workflow
7. Verify: GitHub release page, `brew upgrade ssu`, install script

**Version Strategy section:**
- Semantic versioning (vMAJOR.MINOR.PATCH)
- Pre-release tags: v1.0.0-rc1, v1.0.0-beta1 (auto-detected, skip package manager publishing)
- goreleaser auto-generates changelog from conventional commits (feat:, fix:, etc.)
- Commits prefixed docs:, test:, chore: are excluded from changelog

**What Gets Published section:**
Table showing what each release type triggers:
| Channel | Pre-release (rc/beta) | Stable |
|---------|----------------------|--------|
| GitHub Release | Yes (marked pre-release) | Yes |
| Binaries (8 platforms) | Yes | Yes |
| Checksums | Yes | Yes |
| Homebrew tap | No (skip_upload: auto) | Yes |
| AUR package | No (skip_upload: auto) | Yes |
| go install | Yes (any tag) | Yes |

**Troubleshooting section:**
- "goreleaser check fails": ensure version: 2 at top of .goreleaser.yaml
- "Homebrew tap not updating": check HOMEBREW_TAP_TOKEN has contents:write scope on homebrew-tap repo
- "AUR push fails": check AUR_KEY secret contains full private key including header/footer lines
- "go install can't find module": module must have at least one published tag; run `go install github.com/pxpxltd/ssu/cmd/ssu@v1.0.0` with explicit version if @latest doesn't resolve
- "Workflow doesn't trigger": tags must match `v*` pattern (e.g., v1.0.0 not 1.0.0)

Keep the tone direct and practical. No fluff. Use code blocks for all commands.
  </action>
  <verify>
Test that RELEASING.md exists, has all major sections, and references the actual config files correctly:
- grep for "First Release Setup" in RELEASING.md
- grep for "Routine Release" in RELEASING.md
- grep for "HOMEBREW_TAP_TOKEN" in RELEASING.md
- grep for "goreleaser" in RELEASING.md
- grep for "go install" in RELEASING.md
  </verify>
  <done>RELEASING.md exists with first-release setup, routine checklist, version strategy, publishing matrix, and troubleshooting. All external service setup steps have exact commands or URLs. All goreleaser/workflow references match actual config.</done>
</task>

<task type="auto">
  <name>Task 2: Update README.md Installation section</name>
  <files>README.md</files>
  <action>
Replace the current Installation section in README.md (lines 29-56 approximately) with a comprehensive section listing all installation methods. Keep the existing "From Source" and "Go Install" subsections but add the missing methods and caveats.

New Installation section structure:

### Install Script (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/pxpxltd/ssu/master/scripts/install.sh | bash
```

Works on Linux, macOS, FreeBSD, and Windows (MSYS/MinGW/Cygwin). Auto-detects OS and architecture, verifies SHA256 checksum.

To install a specific version:

```bash
VERSION=v1.0.0 curl -sSL https://raw.githubusercontent.com/pxpxltd/ssu/master/scripts/install.sh | bash
```

### Homebrew (macOS / Linux)

```bash
brew install pxpxltd/tap/ssu
```

### AUR (Arch Linux)

```bash
yay -S ssu-bin
```

### Go Install

```bash
go install github.com/pxpxltd/ssu/cmd/ssu@latest
```

Requires Go 1.21+. Installs to `$GOPATH/bin` (or `$HOME/go/bin`).

> **Note:** `go install @latest` requires at least one published release. Use an explicit version tag if `@latest` doesn't resolve: `go install github.com/pxpxltd/ssu/cmd/ssu@v1.0.0`

### From Source

```bash
git clone https://github.com/pxpxltd/ssu.git
cd ssu
make build
```

(Keep existing copy-to-PATH instructions.)

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/pxpxltd/ssu/releases). Archives available for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- FreeBSD (amd64, arm64)
- Windows (amd64, arm64)

Do NOT change any other section of the README. Only modify the Installation section (between `## Installation` and `## Build`).
  </action>
  <verify>
Verify README.md has all install methods:
- grep for "curl -sSL" in README.md
- grep for "brew install" in README.md
- grep for "yay -S" in README.md
- grep for "go install" in README.md
- grep for "GitHub Releases" in README.md
- grep for "Note.*go install.*requires" in README.md (caveat present)
- Ensure the rest of the README is unchanged (Build section, Commands section, etc. still present)
  </verify>
  <done>README Installation section lists all 6 methods (install script, Homebrew, AUR, go install, from source, pre-built binaries) with the go install caveat about needing a published release. Rest of README unchanged.</done>
</task>

</tasks>

<verification>
- RELEASING.md exists at project root with complete first-release and routine-release documentation
- README.md Installation section covers all distribution channels from .goreleaser.yaml and scripts/install.sh
- go install caveat is documented in both RELEASING.md (troubleshooting) and README.md (inline note)
- All referenced secrets (HOMEBREW_TAP_TOKEN, AUR_KEY) match .github/workflows/release.yml
- All referenced URLs and commands are accurate
</verification>

<success_criteria>
- A maintainer can follow RELEASING.md from zero to first published release without guessing
- A user reading README.md can choose the right installation method for their platform
- The go install limitation (requires published release) is clearly documented
</success_criteria>

<output>
After completion, create `.planning/quick/004-create-a-proper-release-guide-go-install/004-SUMMARY.md`
</output>
