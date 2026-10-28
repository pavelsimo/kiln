// Package gitx is a thin wrapper over the git command line.
package gitx

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pavelsimo/kiln/internal/run"
)

// MemoryPathspec excludes kiln's memory folder, so journal commits never change a verdict.
const MemoryPathspec = ":(exclude).kiln/memory"

// Git runs git in Dir.
type Git struct {
	R   run.Runner
	Dir string
}

// New returns a Git for dir.
func New(r run.Runner, dir string) *Git { return &Git{R: r, Dir: dir} }

// Run runs git with args and returns trimmed stdout.
func (g *Git) Run(ctx context.Context, args ...string) (string, error) {
	out, err := g.R.Run(ctx, g.Dir, nil, "git", args...)
	return strings.TrimSpace(string(out)), err
}

// RunInput runs git with stdin.
func (g *Git) RunInput(ctx context.Context, stdin []byte, args ...string) (string, error) {
	out, err := g.R.Run(ctx, g.Dir, stdin, "git", args...)
	return strings.TrimSpace(string(out)), err
}

// TopLevel returns the root of the working tree, or an error outside a repo.
func (g *Git) TopLevel(ctx context.Context) (string, error) {
	out, err := g.Run(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", errors.New("not a git repository (run kiln inside your repo, or pass -C)")
	}
	return out, nil
}

// MainWorktree returns the path of the main worktree (the original clone).
func (g *Git) MainWorktree(ctx context.Context) (string, error) {
	common, err := g.Run(ctx, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	return filepath.Dir(common), nil
}

// RemoteURL returns the URL of a remote, or "" if it doesn't exist.
func (g *Git) RemoteURL(ctx context.Context, remote string) string {
	out, err := g.Run(ctx, "remote", "get-url", remote)
	if err != nil {
		return ""
	}
	return out
}

var ghRemote = regexp.MustCompile(`github\.com[:/]([^/]+)/([^/]+?)(\.git)?/?$`)

// ParseGitHubRemote extracts owner/name from a GitHub remote URL.
func ParseGitHubRemote(url string) (owner, name string, ok bool) {
	m := ghRemote.FindStringSubmatch(strings.TrimSpace(url))
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// DefaultBranch returns origin's default branch, falling back to "main".
func (g *Git) DefaultBranch(ctx context.Context) string {
	if out, err := g.Run(ctx, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(out, "origin/")
	}
	for _, b := range []string{"main", "master"} {
		if g.RefExists(ctx, "refs/heads/"+b) || g.RefExists(ctx, "refs/remotes/origin/"+b) {
			return b
		}
	}
	return "main"
}

// RefExists reports whether a ref resolves.
func (g *Git) RefExists(ctx context.Context, ref string) bool {
	_, err := g.Run(ctx, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	return err == nil
}

// BaseRef returns origin/<default> when it exists, else <default>.
func (g *Git) BaseRef(ctx context.Context) string {
	b := g.DefaultBranch(ctx)
	if g.RefExists(ctx, "refs/remotes/origin/"+b) {
		return "origin/" + b
	}
	return b
}

// RevParse resolves a revision to a full SHA.
func (g *Git) RevParse(ctx context.Context, rev string) (string, error) {
	return g.Run(ctx, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
}

// CurrentBranch returns the checked-out branch name, or "" when detached.
func (g *Git) CurrentBranch(ctx context.Context) string {
	out, err := g.Run(ctx, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// ChangedFiles lists files changed on HEAD since it forked from base.
func (g *Git) ChangedFiles(ctx context.Context, base, head string) ([]string, error) {
	out, err := g.Run(ctx, "diff", "--name-only", base+"..."+head)
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// SameTreeOutsideMemory reports whether a and b differ only inside .kiln/memory/.
func (g *Git) SameTreeOutsideMemory(ctx context.Context, a, b string) bool {
	_, err := g.Run(ctx, "diff", "--quiet", a, b, "--", ".", MemoryPathspec)
	return err == nil
}

// PatchID returns the stable patch-id of the change from merge-base(base, rev) to rev,
// ignoring .kiln/memory/. An empty change returns "".
func (g *Git) PatchID(ctx context.Context, base, rev string) (string, error) {
	mb, err := g.Run(ctx, "merge-base", base, rev)
	if err != nil {
		return "", err
	}
	diff, err := g.Run(ctx, "diff", "--no-color", mb, rev, "--", ".", MemoryPathspec)
	if err != nil {
		return "", err
	}
	if diff == "" {
		return "", nil
	}
	out, err := g.RunInput(ctx, []byte(diff+"\n"), "patch-id", "--stable")
	if err != nil {
		return "", err
	}
	f := strings.Fields(out)
	if len(f) == 0 {
		return "", nil
	}
	return f[0], nil
}

// Show returns the contents of path at rev.
func (g *Git) Show(ctx context.Context, rev, path string) (string, error) {
	out, err := g.R.Run(ctx, g.Dir, nil, "git", "show", rev+":"+path)
	return string(out), err
}

// ListTree lists file names directly under dir at rev.
func (g *Git) ListTree(ctx context.Context, rev, dir string) ([]string, error) {
	out, err := g.Run(ctx, "ls-tree", "--name-only", rev, strings.TrimSuffix(dir, "/")+"/")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// Tags returns tags matching pattern, highest version first.
func (g *Git) Tags(ctx context.Context, pattern string) ([]string, error) {
	out, err := g.Run(ctx, "tag", "--list", pattern, "--sort=-v:refname")
	if err != nil {
		return nil, err
	}
	return lines(out), nil
}

// RemoteTagExists reports whether origin has the tag. Without a remote it returns false.
func (g *Git) RemoteTagExists(ctx context.Context, tag string) bool {
	if g.RemoteURL(ctx, "origin") == "" {
		return false
	}
	out, err := g.Run(ctx, "ls-remote", "--tags", "origin", "refs/tags/"+tag)
	return err == nil && out != ""
}

// Commit is one entry from Log.
type Commit struct {
	SHA     string
	Subject string
	Body    string
}

// Log returns first-parent commits in rangeSpec, oldest first.
func (g *Git) Log(ctx context.Context, rangeSpec string) ([]Commit, error) {
	out, err := g.Run(ctx, "log", "--first-parent", "--reverse", "--format=%H%x1f%s%x1f%b%x1e", rangeSpec)
	if err != nil {
		return nil, err
	}
	var cs []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimLeft(rec, "\n")
		if rec == "" {
			continue
		}
		p := strings.SplitN(rec, "\x1f", 3)
		if len(p) < 2 {
			continue
		}
		c := Commit{SHA: p[0], Subject: p[1]}
		if len(p) == 3 {
			c.Body = strings.TrimSpace(p[2])
		}
		cs = append(cs, c)
	}
	return cs, nil
}

// LastCommitTouching returns the newest commit that changed any of paths, or "".
func (g *Git) LastCommitTouching(ctx context.Context, paths ...string) string {
	args := append([]string{"log", "-1", "--format=%H", "--"}, paths...)
	out, err := g.Run(ctx, args...)
	if err != nil {
		return ""
	}
	return out
}

// CountCommitsSince counts commits after since that touched any glob.
func (g *Git) CountCommitsSince(ctx context.Context, since string, globs []string) (int, error) {
	args := make([]string, 0, 4+len(globs))
	args = append(args, "rev-list", "--count", since+"..HEAD", "--")
	for _, gl := range globs {
		args = append(args, ":(glob)"+gl)
	}
	out, err := g.Run(ctx, args...)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(out)
}

// Worktree is one entry of `git worktree list`.
type Worktree struct {
	Path      string `json:"path"`
	Head      string `json:"head"`
	Branch    string `json:"branch"`
	Prunable  bool   `json:"prunable"`
	Detached  bool   `json:"detached"`
	IsPrimary bool   `json:"primary"`
}

// Worktrees lists worktrees.
func (g *Git) Worktrees(ctx context.Context) ([]Worktree, error) {
	out, err := g.Run(ctx, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var wts []Worktree
	var cur *Worktree
	for _, l := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(l, "worktree "):
			wts = append(wts, Worktree{Path: strings.TrimPrefix(l, "worktree ")})
			cur = &wts[len(wts)-1]
			cur.IsPrimary = len(wts) == 1
		case cur == nil:
		case strings.HasPrefix(l, "HEAD "):
			cur.Head = strings.TrimPrefix(l, "HEAD ")
		case strings.HasPrefix(l, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(l, "branch "), "refs/heads/")
		case l == "detached":
			cur.Detached = true
		case strings.HasPrefix(l, "prunable"):
			cur.Prunable = true
		}
	}
	return wts, nil
}

// IsClean reports whether the working tree at dir has no changes.
func (g *Git) IsClean(ctx context.Context, dir string) (bool, error) {
	out, err := g.R.Run(ctx, dir, nil, "git", "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "", nil
}

// BranchIssue extracts the issue number from a kiln/<n>-<slug> branch.
func BranchIssue(branch string) (int, bool) {
	rest, ok := strings.CutPrefix(branch, "kiln/")
	if !ok {
		return 0, false
	}
	num, _, _ := strings.Cut(rest, "-")
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// IsReleaseBranch reports whether branch is a kiln/release-v* branch.
func IsReleaseBranch(branch string) bool {
	return strings.HasPrefix(branch, "kiln/release-v")
}

// IsHousekeepingBranch reports whether branch is a kiln branch with no issue behind it
// (kiln/release-v1.4.0, kiln/retro-2026-10-28, kiln/onboard, kiln/verifier-2026-10-28).
func IsHousekeepingBranch(branch string) bool {
	if !strings.HasPrefix(branch, "kiln/") {
		return false
	}
	_, isTask := BranchIssue(branch)
	return !isTask
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Short abbreviates a SHA for display.
func Short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
