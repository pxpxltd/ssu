package stack

import (
	"context"
	"strings"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestExportRecordsBranchesAndDetachedModules(t *testing.T) {
	mock := &git.MockGitService{
		SubmodulePathsFn: func(context.Context, string) ([]string, error) {
			return []string{"b", "a"}, nil
		},
		CurrentBranchFn: func(_ context.Context, dir string) (git.BranchResult, error) {
			if strings.HasSuffix(dir, "/a") {
				return git.BranchResult{Name: "feature/a"}, nil
			}
			return git.BranchResult{Detached: true}, nil
		},
		CurrentSHAFn: func(_ context.Context, dir string) (string, error) {
			if strings.HasSuffix(dir, "/a") {
				return strings.Repeat("a", 40), nil
			}
			return strings.Repeat("b", 40), nil
		},
	}
	f, err := NewService(mock).Export(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(f.Modules) != 2 || f.Modules[0].Path != "a" || f.Modules[0].Branch != "feature/a" {
		t.Fatalf("unexpected modules: %#v", f.Modules)
	}
	if f.Modules[1].Branch != "" {
		t.Fatalf("detached branch should be empty: %#v", f.Modules[1])
	}
}

func TestImportExactSHAFastForwardsExistingBranch(t *testing.T) {
	var checkedOut, resetTo string
	mock := &git.MockGitService{
		RefExistsFn: func(_ context.Context, _, ref string) (bool, error) {
			return ref == testSHA || ref == "refs/heads/develop", nil
		},
		IsAncestorFn: func(context.Context, string, string, string) (bool, error) {
			return true, nil
		},
		CheckoutFn: func(_ context.Context, _, branch string) (git.CheckoutResult, error) {
			checkedOut = branch
			return git.CheckoutResult{Branch: branch}, nil
		},
		ResetHardFn: func(_ context.Context, _, ref string) error {
			resetTo = ref
			return nil
		},
	}
	actions := NewService(mock).Import(context.Background(), []Module{
		{Path: "module", Branch: "develop", SHA: testSHA},
	}, ImportOptions{RootDir: "/repo", Concurrency: 1})
	if actions[0].Status != StatusSynced || checkedOut != "develop" || resetTo != testSHA {
		t.Fatalf("unexpected action=%#v checkout=%q reset=%q", actions[0], checkedOut, resetTo)
	}
}

func TestImportMissingSHAFallsBackToRemoteBranch(t *testing.T) {
	mock := &git.MockGitService{
		RefExistsFn: func(_ context.Context, _, ref string) (bool, error) {
			return ref == "origin/develop", nil
		},
		CheckoutNewBranchFn: func(_ context.Context, _, branch, start string) (git.CheckoutResult, error) {
			if branch != "develop" || start != "origin/develop" {
				t.Fatalf("unexpected create %s at %s", branch, start)
			}
			return git.CheckoutResult{Branch: branch}, nil
		},
	}
	actions := NewService(mock).Import(context.Background(), []Module{
		{Path: "module", Branch: "develop", SHA: testSHA},
	}, ImportOptions{RootDir: "/repo", Concurrency: 1})
	if actions[0].Status != StatusFallback {
		t.Fatalf("expected fallback, got %#v", actions[0])
	}
}

func TestImportSkipsDirtyBeforeCheckout(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		HasLocalChangesFn: func(context.Context, string) (bool, error) { return true, nil },
		CheckoutFn: func(context.Context, string, string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}
	actions := NewService(mock).Import(context.Background(), []Module{
		{Path: "module", Branch: "develop", SHA: testSHA},
	}, ImportOptions{RootDir: "/repo", Concurrency: 1})
	if actions[0].Status != StatusDirty || checkoutCalled {
		t.Fatalf("dirty import action=%#v checkoutCalled=%v", actions[0], checkoutCalled)
	}
}

func TestImportSkipsDivergentTargetBranch(t *testing.T) {
	mock := &git.MockGitService{
		RefExistsFn: func(context.Context, string, string) (bool, error) { return true, nil },
		IsAncestorFn: func(context.Context, string, string, string) (bool, error) {
			return false, nil
		},
	}
	actions := NewService(mock).Import(context.Background(), []Module{
		{Path: "module", Branch: "develop", SHA: testSHA},
	}, ImportOptions{RootDir: "/repo", Concurrency: 1})
	if actions[0].Status != StatusDivergent {
		t.Fatalf("expected divergent, got %#v", actions[0])
	}
}

func TestImportDryRunDoesNotCheckout(t *testing.T) {
	checkoutCalled := false
	mock := &git.MockGitService{
		CheckoutFn: func(context.Context, string, string) (git.CheckoutResult, error) {
			checkoutCalled = true
			return git.CheckoutResult{}, nil
		},
	}
	actions := NewService(mock).Import(context.Background(), []Module{
		{Path: "module", Branch: "develop", SHA: testSHA},
	}, ImportOptions{RootDir: "/repo", Concurrency: 1, DryRun: true})
	if !actions[0].DryRun || checkoutCalled {
		t.Fatalf("dry-run action=%#v checkoutCalled=%v", actions[0], checkoutCalled)
	}
}
