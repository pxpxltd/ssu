package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestCheckout_DetachedWithMatchingBranch(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         "plugins/auth",
			DetachedHead: true,
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
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
	if action.Action != "checked out develop" {
		t.Errorf("expected action 'checked out develop', got %q", action.Action)
	}
	if action.Error != nil {
		t.Errorf("expected no error, got %v", action.Error)
	}
}

func TestCheckout_NotDetachedSkipped(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/blog",
			CurrentBranch: "develop",
			DetachedHead:  false,
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	action := result.Actions[0]
	if action.Action != "skipped (not detached)" {
		t.Errorf("expected 'skipped (not detached)', got %q", action.Action)
	}
	if checkoutCalled {
		t.Error("Checkout should not be called for non-detached submodule")
	}
}

func TestCheckout_NoMatchingBranch(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         "plugins/orphan",
			DetachedHead: true,
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Action != "skipped (no matching branch)" {
		t.Errorf("expected 'skipped (no matching branch)', got %q", action.Action)
	}
	if action.Error != nil {
		t.Errorf("expected no error for skip, got %v", action.Error)
	}
}

func TestCheckout_CheckoutFailure(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Local: []string{"develop"},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			return git.CheckoutResult{}, fmt.Errorf("checkout conflict")
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         "plugins/broken",
			DetachedHead: true,
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout should not return top-level error: %v", err)
	}

	action := result.Actions[0]
	if action.Error == nil {
		t.Error("expected non-nil error for failed checkout")
	}
	if action.Action != "checkout develop failed" {
		t.Errorf("expected 'checkout develop failed', got %q", action.Action)
	}
}

func TestCheckout_RootSkipped(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         ".",
			IsRoot:       true,
			DetachedHead: true,
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions (root skipped), got %d", len(result.Actions))
	}
	if checkoutCalled {
		t.Error("Checkout should not be called for root")
	}
}

func TestCheckout_MultipleParallel(t *testing.T) {
	var checkoutCount int32
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			atomic.AddInt32(&checkoutCount, 1)
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", DetachedHead: true},
		{Path: "plugins/blog", DetachedHead: true},
		{Path: "plugins/pages", DetachedHead: true},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 3,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(result.Actions))
	}
	if atomic.LoadInt32(&checkoutCount) != 3 {
		t.Errorf("expected 3 checkout calls, got %d", checkoutCount)
	}

	for _, action := range result.Actions {
		if action.Error != nil {
			t.Errorf("%s: unexpected error: %v", action.Path, action.Error)
		}
		if action.Action != "checked out develop" {
			t.Errorf("%s: expected 'checked out develop', got %q", action.Path, action.Action)
		}
	}
}

func TestCheckout_ProgressCallback(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Local: []string{"develop"},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	var mu sync.Mutex
	var events []ProgressEvent

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", DetachedHead: true},
		{Path: "plugins/blog", DetachedHead: true},
	}

	_, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
		OnProgress: func(evt ProgressEvent) {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	startedCount := 0
	completedCount := 0
	for _, e := range events {
		switch e.Type {
		case EventStarted:
			startedCount++
		case EventCompleted:
			completedCount++
		}
		if e.Phase != "checkout" {
			t.Errorf("expected Phase 'checkout', got %q", e.Phase)
		}
	}

	if startedCount != 2 {
		t.Errorf("expected 2 EventStarted, got %d", startedCount)
	}
	if completedCount != 2 {
		t.Errorf("expected 2 EventCompleted, got %d", completedCount)
	}
}

func TestCheckout_EmptyTargets(t *testing.T) {
	mock := &git.MockGitService{}

	eng := New(mock)
	result, err := eng.Checkout(context.Background(), nil, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for empty targets, got %d", len(result.Actions))
	}
}

func TestCheckout_FeatureBranchPreferredOverPriority(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Local: []string{"feature/cool-thing", "develop"},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", DetachedHead: true},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Branch != "feature/cool-thing" {
		t.Errorf("expected branch feature/cool-thing, got %s", action.Branch)
	}
	if action.Action != "checked out feature/cool-thing" {
		t.Errorf("expected 'checked out feature/cool-thing', got %q", action.Action)
	}
}

func TestCheckout_ContinueOnError(t *testing.T) {
	mock := &git.MockGitService{
		CurrentSHAFn: func(_ context.Context, dir string) (string, error) {
			if dir == "/project/plugins/fail" {
				return "", fmt.Errorf("corrupt repo")
			}
			return "abc1234", nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{
				Local: []string{"develop"},
			}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/ok", DetachedHead: true},
		{Path: "plugins/fail", DetachedHead: true},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Checkout should not return top-level error: %v", err)
	}

	if len(result.Actions) != 2 {
		t.Fatalf("expected 2 actions (continue-on-error), got %d", len(result.Actions))
	}

	byPath := make(map[string]CheckoutAction)
	for _, a := range result.Actions {
		byPath[a.Path] = a
	}

	fail := byPath["plugins/fail"]
	if fail.Error == nil {
		t.Error("plugins/fail should have error")
	}

	ok := byPath["plugins/ok"]
	if ok.Error != nil {
		t.Errorf("plugins/ok should not have error, got %v", ok.Error)
	}
	if ok.Action != "checked out develop" {
		t.Errorf("plugins/ok: expected 'checked out develop', got %q", ok.Action)
	}
}

// ---------------------------------------------------------------------------
// Reset mode tests
// ---------------------------------------------------------------------------

func TestCheckout_ResetWithMatchingBranch(t *testing.T) {
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, sha string) (git.BranchPointsAtResult, error) {
			if sha == "newsha123" {
				return git.BranchPointsAtResult{
					Local: []string{"develop"},
				}, nil
			}
			return git.BranchPointsAtResult{}, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/auth",
			CurrentBranch: "master",
			CurrentSHA:    "oldsha456",
		},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "newsha123"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Branch != "develop" {
		t.Errorf("expected branch develop, got %s", action.Branch)
	}
	if action.Action != "checked out develop" {
		t.Errorf("expected 'checked out develop', got %q", action.Action)
	}
}

func TestCheckout_ResetNoBranchMatch(t *testing.T) {
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{}, nil
		},
		CheckoutFn: func(_ context.Context, _, ref string) (git.CheckoutResult, error) {
			return git.CheckoutResult{Branch: ref}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", CurrentSHA: "oldsha456"},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "abc1234567890"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if !action.Detached {
		t.Error("expected detached checkout")
	}
	if action.Action != "detached at abc1234" {
		t.Errorf("expected 'detached at abc1234', got %q", action.Action)
	}
}

func TestCheckout_ResetAlreadyAtRecordedSHA(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/auth", CurrentBranch: "develop", CurrentSHA: "same1234", DetachedHead: false},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "same1234"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Action != "already at recorded SHA on develop" {
		t.Errorf("expected skip, got %q", action.Action)
	}
	if checkoutCalled {
		t.Error("Checkout should not be called when already at recorded SHA")
	}
}

func TestCheckout_ResetDirtySkipped(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/dirty", CurrentSHA: "oldsha"},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/dirty": "newsha"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Action != "skipped (dirty working tree)" {
		t.Errorf("expected dirty skip, got %q", action.Action)
	}
	if checkoutCalled {
		t.Error("Checkout should not be called for dirty submodule")
	}
}

func TestCheckout_ResetNotInRecordedSHAs(t *testing.T) {
	mock := &git.MockGitService{}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/unknown", CurrentSHA: "abc1234"},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	action := result.Actions[0]
	if action.Action != "skipped (not in recorded SHAs)" {
		t.Errorf("expected skip, got %q", action.Action)
	}
}

func TestCheckout_ResetRootExcluded(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: ".", IsRoot: true, CurrentSHA: "oldsha"},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{".": "newsha"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions (root excluded), got %d", len(result.Actions))
	}
	if checkoutCalled {
		t.Error("Checkout should not be called for root")
	}
}

func TestCheckout_ResetContinueOnError(t *testing.T) {
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
			return git.BranchPointsAtResult{Local: []string{"develop"}}, nil
		},
		CheckoutFn: func(_ context.Context, dir, branch string) (git.CheckoutResult, error) {
			if dir == "/project/plugins/fail" {
				return git.CheckoutResult{}, fmt.Errorf("checkout failed")
			}
			return git.CheckoutResult{Branch: branch}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{Path: "plugins/ok", CurrentSHA: "old1"},
		{Path: "plugins/fail", CurrentSHA: "old2"},
	}

	result, err := eng.Checkout(context.Background(), targets, CheckoutOpts{
		RootDir:     "/project",
		Concurrency: 1,
		Reset:       true,
		RecordedSHAs: map[string]string{
			"plugins/ok":   "new1",
			"plugins/fail": "new2",
		},
	})
	if err != nil {
		t.Fatalf("should not return top-level error: %v", err)
	}

	if len(result.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(result.Actions))
	}

	byPath := make(map[string]CheckoutAction)
	for _, a := range result.Actions {
		byPath[a.Path] = a
	}

	if byPath["plugins/ok"].Error != nil {
		t.Errorf("plugins/ok: unexpected error: %v", byPath["plugins/ok"].Error)
	}
	if byPath["plugins/fail"].Error == nil {
		t.Error("plugins/fail: expected error")
	}
}

// A remote-only match whose name is already taken by a local branch must not be
// checked out by bare name: git would resolve the local branch, which sits on a
// different commit, leaving HEAD off the recorded SHA.
func TestCheckout_ResetRemoteMatchShadowedByLocalBranch(t *testing.T) {
	var checkedOut string
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, sha string) (git.BranchPointsAtResult, error) {
			if sha == "abc1234567890" {
				return git.BranchPointsAtResult{
					Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
				}, nil
			}
			return git.BranchPointsAtResult{}, nil
		},
		RefExistsFn: func(_ context.Context, _, ref string) (bool, error) {
			return ref == "refs/heads/develop", nil // local develop exists, elsewhere
		},
		CheckoutFn: func(_ context.Context, _, ref string) (git.CheckoutResult, error) {
			checkedOut = ref
			return git.CheckoutResult{Branch: ref}, nil
		},
	}

	eng := New(mock)
	result, err := eng.Checkout(context.Background(), []*SubmoduleInfo{
		{Path: "plugins/auth", CurrentSHA: "oldsha456"},
	}, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "abc1234567890"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if checkedOut != "abc1234567890" {
		t.Errorf("checked out %q, want the recorded SHA", checkedOut)
	}
	action := result.Actions[0]
	if !action.Detached {
		t.Error("expected detached checkout at the recorded SHA")
	}
	if action.Action != "detached at abc1234 (local develop points elsewhere)" {
		t.Errorf("unexpected action %q", action.Action)
	}
}

// With no local branch of that name, git DWIM creates a tracking branch at the
// recorded SHA, so checking out by name is safe and preferred.
func TestCheckout_ResetRemoteMatchWithoutLocalBranch(t *testing.T) {
	var checkedOut string
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, nil
		},
		BranchesPointingAtFn: func(_ context.Context, _, sha string) (git.BranchPointsAtResult, error) {
			if sha == "abc1234567890" {
				return git.BranchPointsAtResult{
					Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
				}, nil
			}
			return git.BranchPointsAtResult{}, nil
		},
		RefExistsFn: func(_ context.Context, _, _ string) (bool, error) {
			return false, nil
		},
		CheckoutFn: func(_ context.Context, _, ref string) (git.CheckoutResult, error) {
			checkedOut = ref
			return git.CheckoutResult{Branch: ref}, nil
		},
	}

	eng := New(mock)
	result, err := eng.Checkout(context.Background(), []*SubmoduleInfo{
		{Path: "plugins/auth", CurrentSHA: "oldsha456"},
	}, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "abc1234567890"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if checkedOut != "develop" {
		t.Errorf("checked out %q, want develop", checkedOut)
	}
	if action := result.Actions[0]; action.Action != "checked out develop" || action.Detached {
		t.Errorf("unexpected action %q (detached=%v)", action.Action, action.Detached)
	}
}

// Reset moves HEAD, so an unreadable working tree must abort rather than be
// treated as clean.
func TestCheckout_ResetDirtyCheckFails(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		HasLocalChangesFn: func(_ context.Context, _ string) (bool, error) {
			return false, errors.New("git status failed")
		},
		CheckoutFn: func(_ context.Context, _, _ string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}

	eng := New(mock)
	result, err := eng.Checkout(context.Background(), []*SubmoduleInfo{
		{Path: "plugins/auth", CurrentSHA: "oldsha456"},
	}, CheckoutOpts{
		RootDir:      "/project",
		Concurrency:  1,
		Reset:        true,
		RecordedSHAs: map[string]string{"plugins/auth": "abc1234567890"},
	})
	if err != nil {
		t.Fatalf("Checkout error: %v", err)
	}

	if checkoutCalled {
		t.Error("must not check out when the working tree state is unknown")
	}
	action := result.Actions[0]
	if action.Error == nil {
		t.Error("expected an error action")
	}
	if action.Action != "error checking working tree" {
		t.Errorf("unexpected action %q", action.Action)
	}
}
