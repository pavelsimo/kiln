package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/hooks"
)

// hookEvent is one transition to announce to the user's hooks.
type hookEvent struct {
	name     string // move | merge
	from, to string
	issue    int
	title    string
	branch   string // the task branch, when the caller already knows it
	env      map[string]string
}

var noHooks bool

func addNoHooksFlag(c *cobra.Command) {
	c.Flags().BoolVar(&noHooks, "no-hooks", false, "don't run your hooks for this command (KILN_NO_HOOKS=1 does the same)")
}

// fireHooks runs the user's hooks that match ev, with report as their stdin.
// Problems are warnings on stderr; they never change what kiln did or its exit code.
func fireHooks(e *env, ev hookEvent, report any) []hooks.Result {
	if noHooks || hooks.Disabled() {
		return nil
	}
	// the local hook file is gitignored, so from a task worktree it's in the main checkout
	main := e.root
	if m, err := e.git.MainWorktree(e.ctx); err == nil {
		main = m
	}
	files, errs := hooks.Load(main)
	for _, err := range errs {
		e.ui.Progress("kiln: warning: %v (skipped)", err)
	}
	hs := hooks.Matching(files, ev.name, ev.from, ev.to)
	if len(hs) == 0 {
		return nil
	}
	stdin, err := json.Marshal(report)
	if err != nil {
		e.ui.Progress("kiln: warning: hooks: %v", err)
		return nil
	}
	res := hooks.Run(e.ctx, hs, hookEnv(e, ev), stdin, e.root, e.ui.Err)
	for _, r := range res {
		if !r.OK {
			e.ui.Progress("kiln: warning: hook %q failed: %s", r.Run, r.Error)
		}
	}
	return res
}

// hookEnv builds the KILN_* variables every hook gets. Branch and worktree are empty
// when the task has no worktree yet.
func hookEnv(e *env, ev hookEvent) map[string]string {
	m := map[string]string{
		"KILN_EVENT":    ev.name,
		"KILN_ISSUE":    "",
		"KILN_TITLE":    ev.title,
		"KILN_REPO":     "",
		"KILN_BRANCH":   ev.branch,
		"KILN_WORKTREE": "",
	}
	if ev.issue > 0 {
		m["KILN_ISSUE"] = itoa(ev.issue)
	}
	if repo, err := e.repoName(); err == nil {
		m["KILN_REPO"] = repo
	}
	wts, _ := e.git.Worktrees(e.ctx)
	for _, w := range wts {
		n, isTask := gitx.BranchIssue(w.Branch)
		if (ev.branch != "" && w.Branch == ev.branch) || (ev.branch == "" && isTask && n == ev.issue) {
			m["KILN_BRANCH"], m["KILN_WORKTREE"] = w.Branch, w.Path
			break
		}
	}
	for k, v := range ev.env {
		m[k] = v
	}
	return m
}
