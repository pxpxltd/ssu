package cli

// Exit code constants for consistent process exit status.
const (
	// ExitSuccess indicates the operation completed successfully.
	ExitSuccess = 0

	// ExitError indicates a general error (invalid args, git failure, etc.).
	ExitError = 1

	// ExitConflict indicates a merge conflict was detected during update.
	ExitConflict = 2
)
