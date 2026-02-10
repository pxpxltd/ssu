package tui

import (
	"fmt"
	"strings"

	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// ---------------------------------------------------------------------------
// SelectorItem interface
// ---------------------------------------------------------------------------

// SelectorItem is the interface each item in the multi-select list must satisfy.
type SelectorItem interface {
	Path() string          // Submodule path (used for filtering)
	Label() string         // Display label
	Metadata() string      // Branch/behind/status badge string
	DetailContent() string // Content for the split-pane detail view
}

// ---------------------------------------------------------------------------
// SubmoduleItem wraps *engine.SubmoduleInfo to implement SelectorItem.
// ---------------------------------------------------------------------------

// SubmoduleItem adapts engine.SubmoduleInfo to the SelectorItem interface.
type SubmoduleItem struct {
	Info *engine.SubmoduleInfo
}

// Path returns the submodule path.
func (s SubmoduleItem) Path() string { return s.Info.Path }

// Label returns the submodule path for display.
func (s SubmoduleItem) Label() string { return s.Info.Path }

// Metadata returns a formatted string showing branch, behind count, and status badge.
func (s SubmoduleItem) Metadata() string {
	status := s.Info.PrimaryStatus()
	style := StyleForStatus(status)

	parts := []string{s.Info.CurrentBranch}

	if s.Info.CommitsBehind > 0 {
		parts = append(parts, fmt.Sprintf("-%d", s.Info.CommitsBehind))
	}
	if s.Info.CommitsAhead > 0 {
		parts = append(parts, fmt.Sprintf("+%d", s.Info.CommitsAhead))
	}

	badge := style.Render(string(status))
	parts = append(parts, badge)

	return strings.Join(parts, " ")
}

// DetailContent returns the changelog lines for the detail pane.
// Returns "No incoming changes" when the Changelog slice is empty.
func (s SubmoduleItem) DetailContent() string {
	if len(s.Info.Changelog) == 0 {
		return "No incoming changes"
	}
	return strings.Join(s.Info.Changelog, "\n")
}

// ---------------------------------------------------------------------------
// SelectorOpts configures the SelectorModel.
// ---------------------------------------------------------------------------

// SelectorOpts configures the selector model.
type SelectorOpts struct {
	Title      string                       // Title displayed at the top
	Subtitle   string                       // Informational line below the title
	ShowDetail bool                         // Show detail pane by default
	FilterFn   func(item SelectorItem) bool // Optional pre-filter
	Operation  string                       // Operation name for confirmation (e.g. "update", "push")
	SelectAll  bool                         // Pre-select all items (user deselects unwanted)
}

// ---------------------------------------------------------------------------
// Shared tea.Msg types for cross-model communication.
// ---------------------------------------------------------------------------

// ScanCompleteMsg is sent when a scan operation completes.
type ScanCompleteMsg struct {
	Result *engine.ScanResult
	Err    error
}

// UpdateProgressMsg is sent for each submodule update progress event.
type UpdateProgressMsg struct {
	Path  string
	Done  int
	Total int
	Err   error
}

// UpdateCompleteMsg is sent when an update operation completes.
type UpdateCompleteMsg struct {
	Result *engine.UpdateResult
	Err    error
}

// PushCompleteMsg is sent when a push operation completes.
type PushCompleteMsg struct {
	Result *engine.PushResult
	Err    error
}

// ---------------------------------------------------------------------------
// Helpers for converting engine types to SelectorItems.
// ---------------------------------------------------------------------------

// SubmoduleItems converts a slice of SubmoduleInfo pointers to SelectorItems.
func SubmoduleItems(infos []*engine.SubmoduleInfo) []SelectorItem {
	items := make([]SelectorItem, len(infos))
	for i, info := range infos {
		items[i] = SubmoduleItem{Info: info}
	}
	return items
}

// FilterPending returns a FilterFn that keeps only items with pending status.
func FilterPending() func(SelectorItem) bool {
	return func(item SelectorItem) bool {
		si, ok := item.(SubmoduleItem)
		if !ok {
			return true
		}
		return si.Info.HasStatus(git.StatusPending)
	}
}

// FilterAhead returns a FilterFn that keeps only items with ahead status.
func FilterAhead() func(SelectorItem) bool {
	return func(item SelectorItem) bool {
		si, ok := item.(SubmoduleItem)
		if !ok {
			return true
		}
		return si.Info.HasStatus(git.StatusAhead)
	}
}
