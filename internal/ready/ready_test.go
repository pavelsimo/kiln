package ready

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gittest"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/run"
)

func prJSON(t *testing.T, mut func(m map[string]any)) string {
	m := map[string]any{
		"number": 57, "title": "✨ feat: add dark mode to settings", "state": "OPEN", "isDraft": false,
		"mergeable": "MERGEABLE", "reviewDecision": "", "headRefName": "kiln/42-dark-mode",
		"headRefOid": "", "baseRefName": "main", "body": "## Why\n...\nCloses #42\n",
		"statusCheckRollup": []map[string]any{
			{"__typename": "CheckRun", "name": "test", "status": "COMPLETED", "conclusion": "SUCCESS"},
			{"__typename": "StatusContext", "context": "deploy/preview", "state": "SUCCESS"},
		},
	}
	mut(m)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const threads = `{"data":{"repository":{"pullRequest":{"reviewThreads":{"nodes":[{"isResolved":true}]}}}}}`

func setup(t *testing.T, mut func(m map[string]any), journal bool) Report {
	r := gittest.New(t)
	r.Git("checkout", "-q", "-b", "kiln/42-dark-mode")
	sha := r.Commit("✨ feat: toggle", map[string]string{"a.css": "x"})
	if journal {
		sha = r.Commit("📝 log", map[string]string{".kiln/memory/tasks/42-dark-mode.md": "## Journal\n" +
			"| 2026-10-27 14:37 | kiln-verify | p | " + sha + " | VERIFIED |\n" +
			"| 2026-10-27 14:40 | kiln-review | r | " + sha + " | PASS |\n"})
	}
	// origin/main is what ready compares against
	r.Git("update-ref", "refs/remotes/origin/main", "main")
	fake := &run.Fake{Responses: map[string]run.Response{
		"gh pr view 57 *":  {Out: prJSON(t, func(m map[string]any) { m["headRefOid"] = sha; mut(m) })},
		"gh api graphql *": {Out: threads},
	}}
	rep, err := Check(context.Background(), ghx.New(fake, r.Dir), gitx.New(run.Exec{}, r.Dir), Options{PR: 57, Repo: "acme/app"})
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func failing(rep Report) []string {
	var out []string
	for _, c := range rep.Checks {
		if !c.OK {
			out = append(out, c.Name)
		}
	}
	return out
}

func TestReady(t *testing.T) {
	rep := setup(t, func(map[string]any) {}, true)
	if !rep.Ready || rep.Issue != 42 {
		t.Fatalf("want ready, failing: %v %+v", failing(rep), rep.Checks)
	}
}

func TestNotReady(t *testing.T) {
	rep := setup(t, func(m map[string]any) {
		m["isDraft"] = true
		m["mergeable"] = "CONFLICTING"
		m["reviewDecision"] = "CHANGES_REQUESTED"
		m["body"] = "no link"
		m["statusCheckRollup"] = []map[string]any{
			{"__typename": "CheckRun", "name": "test", "status": "COMPLETED", "conclusion": "FAILURE"},
			{"__typename": "CheckRun", "name": "lint", "status": "IN_PROGRESS"},
		}
	}, false)
	got := failing(rep)
	want := []string{"open", "conflicts", "ci", "review", "closes", "verdict"}
	if rep.Ready || len(got) != len(want) {
		t.Fatalf("failing = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("failing = %v, want %v", got, want)
		}
	}
}

func TestHousekeepingSkipsVerdict(t *testing.T) {
	rep := setup(t, func(m map[string]any) {
		m["headRefName"] = "kiln/release-v1.4.0"
		m["body"] = "release notes"
	}, false)
	if !rep.Ready || !rep.Housekeeping {
		t.Fatalf("release PR should be ready on CI alone, failing: %v", failing(rep))
	}
}

func TestClosesIssue(t *testing.T) {
	for body, want := range map[string]int{"Closes #42": 42, "fixes #7 and more": 7, "resolved #12": 12, "see #3": 0} {
		got, _ := ClosesIssue(body)
		if got != want {
			t.Errorf("ClosesIssue(%q) = %d, want %d", body, got, want)
		}
	}
}
