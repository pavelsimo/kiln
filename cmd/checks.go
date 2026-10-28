package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/checks"
	"github.com/pavelsimo/kiln/internal/gitx"
)

var checksFlags struct {
	base   string
	labels string
}

type checksReport struct {
	Base    string         `json:"base"`
	Changed []string       `json:"changed"`
	Checks  []checks.Match `json:"checks"`
}

var checksCmd = &cobra.Command{
	Use:   "checks",
	Short: "List the standing checks that apply to this branch",
	Long: `List the standing checks in .kiln/verify/checks.yaml that apply to this branch,
by matching their path globs against git diff --name-only <base>...HEAD and their
labels against the issue's labels.`,
	Example: `  kiln checks
  kiln checks --base origin/main --json
  kiln checks --labels type:feature,size:M`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		base := checksFlags.base
		if base == "" {
			base = e.git.BaseRef(e.ctx)
		}
		changed, err := e.git.ChangedFiles(e.ctx, base, "HEAD")
		if err != nil {
			return err
		}
		labels := splitList(checksFlags.labels)
		cs, err := checks.Load(filepath.Join(e.root, e.cfg.Verify.Dir))
		if err != nil {
			return err
		}
		if checksFlags.labels == "" && needsLabels(cs) {
			if n, ok := gitx.BranchIssue(e.git.CurrentBranch(e.ctx)); ok {
				if is, err := e.gh.Issue(e.ctx, n); err == nil {
					labels = is.LabelNames()
				}
			}
		}
		rep := checksReport{Base: base, Changed: changed, Checks: checks.Apply(cs, changed, labels)}
		if rep.Changed == nil {
			rep.Changed = []string{}
		}
		if rep.Checks == nil {
			rep.Checks = []checks.Match{}
		}
		return e.out(rep, func() {
			u := e.ui
			u.Println(u.Dim(fmt.Sprintf("%d files changed since %s · %d of %d standing checks apply", len(changed), base, len(rep.Checks), len(cs))))
			for _, m := range rep.Checks {
				tag := "advisory"
				if m.Blocking {
					tag = "blocking"
				}
				u.Println(fmt.Sprintf("%s  %s  %s", u.Bold(m.ID), u.Dim(tag), m.Predicate))
				if len(m.Matched) > 0 {
					u.Note("matched %s", strings.Join(first(m.Matched, 3), ", "))
				}
			}
		})
	},
}

func needsLabels(cs []checks.Check) bool {
	for _, c := range cs {
		if len(c.When.Labels) > 0 {
			return true
		}
	}
	return false
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return append(xs[:n:n], fmt.Sprintf("and %d more", len(xs)-n))
	}
	return xs
}

func init() {
	f := checksCmd.Flags()
	f.StringVar(&checksFlags.base, "base", "", "compare against this ref (default: origin/<default branch>)")
	f.StringVar(&checksFlags.labels, "labels", "", "issue labels to match, comma separated (default: read from the branch's issue)")
	rootCmd.AddCommand(checksCmd)
}
