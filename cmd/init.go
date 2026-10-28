package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/install"
)

var initFlags struct {
	agents   string
	noGitHub bool
	force    bool
	dryRun   bool
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install kiln into this repo",
	Long: `Install kiln into this repo.

Writes .kiln/ (skills, config.yaml, kiln.lock, memory and verify skeletons), links
each skill into .claude/skills, .agents/skills and .cursor/skills, adds a pointer
block to AGENTS.md and CLAUDE.md, creates 14 labels, and writes an issue template,
a PR template and a CI workflow.

Safe to re-run. Existing files are left alone. --force replaces skills and GitHub
templates, but never your config, memory or verify files.`,
	Example: `  kiln init
  kiln init --agents claude,codex
  kiln init --no-github --dry-run`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		opts := install.Options{Root: e.root, Force: initFlags.force, DryRun: initFlags.dryRun, Version: Version}
		if initFlags.agents != "" {
			for _, a := range strings.Split(initFlags.agents, ",") {
				if a = strings.TrimSpace(a); a != "" {
					opts.Agents = append(opts.Agents, a)
				}
			}
		}
		login := ""
		if !initFlags.noGitHub {
			if _, _, ok := gitx.ParseGitHubRemote(e.git.RemoteURL(e.ctx, "origin")); !ok {
				return usage("no GitHub remote named origin; add one, or pass --no-github")
			}
			if login, err = e.gh.Login(e.ctx); err != nil {
				return err
			}
			opts.GH = e.gh
		}
		rep, err := install.Init(e.ctx, opts)
		if err != nil {
			return err
		}
		return e.out(rep, func() {
			u := e.ui
			if rep.DryRun {
				u.Println(u.Dim("dry run: nothing was written"))
			}
			if login != "" {
				u.OK("git repo · gh authenticated as %s", login)
			} else {
				u.OK("git repo · GitHub skipped")
			}
			u.OK(".kiln/ written · %d skills · config.yaml · kiln.lock %s", rep.Skills, u.Dim(summary(rep)))
			dirs := make([]string, 0, len(rep.Agents))
			for _, a := range rep.Agents {
				dirs = append(dirs, install.AgentDirs[a])
			}
			u.OK("linked → %s", strings.Join(dirs, "  "))
			if rep.Labels > 0 {
				u.OK("%d labels · issue + PR templates · ci.yml", rep.Labels)
			} else {
				u.OK("issue + PR templates · ci.yml")
			}
			for _, a := range rep.Actions {
				if a.Kind == "skip" && a.Detail != "exists" {
					u.Warn("%s: %s", a.Path, a.Detail)
				}
			}
			u.Println("  next: run " + u.Ember("/kiln-onboard") + " in your agent")
		})
	},
}

func summary(rep install.Report) string {
	parts := []string{}
	if n := rep.Count("create"); n > 0 {
		parts = append(parts, itoa(n)+" created")
	}
	if n := rep.Count("replace"); n > 0 {
		parts = append(parts, itoa(n)+" replaced")
	}
	if n := rep.Count("skip"); n > 0 {
		parts = append(parts, itoa(n)+" kept")
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func init() {
	f := initCmd.Flags()
	f.StringVar(&initFlags.agents, "agents", "", "agents to link, comma separated: claude,codex,cursor (default: detect)")
	f.BoolVar(&initFlags.noGitHub, "no-github", false, "skip gh auth and label creation (files are still written)")
	f.BoolVarP(&initFlags.force, "force", "f", false, "replace existing skills and GitHub templates")
	f.BoolVarP(&initFlags.dryRun, "dry-run", "n", false, "show what would change without writing anything")
	rootCmd.AddCommand(initCmd)
}
