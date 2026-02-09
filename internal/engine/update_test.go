package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// pendingInfo creates a SubmoduleInfo that is behind remote and ready for update.
func pendingInfo(path string, behind int, hasChanges bool) *SubmoduleInfo {
	statuses := []git.SubmoduleStatus{git.StatusPending}
	if hasChanges {
		statuses = append(statuses, git.StatusModified)
	}
	return &SubmoduleInfo{
		Path:          path,
		Statuses:      statuses,
		TargetBranch:  "develop",
		CommitsBehind: behind,
		HasChanges:    hasChanges,
	}
}

// findAction returns the UpdateAction for the given path, or nil.
func findAction(actions []UpdateAction, path string) *UpdateAction {
	for i := range actions {
		if actions[i].Path == path {
			return &actions[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestUpdate_CleanSubmoduleBehindRemote(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/auth", 3, false)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(result.Actions))
	}

	a := result.Actions[0]
	if a.Path != "plugins/auth" {
		t.Errorf("path: expected plugins/auth, got %s", a.Path)
	}
	if !containsStatus(a.AfterStatus, git.StatusCurrent) {
		t.Errorf("AfterStatus should contain current, got %v", a.AfterStatus)
	}
	if !strings.Contains(a.Action, "merged") {
		t.Errorf("Action should contain 'merged', got %q", a.Action)
	}
	if a.Error != nil {
		t.Errorf("unexpected error: %v", a.Error)
	}
	if a.ConflictHint != "" {
		t.Errorf("ConflictHint should be empty, got %q", a.ConflictHint)
	}
}

func TestUpdate_DirtySubmoduleStashMergePop(t *testing.T) {
	var callOrder []string

	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			callOrder = append(callOrder, "stash")
			return git.StashResult{Created: true}, nil
		},
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			callOrder = append(callOrder, "merge")
			return git.MergeResult{Success: true}, nil
		},
		StashPopFn: func(_ context.Context, _ string) (git.StashResult, error) {
			callOrder = append(callOrder, "stash-pop")
			return git.StashResult{Created: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/blog", 2, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusCurrent) {
		t.Errorf("AfterStatus should contain current, got %v", a.AfterStatus)
	}
	if !strings.Contains(a.Action, "stashed") {
		t.Errorf("Action should contain 'stashed', got %q", a.Action)
	}
	if !strings.Contains(a.Action, "merged") {
		t.Errorf("Action should contain 'merged', got %q", a.Action)
	}
	if !strings.Contains(a.Action, "restored") {
		t.Errorf("Action should contain 'restored', got %q", a.Action)
	}

	// Verify call order: stash -> merge -> stash-pop
	expected := []string{"stash", "merge", "stash-pop"}
	if len(callOrder) != len(expected) {
		t.Fatalf("call order: expected %v, got %v", expected, callOrder)
	}
	for i, c := range callOrder {
		if c != expected[i] {
			t.Errorf("call %d: expected %s, got %s", i, expected[i], c)
		}
	}
}

func TestUpdate_DirtySubmoduleMergeConflictAfterStash(t *testing.T) {
	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Conflict: true}, &git.GitError{
				Op:     "merge",
				Stderr: "CONFLICT (content): Merge conflict in file.txt",
				Err:    fmt.Errorf("exit status 1"),
			}
		},
		MergeAbortFn: func(_ context.Context, _ string) error {
			return nil
		},
		StashPopFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/pages", 1, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusConflict) {
		t.Errorf("AfterStatus should contain conflict, got %v", a.AfterStatus)
	}
	if a.ConflictHint == "" {
		t.Error("ConflictHint should not be empty")
	}
	if !strings.Contains(a.ConflictHint, "git merge") {
		t.Errorf("ConflictHint should contain 'git merge', got %q", a.ConflictHint)
	}
	if !strings.Contains(a.ConflictHint, "git stash") {
		t.Errorf("ConflictHint should contain 'git stash', got %q", a.ConflictHint)
	}
	if !strings.Contains(a.ConflictHint, "plugins/pages") {
		t.Errorf("ConflictHint should contain submodule path, got %q", a.ConflictHint)
	}
	if !strings.Contains(a.Action, "conflict") {
		t.Errorf("Action should contain 'conflict', got %q", a.Action)
	}
}

func TestUpdate_DirtySubmoduleStashFails(t *testing.T) {
	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{}, fmt.Errorf("cannot stash: permission denied")
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/payment", 1, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusError) {
		t.Errorf("AfterStatus should contain error, got %v", a.AfterStatus)
	}
	if !strings.Contains(a.Action, "stash failed") {
		t.Errorf("Action should contain 'stash failed', got %q", a.Action)
	}
	if a.Error == nil {
		t.Error("Error should not be nil")
	}
}

func TestUpdate_CleanSubmoduleMergeConflict(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Conflict: true}, &git.GitError{
				Op:     "merge",
				Stderr: "CONFLICT (content): Merge conflict in main.go",
				Err:    fmt.Errorf("exit status 1"),
			}
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("lib/core", 1, false)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusConflict) {
		t.Errorf("AfterStatus should contain conflict, got %v", a.AfterStatus)
	}
	if a.ConflictHint == "" {
		t.Error("ConflictHint should not be empty for clean merge conflict")
	}
	if !strings.Contains(a.ConflictHint, "git merge") {
		t.Errorf("ConflictHint should contain 'git merge', got %q", a.ConflictHint)
	}
	// Clean merge conflict hint should NOT have stash commands
	if strings.Contains(a.ConflictHint, "git stash") {
		t.Errorf("Clean merge ConflictHint should not contain 'git stash', got %q", a.ConflictHint)
	}
}

func TestUpdate_MergeConflictAbortFails(t *testing.T) {
	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Conflict: true}, &git.GitError{
				Op:     "merge",
				Stderr: "CONFLICT",
				Err:    fmt.Errorf("exit status 1"),
			}
		},
		MergeAbortFn: func(_ context.Context, _ string) error {
			return fmt.Errorf("abort failed: lock file exists")
		},
		StashPopFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil // best-effort
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/broken", 1, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusError) {
		t.Errorf("AfterStatus should contain error when abort fails, got %v", a.AfterStatus)
	}
	if a.Error == nil {
		t.Error("Error should not be nil when abort fails")
	}
	if !strings.Contains(a.Action, "abort") {
		t.Errorf("Action should mention abort, got %q", a.Action)
	}
}

func TestUpdate_RootSubmoduleSkipped(t *testing.T) {
	mergeCalled := false
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			mergeCalled = true
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          ".",
			IsRoot:        true,
			Statuses:      []git.SubmoduleStatus{git.StatusPending},
			TargetBranch:  "develop",
			CommitsBehind: 5,
		},
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for root-only targets, got %d", len(result.Actions))
	}
	if mergeCalled {
		t.Error("merge should not be called for root submodule")
	}
}

func TestUpdate_CurrentSubmoduleSkipped(t *testing.T) {
	mergeCalled := false
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			mergeCalled = true
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:         "plugins/uptodate",
			Statuses:     []git.SubmoduleStatus{git.StatusCurrent},
			TargetBranch: "develop",
		},
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for current submodule, got %d", len(result.Actions))
	}
	if mergeCalled {
		t.Error("merge should not be called for current submodule")
	}
}

func TestUpdate_MissingSubmoduleSkipped(t *testing.T) {
	mock := &git.MockGitService{}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:     "plugins/missing",
			Statuses: []git.SubmoduleStatus{git.StatusMissing},
		},
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for missing submodule, got %d", len(result.Actions))
	}
}

func TestUpdate_MultipleSubmodulesParallel(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := make([]*SubmoduleInfo, 5)
	for i := range targets {
		targets[i] = pendingInfo(fmt.Sprintf("mod/%d", i), 1, false)
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 3,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(result.Actions))
	}

	// Every action should be a successful merge.
	for _, a := range result.Actions {
		if a.Error != nil {
			t.Errorf("%s: unexpected error: %v", a.Path, a.Error)
		}
		if !containsStatus(a.AfterStatus, git.StatusCurrent) {
			t.Errorf("%s: AfterStatus should contain current, got %v", a.Path, a.AfterStatus)
		}
	}
}

func TestUpdate_ContinueOnError(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, dir, _ string) (git.MergeResult, error) {
			if strings.HasSuffix(dir, "mod-fail") {
				return git.MergeResult{}, fmt.Errorf("network error")
			}
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		pendingInfo("mod-ok-1", 1, false),
		pendingInfo("mod-fail", 1, false),
		pendingInfo("mod-ok-2", 2, false),
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 3 {
		t.Fatalf("expected 3 actions (continue-on-error), got %d", len(result.Actions))
	}

	fail := findAction(result.Actions, "mod-fail")
	if fail == nil {
		t.Fatal("mod-fail action not found")
	}
	if fail.Error == nil {
		t.Error("mod-fail should have an error")
	}
	if !containsStatus(fail.AfterStatus, git.StatusError) {
		t.Errorf("mod-fail AfterStatus should be error, got %v", fail.AfterStatus)
	}

	ok1 := findAction(result.Actions, "mod-ok-1")
	if ok1 == nil {
		t.Fatal("mod-ok-1 action not found")
	}
	if ok1.Error != nil {
		t.Errorf("mod-ok-1 should not have error: %v", ok1.Error)
	}

	ok2 := findAction(result.Actions, "mod-ok-2")
	if ok2 == nil {
		t.Fatal("mod-ok-2 action not found")
	}
	if ok2.Error != nil {
		t.Errorf("mod-ok-2 should not have error: %v", ok2.Error)
	}
}

func TestUpdate_BeforeStatusRecorded(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:          "plugins/auth",
			Statuses:      []git.SubmoduleStatus{git.StatusPending, git.StatusModified},
			TargetBranch:  "develop",
			CommitsBehind: 3,
			HasChanges:    false, // test as clean despite compound statuses
		},
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	// BeforeStatus should capture the original compound statuses.
	if len(a.BeforeStatus) != 2 {
		t.Fatalf("BeforeStatus should have 2 statuses, got %d", len(a.BeforeStatus))
	}
	if !containsStatus(a.BeforeStatus, git.StatusPending) {
		t.Error("BeforeStatus should contain pending")
	}
	if !containsStatus(a.BeforeStatus, git.StatusModified) {
		t.Error("BeforeStatus should contain modified")
	}
}

func TestUpdate_DirtyMergeFailNonConflict(t *testing.T) {
	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{}, fmt.Errorf("fatal: refusing to merge unrelated histories")
		},
		MergeAbortFn: func(_ context.Context, _ string) error {
			return nil
		},
		StashPopFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/weird", 1, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if !containsStatus(a.AfterStatus, git.StatusError) {
		t.Errorf("AfterStatus should be error for non-conflict merge fail, got %v", a.AfterStatus)
	}
	if a.ConflictHint != "" {
		t.Errorf("ConflictHint should be empty for non-conflict error, got %q", a.ConflictHint)
	}
	if !strings.Contains(a.Action, "merge failed") {
		t.Errorf("Action should say 'merge failed', got %q", a.Action)
	}
}

func TestUpdate_StashPopFailsAfterMerge(t *testing.T) {
	mock := &git.MockGitService{
		StashFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{Created: true}, nil
		},
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
		StashPopFn: func(_ context.Context, _ string) (git.StashResult, error) {
			return git.StashResult{}, fmt.Errorf("stash pop conflict")
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("plugins/popfail", 2, true)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	a := result.Actions[0]
	if a.Error == nil {
		t.Error("Error should not be nil when stash pop fails after merge")
	}
	if !containsStatus(a.AfterStatus, git.StatusError) {
		t.Errorf("AfterStatus should be error, got %v", a.AfterStatus)
	}
	if !strings.Contains(a.Action, "stash pop failed") {
		t.Errorf("Action should mention stash pop failure, got %q", a.Action)
	}
}

func TestUpdate_ProgressEvents(t *testing.T) {
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
	}

	var events []ProgressEvent

	eng := New(mock)
	targets := []*SubmoduleInfo{
		pendingInfo("sub1", 1, false),
		pendingInfo("sub2", 2, false),
	}

	_, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
		OnProgress: func(evt ProgressEvent) {
			events = append(events, evt)
		},
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	started := 0
	completed := 0
	for _, e := range events {
		if e.Phase != "update" {
			t.Errorf("event phase should be 'update', got %q", e.Phase)
		}
		switch e.Type {
		case EventStarted:
			started++
		case EventCompleted:
			completed++
		}
	}

	if started != 2 {
		t.Errorf("expected 2 EventStarted, got %d", started)
	}
	if completed != 2 {
		t.Errorf("expected 2 EventCompleted, got %d", completed)
	}
}

func TestUpdate_SkippedStatusSubmodule(t *testing.T) {
	mock := &git.MockGitService{}

	eng := New(mock)
	targets := []*SubmoduleInfo{
		{
			Path:     "plugins/skipped",
			Statuses: []git.SubmoduleStatus{git.StatusSkipped},
		},
	}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}

	if len(result.Actions) != 0 {
		t.Errorf("expected 0 actions for skipped submodule, got %d", len(result.Actions))
	}
}

func TestUpdate_DefaultConcurrency(t *testing.T) {
	// Verify that concurrency=0 doesn't panic (uses runtime.NumCPU).
	mock := &git.MockGitService{
		MergeFn: func(_ context.Context, _, _ string) (git.MergeResult, error) {
			return git.MergeResult{Success: true}, nil
		},
	}

	eng := New(mock)
	targets := []*SubmoduleInfo{pendingInfo("mod1", 1, false)}

	result, err := eng.Update(context.Background(), targets, UpdateOpts{
		RootDir:     "/project",
		Concurrency: 0, // should default to NumCPU
	})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if len(result.Actions) != 1 {
		t.Errorf("expected 1 action, got %d", len(result.Actions))
	}
}

// containsStatus reports whether the status slice contains the given status.
func containsStatus(statuses []git.SubmoduleStatus, s git.SubmoduleStatus) bool {
	for _, st := range statuses {
		if st == s {
			return true
		}
	}
	return false
}
