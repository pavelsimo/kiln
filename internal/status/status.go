// Package status builds the factory floor: what's in flight and what's waiting on you.
package status

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/pavelsimo/kiln/internal/config"
	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/journal"
	"github.com/pavelsimo/kiln/internal/state"
)

// PRInfo summarizes a task's pull request.
type PRInfo struct {
	Number    int              `json:"number"`
	Checks    ghx.CheckSummary `json:"checks"`
	Mergeable string           `json:"mergeable"`
	Review    string           `json:"review_decision,omitempty"`
}

// Task is one issue on the floor.
type Task struct {
	Issue    int     `json:"issue"`
	Title    string  `json:"title"`
	Size     string  `json:"size"`
	Type     string  `json:"type"`
	State    string  `json:"state"`
	Stage    string  `json:"stage"`
	PR       *PRInfo `json:"pr"`
	Worktree string  `json:"worktree,omitempty"`
	Waiting  string  `json:"waiting"` // "", "plan" or "merge"
	Next     string  `json:"next"`
}

// Backlog counts issues that haven't started.
type Backlog struct {
	Ready       int   `json:"ready"`
	NeedsTriage int   `json:"needs_triage"`
	Blocked     []int `json:"blocked"`
}

// Floor is the full status report.
type Floor struct {
	Repo        string   `json:"repo"`
	Autonomy    string   `json:"autonomy"`
	MaxInFlight int      `json:"max_in_flight"`
	InFlight    []Task   `json:"in_flight"`
	Backlog     Backlog  `json:"backlog"`
	RetroQueue  int      `json:"retro_queue"`
	LastRetro   string   `json:"last_retro"`
	Stale       []string `json:"stale_worktrees"`
}

// WaitingOnYou counts tasks stopped at a gate.
func (f Floor) WaitingOnYou() int {
	n := 0
	for _, t := range f.InFlight {
		if t.Waiting != "" {
			n++
		}
	}
	return n
}

func labelValue(labels []string, prefix string) string {
	for _, l := range labels {
		if v, ok := strings.CutPrefix(l, prefix); ok {
			return v
		}
	}
	return ""
}

// Build reads GitHub, worktrees and journals.
func Build(ctx context.Context, gh *ghx.GH, g *gitx.Git, root, repo string, cfg config.Config) (Floor, error) {
	f := Floor{Repo: repo, Autonomy: cfg.Autonomy, MaxInFlight: cfg.Worktrees.MaxInFlight, InFlight: []Task{}, Backlog: Backlog{Blocked: []int{}}}
	issues, err := gh.OpenIssues(ctx)
	if err != nil {
		return f, err
	}
	prs, err := gh.OpenPRs(ctx)
	if err != nil {
		return f, err
	}
	wts, _ := g.Worktrees(ctx)

	prByIssue := map[int]ghx.PR{}
	for _, p := range prs {
		if n, ok := gitx.BranchIssue(p.HeadRefName); ok {
			prByIssue[n] = p
		}
	}
	wtByIssue := map[int]string{}
	for _, w := range wts {
		if n, ok := gitx.BranchIssue(w.Branch); ok {
			wtByIssue[n] = relTo(root, w.Path)
		}
	}

	for _, is := range issues {
		labels := is.LabelNames()
		st, err := state.Current(labels)
		if err != nil || st == state.None {
			continue
		}
		size := labelValue(labels, "size:")
		switch st {
		case state.NeedsTriage:
			f.Backlog.NeedsTriage++
			continue
		case state.Ready:
			f.Backlog.Ready++
			continue
		case state.Blocked:
			f.Backlog.Blocked = append(f.Backlog.Blocked, is.Number)
			continue
		}
		t := Task{Issue: is.Number, Title: is.Title, Size: size, Type: labelValue(labels, "type:"), State: st, Stage: state.Stage(st), Worktree: wtByIssue[is.Number]}
		if p, ok := prByIssue[is.Number]; ok {
			s := ghx.Summarize(p.StatusCheckRollup)
			t.PR = &PRInfo{Number: p.Number, Checks: s, Mergeable: p.Mergeable, Review: p.ReviewDecision}
		}
		classify(&t, cfg)
		f.InFlight = append(f.InFlight, t)
	}
	sort.Slice(f.InFlight, func(i, j int) bool { return f.InFlight[i].Issue < f.InFlight[j].Issue })
	sort.Ints(f.Backlog.Blocked)

	f.LastRetro, f.RetroQueue = retroQueue(root)
	for _, w := range wts {
		if w.IsPrimary {
			continue
		}
		n, isTask := gitx.BranchIssue(w.Branch)
		if w.Prunable || (isTask && !issueOpen(issues, n)) {
			f.Stale = append(f.Stale, relTo(root, w.Path))
		}
	}
	return f, nil
}

func issueOpen(issues []ghx.Issue, n int) bool {
	for _, i := range issues {
		if i.Number == n {
			return true
		}
	}
	return false
}

func classify(t *Task, cfg config.Config) {
	size := t.Size
	if size == "" {
		size = "M"
	}
	switch t.State {
	case state.Planned:
		if cfg.GateActive("plan", size) {
			t.Waiting, t.Next = "plan", "◆1 approve the plan on #"+strconv.Itoa(t.Issue)
		} else {
			t.Next = "kiln-start #" + strconv.Itoa(t.Issue)
		}
	case state.InProgress:
		t.Next = "kiln-build in " + t.Worktree
		if t.Worktree == "" {
			t.Next = "kiln-resume #" + strconv.Itoa(t.Issue)
		}
	case state.Review:
		if t.PR == nil {
			t.Next = "kiln-pr #" + strconv.Itoa(t.Issue)
			return
		}
		c := t.PR.Checks
		green := c.Failing == 0 && c.Pending == 0 && strings.EqualFold(t.PR.Mergeable, "MERGEABLE")
		if green {
			t.Stage = "ship"
			if cfg.GateActive("merge", size) {
				t.Waiting, t.Next = "merge", "◆2 kiln ready "+strconv.Itoa(t.PR.Number)+", then merge"
			} else {
				t.Next = "kiln-merge #" + strconv.Itoa(t.PR.Number)
			}
			return
		}
		t.Next = "kiln-watch #" + strconv.Itoa(t.PR.Number)
	}
}

func relTo(root, p string) string {
	if r, err := filepath.Rel(root, p); err == nil {
		return r
	}
	return p
}

// retroQueue counts retro notes in journals with activity after lessons.md's last_retro.
func retroQueue(root string) (string, int) {
	last := "never"
	var lastT time.Time
	if b, err := os.ReadFile(filepath.Join(root, ".kiln/memory/lessons.md")); err == nil {
		if rest, ok := strings.CutPrefix(string(b), "---\n"); ok {
			if front, _, found := strings.Cut(rest, "\n---"); found {
				var m struct {
					LastRetro string `yaml:"last_retro"`
				}
				if yaml.Unmarshal([]byte(front), &m) == nil && m.LastRetro != "" {
					last = m.LastRetro
				}
			}
		}
	}
	if t, err := time.Parse("2006-01-02", last); err == nil {
		lastT = t.Add(24 * time.Hour)
	}
	js, err := journal.InDir(root)
	if err != nil {
		return last, 0
	}
	n := 0
	for _, j := range js {
		if t, ok := j.LastActivity(); ok && !lastT.IsZero() && t.Before(lastT) {
			continue
		}
		n += len(j.RetroNotes)
	}
	return last, n
}
