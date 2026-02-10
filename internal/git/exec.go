package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Compile-time interface check.
var _ GitService = (*ExecGit)(nil)

// ExecGit implements GitService by shelling out to the git binary via os/exec.
// Every command uses context.WithTimeout for robust timeout handling and sets
// GIT_TERMINAL_PROMPT=0 to prevent credential prompt hangs.
type ExecGit struct {
	GitBin   string        // Path to git binary; defaults to "git" if empty.
	Timeouts TimeoutConfig // Per-operation timeout configuration.
}

// TimeoutConfig holds per-operation timeout durations.
type TimeoutConfig struct {
	Fetch   time.Duration // Default 120s -- network bound.
	Push    time.Duration // Default 60s -- network bound.
	Merge   time.Duration // Default 30s -- local operation.
	Default time.Duration // Default 30s -- for queries like rev-parse, branch -r.
}

// DefaultTimeouts returns a TimeoutConfig with sensible defaults.
func DefaultTimeouts() TimeoutConfig {
	return TimeoutConfig{
		Fetch:   120 * time.Second,
		Push:    60 * time.Second,
		Merge:   30 * time.Second,
		Default: 30 * time.Second,
	}
}

// NewExecGit returns an ExecGit with default timeouts and system git.
func NewExecGit() *ExecGit {
	return &ExecGit{Timeouts: DefaultTimeouts()}
}

// ---------------------------------------------------------------------------
// Core run helper
// ---------------------------------------------------------------------------

// run executes a git command in the given directory with timeout handling.
// It returns trimmed stdout, raw stderr, and a wrapped error.
func (g *ExecGit) run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, string, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	bin := g.GitBin
	if bin == "" {
		bin = "git"
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout := strings.TrimSpace(outBuf.String())
	stderr := errBuf.String()

	if err != nil {
		op := ""
		if len(args) > 0 {
			op = args[0]
		}
		return stdout, stderr, &GitError{Op: op, Stderr: stderr, Err: err}
	}
	return stdout, stderr, nil
}

// ---------------------------------------------------------------------------
// Repository discovery
// ---------------------------------------------------------------------------

func (g *ExecGit) SubmodulePaths(ctx context.Context, rootDir string) ([]string, error) {
	stdout, _, err := g.run(ctx, rootDir, g.Timeouts.Default,
		"config", "--file", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
	if err != nil {
		// No .gitmodules or no submodules configured -- not an error for callers.
		if isGitExitError(err) {
			return nil, nil
		}
		return nil, err
	}
	if stdout == "" {
		return nil, nil
	}

	var paths []string
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			paths = append(paths, fields[1])
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func (g *ExecGit) IsSubmoduleInitialized(rootDir, subPath string) bool {
	info, err := os.Stat(filepath.Join(rootDir, subPath, ".git"))
	if err != nil {
		return false
	}
	// .git can be a file (gitlink) or directory.
	return info != nil
}

// ---------------------------------------------------------------------------
// Branch and revision queries
// ---------------------------------------------------------------------------

func (g *ExecGit) CurrentBranch(ctx context.Context, dir string) (BranchResult, error) {
	stdout, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return BranchResult{Stderr: stderr}, err
	}
	if stdout == "HEAD" {
		return BranchResult{Detached: true, Stderr: stderr}, nil
	}
	return BranchResult{Name: stdout, Stderr: stderr}, nil
}

func (g *ExecGit) CurrentSHA(ctx context.Context, dir string) (string, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (g *ExecGit) IsDetachedHead(ctx context.Context, dir string) (bool, error) {
	_, _, err := g.run(ctx, dir, g.Timeouts.Default, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		// Exit code 1 means detached -- not a real error.
		if isGitExitError(err) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (g *ExecGit) RemoteBranches(ctx context.Context, dir string) ([]RemoteBranch, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "branch", "-r")
	if err != nil {
		return nil, err
	}
	if stdout == "" {
		return nil, nil
	}

	var branches []RemoteBranch
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		// Format: "origin/develop" or "upstream/feature/foo"
		idx := strings.Index(line, "/")
		if idx < 0 {
			continue
		}
		branches = append(branches, RemoteBranch{
			Remote: line[:idx],
			Branch: line[idx+1:],
		})
	}
	return branches, nil
}

func (g *ExecGit) CommitsBehind(ctx context.Context, dir, localRef, remoteRef string) (int, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "rev-list", "--count", localRef+".."+remoteRef)
	if err != nil {
		return 0, nil // Match bash behavior: return 0 on error.
	}
	n, parseErr := strconv.Atoi(stdout)
	if parseErr != nil {
		return 0, nil
	}
	return n, nil
}

func (g *ExecGit) CommitsAhead(ctx context.Context, dir, remoteRef string) (int, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "rev-list", "--count", remoteRef+"..HEAD")
	if err != nil {
		return 0, nil // Match bash behavior: return 0 on error.
	}
	n, parseErr := strconv.Atoi(stdout)
	if parseErr != nil {
		return 0, nil
	}
	return n, nil
}

func (g *ExecGit) HasRemoteBranch(ctx context.Context, dir, remote, branch string) (bool, error) {
	_, _, err := g.run(ctx, dir, g.Timeouts.Default, "show-ref", "--verify", fmt.Sprintf("refs/remotes/%s/%s", remote, branch))
	if err != nil {
		if isGitExitError(err) {
			return false, nil // Exit non-zero means ref doesn't exist.
		}
		return false, err
	}
	return true, nil
}

func (g *ExecGit) SymbolicRef(ctx context.Context, dir, ref string) (string, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "symbolic-ref", ref)
	if err != nil {
		return "", err
	}
	return stdout, nil
}

func (g *ExecGit) TrackingBranch(ctx context.Context, dir string) (TrackingInfo, error) {
	// First get current branch name.
	branchOut, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return TrackingInfo{Stderr: stderr}, err
	}
	if branchOut == "HEAD" {
		return TrackingInfo{Set: false, Stderr: stderr}, nil
	}

	// Get remote name.
	remote, remoteSterr, remoteErr := g.run(ctx, dir, g.Timeouts.Default,
		"config", "--get", fmt.Sprintf("branch.%s.remote", branchOut))
	combinedStderr := stderr + remoteSterr
	if remoteErr != nil {
		return TrackingInfo{Set: false, Stderr: combinedStderr}, nil
	}

	// Get merge ref.
	merge, mergeSterr, mergeErr := g.run(ctx, dir, g.Timeouts.Default,
		"config", "--get", fmt.Sprintf("branch.%s.merge", branchOut))
	combinedStderr += mergeSterr
	if mergeErr != nil {
		return TrackingInfo{Set: false, Stderr: combinedStderr}, nil
	}

	// Parse merge ref: strip "refs/heads/" prefix.
	branch := strings.TrimPrefix(merge, "refs/heads/")

	return TrackingInfo{
		Remote: remote,
		Branch: branch,
		Set:    true,
		Stderr: combinedStderr,
	}, nil
}

// ---------------------------------------------------------------------------
// Status queries
// ---------------------------------------------------------------------------

func (g *ExecGit) HasLocalChanges(ctx context.Context, dir string) (bool, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return len(stdout) > 0, nil
}

func (g *ExecGit) IncomingChangelog(ctx context.Context, dir, remoteRef string, limit int) ([]string, error) {
	stdout, _, err := g.run(ctx, dir, g.Timeouts.Default,
		"log", "--oneline", "-n", strconv.Itoa(limit), ".."+remoteRef)
	if err != nil {
		return nil, err
	}
	if stdout == "" {
		return nil, nil
	}
	return strings.Split(stdout, "\n"), nil
}

// ---------------------------------------------------------------------------
// Mutating operations
// ---------------------------------------------------------------------------

func (g *ExecGit) Fetch(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error) {
	args := []string{"fetch"}
	if opts.Remote != "" {
		args = append(args, opts.Remote)
	} else {
		args = append(args, "--all")
	}
	if opts.Prune {
		args = append(args, "--prune")
	}

	_, stderr, err := g.run(ctx, dir, g.Timeouts.Fetch, args...)
	if err != nil {
		return FetchResult{Stderr: stderr}, err
	}
	return FetchResult{Stderr: stderr}, nil
}

func (g *ExecGit) Checkout(ctx context.Context, dir, branch string) (CheckoutResult, error) {
	_, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "checkout", branch)
	if err != nil {
		return CheckoutResult{Stderr: stderr}, err
	}
	return CheckoutResult{Branch: branch, Stderr: stderr}, nil
}

func (g *ExecGit) Merge(ctx context.Context, dir, ref string) (MergeResult, error) {
	stdout, stderr, err := g.run(ctx, dir, g.Timeouts.Merge, "merge", ref)
	if err != nil {
		// Git writes CONFLICT markers to stdout, not stderr.
		if strings.Contains(stdout, "CONFLICT") || strings.Contains(stderr, "CONFLICT") {
			return MergeResult{Conflict: true, Stderr: stderr}, err
		}
		return MergeResult{Stderr: stderr}, err
	}
	return MergeResult{Success: true, Stderr: stderr}, nil
}

func (g *ExecGit) Push(ctx context.Context, dir string, opts PushOpts) (PushResult, error) {
	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	setUpstream := opts.SetUpstream

	// If SetUpstream not explicitly requested, check if tracking branch exists.
	if !setUpstream {
		tracking, _ := g.TrackingBranch(ctx, dir)
		if !tracking.Set {
			setUpstream = true
		}
	}

	branch := opts.Branch
	if branch == "" {
		// Detect current branch for the push.
		br, err := g.CurrentBranch(ctx, dir)
		if err != nil {
			return PushResult{}, err
		}
		branch = br.Name
	}

	args := []string{"push"}
	if setUpstream {
		args = append(args, "-u")
	}
	args = append(args, remote, branch)

	_, stderr, err := g.run(ctx, dir, g.Timeouts.Push, args...)
	if err != nil {
		return PushResult{Remote: remote, Branch: branch, Stderr: stderr}, err
	}
	return PushResult{
		Remote:      remote,
		Branch:      branch,
		NewTracking: setUpstream,
		Stderr:      stderr,
	}, nil
}

func (g *ExecGit) Stash(ctx context.Context, dir string) (StashResult, error) {
	stdout, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "stash", "push", "-m", "ssu-auto-stash")
	if err != nil {
		return StashResult{Stderr: stderr}, err
	}
	if strings.Contains(stdout, "No local changes") {
		return StashResult{Created: false, Stderr: stderr}, nil
	}
	return StashResult{Created: true, Stderr: stderr}, nil
}

func (g *ExecGit) StashPop(ctx context.Context, dir string) (StashResult, error) {
	_, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "stash", "pop")
	if err != nil {
		return StashResult{Stderr: stderr}, err
	}
	return StashResult{Created: true, Stderr: stderr}, nil
}

func (g *ExecGit) MergeAbort(ctx context.Context, dir string) error {
	_, _, err := g.run(ctx, dir, g.Timeouts.Default, "merge", "--abort")
	return err
}

func (g *ExecGit) ResetHard(ctx context.Context, dir, ref string) error {
	_, _, err := g.run(ctx, dir, g.Timeouts.Default, "reset", "--hard", ref)
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isGitExitError reports whether err is a *GitError wrapping an exec.ExitError.
// This distinguishes normal git exit codes (e.g. show-ref not found) from
// unexpected failures (e.g. binary not found, timeout).
func isGitExitError(err error) bool {
	var ge *GitError
	if !errors.As(err, &ge) {
		return false
	}
	var exitErr *exec.ExitError
	return errors.As(ge.Err, &exitErr)
}
