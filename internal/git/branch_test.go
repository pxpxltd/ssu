package git_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestDetectBestBranch(t *testing.T) {
	ctx := context.Background()
	dir := "/tmp/repo"

	tests := []struct {
		name     string
		mock     *git.MockGitService
		opts     git.BranchDetectOpts
		want     string
		wantErr  bool
	}{
		{
			name: "override takes priority",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "develop"}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{{Remote: "origin", Branch: "develop"}}, nil
				},
			},
			opts: git.BranchDetectOpts{Override: "custom-branch"},
			want: "custom-branch",
		},
		{
			name: "feature branch preserved when remote exists",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "feature/xyz"}, nil
				},
				HasRemoteBranchFn: func(_ context.Context, _, remote, branch string) (bool, error) {
					if remote == "origin" && branch == "feature/xyz" {
						return true, nil
					}
					return false, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "feature/xyz",
		},
		{
			name: "feature branch without remote falls to priority chain",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "feature/xyz"}, nil
				},
				HasRemoteBranchFn: func(_ context.Context, _, _, _ string) (bool, error) {
					return false, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "develop"},
						{Remote: "origin", Branch: "master"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "develop",
		},
		{
			name: "priority chain: develop wins over master",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "develop"}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "develop"},
						{Remote: "origin", Branch: "master"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "develop",
		},
		{
			name: "priority chain: master when no develop",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "master"}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "master"},
						{Remote: "origin", Branch: "main"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "master",
		},
		{
			name: "priority chain: main as last priority",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "main"}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "main"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "main",
		},
		{
			name: "detached HEAD skips feature check, uses priority chain",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Detached: true}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "develop"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "develop",
		},
		{
			name: "remote HEAD fallback when no priority branches match",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Detached: true}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return nil, nil
				},
				SymbolicRefFn: func(_ context.Context, _, ref string) (string, error) {
					if ref == "refs/remotes/origin/HEAD" {
						return "refs/remotes/origin/staging", nil
					}
					return "", errors.New("not found")
				},
			},
			opts: git.BranchDetectOpts{},
			want: "staging",
		},
		{
			name: "absolute fallback to master",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Detached: true}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return nil, nil
				},
				SymbolicRefFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("not set")
				},
			},
			opts: git.BranchDetectOpts{},
			want: "master",
		},
		{
			name: "first remote branch as fallback",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Detached: true}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "release-1.0"},
					}, nil
				},
				SymbolicRefFn: func(_ context.Context, _, _ string) (string, error) {
					return "", errors.New("not set")
				},
			},
			opts: git.BranchDetectOpts{},
			want: "release-1.0",
		},
		{
			name: "custom priority branches",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "staging"}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "staging"},
						{Remote: "origin", Branch: "production"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{PriorityBranches: []string{"staging", "production"}},
			want: "staging",
		},
		{
			name: "feature branch check uses default remote",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "feature/foo"}, nil
				},
				HasRemoteBranchFn: func(_ context.Context, _, remote, branch string) (bool, error) {
					if remote == "upstream" && branch == "feature/foo" {
						return true, nil
					}
					return false, nil
				},
			},
			opts: git.BranchDetectOpts{DefaultRemote: "upstream"},
			want: "feature/foo",
		},
		{
			name: "HasRemoteBranch error is non-fatal",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Name: "feature/xyz"}, nil
				},
				HasRemoteBranchFn: func(_ context.Context, _, _, _ string) (bool, error) {
					return false, errors.New("network error")
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return []git.RemoteBranch{
						{Remote: "origin", Branch: "develop"},
					}, nil
				},
			},
			opts: git.BranchDetectOpts{},
			want: "develop",
		},
		{
			name: "RemoteBranches error is non-fatal",
			mock: &git.MockGitService{
				CurrentBranchFn: func(_ context.Context, _ string) (git.BranchResult, error) {
					return git.BranchResult{Detached: true}, nil
				},
				RemoteBranchesFn: func(_ context.Context, _ string) ([]git.RemoteBranch, error) {
					return nil, errors.New("network error")
				},
				SymbolicRefFn: func(_ context.Context, _, ref string) (string, error) {
					if ref == "refs/remotes/origin/HEAD" {
						return "refs/remotes/origin/develop", nil
					}
					return "", errors.New("not set")
				},
			},
			opts: git.BranchDetectOpts{},
			want: "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := git.DetectBestBranch(ctx, tt.mock, dir, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("DetectBestBranch() = %q, want %q", got, tt.want)
			}
		})
	}
}
