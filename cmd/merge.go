package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/state"
)

var mergeFlags struct {
	acceptInconclusive bool
	noCleanup          bool
	dryRun             bool
}

type mergeReport struct {
	PR       int      `json:"pr"`
	Merged   bool     `json:"merged"`
	MergeSHA string   `json:"merge_sha,omitempty"`
	Issue    int      `json:"issue,omitempty"`
	Cleanup  []string `json:"cleanup"`
	DryRun   bool     `json:"dry_run"`
}

var mergeCmd = &cobra.Command{
	Use:   "merge <pr>",
	Short: "Squash merge a PR, only if kiln ready says yes",
	Long: `Squash merge a PR, but only when kiln ready passes, whatever the agent believes.

The merge is pinned to the head commit kiln checked, so a late push can't sneak in.
Afterwards kiln reads the PR back to confirm it's MERGED, removes kiln:* labels from
the closed issue, and cleans up: the task's worktree (only if it has no local
changes), the local branch and the remote branch.

Exits 3 when kiln ready says no.`,
	Example: `  kiln merge 57
  kiln merge 57 --accept-inconclusive
  kiln merge 61 --no-cleanup --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		n, err := parseNumber(args[0], "pr")
		if err != nil {
			return err
		}
		rd, err := checkReady(e, n, mergeFlags.acceptInconclusive)
		if err != nil {
			return err
		}
		if !rd.Ready {
			if !g.json {
				printReady(e.ui, rd)
			}
			var left []string
			for _, c := range rd.Checks {
				if !c.OK {
					left = append(left, c.Name+": "+c.Detail)
				}
			}
			return no(fmt.Errorf("refusing to merge #%d: %s", n, strings.Join(left, "; ")))
		}
		rep := mergeReport{PR: n, Issue: rd.Issue, Cleanup: []string{}, DryRun: mergeFlags.dryRun}
		if mergeFlags.dryRun {
			return e.out(rep, func() { e.ui.OK("#%d is ready; would squash merge %s", n, gitx.Short(rd.Head)) })
		}
		if err := e.gh.SquashMerge(e.ctx, n, rd.Head); err != nil {
			return err
		}
		pr, err := readBack(e, n)
		if err != nil {
			return err
		}
		rep.Merged = true
		if pr.MergeCommit != nil {
			rep.MergeSHA = pr.MergeCommit.Oid
		}
		if rd.Issue > 0 {
			if is, err := e.gh.Issue(e.ctx, rd.Issue); err == nil {
				var remove []string
				for _, l := range is.LabelNames() {
					if s, ok := strings.CutPrefix(l, "kiln:"); ok && state.Valid(s) {
						remove = append(remove, l)
					}
				}
				if err := e.gh.EditLabels(e.ctx, rd.Issue, nil, remove); err == nil && len(remove) > 0 {
					rep.Cleanup = append(rep.Cleanup, "removed "+strings.Join(remove, ", ")+" from #"+itoa(rd.Issue))
				}
			}
		}
		if !mergeFlags.noCleanup {
			rep.Cleanup = append(rep.Cleanup, cleanup(e, pr.HeadRefName)...)
		}
		return e.out(rep, func() {
			u := e.ui
			u.OK("#%d squash merged · %s", n, gitx.Short(rep.MergeSHA))
			for _, c := range rep.Cleanup {
				u.OK("%s", c)
			}
		})
	},
}

// readBack polls until GitHub reports the PR as merged.
func readBack(e *env, n int) (ghx.PR, error) {
	var pr ghx.PR
	var err error
	for i := 0; i < 10; i++ {
		pr, err = e.gh.PR(e.ctx, n)
		if err == nil && strings.EqualFold(pr.State, "MERGED") {
			return pr, nil
		}
		select {
		case <-e.ctx.Done():
			return pr, e.ctx.Err()
		case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
		}
	}
	if err != nil {
		return pr, err
	}
	return pr, fmt.Errorf("merged #%d, but GitHub still reports it as %s; check it by hand", n, strings.ToLower(pr.State))
}

// cleanup removes the merged branch's worktree and branches. Failures are reported, not fatal.
func cleanup(e *env, branch string) []string {
	var done []string
	wts, _ := e.git.Worktrees(e.ctx)
	for _, w := range wts {
		if w.IsPrimary || w.Branch != branch {
			continue
		}
		clean, err := e.git.IsClean(e.ctx, w.Path)
		rel, _ := filepath.Rel(e.root, w.Path)
		if err != nil || !clean {
			done = append(done, "kept worktree "+rel+" (it has local changes)")
			continue
		}
		if _, err := e.git.Run(e.ctx, "worktree", "remove", w.Path); err != nil {
			done = append(done, "couldn't remove worktree "+rel+": "+err.Error())
			continue
		}
		done = append(done, "worktree "+rel+" removed")
	}
	if e.git.RefExists(e.ctx, "refs/heads/"+branch) && e.git.CurrentBranch(e.ctx) != branch {
		// squash merges aren't ancestors, so -d would refuse; the read-back proved it merged
		if _, err := e.git.Run(e.ctx, "branch", "-D", branch); err == nil {
			done = append(done, "branch "+branch+" deleted")
		}
	}
	if _, err := e.git.Run(e.ctx, "push", "--quiet", "origin", "--delete", branch); err == nil {
		done = append(done, "remote branch "+branch+" deleted")
	} else if !errors.Is(err, nil) && !strings.Contains(err.Error(), "remote ref does not exist") {
		done = append(done, "remote branch "+branch+" left in place")
	}
	return done
}

func init() {
	f := mergeCmd.Flags()
	f.BoolVar(&mergeFlags.acceptInconclusive, "accept-inconclusive", false, "merge with an INCONCLUSIVE verify (a human decision at gate 2)")
	f.BoolVar(&mergeFlags.noCleanup, "no-cleanup", false, "keep the worktree and branches after merging")
	f.BoolVarP(&mergeFlags.dryRun, "dry-run", "n", false, "run every check, but don't merge")
	rootCmd.AddCommand(mergeCmd)
}
