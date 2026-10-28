// Package install writes kiln into a repository (kiln init) and upgrades it (kiln update).
package install

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/pavelsimo/kiln/internal/assets"
	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/lock"
)

// SkillsDir is where skills are installed, relative to the repo root.
const SkillsDir = ".kiln/skills"

// AgentDirs maps an agent to the skills folder it reads.
var AgentDirs = map[string]string{
	"claude": ".claude/skills",
	"codex":  ".agents/skills",
	"cursor": ".cursor/skills",
}

// Labels are the 14 labels kiln init creates.
var Labels = []ghx.Label{
	{Name: "kiln:needs-triage", Color: "b48767", Description: "kiln: needs acceptance criteria, type and size"},
	{Name: "kiln:ready", Color: "d9a066", Description: "kiln: ready to plan or start"},
	{Name: "kiln:planned", Color: "e0503a", Description: "kiln: plan posted, waiting at gate 1"},
	{Name: "kiln:in-progress", Color: "ff6b2b", Description: "kiln: being built in its own worktree"},
	{Name: "kiln:review", Color: "ffae42", Description: "kiln: PR open, verifying and reviewing"},
	{Name: "kiln:blocked", Color: "b60205", Description: "kiln: blocked, see the latest comment"},
	{Name: "type:feature", Color: "7ccb9c", Description: "New behavior a user can see"},
	{Name: "type:bug", Color: "d73a4a", Description: "Something is wrong today"},
	{Name: "type:refactor", Color: "8fabbb", Description: "Same behavior, better shape"},
	{Name: "type:chore", Color: "cdbdad", Description: "Deps, config, cleanup"},
	{Name: "type:docs", Color: "0075ca", Description: "Documentation"},
	{Name: "size:S", Color: "c2e0c6", Description: "One area, about 100 lines, no public interface change"},
	{Name: "size:M", Color: "fbca04", Description: "Several modules, or a public interface change"},
	{Name: "size:L", Color: "e99695", Description: "Cross-cutting, migrations, security or data"},
}

// Action is one thing init or update did (or would do, in a dry run).
type Action struct {
	Kind   string `json:"kind"` // create, replace, skip, keep, conflict, link, unlink, label, remove
	Path   string `json:"path"`
	Detail string `json:"detail,omitempty"`
}

// Report summarizes a run.
type Report struct {
	Root    string   `json:"root"`
	DryRun  bool     `json:"dry_run"`
	Agents  []string `json:"agents"`
	Skills  int      `json:"skills"`
	Labels  int      `json:"labels"`
	Actions []Action `json:"actions"`
}

func (r *Report) add(kind, path, detail string) {
	r.Actions = append(r.Actions, Action{Kind: kind, Path: path, Detail: detail})
}

// Count returns how many actions of kind happened.
func (r *Report) Count(kinds ...string) int {
	n := 0
	for _, a := range r.Actions {
		if slices.Contains(kinds, a.Kind) {
			n++
		}
	}
	return n
}

// Options control Init and Update.
type Options struct {
	Root    string
	Agents  []string // explicit --agents; empty means detect
	Force   bool
	DryRun  bool
	Version string
	GH      *ghx.GH // nil skips GitHub (labels)
}

type writer struct {
	root   string
	dryRun bool
	rep    *Report
}

func (w writer) abs(rel string) string { return filepath.Join(w.root, filepath.FromSlash(rel)) }

func (w writer) exists(rel string) bool {
	_, err := os.Lstat(w.abs(rel))
	return err == nil
}

func (w writer) write(kind, rel string, data []byte) error {
	w.rep.add(kind, rel, "")
	if w.dryRun {
		return nil
	}
	p := w.abs(rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// writeIfAbsent writes a file unless it exists (or force is set).
func (w writer) writeIfAbsent(rel string, data []byte, force bool) error {
	if w.exists(rel) {
		if !force {
			w.rep.add("skip", rel, "exists")
			return nil
		}
		cur, _ := os.ReadFile(w.abs(rel))
		if bytes.Equal(cur, data) {
			w.rep.add("keep", rel, "")
			return nil
		}
		return w.write("replace", rel, data)
	}
	return w.write("create", rel, data)
}

// DetectAgents guesses which agents a repo uses. With no signal, it returns all of them.
func DetectAgents(root string) []string {
	has := func(p string) bool { _, err := os.Stat(filepath.Join(root, p)); return err == nil }
	var out []string
	if has(".claude") || has("CLAUDE.md") {
		out = append(out, "claude")
	}
	if has(".agents") || has("AGENTS.md") || has(".codex") {
		out = append(out, "codex")
	}
	if has(".cursor") || has(".cursorrules") {
		out = append(out, "cursor")
	}
	if len(out) == 0 {
		return slices.Clone(config.AgentNames)
	}
	return out
}

func resolveAgents(opts Options) ([]string, error) {
	if len(opts.Agents) > 0 {
		for _, a := range opts.Agents {
			if _, ok := AgentDirs[a]; !ok {
				return nil, fmt.Errorf("unknown agent %q (want claude, codex or cursor)", a)
			}
		}
		return opts.Agents, nil
	}
	if cfg, err := config.Load(opts.Root); err == nil && len(cfg.Agents) > 0 {
		return cfg.Agents, nil
	}
	return DetectAgents(opts.Root), nil
}

// Init installs kiln into opts.Root.
func Init(ctx context.Context, opts Options) (Report, error) {
	rep := Report{Root: opts.Root, DryRun: opts.DryRun}
	w := writer{root: opts.Root, dryRun: opts.DryRun, rep: &rep}
	agents, err := resolveAgents(opts)
	if err != nil {
		return rep, err
	}
	rep.Agents = agents

	lk, _ := lock.Load(opts.Root)
	for _, name := range sortedKeys(assets.SkillFiles()) {
		data := assets.SkillFiles()[name]
		rel := SkillsDir + "/" + name
		key := "skills/" + name
		if w.exists(rel) && !opts.Force {
			rep.add("skip", rel, "exists")
			continue
		}
		if err := w.writeIfAbsent(rel, data, opts.Force); err != nil {
			return rep, err
		}
		lk.Files[key] = lock.Hash(data)
	}
	rep.Skills = len(assets.SkillNames())

	// config, memory and verify are yours: written once, never overwritten
	cfg := assets.Template("kiln/config.yaml")
	cfg = setAgentsLine(cfg, agents)
	for rel, data := range assets.Templates("kiln") {
		if rel == "config.yaml" {
			data = cfg
		}
		if err := w.writeIfAbsent(".kiln/"+rel, data, false); err != nil {
			return rep, err
		}
	}

	if err := linkAll(w, agents, assets.SkillNames(), opts.Force); err != nil {
		return rep, err
	}
	if err := pointer(w, agents); err != nil {
		return rep, err
	}

	conf := config.Default()
	if c, err := config.Load(opts.Root); err == nil {
		conf = c
	}
	if conf.GitHub.Templates {
		if !hasPRTemplate(opts.Root) || opts.Force {
			if err := w.writeIfAbsent(".github/pull_request_template.md", assets.Template("github/pull_request_template.md"), opts.Force); err != nil {
				return rep, err
			}
		} else {
			rep.add("skip", ".github/pull_request_template.md", "a PR template already exists")
		}
		if err := w.writeIfAbsent(".github/ISSUE_TEMPLATE/kiln-task.yml", assets.Template("github/ISSUE_TEMPLATE/kiln-task.yml"), opts.Force); err != nil {
			return rep, err
		}
	}
	if conf.GitHub.CI {
		if err := w.writeIfAbsent(".github/workflows/ci.yml", assets.Template("github/workflows/ci.yml"), opts.Force); err != nil {
			return rep, err
		}
	}
	if opts.GH != nil && conf.GitHub.Labels {
		for _, l := range Labels {
			rep.add("label", l.Name, "")
			if opts.DryRun {
				continue
			}
			if err := opts.GH.CreateLabel(ctx, l); err != nil {
				return rep, fmt.Errorf("creating label %s: %w", l.Name, err)
			}
		}
		rep.Labels = len(Labels)
	}

	lk.Version = opts.Version
	if !opts.DryRun {
		if err := os.MkdirAll(filepath.Join(opts.Root, ".kiln"), 0o755); err != nil {
			return rep, err
		}
		if err := lk.Save(opts.Root); err != nil {
			return rep, err
		}
	}
	return rep, nil
}

func hasPRTemplate(root string) bool {
	for _, p := range []string{
		".github/pull_request_template.md", ".github/PULL_REQUEST_TEMPLATE.md", ".github/PULL_REQUEST_TEMPLATE",
		"pull_request_template.md", "PULL_REQUEST_TEMPLATE.md", "docs/pull_request_template.md", "docs/PULL_REQUEST_TEMPLATE.md",
	} {
		if _, err := os.Stat(filepath.Join(root, p)); err == nil {
			return true
		}
	}
	return false
}

func setAgentsLine(cfg []byte, agents []string) []byte {
	lines := strings.Split(string(cfg), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "agents:") {
			lines[i] = "agents: [" + strings.Join(agents, ", ") + "]"
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// linkTarget is the relative symlink target from an agent skills folder to a kiln skill.
func linkTarget(agentDir, skill string) string {
	depth := strings.Count(agentDir, "/") + 1
	return strings.Repeat("../", depth) + SkillsDir + "/" + skill
}

func linkAll(w writer, agents, skills []string, force bool) error {
	for _, a := range agents {
		dir := AgentDirs[a]
		for _, s := range skills {
			if err := link(w, dir, s, force); err != nil {
				return err
			}
		}
	}
	return nil
}

func link(w writer, agentDir, skill string, force bool) error {
	rel := agentDir + "/" + skill
	target := linkTarget(agentDir, skill)
	p := w.abs(rel)
	fi, err := os.Lstat(p)
	switch {
	case err == nil && fi.Mode()&os.ModeSymlink != 0:
		cur, _ := os.Readlink(p)
		if filepath.ToSlash(cur) == target {
			w.rep.add("keep", rel, "")
			return nil
		}
		if !force {
			w.rep.add("skip", rel, "links somewhere else ("+cur+"); pass --force to relink")
			return nil
		}
		if !w.dryRun {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	case err == nil:
		if isKilnCopy(w.abs(SkillsDir+"/"+skill), p) {
			w.rep.add("keep", rel, "copy")
			return nil
		}
		w.rep.add("skip", rel, "a folder you own is already there")
		return nil
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	w.rep.add("link", rel, target)
	if w.dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(filepath.FromSlash(target), p); err != nil {
		// Windows without symlink rights: fall back to a copy
		return copyDir(w.abs(SkillsDir+"/"+skill), p)
	}
	return nil
}

// isKilnCopy reports whether dst is an identical copy of src (the Windows fallback).
func isKilnCopy(src, dst string) bool {
	a, errA := os.ReadFile(filepath.Join(src, "SKILL.md"))
	b, errB := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
}

// Pointer block markers.
const (
	MarkStart = "<!-- kiln:start -->"
	MarkEnd   = "<!-- kiln:end -->"
)

// UpsertBlock replaces the marked block in doc, or appends it.
func UpsertBlock(doc, block string) string {
	block = strings.TrimSpace(block)
	i := strings.Index(doc, MarkStart)
	j := strings.Index(doc, MarkEnd)
	if i >= 0 && j > i {
		return doc[:i] + block + doc[j+len(MarkEnd):]
	}
	if strings.TrimSpace(doc) == "" {
		return block + "\n"
	}
	return strings.TrimRight(doc, "\n") + "\n\n" + block + "\n"
}

func pointer(w writer, agents []string) error {
	targets := []string{"AGENTS.md"}
	if slices.Contains(agents, "claude") {
		targets = append(targets, "CLAUDE.md")
	}
	block := string(assets.Template("pointer.md"))
	seen := map[string]bool{}
	for _, t := range targets {
		p := w.abs(t)
		real := p
		if r, err := filepath.EvalSymlinks(p); err == nil {
			real = r
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		cur, err := os.ReadFile(p)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		next := UpsertBlock(string(cur), block)
		if string(cur) == next {
			w.rep.add("keep", t, "pointer block")
			continue
		}
		kind := "replace"
		if cur == nil {
			kind = "create"
		}
		w.rep.add(kind, t, "pointer block")
		if !w.dryRun {
			if err := os.WriteFile(real, []byte(next), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
