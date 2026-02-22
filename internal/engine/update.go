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

// Update processes selected submodules by merging their target branch.
// It handles dirty submodules with a 3-step stash+merge+stash-pop strategy,
// and provides actionable ConflictHint commands on failure.
//
// The targets parameter is the subset of scan.Submodules the caller selected
// for update -- the engine does NOT decide what to update.
//
// Individual update failures do not abort other updates (continue-on-error).
func (e *Engine) Update(ctx context.Context, targets []*SubmoduleInfo, opts UpdateOpts) (*UpdateResult, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Filter targets: skip root and non-updateable statuses.
	var updateable []*SubmoduleInfo
	for _, info := range targets {
		if info.IsRoot {
			continue
		}
		if isSkippable(info) {
			continue
		}
		updateable = append(updateable, info)
	}

	total := len(updateable)

	fire := func(evt ProgressEvent) {
		if opts.OnProgress != nil {
			opts.OnProgress(evt)
		}
	}

	var mu sync.Mutex
	var actions []UpdateAction
	done := 0

	var g errgroup.Group
	g.SetLimit(concurrency)

	for _, info := range updateable {
		info := info // Go 1.21: capture loop variable

		g.Go(func() error {
			fire(ProgressEvent{Type: EventStarted, Path: info.Path, Phase: "update", Total: total, Done: done})

			action := e.updateOne(ctx, opts.RootDir, info)

			mu.Lock()
			done++
			d := done
			actions = append(actions, action)
			mu.Unlock()

			if action.Error != nil {
				fire(ProgressEvent{Type: EventFailed, Path: info.Path, Phase: "update", Error: action.Error, Total: total, Done: d, Action: action.Action})
			} else {
				fire(ProgressEvent{Type: EventCompleted, Path: info.Path, Phase: "update", Total: total, Done: d, Action: action.Action})
			}
			return nil // continue-on-error
		})
	}

	g.Wait()

	return &UpdateResult{Actions: actions}, nil
}

// isSkippable reports whether a SubmoduleInfo should be skipped during update.
// Submodules that are only current, missing, or skipped have nothing to update.
func isSkippable(info *SubmoduleInfo) bool {
	for _, s := range info.Statuses {
		switch s {
		case git.StatusCurrent, git.StatusMissing, git.StatusSkipped:
			// These are non-updateable -- keep checking
			continue
		default:
			// Found an updateable status (pending, modified, ahead, conflict, error)
			return false
		}
	}
	// All statuses are non-updateable
	return true
}

// updateOne processes a single submodule update. It implements the 3-step
// conflict resolution strategy:
//  1. Stash local changes (if dirty)
//  2. Merge target branch
//  3. Stash pop to restore local changes
//
// On merge conflict: abort merge, restore stash, return ConflictHint with
// copy-paste git commands.
func (e *Engine) updateOne(ctx context.Context, rootDir string, info *SubmoduleInfo) UpdateAction {
	action := UpdateAction{
		Path:         info.Path,
		BeforeStatus: append([]git.SubmoduleStatus(nil), info.Statuses...),
	}

	dir := filepath.Join(rootDir, info.Path)
	remoteRef := "origin/" + info.TargetBranch

	if info.HasChanges {
		e.updateDirty(ctx, dir, remoteRef, info, &action)
	} else {
		e.updateClean(ctx, dir, remoteRef, info, &action)
	}

	return action
}

// updateDirty handles the 3-step stash+merge+stash-pop strategy for submodules
// with uncommitted local changes.
func (e *Engine) updateDirty(ctx context.Context, dir, remoteRef string, info *SubmoduleInfo, action *UpdateAction) {
	// Step 1: Stash local changes.
	stashResult, err := e.git.Stash(ctx, dir)
	if err != nil {
		action.Error = fmt.Errorf("stash failed: %w", err)
		action.Action = "stash failed"
		action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
		return
	}

	// Step 2: Merge on clean state.
	mergeResult, mergeErr := e.git.Merge(ctx, dir, remoteRef)

	if mergeErr == nil && mergeResult.Success {
		// Merge succeeded -- restore stash.
		_, popErr := e.git.StashPop(ctx, dir)
		if popErr != nil {
			action.Action = fmt.Sprintf("merged %d commits but stash pop failed", info.CommitsBehind)
			action.Error = fmt.Errorf("stash pop failed after merge: %w", popErr)
			action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
			return
		}
		action.Action = fmt.Sprintf("stashed, merged %d commits, restored changes", info.CommitsBehind)
		action.AfterStatus = []git.SubmoduleStatus{git.StatusCurrent}
		return
	}

	// Merge failed. Check if it was a conflict.
	isConflict := mergeResult.Conflict || git.IsConflict(mergeErr)

	// Step 3 (failure path): Abort merge and restore stash.
	abortErr := e.git.MergeAbort(ctx, dir)
	if abortErr != nil {
		action.Error = fmt.Errorf("merge abort failed after conflict: %w", abortErr)
		action.Action = "merge abort failed"
		action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
		// Still try to restore stash.
		if stashResult.Created {
			e.git.StashPop(ctx, dir) //nolint:errcheck // best-effort
		}
		return
	}

	// Restore stash after abort.
	if stashResult.Created {
		_, popErr := e.git.StashPop(ctx, dir)
		if popErr != nil {
			action.Error = fmt.Errorf("stash pop failed after merge abort: %w", popErr)
			action.Action = "stash restore failed after conflict"
			action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
			return
		}
	}

	if isConflict {
		action.Action = "conflict after stash+retry"
		action.AfterStatus = []git.SubmoduleStatus{git.StatusConflict}
		action.ConflictHint = fmt.Sprintf("cd %s && git stash && git merge %s && git stash pop", info.Path, remoteRef)
		action.Error = mergeErr
	} else {
		action.Action = "merge failed"
		action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
		action.Error = mergeErr
	}
}

// updateClean handles a straightforward merge for submodules without local changes.
func (e *Engine) updateClean(ctx context.Context, dir, remoteRef string, info *SubmoduleInfo, action *UpdateAction) {
	mergeResult, mergeErr := e.git.Merge(ctx, dir, remoteRef)

	if mergeErr == nil && mergeResult.Success {
		action.Action = fmt.Sprintf("merged %d commits", info.CommitsBehind)
		action.AfterStatus = []git.SubmoduleStatus{git.StatusCurrent}
		return
	}

	isConflict := mergeResult.Conflict || git.IsConflict(mergeErr)
	if isConflict {
		action.Action = "conflict"
		action.AfterStatus = []git.SubmoduleStatus{git.StatusConflict}
		action.ConflictHint = fmt.Sprintf("cd %s && git merge %s", info.Path, remoteRef)
		action.Error = mergeErr
		return
	}

	action.Action = "merge failed"
	action.AfterStatus = []git.SubmoduleStatus{git.StatusError}
	action.Error = mergeErr
}
