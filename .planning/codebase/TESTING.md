# Testing Patterns

**Analysis Date:** 2026-02-09

## Test Framework

**Runner:**
- No formal test framework exists
- Manual testing required on real git repositories with submodules
- ShellCheck available for static analysis (referenced in CLAUDE.md)

**Assertion Library:**
- Not applicable (manual testing only)

**Run Commands:**
```bash
shellcheck ssu              # Static analysis
shellcheck install.sh       # Check installer script
./ssu --status              # Display status without modifications
./ssu --dry-run             # Preview updates without applying
./ssu --auto --dry-run      # Batch preview mode
```

## Test File Organization

**Location:**
- No dedicated test files exist
- Testing is done manually on actual git repositories with submodules
- Code documentation: `CLAUDE.md` contains development guidance (lines 49-118 in `/media/nvme/dev/pxpx/ssu/CLAUDE.md`)

**Naming:**
- No test file pattern (no *.test.sh or *.spec.sh files)

**Structure:**
```
ssu/
├── ssu              # Single executable (all logic in one file)
├── install.sh       # Installer (no tests)
├── README.md        # User documentation
├── CLAUDE.md        # Development guide with manual test scenarios
└── LICENSE
```

## Test Structure

**Manual Testing Approach:**

The codebase uses manual testing with real git repository scenarios. From `CLAUDE.md` (lines 45-80):

```
Testing the Script
# Status check only (no modifications)
./ssu --status

# Dry-run to preview updates
./ssu --dry-run

# Test interactive mode in a git repo with submodules
cd /path/to/project/with/submodules
/path/to/ssu/ssu

# Test batch mode
./ssu --auto --dry-run
```

**Test Scenarios:**

The CLAUDE.md documents how to create test state (lines 73-90):

```bash
# In a submodule:
cd path/to/submodule

# Create "behind" state
git reset --hard HEAD~3

# Create "modified" state
echo "test" >> somefile.txt

# Create "feature branch" state
git checkout -b feature/test

# Create "ahead" state
git commit --allow-empty -m "local commit"
```

**Patterns:**
- Use git commands directly to create test states
- Test each status type individually: pending, current, modified, ahead, conflict
- Verify output and behavior with manual inspection
- No automated assertions - rely on visual inspection

## Mocking

**Framework:** Not used

**Patterns:** Not applicable - tests use real git repositories

**What to Mock:** Not applicable (no mocking framework)

**What NOT to Mock:** Not applicable

## Fixtures and Factories

**Test Data:**
- Manual git repository with submodules serves as fixture
- Test submodules created in-place with git operations
- No predefined factory pattern

**Location:**
- Test scenarios documented in `CLAUDE.md` (lines 73-90)
- No fixture files or setup scripts

**Example fixture creation:**
```bash
# Create a test repository with submodules
mkdir test-project
cd test-project
git init
git submodule add <remote-url> path/to/submodule
```

## Coverage

**Requirements:** None enforced

**View Coverage:** Not applicable (no test runner)

## Test Types

**Unit Tests:**
- Scope: Not applicable (no unit test framework)
- Testing approach: Functions tested via CLI invocation with different arguments
- Example: `./ssu --status` tests scan_submodules() and display_status_table() together

**Integration Tests:**
- Scope: Full workflow testing in real git repositories
- Approach: Manual testing with multiple submodules in different states
- Tested scenarios from CLAUDE.md:
  1. Status display (`--status` flag)
  2. Dry-run preview (`--dry-run` flag)
  3. Interactive mode (no flags - default behavior)
  4. Batch mode (`--auto` flag)
  5. Push workflow (`--push` flag)
  6. Rollback functionality (`--rollback FILE`)
  7. Feature branch detection
  8. Conflict handling and stash/retry

**E2E Tests:**
- Framework: Not formalized
- Approach: Manual end-to-end in actual project repositories
- Test sequence:
  1. Run `./ssu --status` in a multi-submodule project
  2. Verify all submodules display with correct status colors
  3. Run `./ssu --dry-run` and verify no changes are made
  4. Run `./ssu --auto` and verify submodules are updated
  5. Run `./ssu --push --auto` and verify commits are pushed
  6. Test rollback with `./ssu --rollback <backup-file>`

## Common Patterns

**Testing git state detection:**

The code uses multiple helper functions to test repository state. These should be tested manually:

```bash
# is_submodule_initialized (line 523-527)
# Test: Check if submodule is initialized
ls path/to/submodule/.git  # Should exist

# has_local_changes (line 606-614)
# Test: Modify a file and run --status
echo "test" >> path/to/submodule/file.txt
./ssu --status  # Should show "modified"

# has_unpushed_commits (line 616-634)
# Test: Create local commit without pushing
cd path/to/submodule
git commit --allow-empty -m "test"
cd ../..
./ssu --status  # Should show "ahead"

# detect_best_branch (line 535-590)
# Test: With different branch configurations
# Feature branch exists with remote
# Feature branch has no remote (should fallback to develop)
# No develop, fallback to master
# No master, fallback to main
```

**Async Testing:** Not applicable (bash script, no async)

**Error Testing:**
- Test conflict handling: Run with merge conflicts present
- Test missing files: Remove .gitmodules and verify error message
- Test detached HEAD: Checkout a commit and verify "cannot push" warning
- Test permission errors: Try to install to protected directory

**Testing conflict resolution:**

From `ssu` lines 935-963 (`handle_conflict` function):
```bash
# Create a merge conflict scenario:
cd path/to/submodule
git fetch origin
# Create local change that conflicts with remote
# git merge should fail
./ssu  # Should show conflict handling in progress
# Verify stash/retry logic executes
```

## Manual Testing Workflow

**Pre-test Setup:**

1. Create test repository structure:
```bash
mkdir test-ssu
cd test-ssu
git init
git config user.email "test@example.com"
git config user.name "Test User"

# Add submodules (or use existing repos)
git submodule add <repo1> plugins/module1
git submodule add <repo2> plugins/module2
```

2. Create test states in each submodule:
```bash
cd plugins/module1
# Behind: git reset --hard HEAD~3
# Modified: echo "test" >> file.txt
# Ahead: git commit --allow-empty -m "test"
cd ../..
```

3. Run test commands:
```bash
/path/to/ssu --status          # Display current state
/path/to/ssu --dry-run         # Preview changes
/path/to/ssu --auto            # Apply changes
/path/to/ssu --status          # Verify changes
/path/to/ssu --rollback <file> # Rollback if needed
```

**Verification Checklist:**

- [ ] Status table displays root repository first (bold, labeled "(root)")
- [ ] All submodule paths shown correctly
- [ ] Status colors: green=pending, cyan=current, yellow=modified, magenta=ahead, red=conflict
- [ ] "Behind" count shows correct number
- [ ] Feature branch detection works (shows "Yes" for feature branches)
- [ ] Dry-run preview shows what would be updated without making changes
- [ ] Interactive selection allows arrow keys and space toggle
- [ ] Batch mode (`--auto`) updates all pending without prompts
- [ ] Conflict handling stashes and retries gracefully
- [ ] Push mode identifies "ahead" submodules correctly
- [ ] Rollback restores to correct commit SHAs
- [ ] Backup files created in `~/.ssu/<project-name>/`
- [ ] Log file written to `~/.ssu/<project-name>/logs/`
- [ ] Exit codes correct: 0=success, 1=error, 2=conflict/fail-fast

**Testing Special Cases:**

```bash
# Detached HEAD state
cd plugins/module1
git checkout HEAD~1  # Enter detached state
cd ../..
./ssu --push  # Should skip with warning "Cannot push"

# Missing .gitmodules
rm .gitmodules
./ssu  # Should error: "No .gitmodules found"

# Parallel fetch performance
PARALLEL_JOBS=1 ./ssu --status  # Test with single job
PARALLEL_JOBS=16 ./ssu --status # Test with many jobs

# Non-TTY mode (piped)
./ssu | cat  # Should fallback to --auto warning or use --auto mode
```

## ShellCheck Validation

**Static Analysis:**

From `install.sh` line 94:
```bash
# shellcheck disable=SC1091
. /etc/os-release
```

Run ShellCheck on main scripts:
```bash
shellcheck ssu
shellcheck install.sh
```

**Common ShellCheck Rules (not disabled):**
- SC2086: Quote to prevent word splitting (used where applicable)
- SC2076: `[[ ]]` not supported in Bash 3.2 (use `[` instead)
- SC2181: Check exit code of command explicitly
- SC2088: Tilde expansion in quotes requires eval

**Expected Results:**
- `ssu`: Should pass with no errors (Bash 3.2 compatible)
- `install.sh`: Should pass with only SC1091 disabled for OS detection

## Code Validation

**Bash Compatibility Check:**

From CLAUDE.md (lines 94-103), verify Bash 3.2 compatibility:
```bash
# Check for unsupported features
grep -n '\[\[' ssu         # Should not find [[
grep -n '&>' ssu           # Should not find &>
grep -n 'declare -A' ssu   # Should not find associative arrays
grep -n '(())' ssu         # Should not find bash arithmetic (may exist, is ok)
```

**Manual Code Review Areas:**

1. **Array handling** (lines 77-161): Verify Bash 3.2 array simulation works
   - Test with arrays of varying sizes
   - Test `set_array_value` adds new entries correctly
   - Test `get_array_value` finds entries with spaces in paths

2. **TUI control** (lines 168-369): Verify terminal handling
   - Test cursor movement with `tput` commands
   - Verify fallback to ANSI codes when `tput` unavailable
   - Test arrow key input parsing
   - Test space/enter/q key handling

3. **Parallel fetch** (lines 1040-1061): Verify job control
   - Test with `PARALLEL_JOBS=1` (sequential)
   - Test with `PARALLEL_JOBS=8` (default)
   - Test with `PARALLEL_JOBS=100` (high concurrency)
   - Verify `wait -n` compatibility (bash 4.3+ feature fallback to `wait`)

4. **Git integration** (lines 523-701): Verify all git commands
   - Test with network issues (offline mode)
   - Test with large repositories
   - Test with shallow clones
   - Test with various git versions (2.0+)

---

*Testing analysis: 2026-02-09*
