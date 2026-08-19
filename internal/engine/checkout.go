package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pxpxltd/ssu/internal/git"
)

// Checkout re-attaches HEAD to the correct branch for selected submodules.
// Only submodules in detached HEAD state are processed. It finds which branch
// tips match HEAD and checks out the best match. This is completely safe:
// only branches whose tip equals HEAD are considered, so no commits are
// gained or lost.
func (e *Engine) Checkout(ctx context.Context, targets []*SubmoduleInfo, opts CheckoutOpts) (*CheckoutResult, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Filter out root -- root is display-only, never processed by engine.
	var filtered []*SubmoduleInfo
	for _, info := range targets {
		if !info.IsRoot {
			filtered = append(filtered, info)
		}
	}

	total := len(filtered)

	fire := func(evt ProgressEvent) {
		if opts.OnProgress != nil {
			opts.OnProgress(evt)
		}
	}

	var mu sync.Mutex
	var actions []CheckoutAction
	done := 0

	var g errgroup.Group
	g.SetLimit(concurrency)

	for _, info := range filtered {
		info := info

		g.Go(func() error {
			fire(ProgressEvent{Type: EventStarted, Path: info.Path, Phase: "checkout", Total: total, Done: done})

			var action CheckoutAction
			if opts.Reset {
				action = e.checkoutOneReset(ctx, opts, info)
			} else {
				action = e.checkoutOne(ctx, opts.RootDir, info, opts.BranchOpts)
			}

			mu.Lock()
			done++
			d := done
			actions = append(actions, action)
			mu.Unlock()

			if action.Error != nil {
				fire(ProgressEvent{Type: EventFailed, Path: info.Path, Phase: "checkout", Error: action.Error, Total: total, Done: d, Action: action.Action})
			} else {
				fire(ProgressEvent{Type: EventCompleted, Path: info.Path, Phase: "checkout", Total: total, Done: d, Action: action.Action})
			}

			return nil // continue-on-error
		})
	}

	g.Wait()

	return &CheckoutResult{Actions: actions}, nil
}

// checkoutOne resolves and checks out the best branch for a single submodule.
func (e *Engine) checkoutOne(ctx context.Context, rootDir string, info *SubmoduleInfo, branchOpts git.BranchCheckoutOpts) CheckoutAction {
	dir := filepath.Join(rootDir, info.Path)

	// Skip if not detached -- already on a branch.
	if !info.DetachedHead {
		return CheckoutAction{
			Path:   info.Path,
			Branch: info.CurrentBranch,
			Action: "skipped (not detached)",
		}
	}

	// Find best branch matching HEAD.
	branch, _, err := git.ResolveBranchForCheckout(ctx, e.git, dir, branchOpts)
	if err != nil {
		return CheckoutAction{
			Path:   info.Path,
			Action: "checkout failed",
			Error:  err,
		}
	}

	if branch == "" {
		return CheckoutAction{
			Path:   info.Path,
			Action: "skipped (no matching branch)",
		}
	}

	// Checkout the branch. Git DWIM handles creating a local tracking
	// branch from remote if only origin/<branch> exists.
	result, err := e.git.Checkout(ctx, dir, branch)
	if err != nil {
		return CheckoutAction{
			Path:   info.Path,
			Branch: branch,
			Action: fmt.Sprintf("checkout %s failed", branch),
			Error:  err,
		}
	}

	return CheckoutAction{
		Path:   info.Path,
		Branch: result.Branch,
		Action: fmt.Sprintf("checked out %s", result.Branch),
	}
}

// checkoutOneReset handles a single submodule in --reset mode.
// It compares the current SHA against the recorded SHA and resolves
// a named branch at the recorded SHA if possible.
func (e *Engine) checkoutOneReset(ctx context.Context, opts CheckoutOpts, info *SubmoduleInfo) CheckoutAction {
	dir := filepath.Join(opts.RootDir, info.Path)

	// A submodule whose scan failed carries no trustworthy state: DetachedHead
	// and CurrentBranch were never determined, and the recorded commit may not
	// even be in its object store if the fetch is what failed. Report the scan
	// error rather than acting on missing information and surfacing a confusing
	// checkout failure in its place.
	if info.Error != nil {
		return CheckoutAction{
			Path:   info.Path,
			Action: "skipped (scan failed)",
			Error:  info.Error,
		}
	}

	recordedSHA, ok := opts.RecordedSHAs[info.Path]
	if !ok {
		return CheckoutAction{
			Path:   info.Path,
			Action: "skipped (not in recorded SHAs)",
		}
	}

	// Get current SHA.
	currentSHA := info.CurrentSHA
	if currentSHA == "" {
		var err error
		currentSHA, err = e.git.CurrentSHA(ctx, dir)
		if err != nil {
			return CheckoutAction{
				Path:   info.Path,
				Action: "error getting current SHA",
				Error:  err,
			}
		}
	}

	// Already at the recorded SHA and on a branch -- nothing to do.
	if currentSHA == recordedSHA && !info.DetachedHead {
		return CheckoutAction{
			Path:   info.Path,
			Branch: info.CurrentBranch,
			Action: "already at recorded SHA on " + info.CurrentBranch,
		}
	}

	// Check for dirty working tree. Reset moves HEAD, so a failure to determine
	// whether the tree is clean must not be treated as "clean" -- bail out
	// rather than check out over changes we could not see.
	dirty, err := e.git.HasLocalChanges(ctx, dir)
	if err != nil {
		return CheckoutAction{
			Path:   info.Path,
			Action: "error checking working tree",
			Error:  err,
		}
	}
	if dirty {
		return CheckoutAction{
			Path:   info.Path,
			Action: "skipped (dirty working tree)",
		}
	}

	// Resolve a named branch at the recorded SHA.
	branchOpts := opts.BranchOpts
	branchOpts.TargetSHA = recordedSHA
	branch, isLocal, resolveErr := git.ResolveBranchForCheckout(ctx, e.git, dir, branchOpts)
	if resolveErr != nil {
		return CheckoutAction{
			Path:   info.Path,
			Action: "error resolving branch",
			Error:  resolveErr,
		}
	}

	// A remote-only match is not safe to check out by bare name: `git checkout
	// <name>` resolves an existing local branch first, and that branch cannot be
	// at the recorded SHA (it would have resolved as local otherwise). Checking
	// it out would leave HEAD off the recorded SHA while reporting success, so
	// detach at the recorded SHA instead.
	shadowed := ""
	if branch != "" && !isLocal {
		exists, existsErr := e.git.RefExists(ctx, dir, "refs/heads/"+branch)
		if existsErr != nil || exists {
			shadowed = branch
			branch = ""
		}
	}

	if branch == "" {
		// No branch safely points at this SHA -- checkout the SHA directly.
		shortSHA := recordedSHA
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}
		_, coErr := e.git.Checkout(ctx, dir, recordedSHA)
		if coErr != nil {
			return CheckoutAction{
				Path:     info.Path,
				Action:   "checkout failed",
				Detached: true,
				Error:    coErr,
			}
		}
		action := fmt.Sprintf("detached at %s", shortSHA)
		if shadowed != "" {
			action = fmt.Sprintf("detached at %s (local %s points elsewhere)", shortSHA, shadowed)
		}
		return CheckoutAction{
			Path:     info.Path,
			Action:   action,
			Detached: true,
		}
	}

	// Checkout the named branch.
	_, coErr := e.git.Checkout(ctx, dir, branch)
	if coErr != nil {
		return CheckoutAction{
			Path:   info.Path,
			Branch: branch,
			Action: fmt.Sprintf("checkout %s failed", branch),
			Error:  coErr,
		}
	}

	return CheckoutAction{
		Path:   info.Path,
		Branch: branch,
		Action: fmt.Sprintf("checked out %s", branch),
	}
}
