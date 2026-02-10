# Codebase Concerns

**Analysis Date:** 2026-02-09

## Critical Issues for Go Rewrite

### 1. Bash 3.2 Compatibility Constraints (ELIMINATE IN GO)

**Issue:** The current implementation is severely limited by Bash 3.2 compatibility requirements (macOS compatibility).

**Files:** `ssu` (entire file, especially lines 100-162 array helpers, lines 533-590 branch detection)

**Impact:**
- Cannot use associative arrays - forces manual parallel indexed array simulation with linear O(n) lookups
- No regex in conditionals `[[` - must use `[` and `test`
- No `wait -n` for job management (added in Bash 4.3) - implemented workaround at line 1056
- No `&>` redirection - must use `2>&1`
- Array access via `eval` (line 139) is inherently unsafe and hard to debug
- `read -t` timeout handling is unreliable cross-platform (line 311)

**Impact on rewrite:** Go eliminates all these constraints. Use proper data structures, modern concurrency patterns, and type safety from day one.

---

### 2. Array Lookup Performance (O(n) everywhere)

**Issue:** Every data access requires linear search through SUBMODULE_PATHS array.

**Files:** `ssu` lines 101-162 (`_get_path_index`, `set_array_value`, `get_array_value`)

**Impact:**
- Scanning 50 submodules = 50 lookups per access = O(n²) behavior
- `scan_submodules()` calls these helpers hundreds of times
- Not noticeable at <50 modules, but scaling issues begin above 100

**Current behavior:**
```bash
# Every lookup does this:
for i in "${!SUBMODULE_PATHS[@]}"; do
    if [[ "${SUBMODULE_PATHS[$i]}" == "$path" ]]; then
        # found
    fi
done
```

**Fix in Go:** Use maps (`map[string]SubmoduleData`) for O(1) access.

---

### 3. Parallel Job Management is Fragile

**Issue:** Bash background job handling is error-prone and system-dependent.

**Files:** `ssu` lines 1041-1061 (parallel fetch implementation)

**Specific problems:**
- Line 1056: `wait -n` is unsupported in Bash 3.2/4.0/4.1 - fallback to `wait` blocks all jobs
- Semaphore pattern (job_count tracking) can get out of sync if any job fails silently
- No timeout on hung git processes - single slow remote can block everything
- `git fetch --all` in background subshell can't report errors to parent
- No way to know which submodule failed if background job errors

**Current code:**
```bash
for path in "${initialized_paths[@]}"; do
    (
        cd "$PROJECT_ROOT/$path"
        git fetch --all --quiet 2>/dev/null || true  # SILENT FAILURE!
    ) &
    job_count=$((job_count + 1))
    if [[ $job_count -ge $num_jobs ]]; then
        wait -n 2>/dev/null || wait  # Fallback is inefficient
        job_count=$((job_count - 1))
    fi
done
```

**Risk:** If `git fetch` hangs on one submodule, it can block the script indefinitely.

**Fix in Go:** Use goroutine pools with timeouts and proper error propagation.

---

### 4. JSON Parsing is Brittle (Regex-based, no validation)

**Issue:** Manual JSON parsing via regex is fragile and doesn't validate backup format.

**Files:** `ssu` lines 799-846 (backup creation), lines 861-890 (rollback parsing)

**Specific problems:**
- Line 875: BASH_REMATCH regex assumes exact JSON format - breaks if timestamps contain special chars
- No validation that backup file is valid JSON before attempting rollback
- No atomic file writes - interrupted backup creates malformed JSON
- Rollback can silently skip submodules if path doesn't match regex exactly
- No size limits on JSON - could bloat user's home directory

**Backup creation is vulnerable:**
```bash
echo "{" > "$backup_file"
echo '  "timestamp": "'"$(date -Iseconds)"'",' >> "$backup_file"
# ... multiple echo calls ...
echo "}" >> "$backup_file"
# If script is killed between lines, result is invalid JSON
```

**Rollback parsing:**
```bash
if [[ "$line" =~ \"([^\"]+)\":[[:space:]]*\{\"sha\":[[:space:]]*\"([a-f0-9]+)\" ]]; then
    # Regex is too strict - fails if JSON formatting changes slightly
```

**Fix in Go:** Use `encoding/json` package with proper error handling and atomic writes.

---

### 5. Global State and Side Effects (Hard to test)

**Issue:** Heavy reliance on global variables makes code hard to reason about and test.

**Files:** `ssu` lines 68-95 (global state), scattered throughout functions

**Global state that gets mutated:**
- `SUBMODULE_PATHS[]`, `SUBMODULE_STATUS[]`, `SUBMODULE_BRANCH[]` - mutated in `scan_submodules`, used everywhere
- `UPDATED_COUNT`, `SKIPPED_COUNT`, `CONFLICT_COUNT`, `ERROR_COUNT` - incremented in multiple functions
- `TUI_CURSOR`, `TUI_SELECTED[]` - mutated in `tui_select` loop
- `BACKUPS_ENABLED`, `BACKUP_DIR`, `LOG_DIR` - conditionally set during initialization
- `MODE` - changes behavior of `prompt_selection`, `process_updates`, etc.

**Impact:**
- Cannot unit test functions in isolation
- Silent dependencies between functions (e.g., `process_updates` assumes `scan_submodules` already ran)
- Calling functions twice produces different results (counters don't reset)
- Hard to debug which function modified which global

**Current code structure:**
```bash
# Global state scattered everywhere
SUBMODULE_STATUS=()
UPDATED_COUNT=0
MODE="interactive"

# Functions that depend on this state
process_updates() {
    for path in $(get_submodule_paths | sort); do
        local status=$(get_array_value "SUBMODULE_STATUS" "$path")
        # ... relies on previous scan_submodules() call ...
    fi
}
```

**Fix in Go:** Use dependency injection, pass data through function parameters, use struct methods.

---

### 6. Error Handling is Inconsistent and Often Silent

**Issue:** Many operations fail silently without propagating errors correctly.

**Files:** Multiple locations - examples: lines 543-544, 629, 901, 1050

**Specific problems:**

**Silent failures in critical paths:**
- Line 901: `git fetch` failure is silently ignored: `git fetch origin "$branch" --quiet 2>/dev/null`
- Line 1050: Background fetch failure is swallowed: `git fetch --all --quiet 2>/dev/null || true`
- Line 544-545: `cd` into submodule can fail but is not checked

**Error codes aren't used consistently:**
```bash
# Lines 1502-1527: Push function has 3 return codes
push_submodule() {
    # Returns: 0=success, 1=error, 2=detached HEAD
    # But caller doesn't always check
}
```

**Merge conflict handling is incomplete (lines 935-963):**
```bash
handle_conflict() {
    git stash --quiet 2>/dev/null || true  # Stash might fail
    if git merge "origin/$branch" --quiet 2>/dev/null; then
        git stash pop --quiet 2>/dev/null || {
            # Prints message but doesn't track error
            print_status WARNING "Stash could not be applied cleanly..."
        }
        return 0  # Still returns success even if stash pop failed!
    fi
}
```

**Impact:** Script reports success when operations actually fail, leading to data loss.

**Fix in Go:** Use proper error types, error wrapping, and consistent error handling patterns.

---

### 7. TUI Terminal Control is Brittle

**Issue:** Terminal control via ANSI sequences is not reliably portable and lacks proper cleanup.

**Files:** `ssu` lines 167-243 (TUI functions), lines 273-287 (trap handler)

**Specific problems:**
- `tput` commands fallback to ANSI codes but behavior varies across terminal emulators
- Cursor might not be restored if script crashes (no guarantee trap fires)
- Escape sequence parsing for arrow keys is fragile (lines 309-326):
  - `read -rsn1 -t 0.1 key2` timeout is non-standard across shells
  - Different terminals send different escape sequences
  - No handling for function keys or special input

**Current implementation:**
```bash
# Arrow key parsing is brittle
if [ "$key" = $'\x1b' ]; then
    read -rsn1 -t 0.1 key2  # timeout unreliable
    if [ "$key2" = "[" ]; then
        read -rsn1 -t 0.1 key3
        case "$key3" in
            A)  # Up arrow
            B)  # Down arrow
        esac
    fi
fi
```

**Risk:** Terminal gets left in bad state if:
- Script crashes while cursor is hidden
- Piped input interrupts read sequence
- Ctrl+C doesn't properly restore cursor

**Impact on rewrite:** Go has robust terminal libraries (`golang.org/x/term`, `bubbletea`) that handle all of this correctly.

---

### 8. Branch Detection Logic is Complex and Has Bugs

**Issue:** Smart branch detection has special cases and historical fixes that are hard to maintain.

**Files:** `ssu` lines 533-590 (detect_best_branch)

**Known complexity:**
- v1.1.1 was a hotfix: "Fix smart branch detection" (commit d8ce5e5) - indicates previous bugs
- Line 551-559: Feature branch detection has multiple conditions:
  ```bash
  if [[ "$current_branch" != "develop" ]] && [[ "$current_branch" != "master" ]] && \
     [[ "$current_branch" != "main" ]] && [[ "$current_branch" != "HEAD" ]] && \
     [[ "$current_branch" != "detached" ]]; then
  ```
- No clear documentation of priority order vs. current branch behavior
- Fallback logic at lines 575-587 is convoluted:
  ```bash
  default_branch=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@')
  if [[ -z "$default_branch" ]] && [[ -n "$remote_branches" ]]; then
      default_branch=$(echo "$remote_branches" | head -1)
  fi
  if [[ -n "$default_branch" ]]; then
      echo "$default_branch"
  else
      echo "master"  # Ultimate fallback to hardcoded name
  fi
  ```

**Risk:** User expectations unclear - when does SSU use your current branch vs. override it?

**Fix in Go:** Refactor with clear enum for branch priority, unit tests for each decision path.

---

### 9. Status Detection Has Multiple Competing Checks

**Issue:** Status determination (pending/current/modified/ahead) can be ambiguous and has redundant checks.

**Files:** `ssu` lines 1092-1101 (status determination in scan_submodules)

**Order matters, but isn't documented:**
```bash
if has_local_changes "$path"; then
    set_array_value "SUBMODULE_STATUS" "$path" "modified"
elif has_unpushed_commits "$path" 2>/dev/null; then
    set_array_value "SUBMODULE_STATUS" "$path" "ahead"
elif [[ "$behind" -gt 0 ]]; then
    set_array_value "SUBMODULE_STATUS" "$path" "pending"
else
    set_array_value "SUBMODULE_STATUS" "$path" "current"
fi
```

**Problems:**
- What if submodule is BOTH ahead AND behind? (Not possible in normal git, but edge cases exist)
- What if local changes include committed but unpushed files? (`has_unpushed_commits` might miss them)
- `2>/dev/null` on line 1095 hides errors from `has_unpushed_commits`

**Fix in Go:** Create explicit status type with clear precedence and comprehensive tests.

---

### 10. Hardcoded Dependencies and Assumptions

**Issue:** Script makes many hardcoded assumptions about Git and environment.

**Files:** Throughout script

**Hardcoded assumptions:**
- Line 41: `BRANCH_PRIORITY=("develop" "master" "main")` - no "main" first fallback (Git changed defaults)
- Line 44: `DEFAULT_PARALLEL_JOBS=8` - arbitrary number, should be CPU count
- Line 530: `git config --file .gitmodules --get-regexp '^submodule\..*\.path$'` - assumes .gitmodules exists
- Line 564: `sed 's|origin/||'` - assumes all remotes are named "origin"
- Line 586: `echo "master"` - ultimate fallback to "master" (Git changed default to "main")
- Line 1548: `git add plugins/` - hardcoded directory name in suggestion

**Environmental assumptions:**
- Assumes `git` is in PATH (checked in install.sh but not in ssu itself)
- Assumes `awk`, `sed`, `grep` are GNU versions (differences on BSD/macOS)
- Assumes `tput` is available (may not be in minimal containers)
- Assumes `date -Iseconds` format (line 810) - not POSIX, may not work everywhere

**Fix in Go:** Use configuration files, detect environment properly, don't hardcode paths.

---

### 11. Scalability Limits (Not Documented)

**Issue:** Performance not tested beyond ~20 submodules, limits unclear.

**Files:** `ssu` parallel fetch (lines 1041-1061), array lookups throughout

**Scaling problems:**
- Parallel job limiting assumes sequential completion (`wait -n` fallback is O(total jobs))
- Array lookups are O(n) per access
- Display width calculation doesn't account for very long paths (line 1132)
- Status table doesn't paginate - very large monorepos will have unreadable output

**Unknown bottlenecks:**
- No test with 100+ submodules
- No test with deeply nested submodule paths (>5 levels)
- No test with very large branches or commit histories

**Fix in Go:** Add performance benchmarks, support pagination, optimize hot paths.

---

### 12. Log Files Have No Rotation

**Issue:** Logs in `~/.ssu/<project>/logs/` grow indefinitely.

**Files:** `ssu` lines 375-384 (log function), README.md lines 351-357

**Impact:**
- User home directory can fill up over months with a single project
- No documented way to clean up old logs
- No size limits on individual log files

**Current logging:**
```bash
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    mkdir -p "$LOG_DIR"
    echo "[$timestamp] [$level] $message" >> "$LOG_DIR/submodule-update.log"
}
```

**Fix in Go:** Implement log rotation by date/size with configurable retention.

---

### 13. Feature Branch Detection is Incomplete

**Issue:** Feature branch detection doesn't account for all branch naming schemes.

**Files:** `ssu` lines 1076-1079 (feature detection)

**Current logic:**
```bash
if [[ "$current_branch" != "develop" ]] && [[ "$current_branch" != "master" ]] && \
   [[ "$current_branch" != "main" ]] && [[ "$current_branch" != "detached" ]]; then
    is_feature="Yes"
fi
```

**Problems:**
- Doesn't recognize other common "standard" branches: `staging`, `release/*`, `hotfix/*`
- No consideration for git-flow or trunk-based development models
- Can't detect "accidental" feature branches (user forgot to switch back)

**Fix in Go:** Make this configurable, support common branch naming patterns.

---

### 14. Backup Directory Prompt is Problematic

**Issue:** User has to make backup directory decision at runtime, affects reliability.

**Files:** `ssu` lines 735-793 (init_backup_directory), lines 1588-1590 (initialization)

**Problems:**
- Interactive prompt blocks in CI/CD (won't work without `-a` flag)
- No way to pre-configure backup behavior
- Backup directory creation is user action, not automatic
- If user declines once, no reminder for future runs

**Current flow:**
```bash
if [[ ! -d "$project_backup_dir" ]]; then
    if [[ "$MODE" == "auto" ]]; then
        # Auto-create
    else
        # Ask user interactively (will fail in non-TTY)
        read -rp "Create backup directory for this project? [Y/n]: " create_backup
    fi
fi
```

**Fix in Go:** Use configuration file or environment variable, default to creating backups safely.

---

### 15. Conflict Resolution Strategy is Incomplete

**Issue:** Current conflict resolution only tries stash/retry, no manual resolution guidance.

**Files:** `ssu` lines 935-963 (handle_conflict)

**Limitations:**
- Only stash approach, no other merge strategies
- User gets error message but no guidance on how to fix
- Can't detect specific conflict types (file deleted vs. content conflict)
- Conflict resolution is "all or nothing" - can't partial apply updates

**Fix in Go:** Support multiple resolution strategies, provide detailed conflict reports.

---

## Security Considerations

### 1. Backup File Permissions

**Issue:** Backup JSON files created without explicit permission control.

**Files:** `ssu` line 807 (create_backup)

**Risk:** Backups contain git SHAs and paths, could be readable by other users.

**Current code:**
```bash
backup_file="$BACKUP_DIR/${BACKUP_PREFIX}-${timestamp}.json"
echo "{" > "$backup_file"  # Uses default umask
```

**Fix in Go:** Create with restrictive permissions (0600), or encrypt if sensitive.

### 2. Path Traversal in Rollback

**Issue:** Rollback file path is user-provided without validation.

**Files:** `ssu` lines 487-492 (argument parsing), lines 861-890 (rollback_all)

**Risk:** User could be tricked into rolling back from arbitrary file.

**Current code:**
```bash
# No validation that file is actually in backup directory
if [[ -n "$ROLLBACK_FILE" ]]; then
    rollback_all "$ROLLBACK_FILE"  # User can pass any path
    exit 0
fi
```

**Fix in Go:** Validate rollback files are from configured backup directory.

### 3. No SSH Key Protection Check

**Issue:** Script doesn't verify SSH keys are protected or warn if using unencrypted keys.

**Files:** Mentioned in README troubleshooting (lines 380), but not implemented in script

**Risk:** If SSH is misconfigured, script might prompt for passwords repeatedly.

**Fix in Go:** Check SSH key setup before running, provide setup guidance.

---

## Performance Bottlenecks

### 1. Repeated Git Operations

**Issue:** Some git operations are called multiple times unnecessarily.

**Files:** `ssu` scan_submodules and processing phases

**Examples:**
- `git fetch --all` is called in Phase 2 (line 1050), but branch detection calls `git branch -r` which uses cached refs - redundant work
- `has_local_changes` calls `git diff --quiet` twice (line 610)

**Fix in Go:** Cache git operations, batch commands, use git library instead of shelling out.

### 2. Terminal Output is Not Buffered

**Issue:** Each status line is printed separately, can be slow over slow terminals.

**Files:** `ssu` lines 1195-1205 (display_status_table)

**Impact:** Observable delay printing large tables.

**Fix in Go:** Buffer output, print once.

---

## Fragile Areas (High Risk of Regression)

### 1. TUI Selection State Management

**Files:** `ssu` lines 248-369 (tui_select)

**Fragility:**
- Complex state machine with TUI_CURSOR, TUI_SELECTED arrays
- Multiple entry points (escape sequences, regular keys) all modifying state
- No clear invariants (e.g., TUI_CURSOR should always be <= array length)
- Hard to add new keybindings without breaking state

**Safe modification approach:** Add comprehensive test cases for each key input before modifying.

### 2. Array Synchronization

**Files:** `ssu` lines 103-161 (array helpers)

**Fragility:**
- SUBMODULE_PATHS is the "master" array - if any other array gets out of sync, silent corruption
- No validation that all arrays have matching lengths
- `eval` usage in line 139 is inherently unsafe

**Safe modification approach:** Add assertions checking array lengths match, never use eval for data access.

### 3. Bash 3.2 Compatibility Layer

**Files:** Lines 50-62 (color detection), lines 309-326 (escape sequence parsing)

**Fragility:**
- Fallback behaviors differ from primary implementation
- Hard to test on target platforms (older bash versions)
- No CI testing old bash versions

**Safe modification approach:** Keep comprehensive compatibility docs, test on Bash 3.2 VMs.

---

## Test Coverage Gaps

**What's untested:**
- Edge case: submodule with empty .git file (sparse checkouts)
- Edge case: submodule with custom remote names (not "origin")
- Edge case: submodule on detached HEAD trying to push (should warn clearly)
- Edge case: very long submodule paths (>256 chars)
- Edge case: submodules with spaces in path names
- Conflict resolution with multiple file conflicts
- Rollback when submodule directories deleted post-update
- TUI with piped input (should fail gracefully)
- Large monorepos (100+ submodules) performance
- Network failures during parallel fetch phase
- Interrupted backup creation (corrupted JSON)

**Priority:** High - these gaps mean bugs slip to production.

**Fix in Go:** Build comprehensive test suite covering these cases.

---

## Maintainability Issues

### 1. Documentation Drift

The CLAUDE.md (project instructions) documents the architecture in detail, but code has evolved:
- v1.1.0 added TUI selector and root repository display
- v1.1.1 fixed branch detection bugs
- Documentation may be stale

**Fix:** Keep CLAUDE.md in sync with code changes.

### 2. Version Number in Multiple Places

**Files:**
- Line 4 in `ssu`
- Line 5 in README.md
- Git tags: `git tag -a v1.1.1`

**Risk:** Easy to forget to update all three on release.

**Fix in Go:** Single source of truth (build time injection or config file).

### 3. Bash Doesn't Have Built-in Testing

**Impact:** Hard to add tests without significant refactoring to allow function mocking.

**Fix in Go:** Use standard testing library, TDD friendly.

---

## Recommendations for Go Rewrite Priority

**Phase 1 (Critical):**
1. Eliminate bash array lookup O(n) behavior → O(1) with maps
2. Proper error handling and propagation
3. Reliable parallel job management with timeouts
4. Atomic JSON backup with proper validation

**Phase 2 (Important):**
5. Replace TUI terminal control with proper library
6. Comprehensive test suite (unit + integration)
7. Configuration file support
8. Better branch detection with configurable priority

**Phase 3 (Nice-to-have):**
9. Log rotation
10. Performance optimizations for 100+ submodules
11. Detailed conflict resolution strategies
12. SSH key validation

---

*Concerns audit: 2026-02-09*
