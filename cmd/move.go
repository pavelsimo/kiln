package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/hooks"
	"github.com/pavelsimo/kiln/internal/state"
)

var moveFlags struct {
	reason string
	dryRun bool
}

type moveReport struct {
	Issue  int    `json:"issue"`
	From   string `json:"from"`
	To     string `json:"to"`
	Moved  bool   `json:"moved"`
	DryRun bool   `json:"dry_run"`

	Hooks []hooks.Result `json:"hooks,omitempty"`
}

var moveCmd = &cobra.Command{
	Use:   "move <issue> <state>",
	Short: "Move an issue to its next kiln:* state",
	Long: `Move an issue to its next kiln:* state, following the label state machine.

States: needs-triage, ready, planned, in-progress, review, blocked.

  (none)        → needs-triage | ready
  needs-triage  → ready
  ready         → planned | in-progress
  planned       → in-progress | ready
  in-progress   → review
  review        → in-progress
  any state     → blocked (needs --reason; kiln comments where it came from)
  blocked       → the state it came from

Anything else is refused with exit code 3.

After a real move, kiln runs your hooks for it (see kiln doctor for the files it reads).`,
	Example: `  kiln move 42 planned
  kiln move 42 blocked --reason "waiting on the API team's schema"
  kiln move 42 review`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, err := setup(cmd, false)
		if err != nil {
			return err
		}
		n, err := parseNumber(args[0], "issue")
		if err != nil {
			return err
		}
		to := strings.TrimPrefix(args[1], "kiln:")
		if !state.Valid(to) {
			return usage("unknown state %q (want one of %s)", to, strings.Join(state.All, ", "))
		}
		if to == state.Blocked && strings.TrimSpace(moveFlags.reason) == "" {
			return usage("blocking needs a reason: kiln move %d blocked --reason \"...\"", n)
		}
		is, err := e.gh.Issue(e.ctx, n)
		if err != nil {
			return err
		}
		if !strings.EqualFold(is.State, "OPEN") {
			return no(fmt.Errorf("#%d is %s", n, strings.ToLower(is.State)))
		}
		from, err := state.Current(is.LabelNames())
		if err != nil {
			return no(err)
		}
		blockedFrom := ""
		if from == state.Blocked {
			cs, err := e.gh.IssueComments(e.ctx, n)
			if err != nil {
				return err
			}
			bodies := make([]string, 0, len(cs))
			for _, c := range cs {
				bodies = append(bodies, c.Body)
			}
			blockedFrom = state.BlockedFrom(bodies)
		}
		if err := state.Check(from, to, blockedFrom); err != nil {
			return no(fmt.Errorf("#%d: %w", n, err))
		}
		rep := moveReport{Issue: n, From: from, To: to, DryRun: moveFlags.dryRun}
		if !moveFlags.dryRun {
			if to == state.Blocked {
				body := fmt.Sprintf("**Blocked** (was `%s`): %s\n\n%s", state.Label(from), moveFlags.reason, state.BlockedMarker(from))
				if err := e.gh.CommentIssue(e.ctx, n, body); err != nil {
					return err
				}
			}
			var remove []string
			if from != state.None {
				remove = []string{state.Label(from)}
			}
			if err := e.gh.EditLabels(e.ctx, n, []string{state.Label(to)}, remove); err != nil {
				return err
			}
			rep.Moved = true
			rep.Hooks = fireHooks(e, hookEvent{
				name: hooks.Move, from: from, to: to, issue: n, title: is.Title,
				env: map[string]string{"KILN_FROM": from, "KILN_TO": to, "KILN_REASON": moveFlags.reason},
			}, rep)
		}
		return e.out(rep, func() {
			f := "(no kiln label)"
			if from != state.None {
				f = state.Label(from)
			}
			verb := "moved"
			if moveFlags.dryRun {
				verb = "would move"
			}
			e.ui.OK("#%d %s %s → %s", n, verb, f, state.Label(to))
		})
	},
}

func init() {
	moveCmd.Flags().StringVar(&moveFlags.reason, "reason", "", "why the issue is blocked (required for blocked)")
	moveCmd.Flags().BoolVarP(&moveFlags.dryRun, "dry-run", "n", false, "check the move without changing labels")
	addNoHooksFlag(moveCmd)
	rootCmd.AddCommand(moveCmd)
}
