package engine

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

// helper to create a mock that returns the given submodule paths
// and has sensible defaults for all other methods.
func baseMock(paths []string) *git.MockGitService {
	return &git.MockGitService{
		SubmodulePathsFn: func(_ context.Context, _ string) ([]string, error) {
			return paths, nil
		},
	}
}

func TestScan_HappyPath_AllBehind(t *testing.T) {
	mock := baseMock([]string{"plugins/auth", "plugins/blog", "plugins/pages"})
	mock.CommitsBehindFn = func(_ context.Context, _, _, _ string) (int, error) {
		return 2, nil
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if result.Root == nil {
		t.Fatal("Root is nil")
	}
	if !result.Root.IsRoot {
		t.Error("Root.IsRoot should be true")
	}

	if len(result.Submodules) != 3 {
		t.Fatalf("expected 3 submodules, got %d", len(result.Submodules))
	}

	for _, sub := range result.Submodules {
		if !sub.HasStatus(git.StatusPending) {
			t.Errorf("%s: expected StatusPending", sub.Path)
		}
		if sub.CommitsBehind != 2 {
			t.Errorf("%s: expected CommitsBehind=2, got %d", sub.Path, sub.CommitsBehind)
		}
	}
}

func TestScan_MixedStatuses(t *testing.T) {
	mock := baseMock([]string{"mod-pending", "mod-current", "mod-compound"})

	mock.CommitsBehindFn = func(_ context.Context, dir, _, _ string) (int, error) {
		if dir == "/project/mod-pending" {
			return 3, nil
		}
		return 0, nil
	}
	mock.HasLocalChangesFn = func(_ context.Context, dir string) (bool, error) {
		return dir == "/project/mod-compound", nil
	}
	mock.CommitsAheadFn = func(_ context.Context, dir, _ string) (int, error) {
		if dir == "/project/mod-compound" {
			return 1, nil
		}
		return 0, nil
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 4,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Find submodules by path.
	byPath := make(map[string]*SubmoduleInfo)
	for _, s := range result.Submodules {
		byPath[s.Path] = s
	}

	// mod-pending: only pending
	pending := byPath["mod-pending"]
	if pending == nil {
		t.Fatal("mod-pending not found")
	}
	if !pending.HasStatus(git.StatusPending) {
		t.Error("mod-pending should have StatusPending")
	}
	if pending.CommitsBehind != 3 {
		t.Errorf("mod-pending: expected CommitsBehind=3, got %d", pending.CommitsBehind)
	}

	// mod-current: no behind/ahead/changes -> current
	current := byPath["mod-current"]
	if current == nil {
		t.Fatal("mod-current not found")
	}
	if !current.HasStatus(git.StatusCurrent) {
		t.Error("mod-current should have StatusCurrent")
	}

	// mod-compound: modified AND ahead
	compound := byPath["mod-compound"]
	if compound == nil {
		t.Fatal("mod-compound not found")
	}
	if !compound.HasStatus(git.StatusModified) {
		t.Error("mod-compound should have StatusModified")
	}
	if !compound.HasStatus(git.StatusAhead) {
		t.Error("mod-compound should have StatusAhead")
	}
	if compound.PrimaryStatus() != git.StatusModified {
		t.Errorf("mod-compound PrimaryStatus: expected modified, got %s", compound.PrimaryStatus())
	}
}

func TestScan_FetchFailure(t *testing.T) {
	mock := baseMock([]string{"good", "bad"})

	mock.FetchFn = func(_ context.Context, dir string, _ git.FetchOpts) (git.FetchResult, error) {
		if dir == "/project/bad" {
			return git.FetchResult{}, fmt.Errorf("network timeout")
		}
		return git.FetchResult{}, nil
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan should not return error on individual fetch failure: %v", err)
	}

	byPath := make(map[string]*SubmoduleInfo)
	for _, s := range result.Submodules {
		byPath[s.Path] = s
	}

	bad := byPath["bad"]
	if bad == nil {
		t.Fatal("bad not found")
	}
	if !bad.HasStatus(git.StatusError) {
		t.Error("bad should have StatusError")
	}
	if bad.Error == nil {
		t.Error("bad should have non-nil Error")
	}

	good := byPath["good"]
	if good == nil {
		t.Fatal("good not found")
	}
	if good.HasStatus(git.StatusError) {
		t.Error("good should not have StatusError")
	}
}

func TestScan_UninitializedSubmodule(t *testing.T) {
	mock := baseMock([]string{"initialized", "uninitialized"})

	mock.IsSubmoduleInitializedFn = func(_, subPath string) bool {
		return subPath != "uninitialized"
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	byPath := make(map[string]*SubmoduleInfo)
	for _, s := range result.Submodules {
		byPath[s.Path] = s
	}

	uninit := byPath["uninitialized"]
	if uninit == nil {
		t.Fatal("uninitialized not found")
	}
	if !uninit.HasStatus(git.StatusMissing) {
		t.Error("uninitialized should have StatusMissing")
	}

	init := byPath["initialized"]
	if init == nil {
		t.Fatal("initialized not found")
	}
	if init.HasStatus(git.StatusMissing) {
		t.Error("initialized should not have StatusMissing")
	}
}

func TestScan_SkipList(t *testing.T) {
	var fetchMu sync.Mutex
	fetchCalled := make(map[string]bool)

	mock := baseMock([]string{"keep", "skip-me"})
	mock.FetchFn = func(_ context.Context, dir string, _ git.FetchOpts) (git.FetchResult, error) {
		fetchMu.Lock()
		fetchCalled[dir] = true
		fetchMu.Unlock()
		return git.FetchResult{}, nil
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		SkipList:    []string{"skip-me"},
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	byPath := make(map[string]*SubmoduleInfo)
	for _, s := range result.Submodules {
		byPath[s.Path] = s
	}

	skipped := byPath["skip-me"]
	if skipped == nil {
		t.Fatal("skip-me not found")
	}
	if !skipped.HasStatus(git.StatusSkipped) {
		t.Error("skip-me should have StatusSkipped")
	}

	// Verify no fetch was called for skipped submodule.
	fetchMu.Lock()
	skipFetched := fetchCalled["/project/skip-me"]
	fetchMu.Unlock()
	if skipFetched {
		t.Error("fetch should not be called for skipped submodule")
	}

	kept := byPath["keep"]
	if kept == nil {
		t.Fatal("keep not found")
	}
	if kept.HasStatus(git.StatusSkipped) {
		t.Error("keep should not be skipped")
	}
}

func TestScan_RootIncluded(t *testing.T) {
	mock := baseMock([]string{"sub1"})

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if result.Root == nil {
		t.Fatal("Root should not be nil")
	}
	if !result.Root.IsRoot {
		t.Error("Root.IsRoot should be true")
	}
	if result.Root.Path != "." {
		t.Errorf("Root.Path: expected '.', got %q", result.Root.Path)
	}

	// Root should not appear in Submodules slice.
	for _, s := range result.Submodules {
		if s.IsRoot {
			t.Error("Root should not appear in Submodules")
		}
	}
}

func TestScan_EmptyRepo(t *testing.T) {
	mock := baseMock(nil) // no submodules

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if result.Root == nil {
		t.Fatal("Root should still be scanned")
	}
	if len(result.Submodules) != 0 {
		t.Errorf("expected 0 submodules, got %d", len(result.Submodules))
	}
}

func TestScan_DetachedHead(t *testing.T) {
	mock := baseMock([]string{"detached-mod"})

	mock.IsDetachedHeadFn = func(_ context.Context, dir string) (bool, error) {
		if dir == "/project/detached-mod" {
			return true, nil
		}
		return false, nil
	}
	mock.CurrentBranchFn = func(_ context.Context, dir string) (git.BranchResult, error) {
		if dir == "/project/detached-mod" {
			return git.BranchResult{Detached: true}, nil
		}
		return git.BranchResult{Name: "develop"}, nil
	}

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	byPath := make(map[string]*SubmoduleInfo)
	for _, s := range result.Submodules {
		byPath[s.Path] = s
	}

	det := byPath["detached-mod"]
	if det == nil {
		t.Fatal("detached-mod not found")
	}
	if !det.DetachedHead {
		t.Error("detached-mod should have DetachedHead=true")
	}
}

func TestScan_ProgressCallback(t *testing.T) {
	mock := baseMock([]string{"sub1", "sub2"})

	var mu sync.Mutex
	var events []ProgressEvent

	eng := New(mock)
	_, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 1, // serial to make event ordering predictable per-item
		OnProgress: func(evt ProgressEvent) {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Should have events for root + 2 submodules.
	// Each gets at least Started + Completed = 2 events.
	startedCount := 0
	completedCount := 0
	for _, e := range events {
		switch e.Type {
		case EventStarted:
			startedCount++
		case EventCompleted:
			completedCount++
		}
	}

	// 3 items: root, sub1, sub2
	if startedCount < 3 {
		t.Errorf("expected at least 3 EventStarted, got %d", startedCount)
	}
	if completedCount < 3 {
		t.Errorf("expected at least 3 EventCompleted, got %d", completedCount)
	}

	// Verify total is set correctly.
	for _, e := range events {
		if e.Total != 3 {
			t.Errorf("event Total: expected 3, got %d", e.Total)
			break
		}
	}
}

func TestScan_SubmodulesSortedByPath(t *testing.T) {
	mock := baseMock([]string{"z-module", "a-module", "m-module"})

	eng := New(mock)
	result, err := eng.Scan(context.Background(), ScanOpts{
		RootDir:     "/project",
		Concurrency: 2,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(result.Submodules) != 3 {
		t.Fatalf("expected 3 submodules, got %d", len(result.Submodules))
	}

	expected := []string{"a-module", "m-module", "z-module"}
	for i, sub := range result.Submodules {
		if sub.Path != expected[i] {
			t.Errorf("index %d: expected %s, got %s", i, expected[i], sub.Path)
		}
	}
}

func TestSubmoduleInfo_HasStatus(t *testing.T) {
	info := &SubmoduleInfo{
		Statuses: []git.SubmoduleStatus{git.StatusModified, git.StatusAhead},
	}
	if !info.HasStatus(git.StatusModified) {
		t.Error("should have StatusModified")
	}
	if !info.HasStatus(git.StatusAhead) {
		t.Error("should have StatusAhead")
	}
	if info.HasStatus(git.StatusPending) {
		t.Error("should not have StatusPending")
	}
}

func TestSubmoduleInfo_PrimaryStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []git.SubmoduleStatus
		want     git.SubmoduleStatus
	}{
		{"empty defaults to current", nil, git.StatusCurrent},
		{"single status", []git.SubmoduleStatus{git.StatusPending}, git.StatusPending},
		{"error wins over modified", []git.SubmoduleStatus{git.StatusModified, git.StatusError}, git.StatusError},
		{"modified wins over ahead", []git.SubmoduleStatus{git.StatusAhead, git.StatusModified}, git.StatusModified},
		{"conflict wins over ahead", []git.SubmoduleStatus{git.StatusAhead, git.StatusConflict}, git.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &SubmoduleInfo{Statuses: tt.statuses}
			got := info.PrimaryStatus()
			if got != tt.want {
				t.Errorf("PrimaryStatus() = %s, want %s", got, tt.want)
			}
		})
	}
}
