# Requirements: SSU Go Rewrite

**Defined:** 2026-02-09
**Core Value:** Safely update and push git submodules with zero data loss -- smart branch detection, automatic backups, conflict resolution.

## v1 Requirements

### Core Git Operations

- [ ] **CORE-01**: Parallel fetch of all submodule remotes with configurable concurrency
- [ ] **CORE-02**: Smart branch detection (develop > master > main > remote HEAD > fallback)
- [ ] **CORE-03**: Feature branch preservation (submodules on feature branches stay on them)
- [ ] **CORE-04**: Status detection for each submodule (pending, current, modified, ahead, conflict, missing, skipped)
- [ ] **CORE-05**: Changelog preview showing incoming commits per submodule
- [ ] **CORE-06**: Configurable fetch timeout per submodule (kill hung fetch)
- [ ] **CORE-07**: Multiple remote support (don't hardcode "origin")
- [ ] **CORE-08**: Root repository status display (display-only, excluded from operations)

### Smart Conflict Handling

- [ ] **CONF-01**: Detect local uncommitted changes vs local commits vs upstream changes
- [ ] **CONF-02**: Per-submodule choice for dirty submodules: stash+merge, skip, or force
- [ ] **CONF-03**: Auto pull+merge for local commits vs remote commits, alert on conflict
- [ ] **CONF-04**: Merge conflict reporting with actionable resolution hints

### CLI Framework

- [ ] **CLI-01**: Subcommand-based CLI: `ssu status`, `ssu update`, `ssu push`, `ssu rollback`
- [ ] **CLI-02**: Shell completions (bash/zsh/fish) via cobra
- [ ] **CLI-03**: Version command with build info (version, commit, date)
- [ ] **CLI-04**: Backwards compatibility hints (old `--status` -> suggests `ssu status`)
- [ ] **CLI-05**: Meaningful exit codes: 0=success, 1=error, 2=conflict
- [ ] **CLI-06**: `--json` output on `ssu status` for scripting/CI
- [ ] **CLI-07**: `ssu exec <command>` to run command across selected submodules
- [ ] **CLI-08**: `ssu init` wizard for first-time config setup

### Interactive TUI

- [ ] **TUI-01**: Multi-select with checkboxes (arrow/vim keys, space toggle, all/none, confirm/quit)
- [ ] **TUI-02**: Colorized status table with root repo display
- [ ] **TUI-03**: Auto/batch mode (`--auto` or `--all`) for CI/CD, no prompts
- [ ] **TUI-04**: Dry-run preview showing what would change
- [ ] **TUI-05**: Progress bar during parallel fetch (per-submodule status)
- [ ] **TUI-06**: NO_COLOR support and TTY detection
- [ ] **TUI-07**: Graceful Ctrl+C handling (clean terminal state, show partial results)

### Configuration

- [ ] **CFG-01**: YAML config file at `~/.ssu/config.yaml`
- [ ] **CFG-02**: Per-project config override via `.ssu.yaml` in project root
- [ ] **CFG-03**: Configurable skip list, branch priority, parallel jobs
- [ ] **CFG-04**: Config layering: defaults < global < project < env vars < CLI flags

### Safety & Recovery

- [ ] **SAFE-01**: JSON backup before modifications with atomic writes (write temp, rename)
- [ ] **SAFE-02**: Rollback from backup file (compatible with bash-era backups)
- [ ] **SAFE-03**: Fail-fast mode (exit on first error)
- [ ] **SAFE-04**: Backup management: `ssu backup list`, `ssu backup clean --keep N`
- [ ] **SAFE-05**: Log rotation by size/date with configurable limits
- [ ] **SAFE-06**: Logging to `~/.ssu/<project>/logs/`

### Push Operations

- [ ] **PUSH-01**: Detect ahead submodules (unpushed commits)
- [ ] **PUSH-02**: Interactive selection for push
- [ ] **PUSH-03**: Automatic tracking branch setup (`git push -u origin <branch>`)
- [ ] **PUSH-04**: Detached HEAD detection and skip with warning

### Distribution

- [ ] **DIST-01**: `go install` support via `cmd/ssu/main.go`
- [ ] **DIST-02**: goreleaser with cross-platform builds (linux/darwin, amd64/arm64)
- [ ] **DIST-03**: Homebrew tap
- [ ] **DIST-04**: AUR package
- [ ] **DIST-05**: Install script (curl-pipe-bash style)
- [ ] **DIST-06**: Static binaries (CGO_ENABLED=0)

## v2 Requirements

### Enhanced Features

- **EXEC-01**: `ssu exec` with submodule grouping (run command on specific groups)
- **DASH-01**: Rich TUI dashboard with diff stats, commit authors, update age
- **DIFF-01**: `ssu diff` showing changes since last update/backup
- **NOTIF-01**: Webhook/notification on batch completion
- **GROUP-01**: Submodule grouping in config (plugins, vendor, internal)

## Out of Scope

| Feature | Reason |
|---------|--------|
| GUI/web interface | CLI tool -- invest in TUI instead |
| Non-git VCS | Git submodules are git-only |
| Submodule creation/removal | `git submodule add/deinit` works fine |
| Windows native | WSL exists. Target audience has Unix env. |
| Root repo modification | Display-only for safety |
| Interactive conflict resolution UI | Use vimdiff/meld. Not building a merge tool. |
| Plugin/extension system | Tool does one thing well |
| Recursive submodules | Rare, exponential complexity |
| go-git library | Shell out to git via os/exec for 100% compatibility |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| CORE-01 | Phase 3 | Pending |
| CORE-02 | Phase 2 | Pending |
| CORE-03 | Phase 2 | Pending |
| CORE-04 | Phase 3 | Pending |
| CORE-05 | Phase 3 | Pending |
| CORE-06 | Phase 2 | Pending |
| CORE-07 | Phase 2 | Pending |
| CORE-08 | Phase 3 | Pending |
| CONF-01 | Phase 3 | Pending |
| CONF-02 | Phase 3 | Pending |
| CONF-03 | Phase 3 | Pending |
| CONF-04 | Phase 3 | Pending |
| CLI-01 | Phase 1 | Pending |
| CLI-02 | Phase 1 | Pending |
| CLI-03 | Phase 1 | Pending |
| CLI-04 | Phase 1 | Pending |
| CLI-05 | Phase 1 | Pending |
| CLI-06 | Phase 5 | Pending |
| CLI-07 | Phase 5 | Pending |
| CLI-08 | Phase 5 | Pending |
| TUI-01 | Phase 5 | Pending |
| TUI-02 | Phase 5 | Pending |
| TUI-03 | Phase 5 | Pending |
| TUI-04 | Phase 5 | Pending |
| TUI-05 | Phase 5 | Pending |
| TUI-06 | Phase 1 | Pending |
| TUI-07 | Phase 5 | Pending |
| CFG-01 | Phase 4 | Pending |
| CFG-02 | Phase 4 | Pending |
| CFG-03 | Phase 4 | Pending |
| CFG-04 | Phase 4 | Pending |
| SAFE-01 | Phase 4 | Pending |
| SAFE-02 | Phase 4 | Pending |
| SAFE-03 | Phase 4 | Pending |
| SAFE-04 | Phase 4 | Pending |
| SAFE-05 | Phase 4 | Pending |
| SAFE-06 | Phase 4 | Pending |
| PUSH-01 | Phase 3 | Pending |
| PUSH-02 | Phase 5 | Pending |
| PUSH-03 | Phase 2 | Pending |
| PUSH-04 | Phase 3 | Pending |
| DIST-01 | Phase 6 | Pending |
| DIST-02 | Phase 6 | Pending |
| DIST-03 | Phase 6 | Pending |
| DIST-04 | Phase 6 | Pending |
| DIST-05 | Phase 6 | Pending |
| DIST-06 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 47 total
- Mapped to phases: 47
- Unmapped: 0

---
*Requirements defined: 2026-02-09*
*Last updated: 2026-02-09 after roadmap creation*
