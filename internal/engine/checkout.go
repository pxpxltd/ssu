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

			action := e.checkoutOne(ctx, opts.RootDir, info, opts.BranchOpts)

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
