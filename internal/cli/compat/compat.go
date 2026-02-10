// Package compat detects old-style SSU flags (from the bash version) and
// prints friendly migration hints pointing users to the new subcommand syntax.
package compat

import (
	"fmt"
	"io"
)

// OldFlagHints maps deprecated top-level flags to their new subcommand equivalents.
var OldFlagHints = map[string]string{
	"--status":   "ssu status",
	"--push":     "ssu push",
	"--rollback": "ssu rollback",
	"--auto":     "ssu update --auto  (or: ssu push --auto)",
	"--dry-run":  "ssu update --dry-run  (or: ssu status)",
}

// CheckOldFlags inspects the first argument after the binary name. If it
// matches a deprecated flag, a migration hint is printed to w and the
// function returns true. Only args[1] is checked so that valid subcommand
// flags (e.g. "ssu update --dry-run") are not falsely flagged.
func CheckOldFlags(args []string, w io.Writer) bool {
	if len(args) < 2 {
		return false
	}

	suggestion, ok := OldFlagHints[args[1]]
	if !ok {
		return false
	}

	fmt.Fprintf(w, "Hint: Did you mean `%s`?\n", suggestion)
	fmt.Fprintln(w, "Run `ssu help` for the new command syntax.")
	return true
}
