package git

import (
	"context"
	"fmt"
)

// ResolveBranchForCheckout finds the best branch to checkout when HEAD is detached.
// It queries BranchesPointingAt(HEAD) and picks the best match with priority:
//
//  1. Local feature branch (not in priority list)
//  2. Local priority branch (develop > master > main)
//  3. Remote feature branch (not in priority list)
//  4. Remote priority branch (develop > master > main)
//
// Returns the branch name, whether it's a local branch, and any error.
// If no branch points at HEAD, returns ("", false, nil).
func ResolveBranchForCheckout(ctx context.Context, svc GitService, dir string, opts BranchCheckoutOpts) (string, bool, error) {
	if len(opts.PriorityBranches) == 0 {
		opts.PriorityBranches = DefaultBranchPriority
	}
	if opts.DefaultRemote == "" {
		opts.DefaultRemote = "origin"
	}

	sha := opts.TargetSHA
	if sha == "" {
		var shaErr error
		sha, shaErr = svc.CurrentSHA(ctx, dir)
		if shaErr != nil {
			return "", false, fmt.Errorf("resolve checkout branch: %w", shaErr)
		}
	}

	branches, err := svc.BranchesPointingAt(ctx, dir, sha)
	if err != nil {
		return "", false, fmt.Errorf("resolve checkout branch: %w", err)
	}

	// 1. Local feature branch (not in priority list).
	for _, b := range branches.Local {
		if !isInList(b, opts.PriorityBranches) {
			return b, true, nil
		}
	}

	// 2. Local priority branch (first match in priority order).
	for _, prio := range opts.PriorityBranches {
		for _, b := range branches.Local {
			if b == prio {
				return b, true, nil
			}
		}
	}

	// 3. Remote feature branch (not in priority list, on default remote).
	for _, rb := range branches.Remote {
		if rb.Remote == opts.DefaultRemote && !isInList(rb.Branch, opts.PriorityBranches) {
			return rb.Branch, false, nil
		}
	}

	// 4. Remote priority branch (first match in priority order, on default remote).
	for _, prio := range opts.PriorityBranches {
		for _, rb := range branches.Remote {
			if rb.Remote == opts.DefaultRemote && rb.Branch == prio {
				return rb.Branch, false, nil
			}
		}
	}

	// No matching branch found.
	return "", false, nil
}
