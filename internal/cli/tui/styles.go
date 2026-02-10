// Package tui provides shared lipgloss styles and bubbletea components for SSU.
//
// These styles are used inside bubbletea Views (lipgloss only, NOT fatih/color).
// Outside bubbletea (--auto mode, non-TUI), the existing output package with
// fatih/color is used instead.
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/pxpxltd/ssu/internal/git"
)

// Status-specific styles matching the bash version semantics from output/color.go.
var (
	StatusPendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))   // green
	StatusCurrentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))   // cyan
	StatusModifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))   // yellow
	StatusAheadStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))   // magenta
	StatusConflictStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))   // red
	StatusErrorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true) // red bold
	StatusSkippedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray
	StatusMissingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray
)

// General-purpose styles shared across all TUI components.
var (
	RootPathStyle = lipgloss.NewStyle().Bold(true)
	HeaderStyle   = lipgloss.NewStyle().Bold(true)
	MutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	TitleStyle    = lipgloss.NewStyle().Bold(true).Underline(true)
)

// StyleForStatus returns the lipgloss style for the given submodule status.
func StyleForStatus(s git.SubmoduleStatus) lipgloss.Style {
	switch s {
	case git.StatusPending:
		return StatusPendingStyle
	case git.StatusCurrent:
		return StatusCurrentStyle
	case git.StatusModified:
		return StatusModifiedStyle
	case git.StatusAhead:
		return StatusAheadStyle
	case git.StatusConflict:
		return StatusConflictStyle
	case git.StatusError:
		return StatusErrorStyle
	case git.StatusSkipped:
		return StatusSkippedStyle
	case git.StatusMissing:
		return StatusMissingStyle
	default:
		return lipgloss.NewStyle()
	}
}
