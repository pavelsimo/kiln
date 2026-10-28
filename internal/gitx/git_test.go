package gitx

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	for url, want := range map[string]string{
		"git@github.com:acme/app.git":       "acme/app",
		"https://github.com/acme/app":       "acme/app",
		"https://github.com/acme/app.git/":  "acme/app",
		"ssh://git@github.com/acme/my.repo": "acme/my.repo",
		"git@gitlab.com:acme/app.git":       "",
	} {
		o, n, ok := ParseGitHubRemote(url)
		got := ""
		if ok {
			got = o + "/" + n
		}
		if got != want {
			t.Errorf("ParseGitHubRemote(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestBranches(t *testing.T) {
	if n, ok := BranchIssue("kiln/42-dark-mode"); !ok || n != 42 {
		t.Errorf("got %d %v", n, ok)
	}
	for _, b := range []string{"main", "kiln/release-v1.4.0", "kiln/retro-2026-10-28", "feature/42-x"} {
		if _, ok := BranchIssue(b); ok {
			t.Errorf("%s should not be a task branch", b)
		}
	}
	if !IsHousekeepingBranch("kiln/release-v1.4.0") || !IsHousekeepingBranch("kiln/onboard") || IsHousekeepingBranch("kiln/42-x") || IsHousekeepingBranch("main") {
		t.Error("housekeeping detection is off")
	}
}
