// Package output provides color-aware terminal output utilities for SSU.
//
// Color definitions are centralized here. fatih/color handles NO_COLOR,
// TERM=dumb, and non-TTY detection automatically -- no custom logic needed.
package output

import (
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// General-purpose colors.
var (
	Success = color.New(color.FgGreen)
	Error   = color.New(color.FgRed)
	Warning = color.New(color.FgYellow)
	Info    = color.New(color.FgCyan)
	Muted   = color.New(color.FgHiBlack) // Gray for secondary text
	Bold    = color.New(color.Bold)
)

// Status-specific colors matching the bash version semantics.
var (
	Pending  = color.New(color.FgGreen)   // Submodule needs update
	Current  = color.New(color.FgCyan)    // Up to date
	Modified = color.New(color.FgYellow)  // Has local changes
	Ahead    = color.New(color.FgMagenta) // Unpushed commits
	Conflict = color.New(color.FgRed)     // Merge conflict
)

// IsColorEnabled returns whether color output is active.
// fatih/color handles NO_COLOR, TERM=dumb, and non-TTY automatically.
func IsColorEnabled() bool {
	return !color.NoColor
}

// IsTTY returns whether stdout is connected to a terminal.
// This uses mattn/go-isatty for accurate detection, not just color.NoColor
// (a pipe with NO_COLOR unset should still detect non-TTY).
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// ForceDisableColor explicitly turns off all color output.
func ForceDisableColor() {
	color.NoColor = true
}
