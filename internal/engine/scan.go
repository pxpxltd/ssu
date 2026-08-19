package engine

import (
	"context"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pxpxltd/ssu/internal/git"
)

// Scan enumerates submodules, fetches in parallel with bounded concurrency,
// detects per-submodule status (including compound statuses), and includes
// root repository as display-only.
//
// Individual fetch or status-detection failures do not abort the scan;
// they are recorded as StatusError on the affected submodule.
func (e *Engine) Scan(ctx context.Context, opts ScanOpts) (*ScanResult, error) {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	// Build skip set for O(1) lookups.
	skipSet := make(map[string]bool, len(opts.SkipList))
	for _, s := range opts.SkipList {
		skipSet[s] = true
	}

	// Enumerate submodules.
	paths, err := e.git.SubmodulePaths(ctx, opts.RootDir)
	if err != nil {
		return nil, err
	}

	// Total items: submodules + root.
	total := len(paths) + 1

	// Collect results with mutex protection.
	var mu sync.Mutex
	var results []*SubmoduleInfo
	done := 0

	fire := func(evt ProgressEvent) {
		if opts.OnProgress != nil {
			opts.OnProgress(evt)
		}
	}

	// Use zero-value errgroup (NOT WithContext) for continue-on-error.
	var g errgroup.Group
	g.SetLimit(concurrency)

	// Process root repository.
	g.Go(func() error {
		fire(ProgressEvent{Type: EventStarted, Path: ".", Phase: "fetch", Total: total, Done: done})

		info := e.scanOne(ctx, opts.RootDir, ".", opts, false)
		info.IsRoot = true

		mu.Lock()
		done++
		d := done
		results = append(results, info)
		mu.Unlock()

		if info.Error != nil {
			fire(ProgressEvent{Type: EventFailed, Path: ".", Phase: "status", Error: info.Error, Total: total, Done: d})
		} else {
			fire(ProgressEvent{Type: EventCompleted, Path: ".", Phase: "status", Total: total, Done: d})
		}
		return nil
	})

	// Process each submodule.
	for _, path := range paths {
		path := path // Go 1.21: capture loop variable

		g.Go(func() error {
			fire(ProgressEvent{Type: EventStarted, Path: path, Phase: "fetch", Total: total, Done: done})

			// Check skip list.
			if skipSet[path] {
				info := &SubmoduleInfo{
					Path:     path,
					Statuses: []git.SubmoduleStatus{git.StatusSkipped},
				}
				mu.Lock()
				done++
				d := done
				results = append(results, info)
				mu.Unlock()
				fire(ProgressEvent{Type: EventCompleted, Path: path, Phase: "status", Total: total, Done: d})
				return nil
			}

			// Check if submodule is initialized.
			if !e.git.IsSubmoduleInitialized(opts.RootDir, path) {
				info := &SubmoduleInfo{
					Path:     path,
					Statuses: []git.SubmoduleStatus{git.StatusMissing},
				}
				mu.Lock()
				done++
				d := done
				results = append(results, info)
				mu.Unlock()
				fire(ProgressEvent{Type: EventCompleted, Path: path, Phase: "status", Total: total, Done: d})
				return nil
			}

			absDir := filepath.Join(opts.RootDir, path)
			info := e.scanOne(ctx, absDir, path, opts, false)

			mu.Lock()
			done++
			d := done
			results = append(results, info)
			mu.Unlock()

			if info.Error != nil {
				fire(ProgressEvent{Type: EventFailed, Path: path, Phase: "status", Error: info.Error, Total: total, Done: d})
			} else {
				fire(ProgressEvent{Type: EventCompleted, Path: path, Phase: "status", Total: total, Done: d})
			}
			return nil
		})
	}

	g.Wait()

	// Sort results by path and separate root.
	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	result := &ScanResult{}
	for _, info := range results {
		if info.IsRoot {
			result.Root = info
		} else {
			result.Submodules = append(result.Submodules, info)
		}
	}

	return result, nil
}

// scanOne fetches and detects status for a single directory.
// It never returns an error to the caller -- failures are captured in
// the returned SubmoduleInfo.Error and StatusError is added.
func (e *Engine) scanOne(ctx context.Context, absDir, path string, opts ScanOpts, _ bool) *SubmoduleInfo {
	info := &SubmoduleInfo{Path: path}

	// Fetch.
	_, fetchErr := e.git.Fetch(ctx, absDir, git.FetchOpts{Prune: true})
	if fetchErr != nil {
		info.Error = fetchErr
		info.Statuses = append(info.Statuses, git.StatusError)
		return info
	}

	// Detached head check.
	detached, err := e.git.IsDetachedHead(ctx, absDir)
	if err != nil {
		info.Error = err
		info.Statuses = append(info.Statuses, git.StatusError)
		return info
	}
	info.DetachedHead = detached

	// Current branch.
	br, err := e.git.CurrentBranch(ctx, absDir)
	if err != nil {
		info.Error = err
		info.Statuses = append(info.Statuses, git.StatusError)
		return info
	}
	info.CurrentBranch = br.Name

	// Current SHA.
	sha, err := e.git.CurrentSHA(ctx, absDir)
	if err == nil {
		info.CurrentSHA = sha
	}

	// Detect best target branch.
	target, err := git.DetectBestBranch(ctx, e.git, absDir, opts.BranchOpts)
	if err != nil {
		info.Error = err
		info.Statuses = append(info.Statuses, git.StatusError)
		return info
	}
	info.TargetBranch = target

	// Build remote ref.
	remote := opts.BranchOpts.DefaultRemote
	if remote == "" {
		remote = "origin"
	}
	remoteRef := remote + "/" + target

	// Local changes -> StatusModified.
	hasChanges, err := e.git.HasLocalChanges(ctx, absDir)
	if err == nil && hasChanges {
		info.HasChanges = true
		info.Statuses = append(info.Statuses, git.StatusModified)
	}

	// Commits behind -> StatusPending.
	behind, err := e.git.CommitsBehind(ctx, absDir, "HEAD", remoteRef)
	if err == nil && behind > 0 {
		info.CommitsBehind = behind
		info.Statuses = append(info.Statuses, git.StatusPending)
	}

	// Commits ahead -> StatusAhead.
	ahead, err := e.git.CommitsAhead(ctx, absDir, remoteRef)
	if err == nil && ahead > 0 {
		info.CommitsAhead = ahead
		info.Statuses = append(info.Statuses, git.StatusAhead)
	}

	// Changelog.
	changelog, err := e.git.IncomingChangelog(ctx, absDir, remoteRef, 20)
	if err == nil {
		info.Changelog = changelog
	}

	// Default to current if nothing else detected.
	if len(info.Statuses) == 0 {
		info.Statuses = append(info.Statuses, git.StatusCurrent)
	}

	return info
}
