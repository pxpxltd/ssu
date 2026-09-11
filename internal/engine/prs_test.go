package engine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/github"
)

// fakeLister returns canned PRs (or errors) keyed by remote URL.
type fakeLister struct {
	mu    sync.Mutex
	prs   map[string][]github.PullRequest
	errs  map[string]error
	calls []string
}

func (f *fakeLister) OpenPRs(_ context.Context, remoteURL string) ([]github.PullRequest, error) {
	f.mu.Lock()
	f.calls = append(f.calls, remoteURL)
	f.mu.Unlock()
	if err := f.errs[remoteURL]; err != nil {
		return nil, err
	}
	return f.prs[remoteURL], nil
}

// urlMock returns "url:<dir>" as the origin URL of every directory, except
// mod-noremote which has no origin.
func urlMock() *git.MockGitService {
	return &git.MockGitService{
		RemoteURLFn: func(_ context.Context, dir, remote string) (string, error) {
			if remote != "origin" {
				return "", errors.New("unexpected remote " + remote)
			}
			if dir == "/project/mod-noremote" {
				return "", errors.New("no such remote")
			}
			return "url:" + dir, nil
		},
	}
}

func TestAttachPullRequests(t *testing.T) {
	result := &ScanResult{
		Root: &SubmoduleInfo{Path: ".", IsRoot: true, Statuses: []git.SubmoduleStatus{git.StatusCurrent}},
		Submodules: []*SubmoduleInfo{
			{Path: "mod-a", Statuses: []git.SubmoduleStatus{git.StatusCurrent}},
			{Path: "mod-b", Statuses: []git.SubmoduleStatus{git.StatusPending}},
			{Path: "mod-fail", Statuses: []git.SubmoduleStatus{git.StatusCurrent}},
			{Path: "mod-noremote", Statuses: []git.SubmoduleStatus{git.StatusCurrent}},
			{Path: "mod-missing", Statuses: []git.SubmoduleStatus{git.StatusMissing}},
			{Path: "mod-skipped", Statuses: []git.SubmoduleStatus{git.StatusSkipped}},
		},
	}

	lister := &fakeLister{
		prs: map[string][]github.PullRequest{
			"url:/project":       {{Number: 41, Title: "Bump deps"}},
			"url:/project/mod-a": {{Number: 12, Title: "SSO"}, {Number: 9, Title: "Fix"}},
		},
		errs: map[string]error{
			"url:/project/mod-fail": errors.New("not a GitHub remote (gitlab.com)"),
		},
	}

	var mu sync.Mutex
	var events []ProgressEvent
	eng := New(urlMock())
	eng.AttachPullRequests(context.Background(), lister, result, PROpts{
		RootDir:     "/project",
		Concurrency: 2,
		OnProgress: func(evt ProgressEvent) {
			mu.Lock()
			events = append(events, evt)
			mu.Unlock()
		},
	})

	if !result.Root.PRChecked || len(result.Root.PullRequests) != 1 || result.Root.PullRequests[0].Number != 41 {
		t.Errorf("root: expected PR #41, got checked=%v prs=%v", result.Root.PRChecked, result.Root.PullRequests)
	}

	byPath := map[string]*SubmoduleInfo{}
	for _, sm := range result.Submodules {
		byPath[sm.Path] = sm
	}

	if a := byPath["mod-a"]; !a.PRChecked || len(a.PullRequests) != 2 || a.PRError != nil {
		t.Errorf("mod-a: expected 2 PRs, got checked=%v prs=%v err=%v", a.PRChecked, a.PullRequests, a.PRError)
	}
	if b := byPath["mod-b"]; !b.PRChecked || len(b.PullRequests) != 0 || b.PRError != nil {
		t.Errorf("mod-b: expected checked with no PRs, got checked=%v prs=%v err=%v", b.PRChecked, b.PullRequests, b.PRError)
	}
	if f := byPath["mod-fail"]; !f.PRChecked || f.PRError == nil || f.PRError.Error() != "not a GitHub remote (gitlab.com)" {
		t.Errorf("mod-fail: expected lister error, got %v", f.PRError)
	}
	if n := byPath["mod-noremote"]; !n.PRChecked || n.PRError == nil {
		t.Errorf("mod-noremote: expected remote URL error, got %v", n.PRError)
	}
	for _, p := range []string{"mod-missing", "mod-skipped"} {
		if byPath[p].PRChecked {
			t.Errorf("%s: should not be checked", p)
		}
	}

	// Root + 4 looked-up submodules; missing/skipped never reach the lister.
	if len(lister.calls) != 4 {
		t.Errorf("expected 4 lister calls, got %d: %v", len(lister.calls), lister.calls)
	}

	var completed, failed int
	for _, evt := range events {
		if evt.Total != 5 {
			t.Errorf("expected Total=5, got %d", evt.Total)
		}
		switch evt.Type {
		case EventCompleted:
			completed++
		case EventFailed:
			failed++
		}
	}
	if completed != 3 || failed != 2 {
		t.Errorf("expected 3 completed and 2 failed events, got %d and %d", completed, failed)
	}
}

func TestAttachPullRequests_CustomRemote(t *testing.T) {
	var gotRemote string
	mock := &git.MockGitService{
		RemoteURLFn: func(_ context.Context, _, remote string) (string, error) {
			gotRemote = remote
			return "url", nil
		},
	}
	result := &ScanResult{Root: &SubmoduleInfo{Path: ".", IsRoot: true}}

	New(mock).AttachPullRequests(context.Background(), &fakeLister{}, result, PROpts{RootDir: "/project", Remote: "upstream"})

	if gotRemote != "upstream" {
		t.Errorf("expected remote upstream, got %q", gotRemote)
	}
}
