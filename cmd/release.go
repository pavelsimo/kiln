package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/release"
)

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Facts for cutting a release",
	Long:  "Facts for cutting a release. The release itself is the /kiln-release skill.",
}

var planFlags struct {
	version string
	pre     string
	ref     string
}

var releasePlanCmd = &cobra.Command{
	Use:   "plan",
	Short: "Work out the next version from the commits since the last tag",
	Long: `Work out the next version from the gitmoji squash commits since the last tag.
Read-only: it never writes, tags or publishes.

  💥 or ! or BREAKING CHANGE   major (minor before 1.0)   Changed
  ✨ feat                       minor                      Added
  🐛 fix · 🚑️ hotfix            patch                      Fixed
  ⚡️ perf · ⬆️ deps · ♻️ refactor patch                      Changed
  🗑️ deprecate                  patch                      Deprecated
  🔥 remove                     patch                      Removed
  🔒️ security                   patch                      Security
  🔧 ✅ 📝 👷 🎨                  none                       left out

The highest bump wins. The first release is 0.1.0 unless you pass --version.
Exits 3 when nothing releasable merged since the last tag, or the tag exists.`,
	Example: `  kiln release plan
  kiln release plan --json
  kiln release plan --version 2.0.0
  kiln release plan --pre rc`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		ref := planFlags.ref
		if ref == "" {
			ref = e.git.BaseRef(e.ctx)
		}
		repoURL := ""
		if owner, name, ok := gitx.ParseGitHubRemote(e.git.RemoteURL(e.ctx, "origin")); ok {
			repoURL = "https://github.com/" + owner + "/" + name
		}
		p, err := release.Build(e.ctx, e.git, release.Options{
			Ref: ref, Version: strings.TrimPrefix(planFlags.version, "v"), Pre: planFlags.pre,
			TagFormat: e.cfg.Release.Tag, RepoURL: repoURL,
		})
		switch {
		case errors.Is(err, release.ErrNothing):
			since := p.LastTag
			if since == "" {
				since = "the start"
			}
			return no(fmt.Errorf("nothing to release: no user-visible change since %s (pass --version to release anyway)", since))
		case errors.Is(err, release.ErrTagExists):
			return no(err)
		case err != nil:
			return err
		}
		return e.out(p, func() {
			u := e.ui
			from := p.LastTag
			if from == "" {
				from = "(no tags)"
			}
			u.Println(u.Ember(fmt.Sprintf("KILN release · %s → %s", from, p.Tag)))
			u.Println(fmt.Sprintf("rule   %s", p.Rule))
			u.Println(fmt.Sprintf("since  %s · %d commits · %d left out", from, len(p.Commits), p.LeftOut))
			u.Println("")
			u.Println(strings.TrimRight(p.Changelog, "\n"))
		})
	},
}

func init() {
	f := releasePlanCmd.Flags()
	f.StringVar(&planFlags.version, "version", "", "use this version instead of the inferred one")
	f.StringVar(&planFlags.pre, "pre", "", "cut a pre-release with this label, e.g. rc → 1.4.0-rc.1")
	f.StringVar(&planFlags.ref, "ref", "", "the branch to release (default: origin/<default branch>)")
	releaseCmd.AddCommand(releasePlanCmd)
	rootCmd.AddCommand(releaseCmd)
}
