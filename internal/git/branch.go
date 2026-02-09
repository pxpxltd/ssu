package git

import (
	"context"
	"fmt"
	"strings"
)

// DetectBestBranch implements the smart branch detection algorithm.
// It follows a strict 5-level priority chain, matching the original bash
// detect_best_branch() function:
//
//  1. Override: opts.Override (--branch CLI flag)
//  2. Feature branch: current branch preserved if it has a remote
//  3. Priority chain: first match in opts.PriorityBranches with a remote ref
//  4. Remote HEAD: symbolic ref of remote HEAD (e.g. origin/HEAD -> develop)
//  5. Final fallback: first available remote branch, or "master"
//
// This is a standalone function operating on the GitService interface so it
// can be fully unit-tested with MockGitService.
func DetectBestBranch(ctx context.Context, svc GitService, dir string, opts BranchDetectOpts) (string, error) {
	// Apply defaults for unset options.
	if len(opts.PriorityBranches) == 0 {
		opts.PriorityBranches = DefaultBranchPriority
	}
	if opts.DefaultRemote == "" {
		opts.DefaultRemote = "origin"
	}

	// Level 1: CLI override takes absolute priority.
	if opts.Override != "" {
		return opts.Override, nil
	}

	// Level 2: Feature branch preservation.
	br, err := svc.CurrentBranch(ctx, dir)
	if err != nil {
		return "", fmt.Errorf("detect branch: %w", err)
	}

	if !br.Detached && !isInList(br.Name, opts.PriorityBranches) {
		// Current branch is not in priority list -- it's a feature branch.
		// Check if remote has this branch; if so, preserve it.
		has, hasErr := svc.HasRemoteBranch(ctx, dir, opts.DefaultRemote, br.Name)
		if hasErr == nil && has {
			return br.Name, nil
		}
		// HasRemoteBranch error or no remote: fall through to priority chain.
	}

	// Level 3: Priority chain -- first priority branch found on any remote wins.
	remoteBranches, rbErr := svc.RemoteBranches(ctx, dir)

	if rbErr == nil && len(remoteBranches) > 0 {
		for _, prio := range opts.PriorityBranches {
			for _, rb := range remoteBranches {
				if rb.Branch == prio {
					return prio, nil
				}
			}
		}
	}

	// Level 4: Remote HEAD fallback.
	ref := "refs/remotes/" + opts.DefaultRemote + "/HEAD"
	sym, symErr := svc.SymbolicRef(ctx, dir, ref)
	if symErr == nil && sym != "" {
		// Parse "refs/remotes/origin/develop" -> "develop"
		prefix := "refs/remotes/" + opts.DefaultRemote + "/"
		if strings.HasPrefix(sym, prefix) {
			return strings.TrimPrefix(sym, prefix), nil
		}
	}

	// Level 5: Final fallback.
	// Use first remote branch if any were fetched, otherwise "master".
	if rbErr == nil && len(remoteBranches) > 0 {
		return remoteBranches[0].Branch, nil
	}
	return "master", nil
}

// isInList reports whether name appears in list. Simple linear search.
func isInList(name string, list []string) bool {
	for _, s := range list {
		if s == name {
			return true
		}
	}
	return false
}
