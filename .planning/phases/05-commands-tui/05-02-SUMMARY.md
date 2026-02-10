---
phase: 05-commands-tui
plan: 02
subsystem: tui-components
tags: [bubbletea, bubbles, lipgloss, tui, multi-select, progress-bar, elm-architecture]
dependency-graph:
  requires:
    - "02-01: GitService interface (SubmoduleStatus type)"
    - "03-01: engine.SubmoduleInfo, ScanResult, ProgressEvent"
    - "05-01: styles.go (StyleForStatus, TitleStyle, MutedStyle)"
  provides:
    - "SelectorModel: multi-select bubbletea model with all keybindings"
    - "ProgressModel: animated progress bar bubbletea model"
    - "SelectorItem interface + SubmoduleItem adapter"
    - "Shared tea.Msg types for cross-model communication"
    - "FilterPending/FilterAhead pre-filter helpers"
  affects:
    - "05-03+: Command implementations compose these models"
    - "05-04+: ssu exec, ssu init use SelectorModel"
tech-stack:
  added:
    - "github.com/charmbracelet/bubbletea v1.3.0"
    - "github.com/charmbracelet/bubbles v0.20.0"
    - "github.com/charmbracelet/harmonica (transitive)"
  patterns:
    - "Elm Architecture (Model/Update/View) for TUI state management"
    - "Value-receiver methods on bubbletea Models (required by tea.Model interface)"
    - "Pointer-receiver methods for internal mutation (sortItems, resizeViewport, updateDetailPane)"
    - "External goroutine -> p.Send() -> tea.Msg for async communication"
    - "SelectorItem interface for decoupled item rendering"
key-files:
  created:
    - internal/cli/tui/tui.go
    - internal/cli/tui/selector.go
    - internal/cli/tui/selector_keys.go
    - internal/cli/tui/confirm.go
    - internal/cli/tui/progress.go
  modified:
    - go.mod
    - go.sum
key-decisions:
  - "Confirmation is a state within SelectorModel (confirming bool), not a separate bubbletea model"
  - "Selections tracked by original item index (allSelected map) to persist across filter changes"
  - "Sort preserves selections by rebuilding the index map after re-ordering"
  - "ProgressModel uses Result() interface{} for generic result storage (caller type-asserts)"
  - "FetchCompleteMsg.Result is interface{} to avoid circular import between tui and engine"
duration: 6min
completed: 2026-02-09
---

# Phase 5 Plan 2: TUI Components Summary

**Custom bubbletea multi-select selector with 11 keybindings (arrows/j/k, space, a/A, enter, q/esc, /, ?, tab, s), split-pane changelog detail, filter/sort, confirmation step, plus animated gradient progress bar model.**

## Accomplishments

1. **Shared types (tui.go):** SelectorItem interface with Path/Label/Metadata/DetailContent methods. SubmoduleItem adapter wrapping engine.SubmoduleInfo. SelectorOpts with Title, ShowDetail, FilterFn, and Operation fields. SubmoduleItems helper converts info slices. FilterPending/FilterAhead pre-filter helpers. Shared tea.Msg types for scan/update/push communication.

2. **Keybindings (selector_keys.go):** SelectorKeyMap struct with 11 bindings matching bash-era conventions plus new features. DefaultSelectorKeyMap constructor. ShortHelp returns 5 essential bindings. FullHelp returns 2-column layout with all 11 bindings. Implements help.KeyMap interface for bubbles/help integration.

3. **Selector model (selector.go):** Full bubbletea Model with Init/Update/View. Three update modes: normal, filter input, and confirmation. Cursor navigation with scroll windowing. Checkbox toggle with persistent selection across filters. Split-pane layout using lipgloss.JoinHorizontal with viewport for changelog detail. Three sort modes (path, status, behind) with selection preservation. Filter narrows items by path substring. Help overlay toggle. Selected() returns SubmoduleInfo pointers for caller consumption.

4. **Confirmation view (confirm.go):** renderConfirmation helper renders selected item list with y/N prompt. Not a separate model -- rendered as part of SelectorModel.View() when confirming is true.

5. **Progress model (progress.go):** ProgressModel with bubbles/progress gradient bar. Receives FetchProgressMsg/ProcessProgressMsg from external goroutines. Displays "done/total fetching path" label. Failed count shown in red. Ctrl+C sets context.Canceled. PrintProgressLine non-TTY fallback for auto mode.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Shared TUI types and multi-select selector model | cc613f1 | tui.go, selector.go, selector_keys.go, confirm.go |
| 2 | Progress bar model for parallel fetch | 1f6f2b2 | progress.go |

## Files Changed

### Created
- `internal/cli/tui/tui.go` - SelectorItem interface, SubmoduleItem, shared messages, filter helpers
- `internal/cli/tui/selector.go` - SelectorModel with full multi-select, filter, sort, detail pane, confirmation
- `internal/cli/tui/selector_keys.go` - SelectorKeyMap with 11 keybindings + help.KeyMap interface
- `internal/cli/tui/confirm.go` - Confirmation view renderer
- `internal/cli/tui/progress.go` - ProgressModel with gradient bar, progress messages, non-TTY helper

### Modified
- `go.mod` - Added bubbletea v1.3.0, bubbles v0.20.0, harmonica (transitive)
- `go.sum` - Updated checksums

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Confirmation as SelectorModel state, not separate model | Avoids complexity of chaining tea.Programs; y/N is simple enough as a mode within the selector |
| allSelected map keyed by original index | Selections persist when filter narrows/widens the visible set; no re-selecting needed |
| Sort rebuilds index map | Sorting changes item positions; old selected-index -> new selected-index mapping preserves user choices |
| ProgressModel.Result() returns interface{} | Avoids coupling progress model to specific engine result types; caller does type assertion |
| FetchCompleteMsg.Result is interface{} | Prevents circular import between tui package and engine package |
| Value receivers on Model methods | Required by bubbletea's tea.Model interface; pointer receivers used only for internal mutation methods |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing harmonica transitive dependency**
- **Found during:** Task 2
- **Issue:** bubbles/progress imports charmbracelet/harmonica which wasn't in go.sum after initial `go get`
- **Fix:** Ran `go get github.com/charmbracelet/bubbles/progress@v0.20.0 && go mod tidy`
- **Files modified:** go.mod, go.sum
- **Commit:** 1f6f2b2

## Issues & Risks

None. All components compile and pass vet.

## Next Phase Readiness

**Ready for Plan 05-03+** (command implementations):
- SelectorModel can be composed by `ssu update`, `ssu push`, `ssu exec` commands
- ProgressModel can be composed for scan/fetch progress display
- Shared message types enable clean communication between TUI and engine goroutines
- All keybindings match CONTEXT.md requirements
