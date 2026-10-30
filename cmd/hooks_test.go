package cmd_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pavelsimo/kiln/internal/gittest"
)

// hookRepo is a repo with a task worktree for #9 and a fake gh on PATH that knows #9 at kiln:planned.
type hookRepo struct {
	r        *gittest.Repo
	worktree string
	xdg      string // XDG_CONFIG_HOME for kiln, holding the user hook file
	out      string // OUT for hooks, where they leave what they saw
	env      []string
}

func newHookRepo(t *testing.T) *hookRepo {
	t.Helper()
	r := gittest.New(t)
	r.Git("remote", "add", "origin", "git@github.com:acme/app.git")
	wt := filepath.Join(t.TempDir(), "9-add-hooks")
	r.Git("worktree", "add", "-q", wt, "-b", "kiln/9-add-hooks")

	bin, out, xdg := t.TempDir(), t.TempDir(), t.TempDir()
	gh := `#!/bin/sh
echo "gh $*" >> "$OUT/log"
case "$*" in
  "issue view 9 --json number,title,state,labels,url")
    printf '%s' '{"number":9,"title":"✨ add hooks","state":"OPEN","labels":[{"name":"kiln:planned"}],"url":"https://github.com/acme/app/issues/9"}' ;;
  "issue view 9 --json comments") printf '%s' '{"comments":[]}' ;;
  "issue edit 9 "*|"issue comment 9 "*) ;;
  "pr view 57 --json "*) if [ -f "$OUT/merged" ]; then cat "$OUT/pr-merged.json"; else cat "$OUT/pr.json"; fi ;;
  "pr merge 57 --squash --match-head-commit "*) touch "$OUT/merged" ;;
  "api graphql "*) printf '%s' '{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[]}}}}}' ;;
  *) echo "fake gh: unexpected: $*" >&2; exit 1 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(gh), 0o755); err != nil {
		t.Fatal(err)
	}
	return &hookRepo{
		r: r, worktree: wt, xdg: xdg, out: out,
		env: []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"), "XDG_CONFIG_HOME=" + xdg, "OUT=" + out, "KILN_NO_HOOKS="},
	}
}

func (h *hookRepo) userHooks(t *testing.T, yaml string) {
	t.Helper()
	p := filepath.Join(h.xdg, "kiln", "hooks.yaml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (h *hookRepo) localHooks(yaml string) {
	h.r.Write(map[string]string{".kiln/hooks.local.yaml": yaml})
}

// read returns a file the hooks wrote under OUT, or "" if they never wrote it.
func (h *hookRepo) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(h.out, name))
	return string(b)
}

// kiln runs the binary with the hook repo's environment plus extra, keeping stdout and stderr apart.
func (h *hookRepo) kiln(t *testing.T, extra []string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	c := exec.Command(bin, args...)
	c.Dir = h.r.Dir
	c.Env = append(append(os.Environ(), "NO_COLOR=1", "GIT_CONFIG_GLOBAL=/dev/null"), append(h.env, extra...)...)
	var o, e bytes.Buffer
	c.Stdout, c.Stderr = &o, &e
	err := c.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return o.String(), e.String(), code
}

type moveJSON struct {
	Issue  int    `json:"issue"`
	From   string `json:"from"`
	To     string `json:"to"`
	Moved  bool   `json:"moved"`
	DryRun bool   `json:"dry_run"`
	Hooks  []struct {
		Run   string `json:"run"`
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	} `json:"hooks"`
}

func decodeMove(t *testing.T, s string) moveJSON {
	t.Helper()
	var m moveJSON
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, s)
	}
	return m
}

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMoveRunsHooks(t *testing.T) {
	h := newHookRepo(t)
	h.userHooks(t, `hooks:
  - on: move
    to: in-progress
    run: env | grep '^KILN_' | grep -v '^KILN_NO_HOOKS=' | sort > "$OUT/env"; cat > "$OUT/stdin"; echo hook >> "$OUT/log"; echo user >> "$OUT/order"; echo noise
  - on: move
    to: review
    run: echo never >> "$OUT/order"
`)
	h.localHooks("hooks:\n  - on: move\n    run: echo local >> \"$OUT/order\"\n")

	stdout, stderr, code := h.kiln(t, nil, "move", "9", "in-progress", "--json")
	if code != 0 {
		t.Fatalf("move exited %d\n%s", code, stderr)
	}

	// P5: hook output went to stderr, and stdout is still the JSON report
	m := decodeMove(t, stdout)
	if !strings.Contains(stderr, "noise\n") || !strings.HasPrefix(stdout, "{") || !strings.HasSuffix(stdout, "}\n") {
		t.Errorf("hook output leaked: stdout %q, stderr %q", stdout, stderr)
	}
	if !m.Moved || len(m.Hooks) != 2 || !m.Hooks[0].OK || !m.Hooks[1].OK {
		t.Errorf("report = %+v", m)
	}

	// P2: user hooks first, then local ones, each once; the review hook didn't match
	if got := h.read("order"); got != "user\nlocal\n" {
		t.Errorf("order = %q", got)
	}

	// P1: after the label change, with every KILN_* variable
	log := h.read("log")
	edit, hook := strings.Index(log, "gh issue edit 9 --add-label kiln:in-progress --remove-label kiln:planned"), strings.Index(log, "hook")
	if edit < 0 || hook < edit {
		t.Errorf("the hook must run after the labels change:\n%s", log)
	}
	want := []string{
		"KILN_BRANCH=kiln/9-add-hooks",
		"KILN_EVENT=move",
		"KILN_FROM=planned",
		"KILN_ISSUE=9",
		"KILN_REASON=",
		"KILN_REPO=acme/app",
		"KILN_TITLE=✨ add hooks",
		"KILN_TO=in-progress",
		"KILN_WORKTREE=" + realPath(t, h.worktree),
	}
	if got := strings.TrimSpace(h.read("env")); got != strings.Join(want, "\n") {
		t.Errorf("env:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	in := decodeMove(t, h.read("stdin"))
	if in.Issue != 9 || in.From != "planned" || in.To != "in-progress" || !in.Moved {
		t.Errorf("stdin report = %+v", in)
	}
}

func TestMoveToBlockedPassesTheReason(t *testing.T) {
	h := newHookRepo(t)
	h.userHooks(t, "hooks:\n  - on: move\n    to: blocked\n    run: printf '%s' \"$KILN_REASON\" > \"$OUT/reason\"\n")
	if _, stderr, code := h.kiln(t, nil, "move", "9", "blocked", "--reason", "waiting on the api"); code != 0 {
		t.Fatalf("move exited %d\n%s", code, stderr)
	}
	if got := h.read("reason"); got != "waiting on the api" {
		t.Errorf("KILN_REASON = %q", got)
	}
}

func TestMoveSurvivesFailingHooks(t *testing.T) {
	h := newHookRepo(t)
	h.userHooks(t, `hooks:
  - on: move
    run: exit 4
  - on: move
    run: sleep 5
    timeout: 300ms
  - on: move
    run: echo ran >> "$OUT/order"
`)
	stdout, stderr, code := h.kiln(t, nil, "move", "9", "in-progress", "--json")
	if code != 0 {
		t.Fatalf("a failing hook changed the exit code to %d\n%s", code, stderr)
	}
	m := decodeMove(t, stdout)
	if len(m.Hooks) != 3 || m.Hooks[0].OK || m.Hooks[0].Error != "exit status 4" ||
		m.Hooks[1].OK || m.Hooks[1].Error != "timed out after 300ms" || !m.Hooks[2].OK {
		t.Errorf("hooks = %+v", m.Hooks)
	}
	for _, w := range []string{`hook "exit 4" failed: exit status 4`, `hook "sleep 5" failed: timed out after 300ms`} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr has no %q:\n%s", w, stderr)
		}
	}
	if h.read("order") != "ran\n" {
		t.Errorf("the hook after the failures didn't run")
	}
}

func TestMoveWarnsAboutABrokenHookFile(t *testing.T) {
	h := newHookRepo(t)
	h.userHooks(t, "hooks:\n  - on: release\n    run: x\n")
	h.localHooks("hooks:\n  - on: move\n    run: echo local >> \"$OUT/order\"\n")
	_, stderr, code := h.kiln(t, nil, "move", "9", "in-progress")
	if code != 0 || !strings.Contains(stderr, `"release" is not one of move, merge`) || !strings.Contains(stderr, "skipped") {
		t.Fatalf("move (%d), stderr:\n%s", code, stderr)
	}
	if h.read("order") != "local\n" {
		t.Errorf("the good file's hooks should still run")
	}
}

func TestMoveSkipsHooks(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		args  []string
		code  int
	}{
		{"dry run", nil, []string{"move", "9", "in-progress", "--dry-run", "--json"}, 0},
		{"refused move", nil, []string{"move", "9", "review", "--json"}, 3},
		{"--no-hooks", nil, []string{"move", "9", "in-progress", "--no-hooks", "--json"}, 0},
		{"KILN_NO_HOOKS", []string{"KILN_NO_HOOKS=1"}, []string{"move", "9", "in-progress", "--json"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHookRepo(t)
			h.userHooks(t, "hooks:\n  - on: move\n    run: echo ran >> \"$OUT/order\"\n")
			stdout, stderr, code := h.kiln(t, tt.extra, tt.args...)
			if code != tt.code {
				t.Fatalf("exit %d, want %d\n%s", code, tt.code, stderr)
			}
			if h.read("order") != "" {
				t.Errorf("a hook ran")
			}
			if strings.Contains(stdout, `"hooks"`) {
				t.Errorf("report has a hooks key:\n%s", stdout)
			}
		})
	}
}

func TestMoveWithoutHooksIsUnchanged(t *testing.T) {
	h := newHookRepo(t)
	stdout, stderr, code := h.kiln(t, nil, "move", "9", "in-progress", "--json")
	want := "{\n  \"issue\": 9,\n  \"from\": \"planned\",\n  \"to\": \"in-progress\",\n  \"moved\": true,\n  \"dry_run\": false\n}\n"
	if code != 0 || stdout != want || stderr != "" {
		t.Errorf("--json (%d):\n%s\nstderr: %q", code, stdout, stderr)
	}
	stdout, stderr, code = h.kiln(t, nil, "move", "9", "in-progress")
	if code != 0 || stdout != "  ✓ #9 moved kiln:planned → kiln:in-progress\n" || stderr != "" {
		t.Errorf("human (%d): %q, stderr: %q", code, stdout, stderr)
	}
}

// readyPR gives #9's worktree a verified, reviewed head and PR #57 for it that kiln ready accepts.
// Cleanup's remote branch delete goes to a local bare repo, never to GitHub.
func (h *hookRepo) readyPR(t *testing.T) {
	t.Helper()
	wt := func(args ...string) string { return h.r.Git(append([]string{"-C", h.worktree}, args...)...) }
	if err := os.WriteFile(filepath.Join(h.worktree, "hooks.txt"), []byte("on\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt("add", "hooks.txt")
	wt("commit", "-q", "-m", "✨ feat: add hooks")
	code := wt("rev-parse", "--short", "HEAD")
	journal := "---\nissue: 9\n---\n## Journal\n" +
		"| 2026-10-30 10:00 | kiln-verify | 1 predicate | " + code + " | VERIFIED |\n" +
		"| 2026-10-30 10:05 | kiln-review | self | " + code + " | PASS |\n"
	if err := os.MkdirAll(filepath.Join(h.worktree, ".kiln/memory/tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.worktree, ".kiln/memory/tasks/9-add-hooks.md"), []byte(journal), 0o644); err != nil {
		t.Fatal(err)
	}
	wt("add", ".kiln/memory/tasks/9-add-hooks.md")
	wt("commit", "-q", "-m", "📝 docs(kiln): log kiln-review for #9")
	head := wt("rev-parse", "HEAD")

	h.r.Git("update-ref", "refs/remotes/origin/main", "main")
	bare := t.TempDir()
	h.r.Git("init", "-q", "--bare", bare)
	h.r.Git("config", "remote.origin.pushurl", bare)

	pr := `{"number":57,"title":"✨ add hooks","state":"%s","isDraft":false,"mergeable":"MERGEABLE","reviewDecision":"",` +
		`"headRefName":"kiln/9-add-hooks","headRefOid":"` + head + `","baseRefName":"main","body":"Closes #9",` +
		`"url":"https://github.com/acme/app/pull/57","statusCheckRollup":[],"mergeCommit":%s,"closingIssuesReferences":[]}`
	for name, body := range map[string]string{
		"pr.json":        strings.Replace(strings.Replace(pr, "%s", "OPEN", 1), "%s", "null", 1),
		"pr-merged.json": strings.Replace(strings.Replace(pr, "%s", "MERGED", 1), "%s", `{"oid":"feedface00000000000000000000000000000000"}`, 1),
	} {
		if err := os.WriteFile(filepath.Join(h.out, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMergeRunsHooksBeforeCleanup(t *testing.T) {
	h := newHookRepo(t)
	h.readyPR(t)
	h.userHooks(t, `hooks:
  - on: merge
    run: env | grep '^KILN_' | grep -v '^KILN_NO_HOOKS=' | sort > "$OUT/env"; test -d "$KILN_WORKTREE" && echo present > "$OUT/worktree"; cat > "$OUT/stdin"
  - on: move
    run: echo move >> "$OUT/order"
`)
	stdout, stderr, code := h.kiln(t, nil, "merge", "57", "--json")
	if code != 0 {
		t.Fatalf("merge exited %d\n%s\n%s", code, stdout, stderr)
	}
	var rep struct {
		Merged  bool     `json:"merged"`
		Cleanup []string `json:"cleanup"`
		Hooks   []struct {
			OK bool `json:"ok"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(stdout), &rep); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if !rep.Merged || len(rep.Hooks) != 1 || !rep.Hooks[0].OK {
		t.Errorf("report = %+v", rep)
	}

	// the hook saw the worktree, and cleanup removed it afterwards
	if h.read("worktree") != "present\n" {
		t.Errorf("the merge hook ran after the worktree was gone")
	}
	if _, err := os.Stat(h.worktree); !os.IsNotExist(err) || !strings.Contains(strings.Join(rep.Cleanup, "\n"), "removed") {
		t.Errorf("cleanup didn't remove the worktree: %v %v", err, rep.Cleanup)
	}
	if h.read("order") != "" {
		t.Errorf("a move hook ran on merge")
	}

	want := []string{
		"KILN_BRANCH=kiln/9-add-hooks",
		"KILN_EVENT=merge",
		"KILN_ISSUE=9",
		"KILN_MERGE_SHA=feedface00000000000000000000000000000000",
		"KILN_PR=57",
		"KILN_REPO=acme/app",
		"KILN_TITLE=✨ add hooks",
		"KILN_WORKTREE=" + realPath(t, filepath.Dir(h.worktree)) + "/9-add-hooks",
	}
	if got := strings.TrimSpace(h.read("env")); got != strings.Join(want, "\n") {
		t.Errorf("env:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	var in struct {
		PR       int    `json:"pr"`
		Merged   bool   `json:"merged"`
		MergeSHA string `json:"merge_sha"`
	}
	if err := json.Unmarshal([]byte(h.read("stdin")), &in); err != nil || in.PR != 57 || !in.Merged || in.MergeSHA == "" {
		t.Errorf("stdin report = %+v, %v", in, err)
	}
}

func TestMergeSkipsHooks(t *testing.T) {
	for _, args := range [][]string{
		{"merge", "57", "--dry-run", "--json"},
		{"merge", "57", "--no-hooks", "--json"},
	} {
		t.Run(args[2], func(t *testing.T) {
			h := newHookRepo(t)
			h.readyPR(t)
			h.userHooks(t, "hooks:\n  - on: merge\n    run: echo ran >> \"$OUT/order\"\n")
			stdout, stderr, code := h.kiln(t, nil, args...)
			if code != 0 {
				t.Fatalf("exit %d\n%s", code, stderr)
			}
			if h.read("order") != "" || strings.Contains(stdout, `"hooks"`) {
				t.Errorf("a hook ran:\n%s", stdout)
			}
		})
	}
}

func TestMergeWithoutHooksIsUnchanged(t *testing.T) {
	h := newHookRepo(t)
	h.readyPR(t)
	rel, err := filepath.Rel(realPath(t, h.r.Dir), realPath(t, h.worktree))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := h.kiln(t, nil, "merge", "57")
	want := "  ✓ #57 squash merged · feedfac\n" +
		"  ✓ removed kiln:planned from #9\n" +
		"  ✓ worktree " + rel + " removed\n" +
		"  ✓ branch kiln/9-add-hooks deleted\n"
	if code != 0 || stderr != "" || stdout != want {
		t.Errorf("merge (%d):\n%s\nwant:\n%s\nstderr: %q", code, stdout, want, stderr)
	}
}

func TestInitGitignoresLocalHooks(t *testing.T) {
	r := gittest.New(t)
	r.Git("remote", "add", "origin", "git@github.com:acme/app.git")
	if out, code := kiln(t, r.Dir, "init", "--no-github"); code != 0 {
		t.Fatalf("init (%d):\n%s", code, out)
	}
	r.Write(map[string]string{".kiln/hooks.local.yaml": "hooks: []\n"})
	r.Git("check-ignore", "-q", ".kiln/hooks.local.yaml")
}

func TestDoctorListsHookFiles(t *testing.T) {
	h := newHookRepo(t)
	if _, stderr, code := h.kiln(t, nil, "init", "--no-github"); code != 0 {
		t.Fatalf("init (%d):\n%s", code, stderr)
	}
	h.r.Git("add", "-A")
	h.r.Git("commit", "-q", "-m", "🔧 chore: install kiln")
	h.userHooks(t, "hooks:\n  - on: move\n    run: a\n  - on: merge\n    run: b\n")
	h.localHooks("hooks:\n  - on: move\n    run: c\n")

	type finding struct{ Name, Level, Detail, Fix string }
	doctor := func() ([]finding, int) {
		stdout, _, code := h.kiln(t, nil, "doctor", "--no-github", "--json")
		var doc struct{ Findings []finding }
		if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
			t.Fatalf("doctor json: %v\n%s", err, stdout)
		}
		var hs []finding
		for _, f := range doc.Findings {
			if f.Name == "hooks" {
				hs = append(hs, f)
			}
		}
		return hs, code
	}

	hs, code := doctor()
	user := filepath.Join(h.xdg, "kiln", "hooks.yaml")
	if code != 0 || len(hs) != 2 ||
		hs[0] != (finding{"hooks", "ok", user + " · 2 hooks", ""}) ||
		hs[1] != (finding{"hooks", "ok", ".kiln/hooks.local.yaml · 1 hook", ""}) {
		t.Errorf("doctor (%d): %+v", code, hs)
	}

	h.localHooks("hooks:\n  - on: deploy\n    run: c\n")
	hs, code = doctor()
	if code != 3 || len(hs) != 2 || hs[1].Level != "fail" ||
		!strings.Contains(hs[1].Detail, `.kiln/hooks.local.yaml: hooks[0].on: "deploy" is not one of move, merge`) || hs[1].Fix == "" {
		t.Errorf("doctor with a broken file (%d): %+v", code, hs)
	}
}
