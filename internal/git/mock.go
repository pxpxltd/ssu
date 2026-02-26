package git

import "context"

// MockGitService is a test double for GitService. Each method delegates to
// its corresponding Fn field if non-nil, otherwise returns a sensible default.
// This allows tests to override only the methods they care about while
// leaving everything else functional with safe zero-value behavior.
type MockGitService struct {
	SubmodulePathsFn         func(ctx context.Context, rootDir string) ([]string, error)
	IsSubmoduleInitializedFn func(rootDir, subPath string) bool
	CurrentBranchFn          func(ctx context.Context, dir string) (BranchResult, error)
	CurrentSHAFn             func(ctx context.Context, dir string) (string, error)
	IsDetachedHeadFn         func(ctx context.Context, dir string) (bool, error)
	RemoteBranchesFn         func(ctx context.Context, dir string) ([]RemoteBranch, error)
	CommitsBehindFn          func(ctx context.Context, dir, localRef, remoteRef string) (int, error)
	CommitsAheadFn           func(ctx context.Context, dir, remoteRef string) (int, error)
	HasRemoteBranchFn        func(ctx context.Context, dir, remote, branch string) (bool, error)
	SymbolicRefFn            func(ctx context.Context, dir, ref string) (string, error)
	TrackingBranchFn         func(ctx context.Context, dir string) (TrackingInfo, error)
	HasLocalChangesFn        func(ctx context.Context, dir string) (bool, error)
	IncomingChangelogFn      func(ctx context.Context, dir, remoteRef string, limit int) ([]string, error)
	FetchFn                  func(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error)
	CheckoutFn               func(ctx context.Context, dir, branch string) (CheckoutResult, error)
	MergeFn                  func(ctx context.Context, dir, ref string) (MergeResult, error)
	PushFn                   func(ctx context.Context, dir string, opts PushOpts) (PushResult, error)
	StashFn                  func(ctx context.Context, dir string) (StashResult, error)
	StashPopFn               func(ctx context.Context, dir string) (StashResult, error)
	MergeAbortFn             func(ctx context.Context, dir string) error
	ResetHardFn              func(ctx context.Context, dir, ref string) error
	DiffIndexFn              func(ctx context.Context, dir string) ([]string, error)
	StageFilesFn             func(ctx context.Context, dir string, paths []string) error
	CommitFn                 func(ctx context.Context, dir, message string) (CommitResult, error)
	ListTagsFn               func(ctx context.Context, dir string, limit int) ([]string, error)
	CreateTagFn              func(ctx context.Context, dir string, opts TagOpts) error
	BranchesPointingAtFn     func(ctx context.Context, dir, ref string) (BranchPointsAtResult, error)
	SubmoduleDeinitFn        func(ctx context.Context, rootDir, subPath string) error
	RemovePathFn             func(ctx context.Context, rootDir, path string) error
}

func (m *MockGitService) SubmodulePaths(ctx context.Context, rootDir string) ([]string, error) {
	if m.SubmodulePathsFn != nil {
		return m.SubmodulePathsFn(ctx, rootDir)
	}
	return nil, nil
}

func (m *MockGitService) IsSubmoduleInitialized(rootDir, subPath string) bool {
	if m.IsSubmoduleInitializedFn != nil {
		return m.IsSubmoduleInitializedFn(rootDir, subPath)
	}
	return true
}

func (m *MockGitService) CurrentBranch(ctx context.Context, dir string) (BranchResult, error) {
	if m.CurrentBranchFn != nil {
		return m.CurrentBranchFn(ctx, dir)
	}
	return BranchResult{Name: "develop"}, nil
}

func (m *MockGitService) CurrentSHA(ctx context.Context, dir string) (string, error) {
	if m.CurrentSHAFn != nil {
		return m.CurrentSHAFn(ctx, dir)
	}
	return "abc1234", nil
}

func (m *MockGitService) IsDetachedHead(ctx context.Context, dir string) (bool, error) {
	if m.IsDetachedHeadFn != nil {
		return m.IsDetachedHeadFn(ctx, dir)
	}
	return false, nil
}

func (m *MockGitService) RemoteBranches(ctx context.Context, dir string) ([]RemoteBranch, error) {
	if m.RemoteBranchesFn != nil {
		return m.RemoteBranchesFn(ctx, dir)
	}
	return []RemoteBranch{{Remote: "origin", Branch: "develop"}}, nil
}

func (m *MockGitService) CommitsBehind(ctx context.Context, dir, localRef, remoteRef string) (int, error) {
	if m.CommitsBehindFn != nil {
		return m.CommitsBehindFn(ctx, dir, localRef, remoteRef)
	}
	return 0, nil
}

func (m *MockGitService) CommitsAhead(ctx context.Context, dir, remoteRef string) (int, error) {
	if m.CommitsAheadFn != nil {
		return m.CommitsAheadFn(ctx, dir, remoteRef)
	}
	return 0, nil
}

func (m *MockGitService) HasRemoteBranch(ctx context.Context, dir, remote, branch string) (bool, error) {
	if m.HasRemoteBranchFn != nil {
		return m.HasRemoteBranchFn(ctx, dir, remote, branch)
	}
	return true, nil
}

func (m *MockGitService) SymbolicRef(ctx context.Context, dir, ref string) (string, error) {
	if m.SymbolicRefFn != nil {
		return m.SymbolicRefFn(ctx, dir, ref)
	}
	return "refs/remotes/origin/develop", nil
}

func (m *MockGitService) TrackingBranch(ctx context.Context, dir string) (TrackingInfo, error) {
	if m.TrackingBranchFn != nil {
		return m.TrackingBranchFn(ctx, dir)
	}
	return TrackingInfo{Remote: "origin", Branch: "develop", Set: true}, nil
}

func (m *MockGitService) HasLocalChanges(ctx context.Context, dir string) (bool, error) {
	if m.HasLocalChangesFn != nil {
		return m.HasLocalChangesFn(ctx, dir)
	}
	return false, nil
}

func (m *MockGitService) IncomingChangelog(ctx context.Context, dir, remoteRef string, limit int) ([]string, error) {
	if m.IncomingChangelogFn != nil {
		return m.IncomingChangelogFn(ctx, dir, remoteRef, limit)
	}
	return nil, nil
}

func (m *MockGitService) Fetch(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error) {
	if m.FetchFn != nil {
		return m.FetchFn(ctx, dir, opts)
	}
	return FetchResult{}, nil
}

func (m *MockGitService) Checkout(ctx context.Context, dir, branch string) (CheckoutResult, error) {
	if m.CheckoutFn != nil {
		return m.CheckoutFn(ctx, dir, branch)
	}
	return CheckoutResult{Branch: branch}, nil
}

func (m *MockGitService) Merge(ctx context.Context, dir, ref string) (MergeResult, error) {
	if m.MergeFn != nil {
		return m.MergeFn(ctx, dir, ref)
	}
	return MergeResult{Success: true}, nil
}

func (m *MockGitService) Push(ctx context.Context, dir string, opts PushOpts) (PushResult, error) {
	if m.PushFn != nil {
		return m.PushFn(ctx, dir, opts)
	}
	return PushResult{Remote: opts.Remote, Branch: opts.Branch}, nil
}

func (m *MockGitService) Stash(ctx context.Context, dir string) (StashResult, error) {
	if m.StashFn != nil {
		return m.StashFn(ctx, dir)
	}
	return StashResult{Created: true}, nil
}

func (m *MockGitService) StashPop(ctx context.Context, dir string) (StashResult, error) {
	if m.StashPopFn != nil {
		return m.StashPopFn(ctx, dir)
	}
	return StashResult{Created: true}, nil
}

func (m *MockGitService) MergeAbort(ctx context.Context, dir string) error {
	if m.MergeAbortFn != nil {
		return m.MergeAbortFn(ctx, dir)
	}
	return nil
}

func (m *MockGitService) ResetHard(ctx context.Context, dir, ref string) error {
	if m.ResetHardFn != nil {
		return m.ResetHardFn(ctx, dir, ref)
	}
	return nil
}

func (m *MockGitService) DiffIndex(ctx context.Context, dir string) ([]string, error) {
	if m.DiffIndexFn != nil {
		return m.DiffIndexFn(ctx, dir)
	}
	return nil, nil
}

func (m *MockGitService) StageFiles(ctx context.Context, dir string, paths []string) error {
	if m.StageFilesFn != nil {
		return m.StageFilesFn(ctx, dir, paths)
	}
	return nil
}

func (m *MockGitService) Commit(ctx context.Context, dir, message string) (CommitResult, error) {
	if m.CommitFn != nil {
		return m.CommitFn(ctx, dir, message)
	}
	return CommitResult{SHA: "abc1234"}, nil
}

func (m *MockGitService) ListTags(ctx context.Context, dir string, limit int) ([]string, error) {
	if m.ListTagsFn != nil {
		return m.ListTagsFn(ctx, dir, limit)
	}
	return nil, nil
}

func (m *MockGitService) CreateTag(ctx context.Context, dir string, opts TagOpts) error {
	if m.CreateTagFn != nil {
		return m.CreateTagFn(ctx, dir, opts)
	}
	return nil
}

func (m *MockGitService) BranchesPointingAt(ctx context.Context, dir, ref string) (BranchPointsAtResult, error) {
	if m.BranchesPointingAtFn != nil {
		return m.BranchesPointingAtFn(ctx, dir, ref)
	}
	return BranchPointsAtResult{}, nil
}

func (m *MockGitService) SubmoduleDeinit(ctx context.Context, rootDir, subPath string) error {
	if m.SubmoduleDeinitFn != nil {
		return m.SubmoduleDeinitFn(ctx, rootDir, subPath)
	}
	return nil
}

func (m *MockGitService) RemovePath(ctx context.Context, rootDir, path string) error {
	if m.RemovePathFn != nil {
		return m.RemovePathFn(ctx, rootDir, path)
	}
	return nil
}
