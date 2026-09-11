package github

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const prFixture = `[
  {
    "author": {"id": "U_1", "is_bot": false, "login": "alice", "name": "Alice"},
    "baseRefName": "develop",
    "headRefName": "feat/sso",
    "isDraft": false,
    "number": 12,
    "title": "SSO login",
    "url": "https://github.com/pxpxltd/auth/pull/12"
  },
  {
    "author": {"id": "U_2", "is_bot": false, "login": "bob", "name": ""},
    "baseRefName": "develop",
    "headRefName": "fix/refresh",
    "isDraft": true,
    "number": 9,
    "title": "Fix token refresh",
    "url": "https://github.com/pxpxltd/auth/pull/9"
  }
]`

func TestParsePRList(t *testing.T) {
	prs, err := parsePRList([]byte(prFixture))
	if err != nil {
		t.Fatal(err)
	}
	want := []PullRequest{
		{Number: 12, Title: "SSO login", URL: "https://github.com/pxpxltd/auth/pull/12", HeadBranch: "feat/sso", BaseBranch: "develop", Author: "alice"},
		{Number: 9, Title: "Fix token refresh", URL: "https://github.com/pxpxltd/auth/pull/9", HeadBranch: "fix/refresh", BaseBranch: "develop", Author: "bob", Draft: true},
	}
	if len(prs) != len(want) {
		t.Fatalf("expected %d PRs, got %d", len(want), len(prs))
	}
	for i := range want {
		if prs[i] != want[i] {
			t.Errorf("pr[%d] = %+v, want %+v", i, prs[i], want[i])
		}
	}
}

func TestParsePRList_Empty(t *testing.T) {
	prs, err := parsePRList([]byte("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if prs == nil || len(prs) != 0 {
		t.Errorf("expected empty non-nil slice, got %#v", prs)
	}
}

func TestParsePRList_Invalid(t *testing.T) {
	if _, err := parsePRList([]byte("not json")); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// writeScript creates an executable shell script in a temp dir.
func writeScript(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script fakes need a POSIX shell")
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// fakeGH returns a gh stand-in that records its arguments (one per line)
// into the returned file and prints the fixture.
func fakeGH(t *testing.T) (bin, argsFile string) {
	t.Helper()
	argsFile = filepath.Join(t.TempDir(), "args")
	fixture := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(fixture, []byte(prFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	bin = writeScript(t, "gh", `printf '%s\n' "$@" > '`+argsFile+`'
cat '`+fixture+`'
`)
	return bin, argsFile
}

func readArgs(t *testing.T, argsFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func argAfter(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestExecGHOpenPRs(t *testing.T) {
	bin, argsFile := fakeGH(t)
	g := &ExecGH{GHBin: bin}

	prs, err := g.OpenPRs(context.Background(), "https://github.com/pxpxltd/auth.git")
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 2 || prs[0].Number != 12 || !prs[1].Draft {
		t.Errorf("unexpected PRs: %+v", prs)
	}

	args := readArgs(t, argsFile)
	if args[0] != "pr" || args[1] != "list" {
		t.Errorf("expected 'pr list', got %v", args[:2])
	}
	if got := argAfter(args, "-R"); got != "pxpxltd/auth" {
		t.Errorf("-R = %q, want pxpxltd/auth", got)
	}
	if got := argAfter(args, "--state"); got != "open" {
		t.Errorf("--state = %q, want open", got)
	}
	if got := argAfter(args, "--json"); got != prJSONFields {
		t.Errorf("--json = %q, want %q", got, prJSONFields)
	}
}

func TestExecGHOpenPRs_EnterpriseHost(t *testing.T) {
	bin, argsFile := fakeGH(t)
	g := &ExecGH{GHBin: bin}

	if _, err := g.OpenPRs(context.Background(), "https://ghe.example.com/team/app.git"); err != nil {
		t.Fatal(err)
	}
	if got := argAfter(readArgs(t, argsFile), "-R"); got != "ghe.example.com/team/app" {
		t.Errorf("-R = %q, want ghe.example.com/team/app", got)
	}
}

func TestExecGHOpenPRs_SSHAlias(t *testing.T) {
	bin, argsFile := fakeGH(t)
	ssh := writeScript(t, "ssh", `echo "user git"
echo "hostname github.com"
echo "port 22"
`)
	g := &ExecGH{GHBin: bin, SSHBin: ssh}

	if _, err := g.OpenPRs(context.Background(), "git@github-work:pxpxltd/auth.git"); err != nil {
		t.Fatal(err)
	}
	if got := argAfter(readArgs(t, argsFile), "-R"); got != "pxpxltd/auth" {
		t.Errorf("-R = %q, want pxpxltd/auth", got)
	}
}

func TestExecGHOpenPRs_SSHAliasToNonGitHub(t *testing.T) {
	bin, _ := fakeGH(t)
	ssh := writeScript(t, "ssh", `echo "hostname gitlab.com"
`)
	g := &ExecGH{GHBin: bin, SSHBin: ssh}

	_, err := g.OpenPRs(context.Background(), "git@work-gitlab:team/app.git")
	if err == nil || err.Error() != "not a GitHub remote (gitlab.com)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecGHOpenPRs_SSHUnavailableKeepsHost(t *testing.T) {
	bin, argsFile := fakeGH(t)
	g := &ExecGH{GHBin: bin, SSHBin: filepath.Join(t.TempDir(), "no-ssh")}

	if _, err := g.OpenPRs(context.Background(), "git@ghe.example.com:team/app.git"); err != nil {
		t.Fatal(err)
	}
	if got := argAfter(readArgs(t, argsFile), "-R"); got != "ghe.example.com/team/app" {
		t.Errorf("-R = %q, want ghe.example.com/team/app", got)
	}
}

func TestExecGHOpenPRs_GHFailure(t *testing.T) {
	bin := writeScript(t, "gh", `echo "" >&2
echo "To get started with GitHub CLI, please run:  gh auth login" >&2
echo "second line" >&2
exit 4
`)
	g := &ExecGH{GHBin: bin}

	_, err := g.OpenPRs(context.Background(), "git@github.com:pxpxltd/auth.git")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "To get started with GitHub CLI, please run:  gh auth login" {
		t.Errorf("expected first stderr line, got %q", err.Error())
	}
}

func TestExecGHOpenPRs_BadRemote(t *testing.T) {
	g := &ExecGH{GHBin: filepath.Join(t.TempDir(), "never-called")}
	if _, err := g.OpenPRs(context.Background(), "git@gitlab.com:team/app.git"); err == nil {
		t.Error("expected error for non-GitHub remote")
	}
}

func TestExecGHAvailable(t *testing.T) {
	g := &ExecGH{GHBin: filepath.Join(t.TempDir(), "no-gh")}
	err := g.Available()
	if err == nil {
		t.Fatal("expected error for missing gh")
	}
	if !strings.Contains(err.Error(), "gh auth login") {
		t.Errorf("error should explain how to set up gh: %v", err)
	}

	bin, _ := fakeGH(t)
	if err := (&ExecGH{GHBin: bin}).Available(); err != nil {
		t.Errorf("expected fake gh to be available: %v", err)
	}
}
