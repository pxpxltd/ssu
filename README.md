# SSU - Smart Submodule Updater

An intelligent git submodule updater with smart branch detection, interactive workflows, and robust conflict handling.

[![Version](https://img.shields.io/badge/version-1.0.6-blue.svg)](https://github.com/yourusername/ssu)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Bash](https://img.shields.io/badge/bash-3.2%2B-orange.svg)](https://www.gnu.org/software/bash/)

## Features

- **Smart Branch Detection** - Automatically detects the best branch to use (develop → master → main → remote default)
- **Interactive & Batch Modes** - Choose between interactive prompts or fully automated updates
- **Push Ahead Submodules** - Easily push submodules with unpushed commits
- **Dry-Run Preview** - Preview changes before applying them
- **Backup & Rollback** - Automatic backups with one-command rollback capability
- **Intelligent Conflict Handling** - Automatically stash and retry on conflicts
- **Parallel Fetching** - Fast performance with configurable parallel fetch operations
- **Cross-Platform** - Compatible with Bash 3.2+ (macOS and Linux)
- **Feature Branch Detection** - Identifies and warns about non-standard branches

## Requirements

- **Git** 2.0 or higher
- **Bash** 3.2 or higher (macOS compatible)
- Basic Unix tools (awk, sed, grep)

## Installation

### Quick Install

Clone the repository and run the installer:

```bash
git clone https://github.com/pxpxltd/ssu.git
cd ssu
./install.sh
```

The installer will guide you through choosing an installation location:
- `~/.local/bin/ssu` - User-local installation (no sudo required)
- `/usr/local/bin/ssu` - System-wide installation (requires sudo)
- `/usr/bin/ssu` - System installation (requires sudo)

### Manual Installation

If you prefer manual installation:

```bash
# Clone the repository
git clone https://github.com/pxpxltd/ssu.git /opt/ssu

# Create a symlink in your PATH
sudo ln -s /opt/ssu/ssu /usr/local/bin/ssu

# Or for user-local installation
mkdir -p ~/.local/bin
ln -s /opt/ssu/ssu ~/.local/bin/ssu
# Make sure ~/.local/bin is in your PATH
```

### Uninstall

```bash
./install.sh --uninstall
```

## Usage

### Basic Syntax

```bash
ssu [OPTIONS]
```

### Options

| Option | Description |
|--------|-------------|
| `-h, --help` | Show help message |
| `-a, --auto` | Batch mode - update all submodules without prompts |
| `-d, --dry-run` | Preview changes without applying them |
| `-b, --branch BRANCH` | Override branch for all submodules |
| `-f, --fail-fast` | Stop on first conflict or error |
| `-s, --status` | Show status table only (no updates) |
| `-p, --push` | Push submodules with unpushed commits |
| `-r, --rollback FILE` | Rollback from backup file |

### Status Legend

The status table uses color-coded indicators:

- **pending** (green) - Has updates available
- **current** (cyan) - Already up-to-date
- **modified** (yellow) - Has local uncommitted changes
- **ahead** (magenta) - Has unpushed commits
- **conflict** (red) - Merge conflict detected

## Examples

### Check Submodule Status

View the current state of all submodules without making changes:

```bash
ssu --status
```

This displays a table showing each submodule's current branch, how many commits it's behind, and whether it's on a feature branch.

### Interactive Update Workflow

Run in interactive mode to selectively update submodules:

```bash
ssu
```

You'll be prompted to:
1. View the status table
2. Select which submodules to update (all, none, or specific ones)
3. Review incoming changes for each submodule
4. Confirm each update individually

### Batch Update All Submodules

Automatically update all submodules without prompts:

```bash
ssu --auto
```

This is useful in CI/CD pipelines or when you want to quickly sync all submodules.

### Dry-Run Preview

Preview what would be updated without making any changes:

```bash
ssu --dry-run
```

This shows which submodules have updates and displays the incoming commits.

### Push Ahead Submodules

When submodules have unpushed commits (shown as "ahead" in the status table), you can push them with:

```bash
# Interactive mode - select which submodules to push
ssu --push

# Batch mode - push all ahead submodules automatically
ssu --push --auto

# Preview what would be pushed without actually pushing
ssu --push --dry-run
```

This is useful when you've made commits in multiple submodules and want to push them all at once. The script will:
- Detect submodules with unpushed commits
- Push to their tracking branches (or set up tracking if needed)
- Skip submodules in detached HEAD state
- Handle errors gracefully

### Force Specific Branch

Override the smart branch detection and force all submodules to use a specific branch:

```bash
ssu --branch develop
```

This is useful when you want all submodules on the same branch (e.g., for testing).

### Stop on First Error

Exit immediately if any submodule encounters a conflict or error:

```bash
ssu --auto --fail-fast
```

This is useful in automated scripts where you want to catch problems immediately.

### Rollback After Issues

If an update causes problems, rollback to the previous state:

```bash
# Find the backup file (created before updates)
ls ~/.ssu/your-project-name/.submodule-backup-*.json

# Rollback to that state
ssu --rollback ~/.ssu/your-project-name/.submodule-backup-20240315-143022.json
```

**Backups are automatically created before any updates** and stored in `~/.ssu/<project-name>/`.

On first run, you'll be prompted to create the backup directory. If you decline, the script will proceed without creating backups (not recommended).

### Combined Options

Combine options for powerful workflows:

```bash
# Preview automatic update on develop branch
ssu --auto --branch develop --dry-run

# Force all to master and stop on first problem
ssu --auto --branch master --fail-fast
```

## Configuration

### Skip List

To skip specific submodules, edit the `SKIP_LIST` array in the `ssu` script:

```bash
SKIP_LIST=(
    "plugins/deprecated-module"
    "vendor/legacy-lib"
)
```

### Branch Priority

Customize the branch detection order by editing `BRANCH_PRIORITY`:

```bash
BRANCH_PRIORITY=("develop" "master" "main")
```

The script will try branches in this order, falling back to the remote's default branch.

### Parallel Jobs

Control the number of parallel fetch operations:

```bash
# Set environment variable (default is 8)
export PARALLEL_JOBS=16
ssu --auto

# Or inline
PARALLEL_JOBS=4 ssu --status
```

Higher values speed up fetching but use more network connections.

## Backups

SSU automatically creates backups before performing any updates to ensure you can rollback if needed.

### Backup Location

Backups are stored in: `~/.ssu/<project-name>/.submodule-backup-YYYYMMDD-HHMMSS.json`

Where `<project-name>` is the name of your project's root directory. For example:
- Project at `/home/user/my-app` → Backups in `~/.ssu/my-app/`
- Project at `/opt/wordpress` → Backups in `~/.ssu/wordpress/`

### First Run

On the first run, SSU will:
1. Check if `~/.ssu` directory exists (creates it if needed)
2. Check if `~/.ssu/<project-name>` exists
3. In **interactive mode**: prompt you to create the directory
4. In **auto mode** (`--auto`): automatically create the directory

If you decline to create the backup directory, SSU will proceed without creating backups for that session.

### Managing Backups

```bash
# List all backups for your project
ls -lh ~/.ssu/your-project-name/

# View backup contents
cat ~/.ssu/your-project-name/.submodule-backup-20240315-143022.json

# Clean up old backups
find ~/.ssu/your-project-name/ -name ".submodule-backup-*.json" -mtime +30 -delete
```

### Rollback

To restore submodules to a previous state:

```bash
ssu --rollback ~/.ssu/your-project-name/.submodule-backup-YYYYMMDD-HHMMSS.json
```

The rollback will restore all submodules to their exact commit SHAs from the backup. Note that this may leave submodules in detached HEAD state.

## Troubleshooting

### Merge Conflicts

If a submodule has merge conflicts:

1. SSU automatically tries to stash local changes and retry
2. If that fails, the conflict is reported
3. Manually resolve the conflict:
   ```bash
   cd path/to/submodule
   git status
   # Resolve conflicts manually
   git add .
   git merge --continue
   ```
4. Or rollback to the previous state:
   ```bash
   ssu --rollback ~/.ssu/your-project-name/.submodule-backup-*.json
   ```

### Detached HEAD State

If a submodule is in detached HEAD:

1. SSU will attempt to checkout the appropriate branch
2. If you want to keep changes:
   ```bash
   cd path/to/submodule
   git checkout -b my-feature-branch
   git push -u origin my-feature-branch
   ```

### Permission Errors

If you encounter permission errors:

1. Check that submodule directories are writable
2. Ensure you have network access to fetch from remotes
3. Verify SSH keys or credentials are configured for Git

### Log Files

All operations are logged to:

```
~/.ssu/<project-name>/logs/submodule-update.log
```

This keeps the project directory clean. Check this file for detailed information about what happened during an update.

### Feature Branch Warning

If a submodule is on a feature branch (not develop/master/main):

- SSU marks it as "FEATURE: Yes" in the status table
- In interactive mode, you'll see which submodules are on feature branches
- Use `--branch` to override if needed, but be careful not to lose work

### Common Issues

**"No .gitmodules found"**
- You're not in a repository root that contains submodules
- Run `cd` to your project root directory

**"Not initialized" status**
- Run `git submodule update --init` first
- Or `git submodule init && git submodule update`

**Fetch fails**
- Check network connectivity
- Verify you have access to the remote repositories
- Check SSH keys: `ssh -T git@github.com`

## Contributing

Contributions are welcome! Here's how you can help:

### Reporting Issues

1. Check existing issues first
2. Provide the output of `ssu --status`
3. Include relevant log entries from `~/.ssu/<project-name>/logs/submodule-update.log`
4. Describe your environment (OS, Bash version, Git version)

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Test on both macOS and Linux if possible
5. Ensure Bash 3.2 compatibility (no associative arrays, no `[[`, etc.)
6. Update documentation if needed
7. Submit a pull request

### Code Style

- Use Bash 3.2 compatible syntax
- Follow existing indentation (4 spaces)
- Add comments for complex logic
- Use meaningful variable names
- Test thoroughly before submitting

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by the need for better submodule management workflows
- Built with compatibility in mind for macOS and Linux environments
- Thanks to all contributors who help improve this tool

## Links

- [GitHub Repository](https://github.com/yourusername/ssu)
- [Issue Tracker](https://github.com/yourusername/ssu/issues)
- [Git Submodules Documentation](https://git-scm.com/book/en/v2/Git-Tools-Submodules)
