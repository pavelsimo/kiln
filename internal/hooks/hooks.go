// Package hooks runs your own shell commands after kiln move and kiln merge.
// Hooks are personal: they live in your user config and in an untracked file
// in the repo, never in the committed .kiln/config.yaml.
package hooks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/pavelsimo/kiln/internal/state"
)

// LocalPath is the per-repo hook file, relative to the main checkout. kiln init gitignores it,
// so task worktrees don't have their own copy.
const LocalPath = ".kiln/hooks.local.yaml"

// DefaultTimeout is how long a hook may run when it sets no timeout.
const DefaultTimeout = 10 * time.Second

// Events a hook can listen to.
const (
	Move  = "move"
	Merge = "merge"
)

// Events lists every event, in the order kiln documents them.
var Events = []string{Move, Merge}

// Hook is one command to run after an event.
type Hook struct {
	On      string        `yaml:"on"`
	From    string        `yaml:"from,omitempty"`
	To      string        `yaml:"to,omitempty"`
	Run     string        `yaml:"run"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
}

// File is one hook file that was found and parsed.
type File struct {
	Path  string
	Hooks []Hook
}

// Result is what happened when a hook ran.
type Result struct {
	Run   string `json:"run"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Disabled reports whether KILN_NO_HOOKS turns hooks off.
func Disabled() bool { return os.Getenv("KILN_NO_HOOKS") != "" }

// Paths returns where hook files are looked for, in the order they run:
// $XDG_CONFIG_HOME/kiln/hooks.yaml (or ~/.config/kiln/hooks.yaml), then the local file in main,
// the main checkout's root.
func Paths(main string) []string {
	var ps []string
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, ".config")
		}
	}
	if dir != "" {
		ps = append(ps, filepath.Join(dir, "kiln", "hooks.yaml"))
	}
	return append(ps, filepath.Join(main, LocalPath))
}

// Load reads every hook file that exists. A file that can't be read or parsed is
// left out and reported as an error naming it; the others still load.
func Load(main string) ([]File, []error) {
	var files []File
	var errs []error
	for _, p := range Paths(main) {
		b, err := os.ReadFile(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		f, err := parse(p, b)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		files = append(files, f)
	}
	return files, errs
}

func parse(path string, b []byte) (File, error) {
	var doc struct {
		Hooks []Hook `yaml:"hooks"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil && !errors.Is(err, io.EOF) {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return File{}, fmt.Errorf("%s: more than one YAML document", path)
	}
	var bad []string
	for i, h := range doc.Hooks {
		at := fmt.Sprintf("hooks[%d].", i)
		if !slices.Contains(Events, h.On) {
			bad = append(bad, fmt.Sprintf("%son: %q is not one of %s", at, h.On, strings.Join(Events, ", ")))
		}
		if strings.TrimSpace(h.Run) == "" {
			bad = append(bad, at+"run: missing")
		}
		for _, f := range []struct{ name, value string }{{"from", h.From}, {"to", h.To}} {
			if f.value != "" && !state.Valid(f.value) {
				bad = append(bad, fmt.Sprintf("%s%s: %q is not a kiln state", at, f.name, f.value))
			}
		}
		if h.On != Move && (h.From != "" || h.To != "") {
			bad = append(bad, at+"from and to only apply to move")
		}
		if h.Timeout < 0 {
			bad = append(bad, at+"timeout: can't be negative")
		}
	}
	if len(bad) > 0 {
		return File{}, fmt.Errorf("%s: %s", path, strings.Join(bad, "; "))
	}
	return File{Path: path, Hooks: doc.Hooks}, nil
}

// Matching returns the hooks that fire for an event, in file order.
// from and to are the move's states; empty filters match anything.
func Matching(files []File, event, from, to string) []Hook {
	var hs []Hook
	for _, f := range files {
		for _, h := range f.Hooks {
			if h.On == event && (h.From == "" || h.From == from) && (h.To == "" || h.To == to) {
				hs = append(hs, h)
			}
		}
	}
	return hs
}

// Run runs hooks one at a time with sh -c from dir. Each gets the parent's environment
// plus env, and stdin on its stdin. Its stdout and stderr both go to out, so kiln's
// stdout stays data. A hook that fails or times out is reported, never returned as an error.
func Run(ctx context.Context, hs []Hook, env map[string]string, stdin []byte, dir string, out io.Writer) []Result {
	vars := os.Environ()
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vars = append(vars, k+"="+env[k])
	}
	res := make([]Result, 0, len(hs))
	for _, h := range hs {
		r := Result{Run: h.Run, OK: true}
		if err := runOne(ctx, h, vars, stdin, dir, out); err != nil {
			r.OK, r.Error = false, err.Error()
		}
		res = append(res, r)
	}
	return res
}

func runOne(ctx context.Context, h Hook, vars []string, stdin []byte, dir string, out io.Writer) error {
	limit := h.Timeout
	if limit == 0 {
		limit = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", h.Run)
	cmd.Dir = dir
	cmd.Env = vars
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout = out
	cmd.Stderr = out
	ownGroup(cmd)
	// a hook that starts a background process mustn't hold kiln open through the pipes
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("timed out after %s", limit)
	}
	return err
}
