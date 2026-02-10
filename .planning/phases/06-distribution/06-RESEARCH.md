# Phase 6: Distribution - Research

**Researched:** 2026-02-09
**Domain:** Go binary distribution (goreleaser, GitHub Actions, Homebrew, AUR, install scripts)
**Confidence:** HIGH

## Summary

Phase 6 covers distributing SSU as pre-built static binaries across multiple platforms and package managers. The standard approach for Go CLI tools is goreleaser (v2.13+) paired with GitHub Actions, which handles cross-compilation, archive creation, checksum generation, changelog production, and package manager publishing in a single pipeline.

The key tools are goreleaser v2.13 (current stable) for build orchestration, GitHub Actions with `goreleaser/goreleaser-action@v6` for CI/CD, and goreleaser's built-in integrations for Homebrew (`homebrew_casks` section, replacing the deprecated `brews`) and AUR (`aurs` section). The install script follows the established pattern from golangci-lint: detect OS/arch via `uname`, download from GitHub releases, verify SHA256, and install to `/usr/local/bin` or `~/.local/bin`.

Critical discovery: goreleaser v2.10 deprecated the `brews` (Homebrew Formulas) section in favor of `homebrew_casks`. This is the correct approach since goreleaser distributes pre-built binaries, not source. Homebrew added Linux cask support in Feb 2025 (brew#19121), so casks now work on both macOS and Linux.

**Primary recommendation:** Use goreleaser v2.13+ with `homebrew_casks` (not deprecated `brews`), the built-in `aurs` section, and a hand-written install script modeled after golangci-lint's.

## Standard Stack

### Core

| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| goreleaser | v2.13+ | Build, package, release orchestration | De facto standard for Go binary releases; handles cross-compilation, archives, checksums, changelogs, and package manager publishing |
| goreleaser-action | v6 (v6.4.0) | GitHub Actions integration | Official action; handles goreleaser installation and execution |
| actions/checkout | v5 | Repository checkout | Standard; `fetch-depth: 0` required for changelog generation |
| actions/setup-go | v5 | Go toolchain setup | Standard; use `go-version: stable` |

### Supporting

| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| GitHub CLI (`gh`) | any | Manual release testing | Local development/debugging |
| `shellcheck` | any | Install script linting | Validate install.sh before release |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| goreleaser | Manual `GOOS=X go build` + `gh release` | Works but loses changelog, checksums, package manager integration |
| goreleaser AUR integration | Manual PKGBUILD maintenance | Goreleaser auto-generates and pushes PKGBUILD on release |
| Hand-written install script | goreleaser's godownloader (DEPRECATED) | godownloader is officially deprecated; write your own |

**Installation (CI only -- goreleaser is installed by the GitHub Action):**
```bash
# Local testing only
go install github.com/goreleaser/goreleaser/v2@latest
```

## Architecture Patterns

### Recommended File Structure
```
.goreleaser.yaml           # goreleaser configuration (top-level)
.github/
  workflows/
    release.yml            # GitHub Actions release workflow
scripts/
  install.sh               # curl-pipe-bash install script
cmd/
  ssu/
    main.go                # Entry point (already exists)
```

### Pattern 1: goreleaser Configuration Structure
**What:** Single `.goreleaser.yaml` at project root defining all build, archive, checksum, changelog, and publisher configurations.
**When to use:** Every Go project using goreleaser.
**Example:**
```yaml
# Source: goreleaser.com/customization/builds/go/
version: 2

builds:
  - main: ./cmd/ssu
    binary: ssu
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - freebsd
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - formats:
      - tar.gz
    format_overrides:
      - goos: windows
        formats:
          - zip
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    files:
      - LICENSE
      - README.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

release:
  github:
    owner: pxpxltd
    name: ssu
  prerelease: auto
  name_template: "v{{.Version}}"

homebrew_casks:
  - name: ssu
    binaries:
      - ssu
    repository:
      owner: pxpxltd
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://github.com/pxpxltd/ssu"
    description: "Smart Submodule Updater - intelligent git submodule management"
    skip_upload: auto

aurs:
  - name: ssu-bin
    homepage: "https://github.com/pxpxltd/ssu"
    description: "Smart Submodule Updater - intelligent git submodule management"
    maintainers:
      - "pxpxltd"
    license: "MIT"
    private_key: "{{ .Env.AUR_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/ssu-bin.git"
    depends:
      - git
    package: |-
      install -Dm755 "./ssu" "${pkgdir}/usr/bin/ssu"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/ssu/LICENSE"
    skip_upload: auto
    commit_msg_template: "Update to {{ .Tag }}"
```

### Pattern 2: GitHub Actions Release Workflow
**What:** Tag-triggered workflow that runs goreleaser to create GitHub releases with all artifacts.
**When to use:** Every release.
**Example:**
```yaml
# Source: goreleaser.com/ci/actions/
name: release
on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          AUR_KEY: ${{ secrets.AUR_KEY }}
```

### Pattern 3: Install Script Structure
**What:** Bash script for `curl -sSL <url> | bash` installation.
**When to use:** Users who want a quick install without package managers.
**Key functions (modeled after golangci-lint install.sh):**
```bash
#!/usr/bin/env bash
set -euo pipefail

# Wrap everything in a function to prevent partial execution
main() {
    # 1. Detect OS and architecture
    local os arch
    os="$(detect_os)"
    arch="$(detect_arch)"

    # 2. Determine install directory
    local install_dir
    install_dir="$(detect_install_dir)"

    # 3. Resolve latest version from GitHub API
    local version
    version="${VERSION:-$(get_latest_version)}"

    # 4. Download binary and checksum
    local tmpdir
    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT
    download "$tmpdir" "$os" "$arch" "$version"

    # 5. Verify checksum
    verify_checksum "$tmpdir" "$os" "$arch"

    # 6. Install binary
    install_binary "$tmpdir" "$install_dir" "$os" "$arch"
}

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)   echo "linux" ;;
        darwin)  echo "darwin" ;;
        freebsd) echo "freebsd" ;;
        msys*|mingw*|cygwin*) echo "windows" ;;
        *) echo "Error: unsupported OS: $os" >&2; exit 1 ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "Error: unsupported architecture: $arch" >&2; exit 1 ;;
    esac
}

detect_install_dir() {
    if [ -w /usr/local/bin ]; then
        echo "/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        echo "$HOME/.local/bin"
    else
        mkdir -p "$HOME/.local/bin"
        echo "$HOME/.local/bin"
    fi
}

main "$@"
```

### Anti-Patterns to Avoid
- **Using deprecated `brews` section:** goreleaser v2.10 deprecated it; use `homebrew_casks` instead. The `brews` section will be removed in goreleaser v3.
- **Using deprecated `format` (singular):** Use `formats` (plural, as list) in archives and format_overrides.
- **Using `archives.builds`:** Use `archives.ids` instead (deprecated in v2).
- **Hardcoding version in install script:** Always detect from GitHub API or allow `VERSION` env override.
- **Missing `fetch-depth: 0` in checkout:** goreleaser needs full git history for changelog generation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-compilation matrix | Custom Makefile with GOOS/GOARCH loops | goreleaser `builds` section | Handles ldflags, env vars, binary naming, exclusion lists |
| Archive creation | `tar czf` / `zip` commands | goreleaser `archives` section | Handles per-OS format overrides, file inclusion, naming templates |
| Checksum generation | `sha256sum` in Makefile | goreleaser `checksum` section | Automatic, covers all artifacts, consistent naming |
| Changelog from commits | `git log --oneline` parsing | goreleaser `changelog` section | Groups by type, filters noise, includes commit links |
| Homebrew formula/cask | Hand-written Ruby formula | goreleaser `homebrew_casks` section | Auto-updates on release, handles SHA256, versioning |
| AUR PKGBUILD | Hand-written PKGBUILD | goreleaser `aurs` section | Auto-generates and pushes PKGBUILD with correct checksums |
| GitHub release creation | `gh release create` in scripts | goreleaser `release` section | Pre-release auto-detection, artifact upload, release notes |

**Key insight:** goreleaser replaces an entire release pipeline (Makefile + shell scripts + manual steps) with a single YAML config. The only thing to hand-write is the install script, since goreleaser's godownloader is deprecated.

## Common Pitfalls

### Pitfall 1: Missing `fetch-depth: 0` in GitHub Actions Checkout
**What goes wrong:** goreleaser generates empty or incorrect changelogs, or fails with "no previous tag found".
**Why it happens:** `actions/checkout` defaults to `fetch-depth: 1` (shallow clone), so goreleaser cannot see tag history.
**How to avoid:** Always set `fetch-depth: 0` in the checkout step.
**Warning signs:** Empty changelog in GitHub release, goreleaser warnings about git history.

### Pitfall 2: Using GITHUB_TOKEN for Cross-Repo Homebrew Tap Push
**What goes wrong:** goreleaser fails to push the cask to `pxpxltd/homebrew-tap` with a 403 error.
**Why it happens:** `GITHUB_TOKEN` is scoped to the current repository only; it cannot push to other repos.
**How to avoid:** Create a Personal Access Token (PAT) or fine-grained token with `contents:write` on the tap repo. Store as `HOMEBREW_TAP_TOKEN` secret. Reference in goreleaser config as `{{ .Env.HOMEBREW_TAP_TOKEN }}`.
**Warning signs:** 403 Forbidden errors during the Homebrew cask publishing step.

### Pitfall 3: AUR SSH Key Must Be Unencrypted
**What goes wrong:** goreleaser fails to push to AUR with SSH authentication errors.
**Why it happens:** goreleaser does not support passphrase-protected SSH keys.
**How to avoid:** Generate a dedicated, unencrypted SSH key for AUR: `ssh-keygen -t ed25519 -f aur_key -N ""`. Store the private key as `AUR_KEY` secret. Register the public key on the AUR website.
**Warning signs:** SSH authentication failures during AUR publishing step.

### Pitfall 4: Pre-release Tags Not Detected
**What goes wrong:** RC/beta releases get published as latest releases, or get pushed to Homebrew/AUR.
**Why it happens:** `prerelease: auto` and `skip_upload: auto` depend on goreleaser recognizing the tag pattern.
**How to avoid:** Use standard semver pre-release suffixes: `v1.0.0-rc.1`, `v1.0.0-beta.1`, `v1.0.0-alpha.1`. goreleaser detects `-rc`, `-beta`, `-alpha` indicators.
**Warning signs:** Pre-release marked as "Latest" on GitHub, cask/PKGBUILD updated for pre-release.

### Pitfall 5: ldflags Variable Path Mismatch
**What goes wrong:** Version shows "dev" even in goreleaser-built binaries.
**Why it happens:** goreleaser's default ldflags use `-X main.version={{.Version}}` but the variables might be in a different package.
**How to avoid:** SSU already uses `main.version`, `main.commit`, `main.date` in `cmd/ssu/main.go` -- these match goreleaser's defaults exactly. Verify the ldflags in `.goreleaser.yaml` reference `main.version`, `main.commit`, `main.date`.
**Warning signs:** `ssu version` showing "dev" in released binary.

### Pitfall 6: Install Script Partial Execution on Network Error
**What goes wrong:** If piped via `curl | bash` and the download is interrupted, bash may execute a partial script.
**Why it happens:** Bash executes line-by-line as it receives input.
**How to avoid:** Wrap the entire script body in a `main()` function called at the very end. The function call at the bottom ensures nothing executes until the full script is downloaded.
**Warning signs:** Partial error messages from install, corrupted installation.

### Pitfall 7: Homebrew Cask on Linux Requires --cask Flag
**What goes wrong:** `brew install pxpxltd/tap/ssu` may not find the cask on Linux.
**Why it happens:** Linux cask support (Homebrew/brew#19121, merged Feb 2025) requires the `--cask` flag for discovery; casks are not visible via the API on Linux.
**How to avoid:** Document the install command as `brew install --cask pxpxltd/tap/ssu` for Linux users, or `brew install pxpxltd/tap/ssu` for macOS. This is a Homebrew limitation, not a goreleaser issue.
**Warning signs:** "No available formula" error on Linux.

## Code Examples

### Complete .goreleaser.yaml for SSU
```yaml
# Source: goreleaser.com official docs, verified 2026-02-09
version: 2

builds:
  - main: ./cmd/ssu
    binary: ssu
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - freebsd
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - formats:
      - tar.gz
    format_overrides:
      - goos: windows
        formats:
          - zip
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}
    files:
      - LICENSE
      - README.md

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

release:
  github:
    owner: pxpxltd
    name: ssu
  prerelease: auto
  name_template: "v{{.Version}}"

homebrew_casks:
  - name: ssu
    binaries:
      - ssu
    repository:
      owner: pxpxltd
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://github.com/pxpxltd/ssu"
    description: "Smart Submodule Updater - intelligent git submodule management"
    skip_upload: auto

aurs:
  - name: ssu-bin
    homepage: "https://github.com/pxpxltd/ssu"
    description: "Smart Submodule Updater - intelligent git submodule management"
    maintainers:
      - "pxpxltd"
    license: "MIT"
    private_key: "{{ .Env.AUR_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/ssu-bin.git"
    depends:
      - git
    package: |-
      install -Dm755 "./ssu" "${pkgdir}/usr/bin/ssu"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/ssu/LICENSE"
    skip_upload: auto
    commit_msg_template: "Update to {{ .Tag }}"
```

### Complete GitHub Actions Release Workflow
```yaml
# .github/workflows/release.yml
# Source: goreleaser.com/ci/actions/, verified 2026-02-09
name: release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          AUR_KEY: ${{ secrets.AUR_KEY }}
```

### Install Script Key Functions
```bash
# Source: golangci-lint install.sh pattern, adapted for SSU

OWNER="pxpxltd"
REPO="ssu"
GITHUB_DOWNLOAD="https://github.com/${OWNER}/${REPO}/releases/download"

get_latest_version() {
    # Query GitHub API for latest release tag
    local url="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        curl -sSL "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        echo "Error: curl or wget required" >&2
        exit 1
    fi
}

download() {
    local tmpdir="$1" os="$2" arch="$3" version="$4"
    local archive_ext="tar.gz"
    [ "$os" = "windows" ] && archive_ext="zip"

    local filename="${REPO}_${version#v}_${os}_${arch}.${archive_ext}"
    local url="${GITHUB_DOWNLOAD}/${version}/${filename}"
    local checksum_url="${GITHUB_DOWNLOAD}/${version}/checksums.txt"

    http_download "${tmpdir}/${filename}" "$url"
    http_download "${tmpdir}/checksums.txt" "$checksum_url"
}

verify_checksum() {
    local tmpdir="$1" os="$2" arch="$3"
    local archive_ext="tar.gz"
    [ "$os" = "windows" ] && archive_ext="zip"

    local filename="${REPO}_*_${os}_${arch}.${archive_ext}"
    local expected actual

    expected=$(grep "$filename" "${tmpdir}/checksums.txt" | awk '{print $1}')
    actual=$(compute_sha256 "${tmpdir}/"${filename})

    if [ "$expected" != "$actual" ]; then
        echo "Error: checksum mismatch" >&2
        echo "  expected: $expected" >&2
        echo "  actual:   $actual" >&2
        exit 1
    fi
}

compute_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$1" | awk '{print $NF}'
    else
        echo "Error: no SHA256 tool found" >&2
        exit 1
    fi
}

http_download() {
    local dest="$1" url="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -sSL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$dest" "$url"
    else
        echo "Error: curl or wget required" >&2
        exit 1
    fi
}
```

### Testing goreleaser Locally
```bash
# Validate configuration
goreleaser check

# Test build without releasing (snapshot mode)
goreleaser release --snapshot --clean

# Verify built artifacts
ls dist/
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `brews` (Homebrew Formulas) | `homebrew_casks` | goreleaser v2.10 (2025) | Must use `homebrew_casks` for new projects; `brews` deprecated, removal planned for v3 |
| `archives.format` (singular string) | `archives.formats` (list) | goreleaser v2.6 (2024) | Use `formats: [tar.gz]` not `format: tar.gz` |
| `format_overrides.format` (singular) | `format_overrides.formats` (list) | goreleaser v2.6 (2024) | Same plural change in overrides |
| `archives.builds` | `archives.ids` | goreleaser v2.0 (2024) | Use `ids` to filter which builds go into archives |
| `snapshot.name_template` | `snapshot.version_template` | goreleaser v2.0 (2024) | Property rename |
| godownloader (install script generator) | Hand-written install script | godownloader deprecated (2021) | Must write install script manually |
| Homebrew Casks macOS-only | Casks on Linux | Homebrew/brew#19121 (Feb 2025) | Casks now work on Linux, enabling goreleaser `homebrew_casks` for cross-platform |

**Deprecated/outdated:**
- `brews` section: Deprecated in goreleaser v2.10, use `homebrew_casks` instead
- `godownloader`: Officially deprecated, no replacement -- write install scripts manually
- `homebrew_casks.binary` (singular): Use `binaries` (plural, as list)
- `homebrew_casks.manpage` (singular): Use `manpages` (plural)

## Open Questions

1. **AUR as Submodule vs goreleaser Auto-Push**
   - What we know: The user decided "AUR: separate AUR repo, added as a submodule in this repo." goreleaser's `aurs` section auto-pushes the PKGBUILD to `ssh://aur@aur.archlinux.org/ssu-bin.git` -- it does NOT push to a local submodule.
   - What's unclear: Whether the user wants a local git submodule tracking the AUR repo for development purposes (e.g., to edit/review PKGBUILD locally), OR if goreleaser's auto-push is sufficient.
   - Recommendation: Use goreleaser's `aurs` section for automatic PKGBUILD publishing. The "submodule" can be added separately to track the AUR repo locally for review, but goreleaser handles the actual publishing. The submodule would be read-only for development reference.

2. **Homebrew Tap Repository Setup**
   - What we know: Needs `pxpxltd/homebrew-tap` repo on GitHub with a PAT stored as `HOMEBREW_TAP_TOKEN`.
   - What's unclear: Whether this repo already exists or needs creation. Whether a fine-grained PAT or classic PAT is preferred.
   - Recommendation: Plan should include a prerequisite step for creating the tap repo and configuring the secret. Fine-grained PAT with `contents:write` on the tap repo is more secure than a classic PAT.

3. **Linux Homebrew Cask Discovery Limitation**
   - What we know: Homebrew cask support on Linux (merged Feb 2025) requires `--cask` flag for discovery. Without it, `brew install pxpxltd/tap/ssu` may not find the cask on Linux.
   - What's unclear: Whether this limitation has been resolved in more recent Homebrew releases.
   - Recommendation: Document both install commands in README. On macOS: `brew install pxpxltd/tap/ssu`. On Linux: `brew install --cask pxpxltd/tap/ssu`. Test during implementation.

## Sources

### Primary (HIGH confidence)
- [goreleaser.com/customization/builds/go/](https://goreleaser.com/customization/builds/go/) - Go build configuration with CGO_ENABLED, ldflags, goos/goarch
- [goreleaser.com/customization/archive/](https://goreleaser.com/customization/archive/) - Archive formats, naming templates, format overrides
- [goreleaser.com/customization/checksum/](https://goreleaser.com/customization/checksum/) - SHA256 checksum file generation
- [goreleaser.com/customization/changelog/](https://goreleaser.com/customization/changelog/) - Changelog auto-generation, grouping, filtering
- [goreleaser.com/customization/release/](https://goreleaser.com/customization/release/) - GitHub release settings, `prerelease: auto`
- [goreleaser.com/customization/homebrew_casks/](https://goreleaser.com/customization/homebrew_casks/) - Homebrew Cask auto-publishing (replaces deprecated `brews`)
- [goreleaser.com/customization/aur/](https://goreleaser.com/customization/aur/) - AUR PKGBUILD auto-generation and publishing
- [goreleaser.com/ci/actions/](https://goreleaser.com/ci/actions/) - GitHub Actions workflow, `goreleaser-action@v6`
- [goreleaser.com/deprecations/](https://goreleaser.com/deprecations/) - Full deprecation list for v2
- [goreleaser.com/install/](https://goreleaser.com/install/) - Current version confirmed as v2.13.3

### Secondary (MEDIUM confidence)
- [goreleaser.com/blog/goreleaser-v2.10/](https://goreleaser.com/blog/goreleaser-v2.10/) - `brews` to `homebrew_casks` migration rationale
- [github.com/Homebrew/brew/pull/19121](https://github.com/Homebrew/brew/pull/19121) - Linux cask support (merged Feb 2025)
- [github.com/goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action) - Action v6.4.0, supported inputs
- [golangci-lint install.sh](https://github.com/golangci/golangci-lint/blob/master/install.sh) - Reference install script pattern
- [wiki.archlinux.org/title/Go_package_guidelines](https://wiki.archlinux.org/title/Go_package_guidelines) - AUR naming conventions, build flags
- [goreleaser discussion #5563](https://github.com/orgs/goreleaser/discussions/5563) - Casks vs Formulas rationale

### Tertiary (LOW confidence)
- Install script best practices - compiled from multiple blog posts and guides; core patterns are stable but specific details should be validated during implementation

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - goreleaser is the undisputed standard for Go binary distribution; all configuration verified against official docs
- Architecture: HIGH - goreleaser config structure, GitHub Actions workflow, and install script patterns all verified against official sources and real-world examples
- Pitfalls: HIGH - cross-repo token requirements, SSH key constraints, fetch-depth, and deprecation notices all verified from official docs
- Homebrew casks on Linux: MEDIUM - feature is merged (Feb 2025) but relatively new; `--cask` flag requirement on Linux may have been updated since

**Research date:** 2026-02-09
**Valid until:** 2026-03-09 (goreleaser is mature and stable; 30-day validity)
