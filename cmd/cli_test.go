package cmd_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pavelsimo/kiln/internal/gittest"
)

var bin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kiln-bin-")
	if err != nil {
		panic(err)
	}
	bin = filepath.Join(dir, "kiln")
	build := exec.Command("go", "build", "-ldflags", "-X github.com/pavelsimo/kiln/cmd.Version=0.0.0-test", "-o", bin, "..")
	if out, err := build.CombinedOutput(); err != nil {
		panic(string(out))
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func kiln(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "NO_COLOR=1", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := c.CombinedOutput()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

func TestEndToEnd(t *testing.T) {
	r := gittest.New(t)
	r.Git("remote", "add", "origin", "git@github.com:acme/app.git")

	out, code := kiln(t, r.Dir, "init", "--no-github")
	if code != 0 || !strings.Contains(out, "21 skills") || !strings.Contains(out, "/kiln-onboard") {
		t.Fatalf("init (%d):\n%s", code, out)
	}
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "🔧 chore: install kiln")

	out, code = kiln(t, r.Dir, "doctor", "--no-github", "--json")
	var doc struct {
		Healthy  bool `json:"healthy"`
		Findings []struct{ Name, Level, Detail string }
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("doctor json: %v\n%s", err, out)
	}
	if code != 0 || !doc.Healthy {
		t.Fatalf("doctor (%d): %+v", code, doc.Findings)
	}

	// a task branch with a standing check that applies
	r.Write(map[string]string{".kiln/verify/checks.yaml": "- id: css-tokens\n  when: { paths: [\"src/**/*.css\"] }\n  predicate: lint passes\n  blocking: true\n"})
	r.Git("add", "-A")
	r.Git("commit", "-q", "-m", "🔧 chore: add a check")
	r.Git("checkout", "-q", "-b", "kiln/42-dark-mode")
	sha := r.Commit("✨ feat: toggle", map[string]string{"src/theme.css": "dark"})

	out, code = kiln(t, r.Dir, "checks", "--base", "main", "--json")
	if code != 0 || !strings.Contains(out, `"css-tokens"`) || !strings.Contains(out, `"src/theme.css"`) {
		t.Fatalf("checks (%d):\n%s", code, out)
	}

	_, code = kiln(t, r.Dir, "verdict")
	if code != 1 {
		t.Fatalf("verdict with no journal should be an error (1), got %d", code)
	}
	r.Commit("📝 docs(kiln): log verify", map[string]string{".kiln/memory/tasks/42-dark-mode.md": "---\nissue: 42\n---\n## Journal\n" +
		"| 2026-10-28 10:00 | kiln-verify | 2 predicates | " + sha[:7] + " | VERIFIED |\n"})
	out, code = kiln(t, r.Dir, "verdict", "--only", "verify")
	if code != 0 || !strings.Contains(out, "fresh") {
		t.Fatalf("verdict --only verify (%d):\n%s", code, out)
	}
	_, code = kiln(t, r.Dir, "verdict")
	if code != 3 {
		t.Fatalf("verdict without a review should exit 3, got %d", code)
	}

	r.Git("checkout", "-q", "main")
	r.Git("merge", "-q", "--squash", "kiln/42-dark-mode")
	r.Git("commit", "-q", "-m", "✨ feat: add dark mode to settings (#42)")
	out, code = kiln(t, r.Dir, "release", "plan", "--ref", "main", "--json")
	var plan struct{ Version, Rule string }
	if err := json.Unmarshal([]byte(out), &plan); err != nil || code != 0 || plan.Version != "0.1.0" {
		t.Fatalf("release plan (%d): %+v\n%s", code, plan, out)
	}

	out, code = kiln(t, r.Dir, "update", "--dry-run")
	if code != 0 || !strings.Contains(out, "up to date") {
		t.Fatalf("update (%d):\n%s", code, out)
	}
}

func TestExitCodes(t *testing.T) {
	r := gittest.New(t)
	r.Git("tag", "v0.1.0")
	tests := []struct {
		args []string
		want int
	}{
		{[]string{"version"}, 0},
		{[]string{"--bogus"}, 2},
		{[]string{"nope"}, 2},
		{[]string{"move", "42"}, 2},
		{[]string{"move", "x", "ready"}, 2},
		{[]string{"move", "42", "done"}, 2},
		{[]string{"move", "42", "blocked"}, 2},
		{[]string{"verdict"}, 2},                          // main isn't a kiln/<n> branch
		{[]string{"update"}, 1},                           // not installed
		{[]string{"release", "plan", "--ref", "main"}, 3}, // nothing since v0.1.0
	}
	for _, tt := range tests {
		out, code := kiln(t, r.Dir, tt.args...)
		if code != tt.want {
			t.Errorf("kiln %s = %d, want %d\n%s", strings.Join(tt.args, " "), code, tt.want, out)
		}
	}
	out, code := kiln(t, t.TempDir(), "status")
	if code != 1 || !strings.Contains(out, "not a git repository") {
		t.Errorf("outside a repo: %d %s", code, out)
	}
}
