package verdict

import (
	"context"
	"fmt"
	"testing"

	"github.com/pavelsimo/kiln/internal/gittest"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/run"
)

func journalBody(verifySHA, verifyResult, reviewSHA, reviewResult string) string {
	j := "---\nissue: 42\nbranch: kiln/42-dark-mode\n---\n## Journal\n| ts | skill | action | ref | result |\n|---|---|---|---|---|\n"
	if verifySHA != "" {
		j += fmt.Sprintf("| 2026-10-27 14:37 | kiln-verify | 4 predicates | %s | %s |\n", verifySHA, verifyResult)
	}
	if reviewSHA != "" {
		j += fmt.Sprintf("| 2026-10-27 14:40 | kiln-review | reviewer | %s | %s |\n", reviewSHA, reviewResult)
	}
	return j
}

const jpath = ".kiln/memory/tasks/42-dark-mode.md"

func setup(t *testing.T) (*gittest.Repo, *gitx.Git, string) {
	r := gittest.New(t)
	r.Git("checkout", "-q", "-b", "kiln/42-dark-mode")
	sha := r.Commit("✨ feat: add toggle", map[string]string{"src/theme.css": "dark"})
	return r, gitx.New(run.Exec{}, r.Dir), sha
}

func check(t *testing.T, g *gitx.Git, opts Options) Report {
	t.Helper()
	if opts.Issue == 0 {
		opts.Issue = 42
	}
	if opts.Rev == "" {
		opts.Rev = "HEAD"
	}
	if opts.Base == "" {
		opts.Base = "main"
	}
	rep, err := Check(context.Background(), g, opts)
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func TestFreshAfterJournalCommit(t *testing.T) {
	r, g, sha := setup(t)
	// the journal commit comes after verifying, and must not make the verdict stale
	r.Commit("📝 docs(kiln): log verify for #42", map[string]string{jpath: journalBody(sha[:7], "VERIFIED", sha[:7], "PASS")})
	rep := check(t, g, Options{})
	if !rep.Fresh || rep.Verify.How != "same-tree" || rep.Verify.SHA != sha {
		t.Fatalf("want fresh via same-tree, got %+v %+v", rep, rep.Verify)
	}
}

func TestStaleAfterCodeChange(t *testing.T) {
	r, g, sha := setup(t)
	r.Commit("📝 log", map[string]string{jpath: journalBody(sha, "VERIFIED", sha, "PASS")})
	r.Commit("✨ feat: one more thing", map[string]string{"src/other.ts": "x"})
	rep := check(t, g, Options{})
	if rep.Fresh || rep.Verify.Fresh || rep.Verify.How != "stale" {
		t.Fatalf("want stale, got %+v", rep.Verify)
	}
}

func TestFreshAfterRebase(t *testing.T) {
	r, g, sha := setup(t)
	r.Commit("📝 log", map[string]string{jpath: journalBody(sha, "VERIFIED", sha, "PASS")})
	r.Git("checkout", "-q", "main")
	r.Commit("🐛 fix: unrelated", map[string]string{"server/x.go": "y"})
	r.Git("checkout", "-q", "kiln/42-dark-mode")
	r.Git("rebase", "-q", "main")
	rep := check(t, g, Options{})
	if !rep.Fresh || rep.Verify.How != "patch-id" {
		t.Fatalf("want fresh via patch-id, got %+v", rep.Verify)
	}
}

func TestResults(t *testing.T) {
	r, g, sha := setup(t)
	r.Commit("📝 log", map[string]string{jpath: journalBody(sha, "INCONCLUSIVE", sha, "PASS")})
	if rep := check(t, g, Options{}); rep.Fresh {
		t.Fatal("INCONCLUSIVE should not pass")
	}
	if rep := check(t, g, Options{AcceptInconclusive: true}); !rep.Fresh {
		t.Fatalf("accepted INCONCLUSIVE should pass: %v", rep.Reasons)
	}
	r.Commit("📝 log", map[string]string{jpath: journalBody(sha, "VERIFIED", "", "")})
	if rep := check(t, g, Options{}); rep.Fresh {
		t.Fatal("missing review should not pass")
	}
	if rep := check(t, g, Options{Only: "verify"}); !rep.Fresh {
		t.Fatalf("--only verify should ignore the review: %v", rep.Reasons)
	}
	r.Commit("📝 log", map[string]string{jpath: journalBody(sha, "VERIFIED", sha, "FIX")})
	if rep := check(t, g, Options{}); rep.Fresh {
		t.Fatal("FIX should not pass")
	}
}

func TestNoJournal(t *testing.T) {
	_, g, _ := setup(t)
	if _, err := Check(context.Background(), g, Options{Issue: 42, Rev: "HEAD", Base: "main"}); err == nil {
		t.Fatal("want an error with no journal")
	}
}
