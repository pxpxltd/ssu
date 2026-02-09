package backup

import "fmt"

// RollbackOpts configures a rollback operation.
type RollbackOpts struct {
	BackupPath  string // Path to the backup file to restore from
	ProjectRoot string // Git repository root directory
	BackupDir   string // Directory for creating safety backups
	DryRun      bool   // If true, only show what would be restored
}

// RollbackResult holds the outcome of a rollback operation.
type RollbackResult struct {
	SafetyBackupFile string              // Filename of safety backup created before restoring
	RestoredCount    int                 // Number of submodules successfully restored
	Submodules       []RestoredSubmodule // Per-submodule results
}

// RestoredSubmodule holds the result of restoring a single submodule.
type RestoredSubmodule struct {
	Path   string // Submodule relative path
	SHA    string // Target SHA that was (or would be) restored
	Branch string // Target branch that was (or would be) checked out
	Error  error  // Non-nil if this submodule failed to restore
}

// GetCurrentStatesFunc retrieves current submodule states for safety backup.
// It accepts the project root and a list of submodule paths, returning a map
// of path -> SubmoduleState. This is injected to keep the backup package
// independent from the git package.
type GetCurrentStatesFunc func(projectRoot string, paths []string) (map[string]SubmoduleState, error)

// GitCheckoutFunc checks out a branch in a directory.
type GitCheckoutFunc func(dir, branch string) error

// GitResetHardFunc resets a directory to a specific SHA.
type GitResetHardFunc func(dir, sha string) error

// Rollback restores submodules to the state recorded in a backup file.
//
// The process:
//  1. Read the backup file
//  2. Create a safety backup of the current state (unless dry-run)
//  3. For each submodule: checkout branch, then reset to SHA
//  4. Continue on per-submodule errors (the safety backup is the undo mechanism)
//
// Git operations are injected as function parameters to keep the backup package
// independent from the git package. They are wired together in Phase 5.
func Rollback(
	opts RollbackOpts,
	getCurrentStates GetCurrentStatesFunc,
	gitCheckout GitCheckoutFunc,
	gitResetHard GitResetHardFunc,
) (*RollbackResult, error) {
	// 1. Read the backup
	b, err := Read(opts.BackupPath)
	if err != nil {
		return nil, fmt.Errorf("reading backup: %w", err)
	}

	if len(b.Submodules) == 0 {
		return nil, fmt.Errorf("backup contains no submodules")
	}

	// Collect submodule paths
	paths := make([]string, 0, len(b.Submodules))
	for p := range b.Submodules {
		paths = append(paths, p)
	}

	result := &RollbackResult{}

	// 2. In dry-run mode, just list what would be restored
	if opts.DryRun {
		for _, p := range paths {
			state := b.Submodules[p]
			result.Submodules = append(result.Submodules, RestoredSubmodule{
				Path:   p,
				SHA:    state.SHA,
				Branch: state.Branch,
			})
		}
		return result, nil
	}

	// 3. Create a safety backup of current state before restoring
	if getCurrentStates != nil && opts.BackupDir != "" {
		currentStates, err := getCurrentStates(opts.ProjectRoot, paths)
		if err != nil {
			// Safety backup failure is a warning, not a blocker
			// The caller should log this but proceed
			result.SafetyBackupFile = fmt.Sprintf("(failed: %v)", err)
		} else {
			filename, err := Create(opts.BackupDir, currentStates)
			if err != nil {
				result.SafetyBackupFile = fmt.Sprintf("(failed: %v)", err)
			} else {
				result.SafetyBackupFile = filename
			}
		}
	}

	// 4. Restore each submodule
	for _, p := range paths {
		state := b.Submodules[p]
		sub := RestoredSubmodule{
			Path:   p,
			SHA:    state.SHA,
			Branch: state.Branch,
		}

		// Checkout the branch first (avoids detached HEAD)
		if gitCheckout != nil && state.Branch != "" {
			if err := gitCheckout(p, state.Branch); err != nil {
				// Branch might not exist -- try to continue with reset anyway
				sub.Error = fmt.Errorf("checkout %s: %w", state.Branch, err)
			}
		}

		// Reset to exact SHA
		if gitResetHard != nil {
			if err := gitResetHard(p, state.SHA); err != nil {
				sub.Error = fmt.Errorf("reset to %s: %w", state.SHA, err)
			} else if sub.Error != nil {
				// Checkout failed but reset succeeded -- clear error
				// (detached HEAD at correct SHA is acceptable)
				sub.Error = nil
			}
		}

		if sub.Error == nil {
			result.RestoredCount++
		}

		result.Submodules = append(result.Submodules, sub)
	}

	return result, nil
}
