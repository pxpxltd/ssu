package github

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		url   string
		repo  Repo
		ssh   bool
		wantE bool
	}{
		{url: "git@github.com:pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},
		{url: "git@github.com:pxpxltd/ssu", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},
		{url: "github.com:pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},
		{url: "git@github-work:pxpxltd/ssu.git", repo: Repo{"github-work", "pxpxltd", "ssu"}, ssh: true},
		{url: "ssh://git@github.com/pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},
		{url: "ssh://git@github.com:22/pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},
		{url: "https://github.com/pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}},
		{url: "https://github.com/pxpxltd/ssu", repo: Repo{"github.com", "pxpxltd", "ssu"}},
		{url: "https://github.com/pxpxltd/ssu/", repo: Repo{"github.com", "pxpxltd", "ssu"}},
		{url: "https://user:tok@GitHub.com/pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}},
		{url: "https://ghe.example.com/team/app.git", repo: Repo{"ghe.example.com", "team", "app"}},
		{url: "git://github.com/pxpxltd/ssu.git", repo: Repo{"github.com", "pxpxltd", "ssu"}},
		{url: "  git@github.com:pxpxltd/ssu.git\n", repo: Repo{"github.com", "pxpxltd", "ssu"}, ssh: true},

		{url: "", wantE: true},
		{url: "/srv/git/ssu.git", wantE: true},
		{url: "../ssu.git", wantE: true},
		{url: "file:///srv/git/ssu.git", wantE: true},
		{url: "https://github.com/pxpxltd", wantE: true},
		{url: "https://github.com/a/b/c", wantE: true},
		{url: "git@gitlab.com:team/app.git", wantE: true},
		{url: "https://bitbucket.org/team/app.git", wantE: true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := ParseRemoteURL(tt.url)
			if tt.wantE {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Repo != tt.repo {
				t.Errorf("repo = %+v, want %+v", got.Repo, tt.repo)
			}
			if got.SSH != tt.ssh {
				t.Errorf("ssh = %v, want %v", got.SSH, tt.ssh)
			}
		})
	}
}

func TestParseRemoteURL_NonGitHubMessage(t *testing.T) {
	_, err := ParseRemoteURL("git@gitlab.com:team/app.git")
	if err == nil || err.Error() != "not a GitHub remote (gitlab.com)" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRepoSlug(t *testing.T) {
	tests := []struct {
		repo Repo
		want string
	}{
		{Repo{"github.com", "pxpxltd", "ssu"}, "pxpxltd/ssu"},
		{Repo{"", "pxpxltd", "ssu"}, "pxpxltd/ssu"},
		{Repo{"ghe.example.com", "team", "app"}, "ghe.example.com/team/app"},
	}
	for _, tt := range tests {
		if got := tt.repo.Slug(); got != tt.want {
			t.Errorf("Slug(%+v) = %q, want %q", tt.repo, got, tt.want)
		}
	}
}
