package git_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pxpxltd/ssu/internal/git"
)

func TestResolveBranchForCheckout(t *testing.T) {
	ctx := context.Background()
	dir := "/tmp/repo"

	tests := []struct {
		name      string
		mock      *git.MockGitService
		opts      git.BranchCheckoutOpts
		wantBr    string
		wantLocal bool
		wantErr   bool
	}{
		{
			name: "local feature branch preferred over priority",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Local:  []string{"feature/xyz", "develop"},
						Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "feature/xyz",
			wantLocal: true,
		},
		{
			name: "local priority branch when no feature branch",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Local:  []string{"develop", "master"},
						Remote: []git.RemoteBranch{{Remote: "origin", Branch: "develop"}},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "develop",
			wantLocal: true,
		},
		{
			name: "local master when no develop",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Local: []string{"master"},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "master",
			wantLocal: true,
		},
		{
			name: "remote feature branch when no local branches",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Remote: []git.RemoteBranch{
							{Remote: "origin", Branch: "feature/cool"},
							{Remote: "origin", Branch: "develop"},
						},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "feature/cool",
			wantLocal: false,
		},
		{
			name: "remote priority branch when no feature branches",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Remote: []git.RemoteBranch{
							{Remote: "origin", Branch: "develop"},
							{Remote: "origin", Branch: "master"},
						},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "develop",
			wantLocal: false,
		},
		{
			name: "no matching branch returns empty",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "",
			wantLocal: false,
		},
		{
			name: "only non-default remote branches ignored",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Remote: []git.RemoteBranch{
							{Remote: "upstream", Branch: "develop"},
						},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{DefaultRemote: "origin"},
			wantBr:    "",
			wantLocal: false,
		},
		{
			name: "custom priority branches",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Local: []string{"staging", "production"},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{PriorityBranches: []string{"staging", "production"}},
			wantBr:    "staging",
			wantLocal: true,
		},
		{
			name: "CurrentSHA error propagated",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "", fmt.Errorf("not a git repo")
				},
			},
			opts:    git.BranchCheckoutOpts{},
			wantErr: true,
		},
		{
			name: "BranchesPointingAt error propagated",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{}, fmt.Errorf("git error")
				},
			},
			opts:    git.BranchCheckoutOpts{},
			wantErr: true,
		},
		{
			name: "priority order: develop before master in local",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "abc1234", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, _ string) (git.BranchPointsAtResult, error) {
					return git.BranchPointsAtResult{
						Local: []string{"master", "develop"},
					}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{},
			wantBr:    "develop",
			wantLocal: true,
		},
		{
			name: "TargetSHA used instead of CurrentSHA",
			mock: &git.MockGitService{
				CurrentSHAFn: func(_ context.Context, _ string) (string, error) {
					return "current_sha", nil
				},
				BranchesPointingAtFn: func(_ context.Context, _, sha string) (git.BranchPointsAtResult, error) {
					if sha == "target_sha" {
						return git.BranchPointsAtResult{
							Local: []string{"feature/target"},
						}, nil
					}
					return git.BranchPointsAtResult{}, nil
				},
			},
			opts:      git.BranchCheckoutOpts{TargetSHA: "target_sha"},
			wantBr:    "feature/target",
			wantLocal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, isLocal, err := git.ResolveBranchForCheckout(ctx, tt.mock, dir, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != tt.wantBr {
				t.Errorf("branch = %q, want %q", branch, tt.wantBr)
			}
			if isLocal != tt.wantLocal {
				t.Errorf("isLocal = %v, want %v", isLocal, tt.wantLocal)
			}
		})
	}
}
