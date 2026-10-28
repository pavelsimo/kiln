// Package cmd wires kiln's commands.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/run"
	"github.com/pavelsimo/kiln/internal/ui"
)

// Exit codes.
const (
	ExitOK    = 0
	ExitError = 1
	ExitUsage = 2
	ExitNo    = 3 // the answer is no: not ready, stale, illegal move, problems found
)

// ExitErr carries an exit code out of a command.
type ExitErr struct {
	Code int
	Err  error
}

func (e *ExitErr) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitErr) Unwrap() error { return e.Err }

// no returns an exit-3 error. A nil err exits 3 without printing anything more.
func no(err error) error { return &ExitErr{Code: ExitNo, Err: err} }

func usage(format string, a ...any) error {
	return &ExitErr{Code: ExitUsage, Err: fmt.Errorf(format, a...)}
}

type globals struct {
	json    bool
	noColor bool
	dir     string
	verbose bool
}

var g globals

var rootCmd = &cobra.Command{
	Use:   "kiln",
	Short: "A software factory you install into your repo.",
	Long: `kiln installs a software factory into your repo: skills your coding agent follows,
plus exact answers over git and gh that every agent can rely on.

Setup:   kiln init · kiln update · kiln doctor
Facts:   kiln status · kiln checks · kiln verdict · kiln ready · kiln release plan
Writes:  kiln move · kiln merge

Docs: https://pavelsimo.github.io/kiln`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs kiln and returns the process exit code.
func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	rootCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &ExitErr{Code: ExitUsage, Err: err}
	})
	err := rootCmd.ExecuteContext(ctx)
	if err == nil {
		return ExitOK
	}
	code := ExitError
	var ee *ExitErr
	if errors.As(err, &ee) {
		code = ee.Code
	} else if isUsageError(err) {
		code = ExitUsage
	}
	if msg := err.Error(); msg != "" {
		fmt.Fprintln(os.Stderr, "kiln: "+msg)
		if code == ExitUsage {
			fmt.Fprintln(os.Stderr, "run 'kiln --help' for usage")
		}
	}
	return code
}

func isUsageError(err error) bool {
	msg := err.Error()
	for _, p := range []string{"unknown command", "unknown flag", "unknown shorthand", "accepts ", "requires at least", "requires exactly", "invalid argument"} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.BoolVar(&g.json, "json", false, "print machine-readable JSON on stdout")
	pf.BoolVar(&g.noColor, "no-color", false, "disable color (NO_COLOR is respected too)")
	pf.StringVarP(&g.dir, "dir", "C", "", "run as if kiln was started in this directory")
	pf.BoolVarP(&g.verbose, "verbose", "v", false, "print every git and gh command to stderr")
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("kiln {{.Version}}\n")
}

// env is what most commands need: the repo root, git, gh, config and output.
type env struct {
	ctx  context.Context
	root string
	git  *gitx.Git
	gh   *ghx.GH
	cfg  config.Config
	ui   *ui.UI
}

func (e *env) out(v any, human func()) error {
	if g.json {
		return e.ui.JSON(v)
	}
	human()
	return nil
}

func runner() run.Runner {
	var trace io.Writer
	if g.verbose {
		trace = os.Stderr
	}
	return run.Exec{Trace: trace}
}

// setup finds the repo root and loads config. needConfig makes a missing config an error.
func setup(cmd *cobra.Command, needConfig bool) (*env, error) {
	dir := g.dir
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	r := runner()
	gt := gitx.New(r, abs)
	root, err := gt.TopLevel(cmd.Context())
	if err != nil {
		return nil, err
	}
	e := &env{ctx: cmd.Context(), root: root, git: gitx.New(r, root), gh: ghx.New(r, root), ui: ui.New(g.noColor)}
	cfg, err := config.Load(root)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist) && !needConfig:
	case errors.Is(err, os.ErrNotExist):
		// worktrees share the main checkout's config when theirs isn't committed yet
		if main, merr := e.git.MainWorktree(e.ctx); merr == nil && main != root {
			if c, cerr := config.Load(main); cerr == nil {
				cfg, err = c, nil
			}
		}
		if err != nil {
			return nil, config.ErrMissing
		}
	default:
		return nil, err
	}
	e.cfg = cfg
	return e, nil
}

// repoName returns owner/name from the origin remote, falling back to gh.
func (e *env) repoName() (string, error) {
	if owner, name, ok := gitx.ParseGitHubRemote(e.git.RemoteURL(e.ctx, "origin")); ok {
		return owner + "/" + name, nil
	}
	r, err := e.gh.Repo(e.ctx)
	if err != nil {
		return "", fmt.Errorf("can't tell which GitHub repo this is (no github.com origin remote): %w", err)
	}
	return r.NameWithOwner, nil
}

func parseNumber(s, what string) (int, error) {
	s = strings.TrimPrefix(s, "#")
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 || fmt.Sprint(n) != s {
		return 0, usage("%s must be a number, got %q", what, s)
	}
	return n, nil
}

func itoa(n int) string { return fmt.Sprint(n) }
