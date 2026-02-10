# External Integrations

**Analysis Date:** 2026-02-09

## APIs & External Services

**Git Repository Operations:**
- Git (protocol-agnostic) - Core functionality
  - SSH/HTTPS/Git protocol supported (Git handles transport)
  - Submodule fetch operations (lines 896-901)
  - Push operations (lines 965-1003)
  - Merge operations (lines 926-957)

**No Third-Party APIs:**
- No Slack, Discord, Telegram, or notification services
- No cloud service integrations (AWS, Azure, GCP)
- No package registry integrations
- No authentication services beyond git credentials

## Data Storage

**Databases:**
- None - All state is in-memory or filesystem

**File Storage:**
- **Local filesystem only** - No cloud storage integration
  - Backup location: `~/.ssu/<project-name>/` (lines 29, 727-734, 749)
  - Backup format: JSON files (lines 809-846)
  - Log location: `~/.ssu/<project-name>/logs/` (inferred from line 375, explicitly set during runtime)
  - Backup files: `.submodule-backup-YYYYMMDD-HHMMSS.json` (line 807)

**Caching:**
- None explicit
- Git's local cache (`.git/` directory) used implicitly
- Parallel fetch jobs (default 8) configured per environment (line 44)

**Backup Format:**
```json
{
  "timestamp": "2024-01-15T10:30:00+00:00",
  "submodules": {
    "path/to/submodule": {"sha": "abc123", "branch": "develop"}
  }
}
```
- Manual JSON parsing via regex (line 826-846) - No jq or JSON parser dependency
- Rollback via `--rollback <backup-file>` flag (line 75, 861-890)

## Authentication & Identity

**Auth Provider:**
- None - Uses existing git credentials
  - Git SSH keys (for ssh:// URLs)
  - Git HTTPS credentials (for https:// URLs)
  - Git credential helper (configured in user's git config)

**Implementation:**
- Defers entirely to git for authentication
- No custom auth layer
- No token management
- No API key configuration

**Credential Scope:**
- Limited to operations on configured git remotes
- No elevated permissions needed beyond normal git operations

## Monitoring & Observability

**Error Tracking:**
- None - No external error tracking service (Sentry, DataDog, etc.)

**Logs:**
- **File-based logging only**
  - Location: `~/.ssu/<project-name>/logs/submodule-update.log` (line 375, 32)
  - Format: `[YYYY-MM-DD HH:MM:SS] [LEVEL] message` (line 376)
  - Log levels: INFO, SUCCESS, ERROR, WARNING, UPDATED, SKIPPED, CONFLICT, UPTODATE (lines 386-398)
  - Rotation: None - Log file grows indefinitely (no cleanup implemented)
  - Access: User-readable text file in home directory

**Logging Function (`log()` at lines 375-383):**
```bash
log() {
    local level="$1"
    local message="$2"
    [[ "$BACKUPS_ENABLED" == false ]] && return 0
    local timestamp
    timestamp="$(date '+%Y-%m-%d %H:%M:%S')"
    echo "[$timestamp] [$level] $message" >> "$LOG_DIR/submodule-update.log"
}
```

**Console Output:**
- Colorized status messages (lines 386-398)
- Dry-run preview output (lines 1210-1225)
- Status table display (lines 1105-1209)
- TUI interactive selector output (lines 195-244)
- Summary output (lines 1529-1556)

**No structured logging:**
- Plain text logs
- No JSON logging
- No remote log aggregation

## CI/CD & Deployment

**Hosting:**
- User's machine or server running the script
- No cloud deployment platform integration

**CI Pipeline:**
- None detected - No `.github/workflows/`, `.gitlab-ci.yml`, `.circleci/`, or equivalent
- Script designed for both:
  - Interactive use (default mode)
  - Automated use in CI/CD (via `--auto` flag, line 69-70)

**Deployment via Installer:**
- Cross-platform installer (`install.sh`) detects OS and shell:
  - Supported shells: bash, zsh, fish (lines 134-151 in install.sh)
  - Supported OS: Linux (Arch, Debian, Fedora, others), macOS (lines 89-115 in install.sh)
  - Installation options (install.sh lines 173-205):
    1. `~/.local/bin/` (user-local, no sudo)
    2. `/usr/local/bin/` (system-wide, conditional sudo)
    3. `/usr/bin/` (system, requires sudo)
  - Shell config updates (lines 208-233 in install.sh)
    - Adds directory to PATH in `~/.bashrc`, `~/.bash_profile`, `~/.zshrc`, or `~/.config/fish/config.fish`

**No Containerization:**
- No Dockerfile
- No container registry integration
- Not designed for containerized deployment

## Environment Configuration

**Required Environment Variables:**
- None mandatory at runtime
- Optional:
  - `PARALLEL_JOBS` - Number of concurrent git fetch operations (default: 8)
    - Example: `PARALLEL_JOBS=16 ./ssu --status`

**Secrets/Credentials:**
- Git credentials (SSH keys, HTTPS tokens) - Managed by git, not by SSU
- No credentials stored in SSU
- No credential management system

**Configuration Entry Points:**
1. **Script header variables** (`ssu` lines 20-44):
   - `SKIP_LIST` - Submodules to skip
   - `BRANCH_PRIORITY` - Branch detection order
   - `DEFAULT_PARALLEL_JOBS` - Parallel fetch count
   - `SSU_HOME` - Backup directory root

2. **Environment variables** (runtime):
   - `PARALLEL_JOBS` - Override default parallel jobs

3. **Command-line flags** (`ssu` lines 407-450):
   - `--auto` - Batch mode
   - `--dry-run` - Preview without changes
   - `--branch BRANCH` - Override branch
   - `--fail-fast` - Exit on error
   - `--status` - Status only
   - `--push` - Push instead of update
   - `--rollback FILE` - Restore from backup

**File Paths (No configuration files in git):**
- Installation location: Symlink in `$PATH` (chosen by user during install)
- Project root: Current working directory when `ssu` is invoked
- Backup storage: `$HOME/.ssu/<project-name>/`
- Logs: `$HOME/.ssu/<project-name>/logs/`

## Webhooks & Callbacks

**Incoming:**
- None - SSU is not a server and does not listen for webhooks

**Outgoing:**
- None - SSU does not trigger external webhooks or callbacks
- All operations are local file system and git repository operations

**Integration Points:**
- Git fetch (passive retrieval from remotes)
- Git merge (local operation)
- Git push (outgoing to remotes only)
- No callbacks to CI/CD systems, monitoring platforms, or notification services

---

*Integration audit: 2026-02-09*
