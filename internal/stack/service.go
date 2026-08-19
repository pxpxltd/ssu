package stack

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/pxpxltd/ssu/internal/git"
)

type Service struct {
	git git.GitService
}

func NewService(svc git.GitService) *Service {
	return &Service{git: svc}
}

func (s *Service) Export(ctx context.Context, rootDir string) (*File, error) {
	paths, err := s.git.SubmodulePaths(ctx, rootDir)
	if err != nil {
		return nil, fmt.Errorf("listing submodules: %w", err)
	}
	modules := make([]Module, 0, len(paths))
	for _, path := range paths {
		if !s.git.IsSubmoduleInitialized(rootDir, path) {
			continue
		}
		dir := filepath.Join(rootDir, path)
		branch, err := s.git.CurrentBranch(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("%s: reading branch: %w", path, err)
		}
		sha, err := s.git.CurrentSHA(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("%s: reading SHA: %w", path, err)
		}
		modules = append(modules, Module{Path: path, Branch: branch.Name, SHA: sha})
	}
	if len(modules) == 0 {
		return nil, fmt.Errorf("no initialized submodules found")
	}
	return New(modules), nil
}

type ImportStatus string

const (
	StatusSynced    ImportStatus = "synced"
	StatusFallback  ImportStatus = "fallback"
	StatusDirty     ImportStatus = "dirty"
	StatusDivergent ImportStatus = "divergent"
	StatusFailed    ImportStatus = "failed"
)

type ImportAction struct {
	Module Module
	Status ImportStatus
	Target string
	DryRun bool
	Error  error
}

type ImportOptions struct {
	RootDir     string
	Concurrency int
	DryRun      bool
}

func (s *Service) Import(ctx context.Context, modules []Module, opts ImportOptions) []ImportAction {
	concurrency := opts.Concurrency
	if concurrency <= 0 {
		concurrency = runtime.NumCPU()
	}

	var mu sync.Mutex
	actions := make([]ImportAction, 0, len(modules))
	var group errgroup.Group
	group.SetLimit(concurrency)

	for _, module := range modules {
		module := module
		group.Go(func() error {
			action := s.importOne(ctx, module, opts)
			mu.Lock()
			actions = append(actions, action)
			mu.Unlock()
			return nil
		})
	}
	_ = group.Wait()
	sort.Slice(actions, func(i, j int) bool { return actions[i].Module.Path < actions[j].Module.Path })
	return actions
}

func (s *Service) importOne(ctx context.Context, module Module, opts ImportOptions) ImportAction {
	dir := filepath.Join(opts.RootDir, module.Path)
	fail := func(err error) ImportAction {
		return ImportAction{Module: module, Status: StatusFailed, Error: err}
	}

	if _, err := s.git.Fetch(ctx, dir, git.FetchOpts{Prune: true}); err != nil {
		return fail(fmt.Errorf("fetch: %w", err))
	}
	dirty, err := s.git.HasLocalChanges(ctx, dir)
	if err != nil {
		return fail(fmt.Errorf("checking local changes: %w", err))
	}
	if dirty {
		return ImportAction{Module: module, Status: StatusDirty}
	}

	target := module.SHA
	status := StatusSynced
	exists, err := s.git.RefExists(ctx, dir, target)
	if err != nil {
		return fail(fmt.Errorf("checking exported SHA: %w", err))
	}
	if !exists {
		if module.Branch == "" {
			return fail(fmt.Errorf("exported detached SHA %s is unavailable after fetch", shortSHA(module.SHA)))
		}
		target = "origin/" + module.Branch
		exists, err = s.git.RefExists(ctx, dir, target)
		if err != nil {
			return fail(fmt.Errorf("checking fallback branch: %w", err))
		}
		if !exists {
			return fail(fmt.Errorf("exported SHA and fallback branch %s are unavailable", target))
		}
		status = StatusFallback
	}

	if module.Branch == "" {
		current, err := s.git.CurrentSHA(ctx, dir)
		if err != nil {
			return fail(fmt.Errorf("reading current SHA: %w", err))
		}
		safe, err := s.git.IsAncestor(ctx, dir, current, target)
		if err != nil {
			return fail(fmt.Errorf("checking detached history: %w", err))
		}
		if current != target && !safe {
			return ImportAction{Module: module, Status: StatusDivergent, Target: target}
		}
		if opts.DryRun {
			return ImportAction{Module: module, Status: status, Target: target, DryRun: true}
		}
		if _, err := s.git.Checkout(ctx, dir, target); err != nil {
			return fail(fmt.Errorf("checking out detached SHA: %w", err))
		}
		return ImportAction{Module: module, Status: status, Target: target}
	}

	localRef := "refs/heads/" + module.Branch
	localExists, err := s.git.RefExists(ctx, dir, localRef)
	if err != nil {
		return fail(fmt.Errorf("checking local branch: %w", err))
	}
	if localExists {
		safe, err := s.git.IsAncestor(ctx, dir, localRef, target)
		if err != nil {
			return fail(fmt.Errorf("checking branch history: %w", err))
		}
		if !safe {
			return ImportAction{Module: module, Status: StatusDivergent, Target: target}
		}
	}
	if opts.DryRun {
		return ImportAction{Module: module, Status: status, Target: target, DryRun: true}
	}
	if localExists {
		if _, err := s.git.Checkout(ctx, dir, module.Branch); err != nil {
			return fail(fmt.Errorf("checking out %s: %w", module.Branch, err))
		}
		if err := s.git.ResetHard(ctx, dir, target); err != nil {
			return fail(fmt.Errorf("fast-forwarding %s: %w", module.Branch, err))
		}
	} else {
		if _, err := s.git.CheckoutNewBranch(ctx, dir, module.Branch, target); err != nil {
			return fail(fmt.Errorf("creating branch %s: %w", module.Branch, err))
		}
	}
	return ImportAction{Module: module, Status: status, Target: target}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
