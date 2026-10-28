package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/ready"
	"github.com/pavelsimo/kiln/internal/ui"
)

var readyAcceptInconclusive bool

var readyCmd = &cobra.Command{
	Use:   "ready <pr>",
	Short: "Say whether a PR can squash merge right now",
	Long: `Say whether a PR can squash merge right now, and list everything that's left.

Checks: open and not a draft · no conflicts · every CI check passed · no changes
requested · no unresolved review threads · Closes #N in the body · a fresh
VERIFIED and PASS in the task journal. Housekeeping PRs on kiln/ branches with no
issue number (release, retro, onboard) only need the GitHub checks. INCONCLUSIVE
blocks unless you pass --accept-inconclusive.

Exits 3 when the PR isn't ready.`,
	Example: `  kiln ready 57
  kiln ready 57 --json`,
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
		rep, err := checkReady(e, n, readyAcceptInconclusive)
		if err != nil {
			return err
		}
		if err := e.out(rep, func() { printReady(e.ui, rep) }); err != nil {
			return err
		}
		if !rep.Ready {
			return no(errors.New(""))
		}
		return nil
	},
}

func checkReady(e *env, pr int, acceptInconclusive bool) (ready.Report, error) {
	repo, err := e.repoName()
	if err != nil {
		return ready.Report{}, err
	}
	return ready.Check(e.ctx, e.gh, e.git, ready.Options{PR: pr, Repo: repo, AcceptInconclusive: acceptInconclusive})
}

func printReady(u *ui.UI, rep ready.Report) {
	u.Println(u.Dim(fmt.Sprintf("PR #%d · %s · head %s", rep.PR, rep.Branch, gitx.Short(rep.Head))))
	for _, c := range rep.Checks {
		if c.OK {
			u.OK("%-9s %s", c.Name, c.Detail)
		} else {
			u.Fail("%-9s %s", c.Name, c.Detail)
		}
	}
	if rep.Ready {
		u.Println("  " + u.Green("ready to merge"))
	} else {
		u.Println("  " + u.Red("not ready"))
	}
}

func init() {
	readyCmd.Flags().BoolVar(&readyAcceptInconclusive, "accept-inconclusive", false, "count an INCONCLUSIVE verify as passing (a human decision)")
	rootCmd.AddCommand(readyCmd)
}
