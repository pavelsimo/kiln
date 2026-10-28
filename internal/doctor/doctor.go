// Package doctor checks that a kiln install is healthy.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pavelsimo/kiln/internal/assets"
	"github.com/pavelsimo/kiln/internal/checks"
	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/install"
	"github.com/pavelsimo/kiln/internal/lock"
)

// Levels for a finding.
const (
	OK   = "ok"
	Warn = "warn"
	Fail = "fail"
)

// Finding is one check result.
type Finding struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// Driver describes the verify driver's freshness.
type Driver struct {
	Exists   bool           `json:"exists"`
	Features int            `json:"features"`
	Behind   map[string]int `json:"behind"` // feature → commits touching its sources since the driver changed
}

// Report is the full doctor output.
type Report struct {
	Findings []Finding `json:"findings"`
	Driver   Driver    `json:"driver"`
	Healthy  bool      `json:"healthy"`
}

func (r *Report) add(name, level, detail, fix string) {
	r.Findings = append(r.Findings, Finding{Name: name, Level: level, Detail: detail, Fix: fix})
}

// Options control Run.
type Options struct {
	Root    string
	Version string
	GH      *ghx.GH // nil skips GitHub checks
	Git     *gitx.Git
}

// Run performs every check. It never stops early.
func Run(ctx context.Context, opts Options) Report {
	var r Report
	g := opts.Git
	root := opts.Root

	r.add("git", OK, "git repo "+filepath.Base(root), "")
	remote := g.RemoteURL(ctx, "origin")
	if _, _, ok := gitx.ParseGitHubRemote(remote); ok {
		r.add("remote", OK, "origin is "+remote, "")
	} else {
		r.add("remote", Fail, "no GitHub remote named origin", "git remote add origin git@github.com:<owner>/<repo>.git")
	}

	if opts.GH != nil {
		if login, err := opts.GH.Login(ctx); err == nil {
			r.add("gh", OK, "gh authenticated as "+login, "")
			checkLabels(ctx, &r, opts.GH)
		} else {
			r.add("gh", Fail, "gh is not authenticated", "gh auth login")
		}
	}

	cfg, err := config.Load(root)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		r.add("config", Fail, config.Path+" not found", "kiln init")
	case err != nil:
		r.add("config", Fail, err.Error(), "fix "+config.Path)
	default:
		if errs := cfg.Validate(); len(errs) > 0 {
			msgs := make([]string, 0, len(errs))
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			r.add("config", Fail, strings.Join(msgs, "; "), "fix "+config.Path)
		} else {
			r.add("config", OK, fmt.Sprintf("config valid · autonomy %s", cfg.Autonomy), "")
		}
	}

	checkSkills(&r, root, opts.Version)
	checkLinks(&r, root, cfg.Agents)
	checkWorktrees(ctx, &r, g, root, cfg)
	r.Driver = checkDriver(ctx, &r, g, filepath.Join(root, cfg.Verify.Dir), cfg.Verify.Dir)

	r.Healthy = true
	for _, f := range r.Findings {
		if f.Level == Fail {
			r.Healthy = false
		}
	}
	return r
}

func checkLabels(ctx context.Context, r *Report, gh *ghx.GH) {
	ls, err := gh.Labels(ctx)
	if err != nil {
		r.add("labels", Fail, "couldn't list labels: "+err.Error(), "")
		return
	}
	have := map[string]bool{}
	for _, l := range ls {
		have[l.Name] = true
	}
	var missing []string
	for _, l := range install.Labels {
		if !have[l.Name] {
			missing = append(missing, l.Name)
		}
	}
	if len(missing) > 0 {
		r.add("labels", Fail, fmt.Sprintf("%d labels missing: %s", len(missing), strings.Join(missing, ", ")), "kiln init")
		return
	}
	r.add("labels", OK, fmt.Sprintf("%d kiln labels present", len(install.Labels)), "")
}

func checkSkills(r *Report, root, version string) {
	lk, err := lock.Load(root)
	if err != nil {
		r.add("skills", Fail, lock.Path+" not found", "kiln init")
		return
	}
	var missing, edited, pending []string
	for key, h := range lk.Files {
		p := filepath.Join(root, ".kiln", filepath.FromSlash(key))
		cur, err := lock.HashFile(p)
		switch {
		case err != nil:
			missing = append(missing, key)
		case cur != h:
			edited = append(edited, key)
		}
		if _, err := os.Stat(p + install.NewSuffix); err == nil {
			pending = append(pending, key+install.NewSuffix)
		}
	}
	sort.Strings(missing)
	sort.Strings(edited)
	sort.Strings(pending)
	n := len(assets.SkillNames())
	switch {
	case len(missing) > 0:
		r.add("skills", Fail, fmt.Sprintf("%d installed files are missing: %s", len(missing), strings.Join(first(missing, 3), ", ")), "kiln update")
	case len(edited) > 0:
		noun := "files"
		if len(edited) == 1 {
			noun = "file"
		}
		r.add("skills", OK, fmt.Sprintf("%d skills · %d %s edited by you (kiln update keeps them)", n, len(edited), noun), "")
	default:
		r.add("skills", OK, fmt.Sprintf("%d skills, all match kiln.lock", n), "")
	}
	if len(pending) > 0 {
		r.add("updates", Warn, fmt.Sprintf("%d .kiln-new files to merge: %s", len(pending), strings.Join(first(pending, 3), ", ")), "merge each into the file next to it, then delete the .kiln-new")
	}
	if lk.Version != "" && version != "" && version != "dev" && lk.Version != version {
		r.add("version", Warn, fmt.Sprintf("skills installed by kiln %s, this binary is %s", lk.Version, version), "kiln update")
	}
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return append(xs[:n:n], fmt.Sprintf("and %d more", len(xs)-n))
	}
	return xs
}

func checkLinks(r *Report, root string, agents []string) {
	skills := assets.SkillNames()
	for _, a := range agents {
		dir := install.AgentDirs[a]
		var broken []string
		copies := 0
		for _, s := range skills {
			p := filepath.Join(root, dir, s)
			fi, err := os.Lstat(p)
			if err != nil {
				broken = append(broken, s+" (missing)")
				continue
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				if _, err := os.Stat(filepath.Join(p, "SKILL.md")); err != nil {
					broken = append(broken, s+" (dangling)")
				}
				continue
			}
			copies++
			src, _ := os.ReadFile(filepath.Join(root, install.SkillsDir, s, "SKILL.md"))
			dst, _ := os.ReadFile(filepath.Join(p, "SKILL.md"))
			if string(src) != string(dst) {
				broken = append(broken, s+" (copy drifted)")
			}
		}
		switch {
		case len(broken) > 0:
			r.add("links:"+a, Fail, fmt.Sprintf("%s: %s", dir, strings.Join(first(broken, 4), ", ")), "kiln init")
		case copies > 0:
			r.add("links:"+a, OK, fmt.Sprintf("%s: %d skills (%d copies, in sync)", dir, len(skills), copies), "")
		default:
			r.add("links:"+a, OK, fmt.Sprintf("%s: %d skills linked", dir, len(skills)), "")
		}
	}
}

func checkWorktrees(ctx context.Context, r *Report, g *gitx.Git, root string, cfg config.Config) {
	dir := cfg.WorktreeDir(root)
	parent := dir
	if _, err := os.Stat(dir); err != nil {
		parent = filepath.Dir(dir)
	}
	probe, err := os.CreateTemp(parent, ".kiln-probe-*")
	if err != nil {
		r.add("worktrees", Fail, parent+" isn't writable", "check permissions, or change worktrees.dir")
		return
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())
	wts, err := g.Worktrees(ctx)
	if err != nil {
		r.add("worktrees", Warn, "couldn't list worktrees: "+err.Error(), "")
		return
	}
	var stale []string
	active := 0
	for _, w := range wts {
		if w.IsPrimary {
			continue
		}
		if w.Prunable {
			stale = append(stale, w.Path)
			continue
		}
		if _, ok := gitx.BranchIssue(w.Branch); ok {
			active++
		}
	}
	if len(stale) > 0 {
		r.add("worktrees", Warn, fmt.Sprintf("%d stale worktrees: %s", len(stale), strings.Join(stale, ", ")), "git worktree prune")
		return
	}
	shown := dir
	if rel, err := filepath.Rel(root, dir); err == nil {
		shown = rel
	}
	r.add("worktrees", OK, fmt.Sprintf("worktrees go in %s · %d in use", shown, active), "")
}

type featureFront struct {
	Feature string   `yaml:"feature"`
	Sources []string `yaml:"sources"`
}

func checkDriver(ctx context.Context, r *Report, g *gitx.Git, verifyDir, verifyRel string) Driver {
	d := Driver{Behind: map[string]int{}}
	if _, err := checks.Load(verifyDir); err != nil {
		r.add("checks", Fail, err.Error(), "fix "+filepath.Join(verifyRel, checks.File))
	}
	if _, err := os.Stat(filepath.Join(verifyDir, "DRIVER.md")); err != nil {
		r.add("driver", Warn, "no verify driver yet; UI and API checks will be INCONCLUSIVE", "run /kiln-verifier create (or /kiln-onboard)")
		return d
	}
	d.Exists = true
	since := g.LastCommitTouching(ctx, filepath.ToSlash(filepath.Join(verifyRel, "DRIVER.md")), filepath.ToSlash(filepath.Join(verifyRel, "features")))
	entries, _ := os.ReadDir(filepath.Join(verifyDir, "features"))
	total := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(verifyDir, "features", e.Name()))
		if err != nil {
			continue
		}
		var ff featureFront
		if rest, ok := strings.CutPrefix(string(b), "---\n"); ok {
			if front, _, found := strings.Cut(rest, "\n---"); found {
				_ = yaml.Unmarshal([]byte(front), &ff)
			}
		}
		d.Features++
		if ff.Feature == "" {
			ff.Feature = strings.TrimSuffix(e.Name(), ".md")
		}
		if since == "" || len(ff.Sources) == 0 {
			continue
		}
		if n, err := g.CountCommitsSince(ctx, since, ff.Sources); err == nil && n > 0 {
			d.Behind[ff.Feature] = n
			total += n
		}
	}
	switch {
	case since == "":
		r.add("driver", Warn, fmt.Sprintf("DRIVER.md and %d features aren't committed yet", d.Features), "commit .kiln/verify/")
	case total > 0:
		var parts []string
		for f, n := range d.Behind {
			parts = append(parts, fmt.Sprintf("%s +%d", f, n))
		}
		sort.Strings(parts)
		r.add("driver", Warn, fmt.Sprintf("driver is behind: %d commits touched feature sources (%s)", total, strings.Join(parts, ", ")), "kiln-retro runs kiln-verifier maintain")
	default:
		r.add("driver", OK, fmt.Sprintf("driver current · %d features", d.Features), "")
	}
	return d
}
