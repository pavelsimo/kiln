package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/ui"
	"github.com/pavelsimo/kiln/internal/verdict"
)

var verdictFlags struct {
	ref                string
	only               string
	acceptInconclusive bool
}

var verdictCmd = &cobra.Command{
	Use:   "verdict [issue]",
	Short: "Say whether the recorded verify and review results still hold",
	Long: `Read the latest kiln-verify and kiln-review rows from the task journal and say
whether they still hold for HEAD (or --ref).

A result is fresh when it was recorded on the same commit, when the tree is the
same apart from .kiln/memory/, or when the change has the same git patch-id (a
clean rebase). The issue is read from a kiln/<n>-<slug> branch name when omitted.

Exits 3 when verify isn't VERIFIED, review isn't PASS, or either is stale.`,
	Example: `  kiln verdict
  kiln verdict 42 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		var n int
		if len(args) == 1 {
			if n, err = parseNumber(args[0], "issue"); err != nil {
				return err
			}
		} else {
			var ok bool
			branch := e.git.CurrentBranch(e.ctx)
			if n, ok = gitx.BranchIssue(branch); !ok {
				return usage("can't tell the issue from branch %q; pass it: kiln verdict <issue>", branch)
			}
		}
		if o := verdictFlags.only; o != "" && o != "verify" && o != "review" {
			return usage("--only must be verify or review, got %q", o)
		}
		rep, err := verdict.Check(e.ctx, e.git, verdict.Options{
			Issue: n, Rev: verdictFlags.ref, Base: e.git.BaseRef(e.ctx), AcceptInconclusive: verdictFlags.acceptInconclusive,
			Only: verdictFlags.only,
		})
		if err != nil {
			return err
		}
		if err := e.out(rep, func() { printVerdict(e.ui, rep) }); err != nil {
			return err
		}
		if !rep.Fresh {
			return no(errors.New(""))
		}
		return nil
	},
}

func printVerdict(u *ui.UI, rep verdict.Report) {
	u.Println(u.Dim(fmt.Sprintf("#%d · head %s · %s", rep.Issue, gitx.Short(rep.Head), rep.Journal)))
	line := func(name string, r *verdict.Result, want string) {
		if r == nil {
			u.Fail("%-7s none recorded", name)
			return
		}
		msg := fmt.Sprintf("%-7s %s on %s · %s", name, r.Result, gitx.Short(r.SHA), r.How)
		if r.Fresh && r.Result == want {
			u.OK("%s", msg)
		} else {
			u.Fail("%s", msg)
		}
	}
	if rep.Only != "review" {
		line("verify", rep.Verify, verdict.Verified)
	}
	if rep.Only != "verify" {
		line("review", rep.Review, verdict.Pass)
	}
	if rep.Fresh {
		u.Println("  " + u.Green("fresh"))
	} else {
		u.Println("  " + u.Red("not fresh: ") + strings.Join(rep.Reasons, "; "))
	}
}

func init() {
	verdictCmd.Flags().StringVar(&verdictFlags.ref, "ref", "HEAD", "the commit to judge")
	verdictCmd.Flags().StringVar(&verdictFlags.only, "only", "", "judge only one result: verify or review")
	verdictCmd.Flags().BoolVar(&verdictFlags.acceptInconclusive, "accept-inconclusive", false, "count an INCONCLUSIVE verify as passing (a human decision)")
	rootCmd.AddCommand(verdictCmd)
}
