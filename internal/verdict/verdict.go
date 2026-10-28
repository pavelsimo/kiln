// Package verdict decides whether a recorded verify or review result still holds for a commit.
package verdict

import (
	"context"
	"fmt"

	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/journal"
)

// Result is one skill's recorded outcome and whether it is still fresh.
type Result struct {
	Result string `json:"result"`
	SHA    string `json:"sha"`
	TS     string `json:"ts,omitempty"`
	Fresh  bool   `json:"fresh"`
	How    string `json:"how"`
}

// Report is the answer to "is this verdict still valid?".
type Report struct {
	Issue   int      `json:"issue"`
	Head    string   `json:"head"`
	Journal string   `json:"journal"`
	Verify  *Result  `json:"verify"`
	Review  *Result  `json:"review"`
	Only    string   `json:"only,omitempty"`
	Fresh   bool     `json:"fresh"`
	Reasons []string `json:"reasons,omitempty"`
}

// Passing results.
const (
	Verified     = "VERIFIED"
	NotVerified  = "NOT VERIFIED"
	Inconclusive = "INCONCLUSIVE"
	Pass         = "PASS"
	Fix          = "FIX"
)

// Freshness explains how a SHA relates to head: same-sha, same-tree, patch-id or stale.
func Freshness(ctx context.Context, g *gitx.Git, base, sha, head string) (bool, string) {
	full, err := g.RevParse(ctx, sha)
	if err != nil {
		return false, "unknown-sha"
	}
	if full == head {
		return true, "same-sha"
	}
	if g.SameTreeOutsideMemory(ctx, full, head) {
		return true, "same-tree"
	}
	a, errA := g.PatchID(ctx, base, full)
	b, errB := g.PatchID(ctx, base, head)
	if errA == nil && errB == nil && a != "" && a == b {
		return true, "patch-id"
	}
	return false, "stale"
}

// Options control Check.
type Options struct {
	Issue int
	Rev   string // the commit to judge; the journal is read from it too
	Base  string // the branch the change targets, for patch-id
	// AcceptInconclusive lets an INCONCLUSIVE verify count as passing.
	AcceptInconclusive bool
	// Only judges just "verify" or just "review".
	Only string
}

// Check loads the journal at opts.Rev and judges the latest kiln-verify and kiln-review rows.
func Check(ctx context.Context, g *gitx.Git, opts Options) (Report, error) {
	head, err := g.RevParse(ctx, opts.Rev)
	if err != nil {
		return Report{}, fmt.Errorf("can't resolve %s", opts.Rev)
	}
	rep := Report{Issue: opts.Issue, Head: head, Only: opts.Only}
	j, path, err := journal.AtRev(ctx, g, head, opts.Issue)
	if err != nil {
		return rep, err
	}
	rep.Journal = path
	judge := func(skill string) *Result {
		row, ok := j.Latest(skill)
		if !ok {
			return nil
		}
		r := &Result{Result: journal.Normalize(row.Result), SHA: row.Ref, TS: row.TS}
		r.Fresh, r.How = Freshness(ctx, g, opts.Base, row.Ref, head)
		if full, err := g.RevParse(ctx, row.Ref); err == nil {
			r.SHA = full
		}
		return r
	}
	rep.Verify = judge("kiln-verify")
	rep.Review = judge("kiln-review")
	rep.Fresh = true
	fail := func(msg string) {
		rep.Fresh = false
		rep.Reasons = append(rep.Reasons, msg)
	}
	switch v := rep.Verify; {
	case opts.Only == "review":
	case v == nil:
		fail("no kiln-verify row in the journal")
	case !v.Fresh:
		fail(fmt.Sprintf("verify ran on %s, which no longer matches head", gitx.Short(v.SHA)))
	case v.Result == Inconclusive && opts.AcceptInconclusive:
	case v.Result != Verified:
		fail("verify result is " + v.Result)
	}
	switch r := rep.Review; {
	case opts.Only == "verify":
	case r == nil:
		fail("no kiln-review row in the journal")
	case !r.Fresh:
		fail(fmt.Sprintf("review ran on %s, which no longer matches head", gitx.Short(r.SHA)))
	case r.Result != Pass:
		fail("review result is " + r.Result)
	}
	return rep, nil
}
