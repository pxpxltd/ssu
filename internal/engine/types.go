// Package engine implements the core scan/update/push workflows for SSU.
//
// The engine orchestrates git operations through the GitService interface,
// providing parallel execution with bounded concurrency, progress reporting,
// and compound status detection for submodules.
package engine

import "github.com/pxpxltd/ssu/internal/git"

// ---------------------------------------------------------------------------
// Scan types
// ---------------------------------------------------------------------------

// ScanOpts configures a Scan operation.
type ScanOpts struct {
	RootDir     string               // Absolute path to the project root
	SkipList    []string             // Submodule paths to exclude
	Concurrency int                  // Max parallel goroutines (0 = runtime.NumCPU())
	BranchOpts  git.BranchDetectOpts // Passed to DetectBestBranch per submodule
	OnProgress  ProgressFunc         // Optional callback for progress events
}

// ScanResult holds the aggregate outcome of scanning all submodules.
type ScanResult struct {
	Root       *SubmoduleInfo   // Root repository info (display-only, nil if scan failed)
	Submodules []*SubmoduleInfo // Per-submodule scan results, sorted by path
}

// SubmoduleInfo holds scan results for a single submodule (or root).
type SubmoduleInfo struct {
	Path          string                // Relative path from root (e.g. "plugins/auth"), or "." for root
	IsRoot        bool                  // True for root repository
	Statuses      []git.SubmoduleStatus // Compound status set (can have multiple)
	CurrentBranch string                // Branch name, or empty if detached
	TargetBranch  string                // Branch detected by DetectBestBranch
	DetachedHead  bool                  // True if HEAD is detached
	CommitsBehind int                   // Number of commits behind remote
	CommitsAhead  int                   // Number of unpushed commits
	HasChanges    bool                  // Uncommitted local changes exist
	Changelog     []string              // Incoming commit summaries (oneline format)
	Error         error                 // Non-nil if scan failed for this submodule
}

// statusPriority maps each status to a priority value for display ordering.
// Lower number = higher priority (displayed first).
var statusPriority = map[git.SubmoduleStatus]int{
	git.StatusError:    0,
	git.StatusConflict: 1,
	git.StatusModified: 2,
	git.StatusAhead:    3,
	git.StatusPending:  4,
	git.StatusMissing:  5,
	git.StatusSkipped:  6,
	git.StatusCurrent:  7,
}

// HasStatus reports whether info has the given status in its compound set.
func (info *SubmoduleInfo) HasStatus(s git.SubmoduleStatus) bool {
	for _, st := range info.Statuses {
		if st == s {
			return true
		}
	}
	return false
}

// PrimaryStatus returns the highest-priority status for display.
// Priority order: error > conflict > modified > ahead > pending > missing > skipped > current.
func (info *SubmoduleInfo) PrimaryStatus() git.SubmoduleStatus {
	if len(info.Statuses) == 0 {
		return git.StatusCurrent
	}
	best := info.Statuses[0]
	bestPrio := statusPriority[best]
	for _, s := range info.Statuses[1:] {
		p := statusPriority[s]
		if p < bestPrio {
			best = s
			bestPrio = p
		}
	}
	return best
}

// ---------------------------------------------------------------------------
// Update types (used by Plan 02)
// ---------------------------------------------------------------------------

// UpdateOpts configures an Update operation.
type UpdateOpts struct {
	RootDir     string
	Concurrency int
	OnProgress  ProgressFunc
}

// UpdateResult holds the aggregate outcome of updating submodules.
type UpdateResult struct {
	Actions []UpdateAction
}

// UpdateAction records what happened to a single submodule during update.
type UpdateAction struct {
	Path         string
	BeforeStatus []git.SubmoduleStatus
	AfterStatus  []git.SubmoduleStatus
	Action       string // e.g. "merged", "conflict-resolved", "skipped", "error"
	Error        error
	ConflictHint string // Human-readable hint if conflict occurred
}

// ---------------------------------------------------------------------------
// Push types (used by Plan 03)
// ---------------------------------------------------------------------------

// PushOpts configures a Push operation.
type PushOpts struct {
	RootDir     string
	Concurrency int
	OnProgress  ProgressFunc
}

// PushResult holds the aggregate outcome of pushing submodules.
type PushResult struct {
	Actions []PushAction
}

// PushAction records what happened to a single submodule during push.
type PushAction struct {
	Path   string
	Branch string
	Action string // e.g. "pushed", "set-upstream", "skipped", "error"
	Error  error
}

// ---------------------------------------------------------------------------
// Checkout types
// ---------------------------------------------------------------------------

// CheckoutOpts configures a Checkout operation.
type CheckoutOpts struct {
	RootDir     string
	Concurrency int
	BranchOpts  git.BranchCheckoutOpts
	OnProgress  ProgressFunc
}

// CheckoutResult holds the aggregate outcome of checking out submodules.
type CheckoutResult struct {
	Actions []CheckoutAction
}

// CheckoutAction records what happened to a single submodule during checkout.
type CheckoutAction struct {
	Path   string
	Branch string // Branch checked out (or empty if skipped)
	Action string // e.g. "checked out develop", "skipped (not detached)", "skipped (no matching branch)"
	Error  error
}
