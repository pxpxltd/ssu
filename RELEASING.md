# Releasing SSU

## Prerequisites

- **goreleaser v2** installed locally (for snapshot testing):
  ```bash
  go install github.com/goreleaser/goreleaser/v2@latest
  ```
- Push access to the `pxpxltd/ssu` GitHub repository
- Three GitHub Actions secrets configured:

| Secret | Source | Setup |
|--------|--------|-------|
| `GITHUB_TOKEN` | Automatic | Provided by GitHub Actions (no setup needed) |
| `HOMEBREW_TAP_TOKEN` | Manual | Fine-grained PAT scoped to `pxpxltd/homebrew-tap` |
| `AUR_KEY` | Manual | SSH private key registered with AUR |

## First Release Setup

Follow these steps once to configure external services before publishing the first release.

### 1. Create the Homebrew Tap Repository

1. Go to https://github.com/organizations/pxpxltd/repositories/new
2. Create a **public** repository named `homebrew-tap`
3. Initialize with a README (or leave empty)
4. Generate a fine-grained personal access token:
   - Go to https://github.com/settings/tokens?type=beta
   - Token name: `homebrew-tap-goreleaser`
   - Resource owner: `pxpxltd`
   - Repository access: **Only select repositories** -> `pxpxltd/homebrew-tap`
   - Permissions: **Contents: Read and write**
   - Generate and copy the token
5. Add the token as a secret in the SSU repository:
   - Go to https://github.com/pxpxltd/ssu/settings/secrets/actions
   - New repository secret: Name `HOMEBREW_TAP_TOKEN`, paste the token value

### 2. AUR Setup (Optional - Arch Linux)

Skip this if you don't need Arch Linux distribution.

1. Generate an SSH key pair:
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/aur -C "aur"
   ```
2. Register at https://aur.archlinux.org if you don't have an account
3. Add the public key to your AUR account:
   - Go to https://aur.archlinux.org/account (My Account)
   - Paste contents of `~/.ssh/aur.pub` into the SSH Public Key field
4. Add the private key as a secret in the SSU repository:
   - Go to https://github.com/pxpxltd/ssu/settings/secrets/actions
   - New repository secret: Name `AUR_KEY`
   - Paste the **full** contents of `~/.ssh/aur` (including `-----BEGIN` and `-----END` lines)

### 3. Validate goreleaser Config Locally

```bash
goreleaser check
goreleaser release --snapshot --clean
```

Verify the `dist/` directory contains:
- 8 binaries (4 OS x 2 arch: linux/darwin/freebsd/windows x amd64/arm64)
- Archives (`.tar.gz` for Unix, `.zip` for Windows)
- `checksums.txt`

### 4. First Release Candidate

```bash
git tag -a v1.0.0-rc1 -m "Release v1.0.0-rc1"
git push origin v1.0.0-rc1
```

What happens:
- The `v*` tag pattern triggers `.github/workflows/release.yml`
- goreleaser's `prerelease: auto` setting detects `rc1` and marks it as a pre-release
- goreleaser's `skip_upload: auto` on Homebrew and AUR means package managers are **NOT** updated for pre-releases

Verify:
1. Go to https://github.com/pxpxltd/ssu/actions -- release workflow should be running
2. Wait for green check
3. Go to https://github.com/pxpxltd/ssu/releases -- release should exist, marked "Pre-release"
4. Confirm 8 archive files + `checksums.txt` are attached

### 5. First Stable Release

After validating the release candidate:

```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

This triggers full distribution:
- GitHub Release with 8 platform archives + checksums
- Homebrew tap updated (`pxpxltd/homebrew-tap`)
- AUR package updated (`ssu-bin`)
- `go install github.com/pxpxltd/ssu/cmd/ssu@latest` becomes available

Verify all three channels:
1. GitHub release page has binaries
2. `brew install pxpxltd/tap/ssu` works (or `brew upgrade ssu` if already installed)
3. `yay -S ssu-bin` works on Arch Linux (if AUR was configured)

## Routine Release Checklist

For subsequent releases after initial setup:

1. Ensure all changes are merged to the release branch (master/main)
2. Run tests:
   ```bash
   make test
   ```
3. (Optional) Run a local snapshot to sanity-check:
   ```bash
   goreleaser release --snapshot --clean
   ```
4. Tag the release:
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   ```
5. Push the tag:
   ```bash
   git push origin vX.Y.Z
   ```
6. Monitor: https://github.com/pxpxltd/ssu/actions -- watch the release workflow
7. Verify:
   - GitHub release page has binaries and changelog
   - `brew upgrade ssu` pulls the new version
   - Install script fetches the new version:
     ```bash
     curl -sSL https://raw.githubusercontent.com/pxpxltd/ssu/master/scripts/install.sh | bash
     ```

## Version Strategy

- **Semantic versioning**: `vMAJOR.MINOR.PATCH`
- **Pre-release tags**: `v1.0.0-rc1`, `v1.0.0-beta1`, `v1.0.0-alpha1`
  - Auto-detected by goreleaser (`prerelease: auto` in `.goreleaser.yaml`)
  - Marked as pre-release on GitHub
  - Package managers (Homebrew, AUR) skip pre-releases (`skip_upload: auto`)
- **Changelog**: Auto-generated from conventional commits
  - Included: `feat:`, `fix:` prefixed commits
  - Excluded: `docs:`, `test:`, `chore:` prefixed commits (configured in `.goreleaser.yaml`)

## What Gets Published

| Channel | Pre-release (rc/beta/alpha) | Stable |
|---------|-----------------------------|--------|
| GitHub Release | Yes (marked pre-release) | Yes |
| Binaries (8 platforms) | Yes | Yes |
| Checksums | Yes | Yes |
| Homebrew tap | No (`skip_upload: auto`) | Yes |
| AUR package | No (`skip_upload: auto`) | Yes |
| `go install` | Yes (any tag) | Yes |

## Troubleshooting

**`goreleaser check` fails**
- Ensure `version: 2` is at the top of `.goreleaser.yaml`. goreleaser v2 requires this.

**Homebrew tap not updating on stable release**
- Check `HOMEBREW_TAP_TOKEN` secret has `contents:write` scope on the `pxpxltd/homebrew-tap` repository.
- Check the token hasn't expired (fine-grained tokens have expiration dates).
- Check the workflow logs for specific error messages.

**AUR push fails**
- Check `AUR_KEY` secret contains the full private key including `-----BEGIN OPENSSH PRIVATE KEY-----` and `-----END OPENSSH PRIVATE KEY-----` lines.
- Verify the corresponding public key is registered at https://aur.archlinux.org/account.

**`go install` can't find module**
- The module must have at least one published tag. Before the first release, `@latest` won't resolve.
- Use an explicit version: `go install github.com/pxpxltd/ssu/cmd/ssu@v1.0.0`

**Workflow doesn't trigger**
- Tags must match the `v*` pattern (e.g., `v1.0.0` not `1.0.0`).
- Check `.github/workflows/release.yml` is on the default branch.

**Snapshot build produces wrong number of binaries**
- Verify `.goreleaser.yaml` has all 4 OS entries under `goos` and both `amd64`/`arm64` under `goarch`.
- Check `CGO_ENABLED=0` is set (required for cross-compilation without C toolchain).
