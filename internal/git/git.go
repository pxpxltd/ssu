// Package git defines the GitService interface and all types for SSU's git layer.
//
// GitService is the single abstraction through which the Engine and Commands
// interact with git. Two implementations exist: ExecGit (production, os/exec)
// and MockGitService (testing, configurable stubs).
package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// GitService abstracts all git operations SSU needs.
// Implementations: ExecGit (production), MockGitService (testing).
type GitService interface {
	// Repository discovery
	SubmodulePaths(ctx context.Context, rootDir string) ([]string, error)
	IsSubmoduleInitialized(rootDir, subPath string) bool
	SubmoduleInit(ctx context.Context, rootDir string, paths []string) error

	// Branch and revision queries
	CurrentBranch(ctx context.Context, dir string) (BranchResult, error)
	CurrentSHA(ctx context.Context, dir string) (string, error)
	IsDetachedHead(ctx context.Context, dir string) (bool, error)
	RemoteBranches(ctx context.Context, dir string) ([]RemoteBranch, error)
	CommitsBehind(ctx context.Context, dir, localRef, remoteRef string) (int, error)
	CommitsAhead(ctx context.Context, dir, remoteRef string) (int, error)
	HasRemoteBranch(ctx context.Context, dir, remote, branch string) (bool, error)
	SymbolicRef(ctx context.Context, dir, ref string) (string, error)
	TrackingBranch(ctx context.Context, dir string) (TrackingInfo, error)

	// Status queries
	HasLocalChanges(ctx context.Context, dir string) (bool, error)
	IncomingChangelog(ctx context.Context, dir, remoteRef string, limit int) ([]string, error)

	// Mutating operations
	Fetch(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error)
	Checkout(ctx context.Context, dir, branch string) (CheckoutResult, error)
	Merge(ctx context.Context, dir, ref string) (MergeResult, error)
	Push(ctx context.Context, dir string, opts PushOpts) (PushResult, error)
	Stash(ctx context.Context, dir string) (StashResult, error)
	StashPop(ctx context.Context, dir string) (StashResult, error)
	MergeAbort(ctx context.Context, dir string) error
	ResetHard(ctx context.Context, dir, ref string) error

	// Branch queries
	BranchesPointingAt(ctx context.Context, dir, ref string) (BranchPointsAtResult, error)

	// Submodule removal
	SubmoduleDeinit(ctx context.Context, rootDir, subPath string) error
	RemovePath(ctx context.Context, rootDir, path string) error

	// Root repository operations
	DiffIndex(ctx context.Context, dir string) ([]string, error)
	StageFiles(ctx context.Context, dir string, paths []string) error
	Commit(ctx context.Context, dir, message string) (CommitResult, error)
	ListTags(ctx context.Context, dir string, limit int) ([]string, error)
	CreateTag(ctx context.Context, dir string, opts TagOpts) error
}

// ---------------------------------------------------------------------------
// Result types -- every result carries Stderr for verbose/debug output.
// ---------------------------------------------------------------------------

// BranchResult holds the outcome of a current-branch query.
type BranchResult struct {
	Name     string // Branch name, or empty if detached
	Detached bool   // True when HEAD is detached
	Stderr   string
}

// TrackingInfo describes the upstream tracking branch for a local branch.
type TrackingInfo struct {
	Remote string // e.g. "origin"
	Branch string // e.g. "develop"
	Set    bool   // False if no tracking branch is configured
	Stderr string
}

// RemoteBranch represents a branch on a remote. This is a data type,
// not an operation result, so it does not carry Stderr.
type RemoteBranch struct {
	Remote string // e.g. "origin"
	Branch string // e.g. "develop"
}

// FetchResult holds the outcome of a git fetch.
type FetchResult struct {
	Stderr string
}

// CheckoutResult holds the outcome of a git checkout.
type CheckoutResult struct {
	Branch string
	Stderr string
}

// MergeResult holds the outcome of a git merge.
type MergeResult struct {
	Success  bool
	Conflict bool // True if merge failed due to conflict
	Stderr   string
}

// PushResult holds the outcome of a git push.
type PushResult struct {
	Remote      string // Which remote was pushed to
	Branch      string // Which branch was pushed
	NewTracking bool   // True if -u was used to set up tracking
	Stderr      string
}

// StashResult holds the outcome of a stash or stash pop operation.
type StashResult struct {
	Created bool // For Stash: true if a stash was created; for StashPop: true if pop was applied
	Stderr  string
}

// ---------------------------------------------------------------------------
// Option types
// ---------------------------------------------------------------------------

// FetchOpts configures a git fetch operation.
type FetchOpts struct {
	Remote string // Empty means all remotes
	Prune  bool
}

// PushOpts configures a git push operation.
type PushOpts struct {
	Remote      string
	Branch      string
	SetUpstream bool
}

// CommitResult holds the outcome of a git commit.
type CommitResult struct {
	SHA    string // Short SHA of the new commit
	Stderr string
}

// TagOpts configures a git tag creation.
type TagOpts struct {
	Name    string // Tag name (e.g. "v1.2.3")
	Message string // Annotation message
}

// BranchDetectOpts configures the smart branch detection algorithm.
type BranchDetectOpts struct {
	Override         string   // --branch CLI flag (highest priority)
	PriorityBranches []string // Default: ["develop", "master", "main"]
	DefaultRemote    string   // Default: "origin"
}

// BranchCheckoutOpts configures branch resolution for checkout.
type BranchCheckoutOpts struct {
	PriorityBranches []string // Default: ["develop", "master", "main"]
	DefaultRemote    string   // Default: "origin"
}

// BranchPointsAtResult holds the branches whose tip equals a given ref.
type BranchPointsAtResult struct {
	Local  []string       // Local branch names (e.g. "develop", "feature/foo")
	Remote []RemoteBranch // Remote branches (e.g. origin/develop)
}

// DefaultBranchPriority is the default branch priority chain for detection.
var DefaultBranchPriority = []string{"develop", "master", "main"}

// ---------------------------------------------------------------------------
// Status enum
// ---------------------------------------------------------------------------

// SubmoduleStatus represents the state of a submodule.
type SubmoduleStatus string

const (
	StatusPending  SubmoduleStatus = "pending"
	StatusCurrent  SubmoduleStatus = "current"
	StatusModified SubmoduleStatus = "modified"
	StatusAhead    SubmoduleStatus = "ahead"
	StatusConflict SubmoduleStatus = "conflict"
	StatusMissing  SubmoduleStatus = "missing"
	StatusSkipped  SubmoduleStatus = "skipped"
	StatusError    SubmoduleStatus = "error"
)

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

// GitError wraps a git operation failure with context.
type GitError struct {
	Op     string // Operation name: "fetch", "push", "merge", etc.
	Stderr string // Raw stderr output from git
	Err    error  // Underlying error (exec.ExitError, context.DeadlineExceeded, etc.)
}

// Error returns a human-readable error message.
func (e *GitError) Error() string {
	msg := fmt.Sprintf("git %s: %v", e.Op, e.Err)
	if trimmed := strings.TrimSpace(e.Stderr); trimmed != "" {
		msg += "\nstderr: " + trimmed
	}
	return msg
}

// Unwrap returns the underlying error for use with errors.Is and errors.As.
func (e *GitError) Unwrap() error {
	return e.Err
}

// IsTimeout reports whether err is or wraps a context deadline exceeded error.
func IsTimeout(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// IsConflict reports whether err represents a git merge conflict.
func IsConflict(err error) bool {
	var ge *GitError
	if errors.As(err, &ge) {
		return ge.Op == "merge" && strings.Contains(ge.Stderr, "CONFLICT")
	}
	return false
}
