package git_test

import (
	"context"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

// Compile-time interface satisfaction check.
var _ git.GitService = (*git.MockGitService)(nil)

func TestMockDefaults(t *testing.T) {
	m := &git.MockGitService{}
	ctx := context.Background()

	t.Run("CurrentBranch returns develop", func(t *testing.T) {
		br, err := m.CurrentBranch(ctx, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if br.Name != "develop" {
			t.Errorf("expected branch name %q, got %q", "develop", br.Name)
		}
		if br.Detached {
			t.Error("expected Detached=false for default")
		}
	})

	t.Run("IsDetachedHead returns false", func(t *testing.T) {
		detached, err := m.IsDetachedHead(ctx, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if detached {
			t.Error("expected false for default IsDetachedHead")
		}
	})

	t.Run("HasRemoteBranch returns true", func(t *testing.T) {
		has, err := m.HasRemoteBranch(ctx, "/tmp", "origin", "develop")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected true for default HasRemoteBranch")
		}
	})

	t.Run("CurrentSHA returns abc1234", func(t *testing.T) {
		sha, err := m.CurrentSHA(ctx, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sha != "abc1234" {
			t.Errorf("expected SHA %q, got %q", "abc1234", sha)
		}
	})

	t.Run("SubmodulePaths returns nil", func(t *testing.T) {
		paths, err := m.SubmodulePaths(ctx, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if paths != nil {
			t.Errorf("expected nil paths, got %v", paths)
		}
	})

	t.Run("IsSubmoduleInitialized returns true", func(t *testing.T) {
		if !m.IsSubmoduleInitialized("/tmp", "sub") {
			t.Error("expected true for default IsSubmoduleInitialized")
		}
	})

	t.Run("Merge returns success", func(t *testing.T) {
		result, err := m.Merge(ctx, "/tmp", "origin/develop")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("expected Success=true for default Merge")
		}
	})

	t.Run("Stash returns created", func(t *testing.T) {
		result, err := m.Stash(ctx, "/tmp")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Created {
			t.Error("expected Created=true for default Stash")
		}
	})

	t.Run("Checkout returns requested branch", func(t *testing.T) {
		result, err := m.Checkout(ctx, "/tmp", "main")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Branch != "main" {
			t.Errorf("expected branch %q, got %q", "main", result.Branch)
		}
	})

	t.Run("Push returns remote and branch from opts", func(t *testing.T) {
		result, err := m.Push(ctx, "/tmp", git.PushOpts{Remote: "upstream", Branch: "develop"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Remote != "upstream" {
			t.Errorf("expected remote %q, got %q", "upstream", result.Remote)
		}
		if result.Branch != "develop" {
			t.Errorf("expected branch %q, got %q", "develop", result.Branch)
		}
	})
}

func TestMockOverride(t *testing.T) {
	m := &git.MockGitService{
		CurrentBranchFn: func(ctx context.Context, dir string) (git.BranchResult, error) {
			return git.BranchResult{Name: "feature/xyz"}, nil
		},
	}
	ctx := context.Background()

	br, err := m.CurrentBranch(ctx, "/tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if br.Name != "feature/xyz" {
		t.Errorf("expected branch name %q, got %q", "feature/xyz", br.Name)
	}
}
