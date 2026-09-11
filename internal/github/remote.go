// Package github lists pull requests for submodule repositories through the
// GitHub CLI (gh), reusing whatever login the user already has.
package github

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultHost is the public GitHub host. Repos on any other host are passed
// to gh with an explicit host prefix so GitHub Enterprise logins work.
const DefaultHost = "github.com"

// nonGitHubHosts are well-known forges gh cannot talk to. Rejecting them up
// front gives a clearer message than gh's authentication error.
var nonGitHubHosts = map[string]bool{
	"gitlab.com":    true,
	"bitbucket.org": true,
}

// Repo identifies a repository on a GitHub host.
type Repo struct {
	Host  string // e.g. "github.com"
	Owner string // e.g. "pxpxltd"
	Name  string // e.g. "ssu"
}

// Slug returns the repo in the form gh accepts for -R: "owner/name" on
// github.com, "host/owner/name" everywhere else.
func (r Repo) Slug() string {
	if r.Host == "" || r.Host == DefaultHost {
		return r.Owner + "/" + r.Name
	}
	return r.Host + "/" + r.Owner + "/" + r.Name
}

// RemoteURL describes a parsed git remote URL.
type RemoteURL struct {
	Repo
	SSH bool // True for scp-style and ssh:// URLs (host may be an ssh alias)
}

// ParseRemoteURL extracts host, owner and repo name from a git remote URL.
// Supported forms:
//
//	git@github.com:owner/repo.git
//	ssh://git@github.com:22/owner/repo.git
//	https://user:token@github.com/owner/repo.git
//	git://github.com/owner/repo
func ParseRemoteURL(raw string) (RemoteURL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RemoteURL{}, fmt.Errorf("empty remote URL")
	}

	var host, path string
	var ssh bool

	if strings.Contains(raw, "://") {
		// The parse error embeds the raw URL, so it is dropped rather than
		// wrapped: see redactURL.
		u, err := url.Parse(raw)
		if err != nil {
			return RemoteURL{}, fmt.Errorf("invalid remote URL %q", redactURL(raw))
		}
		switch u.Scheme {
		case "ssh", "git+ssh", "ssh+git":
			ssh = true
		case "https", "http", "git":
		default:
			return RemoteURL{}, fmt.Errorf("unsupported remote URL %q", redactURL(raw))
		}
		host = u.Hostname()
		path = u.Path
	} else {
		// scp-style: [user@]host:owner/repo. A colon is required, and a slash
		// before it means this is a local path rather than a remote.
		colon := strings.Index(raw, ":")
		if colon < 0 || strings.Contains(raw[:colon], "/") {
			return RemoteURL{}, fmt.Errorf("unsupported remote URL %q", redactURL(raw))
		}
		host = raw[:colon]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		path = raw[colon+1:]
		ssh = true
	}

	host = strings.ToLower(host)
	path = strings.TrimSuffix(strings.Trim(path, "/"), ".git")
	parts := strings.Split(path, "/")
	if host == "" || len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return RemoteURL{}, fmt.Errorf("cannot find owner/repo in remote URL %q", redactURL(raw))
	}
	if nonGitHubHosts[host] {
		return RemoteURL{}, fmt.Errorf("not a GitHub remote (%s)", host)
	}

	return RemoteURL{
		Repo: Repo{Host: host, Owner: parts[0], Name: parts[1]},
		SSH:  ssh,
	}, nil
}

// redactURL strips user info from a remote URL so it is safe to show in
// errors, which end up in terminal and JSON output. HTTPS remotes can carry
// a token there (https://user:token@host/...).
func redactURL(raw string) string {
	prefix, rest := "", raw
	if i := strings.Index(raw, "://"); i >= 0 {
		prefix, rest = raw[:i+3], raw[i+3:]
	}
	end := strings.Index(rest, "/")
	if end < 0 {
		end = len(rest)
	}
	if at := strings.LastIndex(rest[:end], "@"); at >= 0 {
		rest = rest[at+1:]
	}
	return prefix + rest
}
