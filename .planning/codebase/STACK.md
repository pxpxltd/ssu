# Technology Stack

**Analysis Date:** 2026-02-09

## Languages

**Primary:**
- Bash 3.2+ - Single executable script (`ssu`), 1656 lines, macOS and Linux compatible

**Secondary:**
- Shell script - Installation script (`install.sh`), 558 lines, supports bash/zsh/fish shells

## Runtime

**Environment:**
- Bash 3.2+ (tested on macOS Bash 3.2, Linux Bash 4.x and 5.x)
- No Node.js, Python, Go, or Ruby runtime required

**Package Manager:**
- None - Zero external dependencies beyond standard Unix tools

**Execution:**
- Direct script execution: `./ssu [OPTIONS]`
- System installation via symlink: `ssu [OPTIONS]` (after install.sh)

## Frameworks

**Build/Dev:**
- ShellCheck - Static bash analysis (optional, for code quality)
- No formal test framework - Manual testing with git repositories

**Terminal/UI:**
- ANSI escape sequences (lines 50-63 in `ssu`) - Color output
- TPUT (terminal utility) - Terminal control with fallback to ANSI codes
  - `tput civis/cnorm` - Cursor visibility (lines 169-174)
  - `tput clear` - Screen clearing (line 182)
  - `tput lines` - Terminal height detection (line 186)
  - `tput cup` - Cursor positioning (line 178)

**Core Dependencies:**
- Git 2.0+ - Submodule operations, fetch, merge, branch detection
- Standard Unix utilities (all present in POSIX systems):
  - `awk` - Text processing (line 530)
  - `sed` - Stream editor (lines 564, 577)
  - `grep` - Pattern matching (line 165 in install.sh, throughout ssu)
  - `tr` - Character translation (line 564)
  - `sort` - Sorting (lines 530, 566)
  - `date` - Timestamp generation (lines 806, 810)
  - `mkdir` - Directory creation (lines 754, 776, 183 in install.sh)
  - `ln` - Symlink creation (line 273 in install.sh)

## Key Dependencies

**Critical:**
- Git 2.0+ - Required for all git operations: `git config`, `git fetch`, `git merge`, `git checkout`, `git push`, `git rev-parse`, `git branch`, `git log`, `git stash`, `git symbolic-ref`, `git show-ref`, `git rev-list`
  - Usage frequency: 51 occurrences across script (line 24 of ssu)
  - All critical git commands with explicit error handling via 2>&1 redirection

**Bash Core:**
- Echo/Printf (103 occurrences) - Output and formatting
- Read (8 occurrences) - User input for interactive selection
- Arrays - Bash 3.2 compatible indexed arrays with parallel value arrays (lines 77-84)
- Eval (4 occurrences) - Dynamic variable access for Bash 3.2 compatibility

**Unix Tools:**
- AWK (1 occurrence) - Parsing git config output (line 530: `git config ... | awk '{print $2}'`)
- SED (2 occurrences) - Text transformation:
  - Line 564: Removing `origin/` prefix from branch names
  - Line 577: Extracting remote HEAD default branch

## Configuration

**Environment Variables:**
- `PARALLEL_JOBS` - Number of concurrent git fetch jobs (default: 8, line 44)
  - Configurable at runtime: `PARALLEL_JOBS=16 ./ssu --status`
- `HOME` - Used for backup directory location (`~/.ssu/`)
- `SHELL` - Detected by installer to determine shell config file (install.sh line 126)
- `PATH` - User PATH, modified by installer to include installation directory

**Script Configuration Files:**
Located at top of `ssu` script:
- `SKIP_LIST` (lines 35-38) - Array of submodule paths to skip
  - Configurable by editing array directly, e.g., `"plugins/deprecated-module"`
- `BRANCH_PRIORITY` (line 41) - Branch detection priority order
  - Default: `["develop", "master", "main"]`
  - Configurable by editing array
- `DEFAULT_PARALLEL_JOBS` (line 44) - Default parallel fetch jobs (8)
- `SSU_HOME` (line 29) - Backup directory root (`~/.ssu/`)

**Backup Directory Structure:**
- Created in: `~/.ssu/<project-name>/` (project name from git root directory, line 727-734)
- Subdirectories:
  - `logs/` - Log files (not directly referenced as created, but referenced in log paths)
  - Backup files: `.submodule-backup-YYYYMMDD-HHMMSS.json` (line 807)

**No config files in repo:**
- No `.env`, `.yaml`, `.toml`, or `.json` config files needed
- All configuration via:
  1. Environment variables
  2. Script header variables (edit-to-configure)
  3. Command-line flags (--auto, --dry-run, --branch, etc.)

## Platform Requirements

**Development:**
- Git 2.0+
- Bash 3.2+ (macOS compatible)
- GNU coreutils (awk, sed, grep) - Present on standard Linux/macOS
- Tput (terminal utility) - Part of ncurses package on Linux, built-in macOS

**Production:**
- Same requirements as development
- No compilation or build step needed
- Installation via:
  1. Symlink to `/usr/local/bin/`, `~/.local/bin/`, or `/usr/bin/` (via install.sh)
  2. Manual symlink (documented in README.md lines 52-61)

**Tested Platforms:**
- macOS (Bash 3.2+)
- Linux distributions:
  - Arch Linux
  - Debian/Ubuntu
  - Fedora/RHEL
- Other Linux with Bash 3.2+ and git 2.0+

**Known Limitations:**
- No Windows support (WSL/Git Bash untested)
- Requires GNU sed/awk (some BSD versions may not be compatible)

---

*Stack analysis: 2026-02-09*
