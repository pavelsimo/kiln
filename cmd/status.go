package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/status"
	"github.com/pavelsimo/kiln/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the factory floor: what's in flight and what's waiting on you",
	Long: `Show the factory floor. Reads open issues with kiln:* labels, open PRs and their
checks, git worktrees and task journals. Writes nothing, so it's safe anywhere.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		repo, err := e.repoName()
		if err != nil {
			return err
		}
		f, err := status.Build(e.ctx, e.gh, e.git, e.root, repo, e.cfg)
		if err != nil {
			return err
		}
		return e.out(f, func() { printFloor(e.ui, f) })
	},
}

func printFloor(u *ui.UI, f status.Floor) {
	head := fmt.Sprintf("KILN · %s · %d in flight", f.Repo, len(f.InFlight))
	if w := f.WaitingOnYou(); w > 0 {
		head += fmt.Sprintf(" · %d waiting on you", w)
	}
	u.Println(u.Ember(head))
	width := 0
	for _, t := range f.InFlight {
		width = max(width, len([]rune(t.Title)))
	}
	width = min(width, 34)
	for _, t := range f.InFlight {
		title := t.Title
		if r := []rune(title); len(r) > width {
			title = string(r[:width-1]) + "…"
		}
		pr := "no PR"
		ci := ""
		if t.PR != nil {
			pr = fmt.Sprintf("PR #%d", t.PR.Number)
			c := t.PR.Checks
			switch {
			case c.Total == 0:
				ci = u.Dim("no checks")
			case c.Failing > 0:
				ci = u.Red(fmt.Sprintf("✗ CI %d failing", c.Failing))
			case c.Pending > 0:
				ci = u.Yellow(fmt.Sprintf("● CI %d/%d", c.Passed, c.Total))
			default:
				ci = u.Green("✓ green")
			}
		}
		wait := ""
		switch t.Waiting {
		case "plan":
			wait = u.Ember("◆1 you · plan ready")
		case "merge":
			wait = u.Ember("◆2 you")
		}
		size := t.Size
		if size == "" {
			size = "?"
		}
		cols := []string{fmt.Sprintf("#%-4d", t.Issue), ui.Pad(title, width), ui.Pad(size, 2), ui.Pad(t.Stage, 6), ui.Pad(pr, 7)}
		if ci != "" {
			cols = append(cols, ci)
		}
		if wait != "" {
			if ci != "" {
				cols = append(cols, "·")
			}
			cols = append(cols, wait)
		}
		u.Println(strings.Join(cols, " "))
	}
	b := f.Backlog
	line := fmt.Sprintf("backlog  %d ready · %d need triage · %d blocked", b.Ready, b.NeedsTriage, len(b.Blocked))
	if len(b.Blocked) > 0 {
		ids := make([]string, 0, len(b.Blocked))
		for _, n := range b.Blocked {
			ids = append(ids, fmt.Sprintf("#%d", n))
		}
		line += " (" + strings.Join(ids, " ") + ")"
	}
	u.Println(u.Dim(line))
	if f.RetroQueue > 0 {
		noun := "notes"
		if f.RetroQueue == 1 {
			noun = "note"
		}
		u.Println(u.Dim(fmt.Sprintf("memory   %d retro %s queued for kiln-retro", f.RetroQueue, noun)))
	}
	if len(f.Stale) > 0 {
		u.Println(u.Dim("stale    " + strings.Join(f.Stale, ", ") + " (git worktree remove)"))
	}
}

func init() { rootCmd.AddCommand(statusCmd) }
