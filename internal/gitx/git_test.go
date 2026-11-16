package gitx

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pavelsimo/kiln/internal/gittest"
	"github.com/pavelsimo/kiln/internal/run"
)

func TestMainWorktree(t *testing.T) {
	real := func(p string) string {
		t.Helper()
		r, err := filepath.EvalSymlinks(p)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	check := func(name, from, want string) {
		t.Helper()
		got, err := New(run.Exec{}, from).MainWorktree(context.Background())
		if err != nil || real(got) != real(want) {
			t.Errorf("%s: MainWorktree = %q, %v; want %q", name, got, err, want)
		}
	}
	r := gittest.New(t)
	wt := filepath.Join(t.TempDir(), "wt")
	r.Git("worktree", "add", "-q", wt, "-b", "kiln/9-x")
	check("main checkout", r.Dir, r.Dir)
	check("task worktree", wt, r.Dir)

	// the git dir lives elsewhere, as it does for a submodule
	sep := filepath.Join(t.TempDir(), "sep")
	r.Git("init", "-q", "--separate-git-dir", filepath.Join(t.TempDir(), "gitdir"), sep)
	check("separate git dir", sep, sep)
}

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
