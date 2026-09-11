package engine

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/github"
)

// PRLister lists open pull requests for the repository behind a remote URL.
// Implementations: github.ExecGH (production), fakes in tests.
type PRLister interface {
	OpenPRs(ctx context.Context, remoteURL string) ([]github.PullRequest, error)
}

// PROpts configures an AttachPullRequests operation.
type PROpts struct {
	RootDir     string       // Absolute path to the project root
	Remote      string       // Remote whose URL identifies the repo (default "origin")
	Concurrency int          // Max parallel lookups (0 = runtime.NumCPU())
	OnProgress  ProgressFunc // Optional callback for progress events
}

// AttachPullRequests looks up open pull requests for the root repository and
// every scanned submodule, storing them on each SubmoduleInfo.
//
// Missing and skipped submodules are not looked up. A failed lookup is
// recorded in SubmoduleInfo.PRError and never aborts the others.
func (e *Engine) AttachPullRequests(ctx context.Context, lister PRLister, result *ScanResult, opts PROpts) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}
	remote := opts.Remote
	if remote == "" {
		remote = "origin"
	}

	var targets []*SubmoduleInfo
	if result.Root != nil {
		targets = append(targets, result.Root)
	}
	for _, sm := range result.Submodules {
		if sm.HasStatus(git.StatusMissing) || sm.HasStatus(git.StatusSkipped) {
			continue
		}
		targets = append(targets, sm)
	}

	total := len(targets)
	var mu sync.Mutex
	done := 0

	fire := func(evt ProgressEvent) {
		if opts.OnProgress != nil {
			opts.OnProgress(evt)
		}
	}

	// Use zero-value errgroup (NOT WithContext) for continue-on-error.
	var g errgroup.Group
	g.SetLimit(concurrency)

	for _, info := range targets {
		info := info // Go 1.21: capture loop variable

		g.Go(func() error {
			mu.Lock()
			d := done
			mu.Unlock()
			fire(ProgressEvent{Type: EventStarted, Path: info.Path, Phase: "pr", Total: total, Done: d})

			dir := opts.RootDir
			if !info.IsRoot {
				dir = filepath.Join(opts.RootDir, info.Path)
			}

			// Each goroutine writes only to its own info.
			info.PRChecked = true
			url, err := e.git.RemoteURL(ctx, dir, remote)
			if err != nil {
				err = fmt.Errorf("cannot read URL of remote %q: %w", remote, err)
			} else {
				info.PullRequests, err = lister.OpenPRs(ctx, url)
			}
			info.PRError = err

			mu.Lock()
			done++
			d = done
			mu.Unlock()

			if err != nil {
				fire(ProgressEvent{Type: EventFailed, Path: info.Path, Phase: "pr", Error: err, Total: total, Done: d})
			} else {
				fire(ProgressEvent{Type: EventCompleted, Path: info.Path, Phase: "pr", Total: total, Done: d})
			}
			return nil
		})
	}

	g.Wait()
}
