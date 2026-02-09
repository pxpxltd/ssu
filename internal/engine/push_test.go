package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestPush_SimplePush(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{
				Remote:      "origin",
				Branch:      "develop",
				NewTracking: false,
			}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/auth",
			Statuses:      []git.SubmoduleStatus{git.StatusAhead},
			CurrentBranch: "develop",
			CommitsAhead:  2,
			DetachedHead:  false,
		},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.Path != "plugins/auth" {
		t.Errorf("expected path plugins/auth, got %s", action.Path)
	}
	if action.Branch != "develop" {
		t.Errorf("expected branch develop, got %s", action.Branch)
	}
	if action.Action != "pushed" {
		t.Errorf("expected action 'pushed', got %s", action.Action)
	}
	if action.Error != nil {
		t.Errorf("expected no error, got %v", action.Error)
	}
}

func TestPush_NewTracking(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{
				Remote:      "origin",
				Branch:      "feature/new",
				NewTracking: true,
			}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/blog",
			Statuses:      []git.SubmoduleStatus{git.StatusAhead},
			CurrentBranch: "feature/new",
			CommitsAhead:  1,
		},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.Action != "set up tracking + pushed" {
		t.Errorf("expected action containing 'tracking', got %q", action.Action)
	}
	if action.Branch != "feature/new" {
		t.Errorf("expected branch feature/new, got %s", action.Branch)
	}
}

func TestPush_DetachedHeadSkipped(t *testing.T) {
	pushCalled := false
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			pushCalled = true
			return git.PushResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         "plugins/detached",
			Statuses:     []git.SubmoduleStatus{git.StatusAhead},
			DetachedHead: true,
			CommitsAhead: 3,
		},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.Action != "skipped (detached HEAD)" {
		t.Errorf("expected action 'skipped (detached HEAD)', got %q", action.Action)
	}
	if action.Error != nil {
		t.Errorf("detached HEAD skip should not have error, got %v", action.Error)
	}
	if pushCalled {
		t.Error("Push mock should not be called for detached HEAD submodule")
	}
}

func TestPush_PushFailure(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{}, fmt.Errorf("permission denied")
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/locked",
			Statuses:      []git.SubmoduleStatus{git.StatusAhead},
			CurrentBranch: "develop",
			CommitsAhead:  1,
		},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push should not return top-level error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.Action != "push failed" {
		t.Errorf("expected action 'push failed', got %q", action.Action)
	}
	if action.Error == nil {
		t.Error("expected non-nil error for failed push")
	}
}

func TestPush_RootSkipped(t *testing.T) {
	pushCalled := false
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			pushCalled = true
			return git.PushResult{Branch: "develop"}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          ".",
			IsRoot:        true,
			Statuses:      []git.SubmoduleStatus{git.StatusAhead},
			CurrentBranch: "develop",
			CommitsAhead:  5,
		},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions (root skipped), got %d", len(result.Actions))
	}
	if pushCalled {
		t.Error("Push mock should not be called for root submodule")
	}
}

func TestPush_MultipleSubmodulesParallel(t *testing.T) {
	var pushCount int32
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			atomic.AddInt32(&pushCount, 1)
			return git.PushResult{Remote: "origin", Branch: "develop"}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
		{Path: "plugins/blog", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 2},
		{Path: "plugins/pages", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 3},
		{Path: "plugins/payment", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 4,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 4 {
		t.Fatalf("expected 4 actions, got %d", len(result.Actions))
	}
	if atomic.LoadInt32(&pushCount) != 4 {
		t.Errorf("expected 4 push calls, got %d", pushCount)
	}

	// All should succeed.
	for _, action := range result.Actions {
		if action.Error != nil {
			t.Errorf("%s: unexpected error: %v", action.Path, action.Error)
		}
		if action.Action != "pushed" {
			t.Errorf("%s: expected action 'pushed', got %q", action.Path, action.Action)
		}
	}
}

func TestPush_ContinueOnError(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, dir string, _ git.PushOpts) (git.PushResult, error) {
			if dir == "/project/plugins/fail" {
				return git.PushResult{}, fmt.Errorf("network error")
			}
			return git.PushResult{Remote: "origin", Branch: "develop"}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/ok1", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
		{Path: "plugins/fail", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
		{Path: "plugins/ok2", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1, // serial for deterministic mock dir matching
	})
	if err != nil {
		t.Fatalf("Push should not return top-level error: %v", err)
	}

	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions (continue-on-error), got %d", len(result.Actions))
	}

	// Build lookup by path.
	byPath := make(map[string]PushAction)
	for _, a := range result.Actions {
		byPath[a.Path] = a
	}

	// Failing submodule should have error.
	fail := byPath["plugins/fail"]
	if fail.Error == nil {
		t.Error("plugins/fail should have error")
	}
	if fail.Action != "push failed" {
		t.Errorf("plugins/fail: expected 'push failed', got %q", fail.Action)
	}

	// Successful submodules should have no error.
	for _, path := range []string{"plugins/ok1", "plugins/ok2"} {
		ok := byPath[path]
		if ok.Error != nil {
			t.Errorf("%s should not have error, got %v", path, ok.Error)
		}
		if ok.Action != "pushed" {
			t.Errorf("%s: expected 'pushed', got %q", path, ok.Action)
		}
	}
}

func TestPush_ProgressCallback(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{Remote: "origin", Branch: "develop"}, nil
		},
	}

	var mu sync.Mutex
	var events []ProgressEvent

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
		{Path: "plugins/blog", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	_, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
		OnProgress: func(evt ProgressEvent) {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	// Each submodule should fire EventStarted + EventCompleted = 2 events.
	startedCount := 0
	completedCount := 0
	for _, e := range events {
		switch e.Type {
		case EventStarted:
			startedCount++
		case EventCompleted:
			completedCount++
		}
		// All events should have Phase="push".
		if e.Phase != "push" {
			t.Errorf("expected Phase 'push', got %q", e.Phase)
		}
	}

	if startedCount != 2 {
		t.Errorf("expected 2 EventStarted, got %d", startedCount)
	}
	if completedCount != 2 {
		t.Errorf("expected 2 EventCompleted, got %d", completedCount)
	}

	// Verify Total is correct.
	for _, e := range events {
		if e.Total != 2 {
			t.Errorf("event Total: expected 2, got %d", e.Total)
			break
		}
	}
}

func TestPush_ProgressCallbackFiresFailedEvent(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{}, fmt.Errorf("push rejected")
		},
	}

	var mu sync.Mutex
	var events []ProgressEvent

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/broken", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	_, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
		OnProgress: func(evt ProgressEvent) {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	failedCount := 0
	for _, e := range events {
		if e.Type == EventFailed {
			failedCount++
			if e.Error == nil {
				t.Error("EventFailed should have non-nil Error")
			}
		}
	}

	if failedCount != 1 {
		t.Errorf("expected 1 EventFailed, got %d", failedCount)
	}
}

func TestPush_EmptyTargets(t *testing.T) {
	mock := &git.MockGitService{}

	eng := New(mock)
	result, err := eng.Push(context.Background(), nil, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for empty targets, got %d", len(result.Actions))
	}
}

func TestPush_DefaultConcurrency(t *testing.T) {
	// Verify that passing Concurrency=0 doesn't panic (uses runtime.NumCPU).
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, _ string, _ git.PushOpts) (git.PushResult, error) {
			return git.PushResult{Remote: "origin", Branch: "develop"}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 0, // should default to runtime.NumCPU()
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(result.Actions))
	}
}

func TestPush_MixedScenarios(t *testing.T) {
	mock := &git.MockGitService{
		PushFn: func(_ context.Context, dir string, _ git.PushOpts) (git.PushResult, error) {
			switch dir {
			case "/project/plugins/new-track":
				return git.PushResult{Remote: "origin", Branch: "feature/x", NewTracking: true}, nil
			case "/project/plugins/normal":
				return git.PushResult{Remote: "origin", Branch: "develop"}, nil
			default:
				return git.PushResult{}, fmt.Errorf("unexpected dir: %s", dir)
			}
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: ".", IsRoot: true, Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
		{Path: "plugins/detached", DetachedHead: true, Statuses: []git.SubmoduleStatus{git.StatusAhead}, CommitsAhead: 1},
		{Path: "plugins/new-track", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "feature/x", CommitsAhead: 2},
		{Path: "plugins/normal", Statuses: []git.SubmoduleStatus{git.StatusAhead}, CurrentBranch: "develop", CommitsAhead: 1},
	}

	result, err := eng.Push(context.Background(), targets, PushOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Push error: %v", err)
	}

	// Root should be filtered out: 3 actions expected (detached + new-track + normal).
	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions (root filtered), got %d", len(result.Actions))
	}

	byPath := make(map[string]PushAction)
	for _, a := range result.Actions {
		byPath[a.Path] = a
	}

	// Root should not appear.
	if _, ok := byPath["."]; ok {
		t.Error("root should not appear in actions")
	}

	// Detached HEAD: skipped, no error.
	det := byPath["plugins/detached"]
	if det.Action != "skipped (detached HEAD)" {
		t.Errorf("detached: expected 'skipped (detached HEAD)', got %q", det.Action)
	}
	if det.Error != nil {
		t.Errorf("detached: expected no error, got %v", det.Error)
	}

	// New tracking: set up tracking + pushed.
	track := byPath["plugins/new-track"]
	if track.Action != "set up tracking + pushed" {
		t.Errorf("new-track: expected 'set up tracking + pushed', got %q", track.Action)
	}
	if track.Branch != "feature/x" {
		t.Errorf("new-track: expected branch feature/x, got %s", track.Branch)
	}

	// Normal: pushed.
	norm := byPath["plugins/normal"]
	if norm.Action != "pushed" {
		t.Errorf("normal: expected 'pushed', got %q", norm.Action)
	}
}
