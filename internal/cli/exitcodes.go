package cli

import "fmt"

// Exit code constants for consistent process exit status.
const (
	// ExitSuccess indicates the operation completed successfully.
	ExitSuccess = 0

	// ExitError indicates a general error (invalid args, git failure, etc.).
	ExitError = 1

	// ExitConflict indicates a merge conflict was detected during update.
	ExitConflict = 2
)

// exitError implements error and carries a process exit code.
// Used by update and push commands to signal non-zero exit.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}
