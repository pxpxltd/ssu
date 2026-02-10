---
phase: quick-003
plan: 01
subsystem: cli
tags: [exec, tui, selector, shell, progress-bar]
dependency-graph:
  requires: [05-commands-tui]
  provides: [working-exec-command, select-all-selector-option]
  affects: []
tech-stack:
  added: []
  patterns: [sh-c-shell-execution, select-all-pre-population]
file-tracking:
  key-files:
    created: []
    modified:
      - internal/cli/tui/tui.go
      - internal/cli/tui/selector.go
      - internal/cli/exec.go
decisions:
  - id: quick-003-1
    description: "SelectAll pre-populates allSelected map in NewSelectorModel (not a separate Init step)"
  - id: quick-003-2
    description: "Shell execution uses sh -c with strings.Join(args) for both single-arg and multi-arg forms"
metrics:
  duration: 1.75min
  completed: 2026-02-09
---

# Quick Task 003: Exec Command Fixes Summary

**One-liner:** Fix exec command with progress bar scan, pre-selected TUI items via SelectAll, and sh -c shell interpretation.

## What Changed

### 1. SelectAll option for TUI selector (tui.go + selector.go)
- Added `SelectAll bool` field to `SelectorOpts` struct
- When `SelectAll=true`, `NewSelectorModel` pre-populates `allSelected` map with all filtered items set to `true`
- Zero value (`false`) preserves existing behavior for update/push commands

### 2. Exec command progress bar (exec.go)
- Replaced direct `eng.Scan()` call with `runScanWithProgress()` for TTY mode
- Moved `autoMode` and `isTTY` variable declarations before the scan call
- Non-TTY and auto mode fall back to direct `eng.Scan()` (no progress bar)

### 3. Shell command execution (exec.go)
- Replaced `exec.CommandContext(ctx, args[0], args[1:]...)` with `exec.CommandContext(ctx, "sh", "-c", strings.Join(args, " "))`
- This allows both `ssu exec 'git status'` (single quoted arg) and `ssu exec git status` (multi-arg) to work correctly
- Added `"strings"` to imports

### 4. Pre-selected submodules in exec (exec.go)
- Added `SelectAll: true` to the `SelectorOpts` passed to `NewSelectorModel`
- Users see all submodules checked by default and deselect the ones they don't want

### 5. Updated help examples (exec.go)
- Added `ssu exec 'git log --oneline -5'` example showing quoted compound commands

## Commits

| # | Hash | Message |
|---|------|---------|
| 1 | af6f6a8 | feat(quick-003): add SelectAll option to TUI selector |
| 2 | 8e6fd96 | fix(quick-003): exec command progress bar, select-all, shell execution |

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `go build ./...` passes
- `go vet ./...` passes
- `go test ./...` all tests pass (no regressions)
- Existing update/push callers unaffected (they don't set SelectAll)
