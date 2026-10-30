package hooks

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadReadsUserThenLocal(t *testing.T) {
	xdg, root := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	write(t, filepath.Join(xdg, "kiln", "hooks.yaml"), "hooks:\n  - on: move\n    to: in-progress\n    run: echo user\n")
	write(t, filepath.Join(root, LocalPath), "hooks:\n  - on: merge\n    run: echo local\n    timeout: 2s\n")

	files, errs := Load(root)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(files) != 2 {
		t.Fatalf("want 2 files, got %d", len(files))
	}
	if files[0].Path != filepath.Join(xdg, "kiln", "hooks.yaml") || files[0].Hooks[0].Run != "echo user" {
		t.Errorf("first file = %+v", files[0])
	}
	if files[1].Path != filepath.Join(root, LocalPath) || files[1].Hooks[0].Timeout != 2*time.Second {
		t.Errorf("second file = %+v", files[1])
	}
}

func TestLoadFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".config", "kiln", "hooks.yaml"), "hooks:\n  - on: merge\n    run: echo hi\n")
	files, errs := Load(t.TempDir())
	if len(errs) != 0 || len(files) != 1 {
		t.Fatalf("files %+v, errs %v", files, errs)
	}
}

func TestLoadWithNoFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	files, errs := Load(t.TempDir())
	if len(files) != 0 || len(errs) != 0 {
		t.Fatalf("files %+v, errs %v", files, errs)
	}
}

func TestLoadRejectsBadFiles(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{"not yaml", "hooks: [", "yaml"},
		{"unknown key", "hooks:\n  - on: move\n    run: x\n    when: now\n", "field when not found"},
		{"bad event", "hooks:\n  - on: release\n    run: x\n", `on: "release" is not one of move, merge`},
		{"no run", "hooks:\n  - on: move\n", "run: missing"},
		{"bad state", "hooks:\n  - on: move\n    to: done\n    run: x\n", `to: "done" is not a kiln state`},
		{"filter on merge", "hooks:\n  - on: merge\n    from: review\n    run: x\n", "from and to only apply to move"},
		{"negative timeout", "hooks:\n  - on: move\n    run: x\n    timeout: -1s\n", "timeout: must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			root := t.TempDir()
			write(t, filepath.Join(root, LocalPath), tt.body)
			files, errs := Load(root)
			if len(files) != 0 || len(errs) != 1 {
				t.Fatalf("files %+v, errs %v", files, errs)
			}
			msg := errs[0].Error()
			if !strings.Contains(msg, LocalPath) || !strings.Contains(msg, tt.want) {
				t.Errorf("error %q should name the file and say %q", msg, tt.want)
			}
		})
	}
}

func TestMatching(t *testing.T) {
	files := []File{
		{Hooks: []Hook{
			{On: "move", Run: "any move"},
			{On: "move", To: "in-progress", Run: "to in-progress"},
			{On: "move", From: "review", To: "in-progress", Run: "back from review"},
			{On: "merge", Run: "merged"},
		}},
		{Hooks: []Hook{{On: "move", To: "blocked", Run: "blocked"}}},
	}
	runs := func(hs []Hook) string {
		var s []string
		for _, h := range hs {
			s = append(s, h.Run)
		}
		return strings.Join(s, ", ")
	}
	tests := []struct{ event, from, to, want string }{
		{"move", "planned", "in-progress", "any move, to in-progress"},
		{"move", "review", "in-progress", "any move, to in-progress, back from review"},
		{"move", "in-progress", "blocked", "any move, blocked"},
		{"merge", "", "", "merged"},
	}
	for _, tt := range tests {
		if got := runs(Matching(files, tt.event, tt.from, tt.to)); got != tt.want {
			t.Errorf("%s %s→%s = %q, want %q", tt.event, tt.from, tt.to, got, tt.want)
		}
	}
}

func TestRunPassesEnvAndStdin(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	hs := []Hook{{On: "merge", Run: `printf '%s %s %s|' "$KILN_EVENT" "$KILN_PR" "$KILN_MERGE_SHA"; cat; pwd`}}
	env := map[string]string{"KILN_EVENT": "merge", "KILN_PR": "57", "KILN_MERGE_SHA": "abc1234"}
	res := Run(context.Background(), hs, env, []byte(`{"pr":57}`), dir, &out)
	if len(res) != 1 || !res[0].OK || res[0].Error != "" || res[0].Run != hs[0].Run {
		t.Fatalf("results = %+v", res)
	}
	real, _ := filepath.EvalSymlinks(dir)
	want := `merge 57 abc1234|{"pr":57}` + real + "\n"
	if got := out.String(); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestRunKeepsGoingAfterFailures(t *testing.T) {
	var out bytes.Buffer
	hs := []Hook{
		{On: "move", Run: "echo first; exit 4"},
		{On: "move", Run: "sleep 5", Timeout: 200 * time.Millisecond},
		{On: "move", Run: "echo last"},
	}
	start := time.Now()
	res := Run(context.Background(), hs, nil, nil, t.TempDir(), &out)
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("a timed-out hook held kiln for %s", took)
	}
	if len(res) != 3 {
		t.Fatalf("results = %+v", res)
	}
	if res[0].OK || res[0].Error != "exit status 4" {
		t.Errorf("first = %+v", res[0])
	}
	if res[1].OK || res[1].Error != "timed out after 200ms" {
		t.Errorf("second = %+v", res[1])
	}
	if !res[2].OK {
		t.Errorf("last = %+v", res[2])
	}
	if out.String() != "first\nlast\n" {
		t.Errorf("output = %q", out.String())
	}
}

func TestDisabled(t *testing.T) {
	t.Setenv("KILN_NO_HOOKS", "")
	if Disabled() {
		t.Error("hooks off with KILN_NO_HOOKS empty")
	}
	t.Setenv("KILN_NO_HOOKS", "1")
	if !Disabled() {
		t.Error("hooks on with KILN_NO_HOOKS=1")
	}
}
