# SSU - Smart Submodule Updater

An intelligent git submodule management CLI with smart branch detection, interactive TUI, and robust conflict handling.

[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://go.dev/)

## Features

- **TUI Interactive Selector** - Navigate with arrow keys, toggle selections with space, filter by name, sort by status
- **Root Repository Display** - Shows root repository status alongside submodules
- **Smart Branch Detection** - Automatically detects the best branch (develop -> master -> main -> remote default)
- **Interactive & Batch Modes** - Choose between interactive TUI or fully automated updates (`--auto`)
- **Push Ahead Submodules** - Push submodules with unpushed commits to their tracking branches
- **Execute Commands** - Run arbitrary commands across submodules (`ssu exec git status`)
- **Dry-Run Preview** - Preview changes before applying them
- **Backup & Rollback** - Automatic backups with one-command rollback
- **Intelligent Conflict Handling** - Automatically stash, retry, and restore on conflicts
- **Parallel Fetching** - Configurable parallel fetch with progress bar
- **Layered Configuration** - YAML config with project, global, env, and CLI flag layers
- **Shell Completion** - Bash, Zsh, Fish, and PowerShell completion scripts
- **Claude Code Integration** - Slash commands for AI-assisted submodule management

## Requirements

- **Git** 2.0 or higher
- **Go** 1.21 or higher (to build from source)

## Installation

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

This produces an `ssu` binary in the project root. Copy it to your PATH:

```bash
# User-local
cp ssu ~/.local/bin/

# System-wide
sudo cp ssu /usr/local/bin/
```

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/pxpxltd/ssu/releases). Archives available for:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- FreeBSD (amd64, arm64)
- Windows (amd64, arm64)

## Build

SSU uses a Makefile with ldflags for version injection:

```bash
make build      # Build binary with version/commit/date
make test       # Run all tests
make lint       # Run golangci-lint
make clean      # Remove built binary
```

To build manually:

```bash
go build -ldflags "-s -w -X main.version=1.0.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o ssu ./cmd/ssu
```

## Commands

SSU uses subcommands. Run `ssu` without arguments for an interactive menu.

| Command | Description |
|---------|-------------|
| `ssu status` | Show submodule status table |
| `ssu export <file>.json` | Export exact submodule branches and commits |
| `ssu import <file>.json` | Synchronize submodules from an exported stack |
| `ssu update` | Fetch and merge updates for submodules |
| `ssu push` | Push submodules with unpushed commits |
| `ssu project` | Commit submodule pointer changes in the root repo |
| `ssu exec <cmd>` | Run a command in selected submodules |
| `ssu init` | Create `.ssu.yaml` configuration interactively |
| `ssu rollback [file]` | Restore submodules from a backup |
| `ssu backup` | List or clean up backup files |
| `ssu config show` | Display merged configuration with sources |
| `ssu version` | Print version, commit, build date, and Go version |
| `ssu completion` | Generate shell completion scripts |
| `ssu claude` | Claude Code integration (install slash commands, generate snippets) |

### Global Flags

These flags work with any subcommand:

| Flag | Short | Description |
|------|-------|-------------|
| `--auto` | `-a` | Automatic mode - no prompts, for CI/CD |
| `--dry-run` | `-n` | Preview changes without modifying anything |
| `--verbose` | `-v` | Enable verbose output |
| `--jobs N` | `-j N` | Number of parallel fetch jobs (default: 8) |

### Status

View the current state of all submodules:

```bash
ssu status          # Colorized table
ssu status --json   # JSON output (for scripting)
```

The status table shows:
- **Root repository** status (displayed first as "(root)")
- Each submodule's current branch, commits behind, and status

Status indicators:
- **pending** (green) - Has updates available
- **current** (cyan) - Already up to date
- **modified** (yellow) - Has local uncommitted changes
- **ahead** (magenta) - Has unpushed commits
- **conflict** (red) - Merge conflict detected

### Share a Development Stack

Export the exact branch and commit of every initialized submodule to a file
that can be committed to the root repository:

```bash
ssu export .ssu-stack.json
```

Another developer can synchronize to that stack:

```bash
ssu import .ssu-stack.json          # Pick modules interactively (all pre-selected)
ssu import .ssu-stack.json --auto   # Import every module without prompts
ssu import .ssu-stack.json --dry-run
```

Import fetches each selected module before switching it. Dirty modules and
clean target branches with unpushed or divergent commits are never overwritten;
they are skipped and listed in the final summary. If an exported commit is not
available after fetching, SSU falls back to `origin/<branch>` and warns that the
exporting developer may have forgotten to push.

Stack files use a versioned JSON format:

```json
{
  "version": 1,
  "generated_at": "2026-06-28T12:00:00+02:00",
  "modules": [
    {
      "path": "services/api",
      "branch": "feature/example",
      "sha": "0123456789abcdef0123456789abcdef01234567"
    }
  ]
}
```

### Update

Fetch and merge updates for submodules:

```bash
ssu update              # Interactive - TUI selector to pick submodules
ssu update --auto       # Batch - update all pending submodules
ssu update --dry-run    # Preview what would be updated
```

In interactive mode, the TUI selector lets you:
- **Up/Down** or **j/k** - Navigate
- **Space** - Toggle selection
- **a** - Select all, **A** - Deselect all
- **/** - Filter by name
- **s** - Cycle sort (path / status / behind)
- **d** - Toggle detail pane (changelog)
- **Enter** - Confirm, **q** - Quit

A backup is automatically created before any updates.

### Push

Push submodules with unpushed commits:

```bash
ssu push            # Interactive - select which submodules to push
ssu push --auto     # Push all ahead submodules
```

Features:
- Detects submodules with unpushed commits
- Sets up tracking branches automatically if needed
- Skips submodules in detached HEAD state

### Project

Commit changed submodule pointers in the root repository and optionally create a semver tag. Only submodule pointer changes are staged -- other modified files in the root repo are left untouched.

```bash
ssu project                      # Interactive: select submodules, prompt for message
ssu project -m "chore: bump"     # Interactive select, use provided message
ssu project --auto               # Stage all changed submodules, auto-generate message
ssu project --auto -m "msg"      # Stage all, use provided message
ssu project --tag                # Interactive commit + tag flow
ssu project --tag --auto         # Auto commit + auto-increment patch tag
ssu project --dry-run            # Preview what would be committed
ssu project --dry-run --tag      # Preview including tag suggestion
```

When `--tag` is used, SSU finds the latest semver tag (e.g. `v1.2.3`) and suggests the next patch version (`v1.2.4`). If no semver tags exist, it suggests `v0.0.1`. In interactive mode you can accept or override the suggestion.

### Exec

Run an arbitrary command in submodules:

```bash
ssu exec git status                 # Multi-arg form
ssu exec 'git log --oneline -5'    # Quoted shell command
ssu exec --auto npm install         # All submodules, no selector
ssu exec ls -la
```

In interactive mode, all submodules are pre-selected. Deselect the ones you want to skip.

### Rollback

Restore submodules to a previous state from a backup:

```bash
ssu rollback                                # List recent backups
ssu rollback backup-20260209-103000.json    # Restore from file
ssu rollback --dry-run backup.json          # Preview restore
```

A safety backup is automatically created before restoring.

### Backup Management

```bash
ssu backup              # List available backups
ssu backup list         # Same as above
ssu backup clean --keep 5     # Keep 5 most recent
ssu backup clean --keep 7d    # Keep last 7 days
```

### Init

Create a `.ssu.yaml` config file interactively:

```bash
ssu init
```

### Shell Completion

```bash
# Bash
source <(ssu completion bash)

# Zsh
ssu completion zsh > "${fpath[1]}/_ssu"

# Fish
ssu completion fish > ~/.config/fish/completions/ssu.fish
```

## Configuration

SSU loads configuration in priority order (highest wins):

1. **CLI flags** (`--jobs`, `--verbose`, etc.)
2. **Environment variables** (`SSU_GIT_PARALLEL_JOBS`, etc.)
3. **Project config** (`.ssu.yaml` in project root)
4. **Global config** (`~/.ssu/config.yaml`)
5. **Built-in defaults**

Use `ssu config show` to see the merged configuration and which layer set each value.

### Example `.ssu.yaml`

```yaml
git:
  parallel_jobs: 8
  fail_fast: false
  skip:
    - plugins/deprecated-module
    - vendor/legacy-lib

branches:
  priority:
    - develop
    - master
    - main
  # override: staging    # Force all submodules to this branch

backup:
  enabled: true
  max_backups: 10

log:
  max_size_mb: 10
  max_backups: 5
```

### Environment Variables

All config keys can be set via environment variables with the `SSU_` prefix:

```bash
SSU_GIT_PARALLEL_JOBS=16 ssu status
SSU_BRANCHES_OVERRIDE=staging ssu update --auto
```

The legacy `PARALLEL_JOBS` variable (without prefix) is also supported for backward compatibility.

## Backups

SSU automatically creates backups before any update or rollback operation.

### Backup Location

```
~/.ssu/<project-name>/backups/backup-YYYYMMDD-HHMMSS.json
```

Where `<project-name>` is the name of your project's root directory.

### Backup Format

```json
{
  "timestamp": "2026-02-09T10:30:00Z",
  "submodules": {
    "plugins/module1": {"sha": "abc1234", "branch": "develop"},
    "plugins/module2": {"sha": "def5678", "branch": "main"}
  }
}
```

## Logging

Operations are logged to:

```
~/.ssu/<project-name>/logs/submodule-update.log
```

Log rotation is automatic (configurable via `log.max_size_mb` and `log.max_backups`).

## Troubleshooting

### Merge Conflicts

SSU automatically tries to resolve conflicts by stashing local changes and retrying. If that fails:

```bash
cd path/to/submodule
git status
# Resolve conflicts manually
git add .
git merge --continue
```

Or rollback:

```bash
ssu rollback
```

### Detached HEAD

If a submodule is in detached HEAD state, push operations will skip it. To fix:

```bash
cd path/to/submodule
git checkout -b my-branch
git push -u origin my-branch
```

### Common Issues

**"No submodules found"**
- Ensure you're in a git repository root with `.gitmodules`
- Run `git submodule init && git submodule update` if submodules aren't initialized

**Network errors during fetch**
- Check connectivity and SSH keys: `ssh -T git@github.com`
- Reduce parallel jobs: `ssu status -j 4`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `make test`
4. Run linter: `make lint`
5. Submit a pull request

## License

MIT - see [LICENSE](LICENSE) for details.
