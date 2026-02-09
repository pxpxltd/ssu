# Quick Task 002: Green Progress Bar + Selector Header

## Changes
- Changed progress bar gradient from default (blue/purple) to green shades (#76B947 -> #2E7D32)
- Added `Subtitle` field to `SelectorOpts` struct
- Added `subtitle` field to `SelectorModel` struct
- Rendered subtitle as muted text below title in selector View
- Updated headerLines calculation in both View and resizeViewport
- Wired subtitles in all three selector callers:
  - update: "5 pending of 12 submodules scanned"
  - push: "3 ahead of 12 submodules scanned"
  - exec: "12 submodules available"

## Files Modified
- `internal/cli/tui/progress.go`
- `internal/cli/tui/selector.go`
- `internal/cli/tui/tui.go`
- `internal/cli/update.go`
- `internal/cli/push.go`
- `internal/cli/exec.go`

## Commit
324e153
