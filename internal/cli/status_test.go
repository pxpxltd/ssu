package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fatih/color"

	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/github"
)

// disableColor turns off fatih/color for the duration of a test so output
// can be compared as plain text.
func disableColor(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = prev })
}

// prScanResult builds a scan result after a --pr lookup: a failed module
// sorted first, root with one PR, a module with two (one draft), one with
// none, and one missing.
func prScanResult() *engine.ScanResult {
	current := []git.SubmoduleStatus{git.StatusCurrent}
	return &engine.ScanResult{
		Root: &engine.SubmoduleInfo{
			Path: ".", IsRoot: true, CurrentBranch: "develop", Statuses: current,
			PRChecked: true,
			PullRequests: []github.PullRequest{
				{Number: 41, Title: "Bump deps", URL: "https://github.com/pxpxltd/app/pull/41", BaseBranch: "develop", HeadBranch: "chore/deps", Author: "alice"},
			},
		},
		Submodules: []*engine.SubmoduleInfo{
			{
				Path: "lib/legacy", CurrentBranch: "main", Statuses: current,
				PRChecked: true, PRError: errors.New("not a GitHub remote (gitlab.com)\nsecond line"),
			},
			{
				Path: "plugins/auth", CurrentBranch: "feat/sso", Statuses: []git.SubmoduleStatus{git.StatusAhead},
				PRChecked: true,
				PullRequests: []github.PullRequest{
					{Number: 12, Title: "SSO login", URL: "https://github.com/pxpxltd/auth/pull/12", BaseBranch: "develop", HeadBranch: "feat/sso", Author: "bob"},
					{Number: 9, Title: "Fix token refresh", URL: "https://github.com/pxpxltd/auth/pull/9", BaseBranch: "develop", HeadBranch: "fix/refresh", Author: "carol", Draft: true},
				},
			},
			{Path: "plugins/blog", CurrentBranch: "develop", Statuses: current, PRChecked: true},
			{Path: "vendor/missing", Statuses: []git.SubmoduleStatus{git.StatusMissing}},
		},
	}
}

// tableRow returns the rendered table line that starts with the given path cell.
func tableRow(t *testing.T, out, path string) []string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		// "│a│b│" splits into ["", "a", "b", ""]; keep the inner cells.
		cells := strings.Split(line, "│")
		if len(cells) < 3 {
			continue
		}
		cells = cells[1 : len(cells)-1]
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if cells[0] == path {
			return cells
		}
	}
	t.Fatalf("row %q not found in:\n%s", path, out)
	return nil
}

func TestPrintStatusTable_WithoutPRs(t *testing.T) {
	var buf bytes.Buffer
	if err := printStatusTable(&buf, prScanResult(), false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "PRs") {
		t.Errorf("PRs column should be absent without --pr:\n%s", out)
	}
	if got := tableRow(t, out, "plugins/auth"); len(got) != 5 || got[4] != "ahead" {
		t.Errorf("unexpected row: %q", got)
	}
}

func TestPrintStatusTable_WithPRs(t *testing.T) {
	var buf bytes.Buffer
	if err := printStatusTable(&buf, prScanResult(), true); err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	header := tableRow(t, out, "Path")
	if len(header) != 6 || header[4] != "PRs" || header[5] != "Status" {
		t.Errorf("unexpected header: %q", header)
	}

	tests := []struct {
		path, prs, status string
	}{
		{"(root)", "1", "current"},
		{"plugins/auth", "2", "ahead"},
		{"plugins/blog", "-", "current"},
		{"lib/legacy", "?", "current"},
		{"vendor/missing", "", "missing"},
	}
	for _, tt := range tests {
		row := tableRow(t, out, tt.path)
		if row[4] != tt.prs || row[5] != tt.status {
			t.Errorf("%s: PRs=%q status=%q, want %q %q", tt.path, row[4], row[5], tt.prs, tt.status)
		}
	}
}

func TestPrintPRList(t *testing.T) {
	disableColor(t)
	var buf bytes.Buffer
	printPRList(&buf, prScanResult())
	out := buf.String()

	want := `
Open pull requests

(root)
  #41  Bump deps  develop <- chore/deps  @alice
       https://github.com/pxpxltd/app/pull/41
plugins/auth
  #12  SSO login                  develop <- feat/sso     @bob
       https://github.com/pxpxltd/auth/pull/12
  #9   Fix token refresh (draft)  develop <- fix/refresh  @carol
       https://github.com/pxpxltd/auth/pull/9

Skipped

lib/legacy
  ! not a GitHub remote (gitlab.com)
`
	if out != want {
		t.Errorf("unexpected PR list.\ngot:\n%s\nwant:\n%s", out, want)
	}
}

func TestPrintPRList_None(t *testing.T) {
	disableColor(t)
	result := &engine.ScanResult{
		Root:       &engine.SubmoduleInfo{Path: ".", IsRoot: true, PRChecked: true},
		Submodules: []*engine.SubmoduleInfo{{Path: "a", PRChecked: true}, {Path: "b"}},
	}
	var buf bytes.Buffer
	printPRList(&buf, result)
	if !strings.Contains(buf.String(), "No open pull requests.") {
		t.Errorf("expected empty message, got:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "Skipped") {
		t.Errorf("Skipped section should be absent without failures:\n%s", buf.String())
	}
}

func TestPrintPRList_NoneButFailed(t *testing.T) {
	disableColor(t)
	result := &engine.ScanResult{
		Submodules: []*engine.SubmoduleInfo{
			{Path: "a", PRChecked: true, PRError: errors.New("boom")},
			{Path: "b", PRChecked: true},
		},
	}
	var buf bytes.Buffer
	printPRList(&buf, result)

	want := `
Open pull requests

No open pull requests.

Skipped

a
  ! boom
`
	if buf.String() != want {
		t.Errorf("unexpected PR list.\ngot:\n%s\nwant:\n%s", buf.String(), want)
	}
}

func TestPrintPRList_TruncatesLongTitles(t *testing.T) {
	disableColor(t)
	long := strings.Repeat("x", 80)
	result := &engine.ScanResult{
		Submodules: []*engine.SubmoduleInfo{{
			Path: "a", PRChecked: true,
			PullRequests: []github.PullRequest{{Number: 1, Title: long, URL: "u", BaseBranch: "main", HeadBranch: "f"}},
		}},
	}
	var buf bytes.Buffer
	printPRList(&buf, result)
	if strings.Contains(buf.String(), long) {
		t.Error("long title should be truncated")
	}
	if !strings.Contains(buf.String(), strings.Repeat("x", prTitleMaxRunes-1)+"…") {
		t.Errorf("expected truncated title with ellipsis, got:\n%s", buf.String())
	}
}

func TestPrintStatusJSON_PullRequests(t *testing.T) {
	var buf bytes.Buffer
	if err := printStatusJSON(&buf, prScanResult()); err != nil {
		t.Fatal(err)
	}

	var out struct {
		Root       map[string]json.RawMessage   `json:"root"`
		Submodules []map[string]json.RawMessage `json:"submodules"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}

	var rootPRs []pullRequestJSON
	if err := json.Unmarshal(out.Root["pull_requests"], &rootPRs); err != nil {
		t.Fatalf("root pull_requests: %v", err)
	}
	want := pullRequestJSON{Number: 41, Title: "Bump deps", URL: "https://github.com/pxpxltd/app/pull/41", HeadBranch: "chore/deps", BaseBranch: "develop", Author: "alice"}
	if len(rootPRs) != 1 || rootPRs[0] != want {
		t.Errorf("root pull_requests = %+v", rootPRs)
	}

	byPath := map[string]map[string]json.RawMessage{}
	for _, sm := range out.Submodules {
		var p string
		_ = json.Unmarshal(sm["path"], &p)
		byPath[p] = sm
	}

	if got := string(byPath["plugins/blog"]["pull_requests"]); got != "[]" {
		t.Errorf("checked module with no PRs should emit [], got %q", got)
	}
	if got := string(byPath["lib/legacy"]["pr_error"]); got != `"not a GitHub remote (gitlab.com)"` {
		t.Errorf("unexpected pr_error: %s", got)
	}
	if _, ok := byPath["lib/legacy"]["pull_requests"]; ok {
		t.Error("failed lookup should not emit pull_requests")
	}
	if _, ok := byPath["vendor/missing"]["pull_requests"]; ok {
		t.Error("unchecked module should not emit pull_requests")
	}
}

func TestPrintStatusJSON_WithoutPRLookup(t *testing.T) {
	result := &engine.ScanResult{
		Root:       &engine.SubmoduleInfo{Path: ".", IsRoot: true},
		Submodules: []*engine.SubmoduleInfo{{Path: "a"}},
	}
	var buf bytes.Buffer
	if err := printStatusJSON(&buf, result); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "pull_requests") || strings.Contains(buf.String(), "pr_error") {
		t.Errorf("PR fields should be absent without --pr:\n%s", buf.String())
	}
}
