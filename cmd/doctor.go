package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/doctor"
)

var doctorNoGitHub bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check that kiln is set up correctly",
	Long: `Check that kiln is set up correctly: git, the GitHub remote, gh auth, labels,
the config, installed skills against kiln.lock, agent links, the worktree folder,
standing checks, and whether the verify driver has fallen behind the code.

Exits 3 when something needs fixing. Warnings don't change the exit code.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		opts := doctor.Options{Root: e.root, Version: Version, Git: e.git}
		if !doctorNoGitHub {
			opts.GH = e.gh
		}
		rep := doctor.Run(e.ctx, opts)
		err = e.out(rep, func() {
			u := e.ui
			for _, f := range rep.Findings {
				switch f.Level {
				case doctor.OK:
					u.OK("%s", f.Detail)
				case doctor.Warn:
					u.Warn("%s", f.Detail)
				default:
					u.Fail("%s", f.Detail)
				}
				if f.Fix != "" && f.Level != doctor.OK {
					u.Note("fix: %s", f.Fix)
				}
			}
		})
		if err != nil {
			return err
		}
		if !rep.Healthy {
			return no(errors.New(""))
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorNoGitHub, "no-github", false, "skip the checks that call GitHub")
	rootCmd.AddCommand(doctorCmd)
}
