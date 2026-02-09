# Quick Task 001: Remove Target Column + Add Progress Bar

## Changes
- Removed Target column from `printStatusTable` (5 columns: Path, Branch, Behind, Feature, Status)
- Reduced table width from 120 to 100
- Shifted status column styling index from col 5 to col 4
- Added `runScanWithProgress` call in `runStatus` when TTY and not --json
- JSON output unchanged (target_branch still present)

## Files Modified
- `internal/cli/status.go`

## Commit
18150b4
