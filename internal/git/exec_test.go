package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/pxpxltd/ssu/internal/git"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// setupTestRepo creates a bare-minimum git repo in a temp directory with one
// commit. It returns the directory path. The repo is automatically cleaned up
// when the test finishes.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		run := exec.Command(c[0], c[1:]...)
		run.Dir = dir
		if out, err := run.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", c, err, out)
		}
	}

	// Create initial commit.
	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("# test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "initial"},
	} {
		run := exec.Command(c[0], c[1:]...)
		run.Dir = dir
		if out, err := run.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %v\n%s", c, err, out)
		}
	}
	return dir
}

// setupClonedRepo creates a "remote" repo and clones it into a "local" repo.
// The remote is configured to accept pushes to its current branch.
// Returns (localDir, remoteDir). Both are cleaned up when the test finishes.
func setupClonedRepo(t *testing.T) (string, string) {
	t.Helper()
	remoteDir := setupTestRepo(t)

	// Allow pushing to the checked-out branch on the remote (needed for tests).
	cfg := exec.Command("git", "config", "receive.denyCurrentBranch", "ignore")
	cfg.Dir = remoteDir
	if out, err := cfg.CombinedOutput(); err != nil {
		t.Fatalf("config remote: %v\n%s", err, out)
	}

	localDir := t.TempDir()

	clone := exec.Command("git", "clone", remoteDir, localDir)
	if out, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("clone: %v\n%s", err, out)
	}

	// Set user config in local.
	for _, c := range [][]string{
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	} {
		run := exec.Command(c[0], c[1:]...)
		run.Dir = localDir
		if out, err := run.CombinedOutput(); err != nil {
			t.Fatalf("setup local %v: %v\n%s", c, err, out)
		}
	}
	return localDir, remoteDir
}

// addCommit creates an empty commit in dir with the given message.
func addCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "--allow-empty", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface check (belt-and-suspenders with exec.go)
// ---------------------------------------------------------------------------

var _ git.GitService = (*git.ExecGit)(nil)

// ---------------------------------------------------------------------------
// Integration tests
// ---------------------------------------------------------------------------

func TestExecGitCurrentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	br, err := g.CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if br.Detached {
		t.Error("expected attached HEAD")
	}
	if br.Name == "" {
		t.Error("expected non-empty branch name")
	}
}

func TestExecGitCurrentBranchDetached(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)

	cmd := exec.Command("git", "checkout", "--detach", "HEAD")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("detach: %v\n%s", err, out)
	}

	g := git.NewExecGit()
	br, err := g.CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !br.Detached {
		t.Error("expected detached HEAD")
	}
}

func TestExecGitCurrentSHA(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	sha, err := g.CurrentSHA(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{40}$`, sha); !matched {
		t.Errorf("expected 40-char hex SHA, got %q", sha)
	}
}

func TestExecGitRefExistsAndIsAncestor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()
	ctx := context.Background()

	initial, err := g.CurrentSHA(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	exists, err := g.RefExists(ctx, dir, initial)
	if err != nil || !exists {
		t.Fatalf("RefExists(initial) = %v, %v", exists, err)
	}
	exists, err = g.RefExists(ctx, dir, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil || exists {
		t.Fatalf("RefExists(missing) = %v, %v", exists, err)
	}

	addCommit(t, dir, "second")
	head, err := g.CurrentSHA(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	ancestor, err := g.IsAncestor(ctx, dir, initial, head)
	if err != nil || !ancestor {
		t.Fatalf("IsAncestor(initial, head) = %v, %v", ancestor, err)
	}
	ancestor, err = g.IsAncestor(ctx, dir, head, initial)
	if err != nil || ancestor {
		t.Fatalf("IsAncestor(head, initial) = %v, %v", ancestor, err)
	}
}

func TestExecGitCheckoutNewBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()
	result, err := g.CheckoutNewBranch(context.Background(), dir, "feature/imported", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch != "feature/imported" {
		t.Fatalf("branch = %q", result.Branch)
	}
	current, err := g.CurrentBranch(context.Background(), dir)
	if err != nil || current.Name != "feature/imported" {
		t.Fatalf("current branch = %#v, %v", current, err)
	}
}

func TestExecGitHasLocalChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// Clean tree.
	has, err := g.HasLocalChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected no local changes on clean tree")
	}

	// Create untracked file.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	has, err = g.HasLocalChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected local changes after creating new file")
	}
}

func TestExecGitRemoteBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	branches, err := g.RemoteBranches(context.Background(), localDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) == 0 {
		t.Fatal("expected at least one remote branch")
	}

	foundOrigin := false
	for _, b := range branches {
		if b.Remote == "origin" {
			foundOrigin = true
			break
		}
	}
	if !foundOrigin {
		t.Errorf("expected a branch with Remote=origin, got %+v", branches)
	}
}

func TestExecGitCommitsBehind(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, remoteDir := setupClonedRepo(t)

	// Add 3 commits to remote.
	for i := 0; i < 3; i++ {
		addCommit(t, remoteDir, "remote commit")
	}

	g := git.NewExecGit()

	// Fetch in local.
	_, err := g.Fetch(context.Background(), localDir, git.FetchOpts{Remote: "origin"})
	if err != nil {
		t.Fatal(err)
	}

	// Get current branch name for the local ref.
	br, err := g.CurrentBranch(context.Background(), localDir)
	if err != nil {
		t.Fatal(err)
	}

	behind, err := g.CommitsBehind(context.Background(), localDir, "HEAD", "origin/"+br.Name)
	if err != nil {
		t.Fatal(err)
	}
	if behind != 3 {
		t.Errorf("expected 3 commits behind, got %d", behind)
	}
}

func TestExecGitFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, remoteDir := setupClonedRepo(t)

	// Add commit to remote.
	addCommit(t, remoteDir, "new remote commit")

	g := git.NewExecGit()
	_, err := g.Fetch(context.Background(), localDir, git.FetchOpts{Remote: "origin"})
	if err != nil {
		t.Fatal(err)
	}

	// After fetch, should see commits behind.
	br, _ := g.CurrentBranch(context.Background(), localDir)
	behind, _ := g.CommitsBehind(context.Background(), localDir, "HEAD", "origin/"+br.Name)
	if behind < 1 {
		t.Errorf("expected at least 1 commit behind after fetch, got %d", behind)
	}
}

func TestExecGitCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)

	// Create a branch.
	cmd := exec.Command("git", "branch", "test-branch")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("branch: %v\n%s", err, out)
	}

	g := git.NewExecGit()
	result, err := g.Checkout(context.Background(), dir, "test-branch")
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch != "test-branch" {
		t.Errorf("expected Branch=test-branch, got %q", result.Branch)
	}

	// Verify we're on test-branch.
	br, err := g.CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if br.Name != "test-branch" {
		t.Errorf("expected current branch test-branch, got %q", br.Name)
	}
}

func TestExecGitStashAndPop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// Modify a tracked file.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Stash.
	stashResult, err := g.Stash(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !stashResult.Created {
		t.Error("expected stash to be created")
	}

	// Working tree should be clean now.
	has, err := g.HasLocalChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected clean tree after stash")
	}

	// Pop.
	popResult, err := g.StashPop(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !popResult.Created {
		t.Error("expected stash pop to report applied")
	}

	// Changes should be back.
	has, err = g.HasLocalChanges(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected local changes after stash pop")
	}
}

func TestExecGitStashNoChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	result, err := g.Stash(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if result.Created {
		t.Error("expected Created=false for clean working tree")
	}
}

func TestExecGitIsDetachedHead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// Not detached initially.
	detached, err := g.IsDetachedHead(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if detached {
		t.Error("expected not detached")
	}

	// Detach.
	cmd := exec.Command("git", "checkout", "--detach", "HEAD")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("detach: %v\n%s", err, out)
	}

	detached, err = g.IsDetachedHead(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !detached {
		t.Error("expected detached after checkout --detach")
	}
}

func TestExecGitHasRemoteBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	// Get the default branch name.
	br, _ := g.CurrentBranch(context.Background(), localDir)

	has, err := g.HasRemoteBranch(context.Background(), localDir, "origin", br.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Errorf("expected remote branch origin/%s to exist", br.Name)
	}

	// Check a nonexistent branch.
	has, err = g.HasRemoteBranch(context.Background(), localDir, "origin", "nonexistent-branch-xyz")
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected nonexistent branch to not be found")
	}
}

func TestExecGitTrackingBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	info, err := g.TrackingBranch(context.Background(), localDir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Set {
		t.Error("expected tracking branch to be set in cloned repo")
	}
	if info.Remote != "origin" {
		t.Errorf("expected remote=origin, got %q", info.Remote)
	}
	if info.Branch == "" {
		t.Error("expected non-empty tracking branch name")
	}
}

func TestExecGitTrackingBranchNone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// A standalone repo has no tracking branch.
	info, err := g.TrackingBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Set {
		t.Error("expected no tracking branch in standalone repo")
	}
}

func TestExecGitSubmodulePaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// No submodules -- should return nil, nil.
	paths, err := g.SubmodulePaths(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if paths != nil {
		t.Errorf("expected nil paths for repo without submodules, got %v", paths)
	}
}

func TestExecGitIsSubmoduleInitialized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)
	g := git.NewExecGit()

	// The repo root has .git, so (rootDir=parent, subPath=basename) should find it.
	parent := filepath.Dir(dir)
	base := filepath.Base(dir)
	if !g.IsSubmoduleInitialized(parent, base) {
		t.Error("expected initialized=true for dir with .git")
	}

	// Non-existent path.
	if g.IsSubmoduleInitialized(dir, "nonexistent") {
		t.Error("expected initialized=false for missing path")
	}
}

func TestExecGitTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	dir := setupTestRepo(t)

	g := &git.ExecGit{
		Timeouts: git.TimeoutConfig{
			Default: 1 * time.Nanosecond,
		},
	}

	// With a 1ns timeout, any real git command should fail with deadline exceeded.
	_, err := g.CurrentSHA(context.Background(), dir)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !git.IsTimeout(err) {
		t.Errorf("expected timeout error, got: %v", err)
	}
}

func TestExecGitMerge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, remoteDir := setupClonedRepo(t)

	// Add a commit to remote.
	addCommit(t, remoteDir, "mergeable commit")

	g := git.NewExecGit()

	// Fetch first.
	_, err := g.Fetch(context.Background(), localDir, git.FetchOpts{Remote: "origin"})
	if err != nil {
		t.Fatal(err)
	}

	// Get branch name.
	br, _ := g.CurrentBranch(context.Background(), localDir)

	// Merge.
	result, err := g.Merge(context.Background(), localDir, "origin/"+br.Name)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Error("expected successful merge")
	}
	if result.Conflict {
		t.Error("expected no conflict")
	}
}

func TestExecGitIncomingChangelog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, remoteDir := setupClonedRepo(t)

	// Add commits to remote.
	addCommit(t, remoteDir, "log entry 1")
	addCommit(t, remoteDir, "log entry 2")

	g := git.NewExecGit()
	_, err := g.Fetch(context.Background(), localDir, git.FetchOpts{Remote: "origin"})
	if err != nil {
		t.Fatal(err)
	}

	br, _ := g.CurrentBranch(context.Background(), localDir)
	lines, err := g.IncomingChangelog(context.Background(), localDir, "origin/"+br.Name, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Errorf("expected 2 changelog lines, got %d: %v", len(lines), lines)
	}
}

func TestExecGitCommitsAhead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	// Add local commits.
	addCommit(t, localDir, "ahead 1")
	addCommit(t, localDir, "ahead 2")

	br, _ := g.CurrentBranch(context.Background(), localDir)
	ahead, err := g.CommitsAhead(context.Background(), localDir, "origin/"+br.Name)
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 2 {
		t.Errorf("expected 2 commits ahead, got %d", ahead)
	}
}

func TestExecGitPush(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	// Add a local commit.
	addCommit(t, localDir, "push test")

	br, _ := g.CurrentBranch(context.Background(), localDir)
	result, err := g.Push(context.Background(), localDir, git.PushOpts{
		Remote: "origin",
		Branch: br.Name,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Remote != "origin" {
		t.Errorf("expected Remote=origin, got %q", result.Remote)
	}
	if result.Branch != br.Name {
		t.Errorf("expected Branch=%s, got %q", br.Name, result.Branch)
	}

	// Verify no commits ahead after push.
	ahead, _ := g.CommitsAhead(context.Background(), localDir, "origin/"+br.Name)
	if ahead != 0 {
		t.Errorf("expected 0 commits ahead after push, got %d", ahead)
	}
}

func TestExecGitPushNewTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, _ := setupClonedRepo(t)
	g := git.NewExecGit()

	// Create a new branch with no tracking.
	for _, c := range [][]string{
		{"git", "checkout", "-b", "new-feature"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = localDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", c, err, out)
		}
	}

	addCommit(t, localDir, "feature commit")

	result, err := g.Push(context.Background(), localDir, git.PushOpts{
		Remote:      "origin",
		Branch:      "new-feature",
		SetUpstream: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.NewTracking {
		t.Error("expected NewTracking=true when using SetUpstream")
	}

	// Verify tracking is now set.
	info, _ := g.TrackingBranch(context.Background(), localDir)
	if !info.Set {
		t.Error("expected tracking branch to be set after push -u")
	}
}

func TestExecGitMergeAbort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	localDir, remoteDir := setupClonedRepo(t)
	g := git.NewExecGit()

	// Create conflicting changes.
	if err := os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("remote change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "remote conflict"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = remoteDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", c, err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(localDir, "README.md"), []byte("local change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "local conflict"},
	} {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = localDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", c, err, out)
		}
	}

	// Fetch and attempt merge (should conflict).
	_, _ = g.Fetch(context.Background(), localDir, git.FetchOpts{Remote: "origin"})
	br, _ := g.CurrentBranch(context.Background(), localDir)
	mergeResult, mergeErr := g.Merge(context.Background(), localDir, "origin/"+br.Name)
	if mergeErr == nil {
		t.Fatal("expected merge error due to conflict")
	}
	if !mergeResult.Conflict {
		t.Error("expected Conflict=true")
	}

	// Abort the merge.
	if err := g.MergeAbort(context.Background(), localDir); err != nil {
		t.Fatalf("merge abort: %v", err)
	}

	// After abort, should not have local changes (the conflicted merge is gone).
	has, _ := g.HasLocalChanges(context.Background(), localDir)
	if has {
		t.Error("expected clean tree after merge abort")
	}
}
