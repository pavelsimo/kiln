// Package gittest builds throwaway git repositories for tests.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a temporary git repository.
type Repo struct {
	T   *testing.T
	Dir string
}

// New creates an empty repo on branch main with one initial commit.
func New(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()
	r := &Repo{T: t, Dir: dir}
	r.Git("init", "-q", "-b", "main")
	r.Git("config", "user.email", "test@example.com")
	r.Git("config", "user.name", "Test")
	r.Git("config", "commit.gpgsign", "false")
	r.Git("config", "tag.gpgsign", "false")
	r.Commit("🎉 init", map[string]string{"README.md": "# test\n"})
	return r
}

// Git runs git in the repo and returns trimmed output, failing the test on error.
func (r *Repo) Git(args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// Write writes files relative to the repo root.
func (r *Repo) Write(files map[string]string) {
	r.T.Helper()
	for p, c := range files {
		full := filepath.Join(r.Dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			r.T.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			r.T.Fatal(err)
		}
	}
}

// Commit writes files and commits them, returning the new HEAD SHA.
func (r *Repo) Commit(msg string, files map[string]string) string {
	r.T.Helper()
	r.Write(files)
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", msg)
	return r.Git("rev-parse", "HEAD")
}
