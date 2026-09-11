package github

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// prListLimit caps how many open PRs are requested per repository.
const prListLimit = 100

// prJSONFields are the gh --json fields decoded by parsePRList.
const prJSONFields = "number,title,url,headRefName,baseRefName,isDraft,author"

// PullRequest is an open pull request on a module's repository.
type PullRequest struct {
	Number     int
	Title      string
	URL        string
	HeadBranch string
	BaseBranch string
	Author     string
	Draft      bool
}

// ExecGH lists pull requests by shelling out to the gh binary.
type ExecGH struct {
	GHBin   string        // Path to gh binary; defaults to "gh" if empty.
	SSHBin  string        // Path to ssh binary used to resolve host aliases; defaults to "ssh".
	Timeout time.Duration // Per-call timeout; 0 means no timeout.
}

// NewExecGH returns an ExecGH with the system gh and a 30s timeout.
func NewExecGH() *ExecGH {
	return &ExecGH{Timeout: 30 * time.Second}
}

func (g *ExecGH) ghBin() string {
	if g.GHBin == "" {
		return "gh"
	}
	return g.GHBin
}

func (g *ExecGH) sshBin() string {
	if g.SSHBin == "" {
		return "ssh"
	}
	return g.SSHBin
}

// Available reports whether the gh binary can be found.
func (g *ExecGH) Available() error {
	if _, err := exec.LookPath(g.ghBin()); err != nil {
		return errors.New("--pr requires the GitHub CLI (gh): install from https://cli.github.com and run 'gh auth login'")
	}
	return nil
}

// OpenPRs returns the open pull requests of the repository behind remoteURL.
func (g *ExecGH) OpenPRs(ctx context.Context, remoteURL string) ([]PullRequest, error) {
	remote, err := ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}
	repo := remote.Repo
	if remote.SSH && repo.Host != DefaultHost {
		repo.Host = g.resolveSSHHost(ctx, repo.Host)
		if nonGitHubHosts[repo.Host] {
			return nil, fmt.Errorf("not a GitHub remote (%s)", repo.Host)
		}
	}

	if g.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, g.ghBin(), "pr", "list",
		"-R", repo.Slug(),
		"--state", "open",
		"--limit", fmt.Sprint(prListLimit),
		"--json", prJSONFields)
	cmd.WaitDelay = 5 * time.Second
	cmd.Env = append(os.Environ(), "GH_PROMPT_DISABLED=1", "NO_COLOR=1")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("gh pr list for %s timed out", repo.Slug())
		}
		if msg := firstLine(errBuf.String()); msg != "" {
			return nil, errors.New(msg)
		}
		return nil, fmt.Errorf("gh pr list for %s: %w", repo.Slug(), err)
	}

	return parsePRList(outBuf.Bytes())
}

// resolveSSHHost maps an ssh alias (e.g. "github-work" from ~/.ssh/config)
// to its real hostname using `ssh -G`, which reads config without connecting.
// The original host is returned if ssh is unavailable or has no mapping.
func (g *ExecGH) resolveSSHHost(ctx context.Context, host string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, g.sshBin(), "-G", host).Output()
	if err != nil {
		return host
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "hostname" {
			return strings.ToLower(fields[1])
		}
	}
	return host
}

// ghPR mirrors the subset of `gh pr list --json` output that SSU uses.
type ghPR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	IsDraft     bool   `json:"isDraft"`
	Author      struct {
		Login string `json:"login"`
	} `json:"author"`
}

// parsePRList decodes `gh pr list --json` output.
func parsePRList(data []byte) ([]PullRequest, error) {
	var raw []ghPR
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing gh output: %w", err)
	}
	prs := make([]PullRequest, 0, len(raw))
	for _, p := range raw {
		prs = append(prs, PullRequest{
			Number:     p.Number,
			Title:      p.Title,
			URL:        p.URL,
			HeadBranch: p.HeadRefName,
			BaseBranch: p.BaseRefName,
			Author:     p.Author.Login,
			Draft:      p.IsDraft,
		})
	}
	return prs, nil
}

// firstLine returns the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
