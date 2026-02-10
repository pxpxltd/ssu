package engine

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pxpxltd/ssu/internal/git"
)

// Push pushes selected submodules in parallel with bounded concurrency.
// It skips root and detached-HEAD submodules, auto-sets up tracking branches
// when missing, and continues on error (one failure does not abort others).
func (e *Engine) Push(ctx context.Context, targets []*SubmoduleInfo, opts PushOpts) (*PushResult, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Filter out root -- root is display-only, never pushed by engine.
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
	var actions []PushAction
	done := 0

	var g errgroup.Group
	g.SetLimit(concurrency)

	for _, info := range filtered {
		info := info // Go 1.21: capture loop variable

		g.Go(func() error {
			fire(ProgressEvent{Type: EventStarted, Path: info.Path, Phase: "push", Total: total, Done: done})

			action := e.pushOne(ctx, opts.RootDir, info)

			mu.Lock()
			done++
			d := done
			actions = append(actions, action)
			mu.Unlock()

			if action.Error != nil {
				fire(ProgressEvent{Type: EventFailed, Path: info.Path, Phase: "push", Error: action.Error, Total: total, Done: d})
			} else {
				fire(ProgressEvent{Type: EventCompleted, Path: info.Path, Phase: "push", Total: total, Done: d})
			}

			return nil // continue-on-error
		})
	}

	g.Wait()

	return &PushResult{Actions: actions}, nil
}

// pushOne pushes a single submodule and returns the resulting action.
func (e *Engine) pushOne(ctx context.Context, rootDir string, info *SubmoduleInfo) PushAction {
	dir := filepath.Join(rootDir, info.Path)

	// Detached HEAD: cannot push, skip with descriptive action.
	if info.DetachedHead {
		return PushAction{
			Path:   info.Path,
			Branch: info.CurrentBranch,
			Action: "skipped (detached HEAD)",
		}
	}

	// Push via GitService. The ExecGit.Push implementation handles
	// auto-detecting missing tracking branch and using -u flag.
	result, err := e.git.Push(ctx, dir, git.PushOpts{})
	if err != nil {
		return PushAction{
			Path:   info.Path,
			Branch: info.CurrentBranch,
			Action: "push failed",
			Error:  err,
		}
	}

	action := "pushed"
	if result.NewTracking {
		action = "set up tracking + pushed"
	}

	return PushAction{
		Path:   info.Path,
		Branch: result.Branch,
		Action: action,
	}
}
