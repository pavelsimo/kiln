package cmd

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/install"
)

var updateDryRun bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Upgrade the installed skills to this binary's version",
	Long: `Upgrade the installed skills to the ones built into this binary.

Files whose hash still matches .kiln/kiln.lock are replaced. Files you edited are
kept, and the new version is written next to them as <file>.kiln-new for you to
merge. New skills are linked into your agents; removed ones are unlinked.`,
	Example: `  kiln update --dry-run
  kiln update`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, true)
		if err != nil {
			return err
		}
		rep, err := install.Update(e.ctx, install.Options{Root: e.root, DryRun: updateDryRun, Version: Version})
		if err != nil {
			return err
		}
		return e.out(rep, func() {
			u := e.ui
			if rep.DryRun {
				u.Println(u.Dim("dry run: nothing was written"))
			}
			changed := 0
			for _, a := range rep.Actions {
				switch a.Kind {
				case "create", "replace", "remove", "link", "unlink":
					u.OK("%-8s %s", a.Kind, a.Path)
					changed++
				case "conflict":
					u.Warn("you edited %s · new version in %s", a.Path[:len(a.Path)-len(install.NewSuffix)], a.Path)
					changed++
				}
			}
			if changed == 0 {
				u.OK("already up to date · %d skills", rep.Skills)
				return
			}
			u.Println("  " + u.Dim(strconv.Itoa(changed)+" changes · kiln "+Version))
		})
	},
}

func init() {
	updateCmd.Flags().BoolVarP(&updateDryRun, "dry-run", "n", false, "show what would change without writing anything")
	rootCmd.AddCommand(updateCmd)
}
