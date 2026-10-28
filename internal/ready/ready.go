// Package ready answers "can this PR squash merge right now?".
package ready

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/verdict"
)

// Item is one condition and whether it holds.
type Item struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Report is the answer.
type Report struct {
	PR           int             `json:"pr"`
	Title        string          `json:"title"`
	Branch       string          `json:"branch"`
	Head         string          `json:"head"`
	Issue        int             `json:"issue,omitempty"`
	Housekeeping bool            `json:"housekeeping"`
	Ready        bool            `json:"ready"`
	Checks       []Item          `json:"checks"`
	Verdict      *verdict.Report `json:"verdict,omitempty"`
}

// Options control Check.
type Options struct {
	PR                 int
	Repo               string // owner/name
	AcceptInconclusive bool
}

var closesRe = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)\b`)

// ClosesIssue returns the first issue a PR body closes.
func ClosesIssue(body string) (int, bool) {
	m := closesRe.FindStringSubmatch(body)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// Check gathers every merge condition. It never stops at the first failure, so the
// report lists everything that's left.
func Check(ctx context.Context, gh *ghx.GH, g *gitx.Git, opts Options) (Report, error) {
	pr, err := gh.PR(ctx, opts.PR)
	if err != nil {
		return Report{}, err
	}
	rep := Report{PR: pr.Number, Title: pr.Title, Branch: pr.HeadRefName, Head: pr.HeadRefOid, Housekeeping: gitx.IsHousekeepingBranch(pr.HeadRefName)}
	add := func(name string, ok bool, detail string) {
		rep.Checks = append(rep.Checks, Item{Name: name, OK: ok, Detail: detail})
	}

	switch {
	case !strings.EqualFold(pr.State, "OPEN"):
		add("open", false, "PR is "+strings.ToLower(pr.State))
	case pr.IsDraft:
		add("open", false, "PR is a draft; kiln opens ready PRs")
	default:
		add("open", true, "open, not a draft")
	}

	switch strings.ToUpper(pr.Mergeable) {
	case "MERGEABLE":
		add("conflicts", true, "no conflicts with "+pr.BaseRefName)
	case "CONFLICTING":
		add("conflicts", false, "conflicts with "+pr.BaseRefName+"; rebase first")
	default:
		add("conflicts", false, "GitHub is still computing mergeability; ask again in a moment")
	}

	s := ghx.Summarize(pr.StatusCheckRollup)
	switch {
	case s.Total == 0:
		add("ci", true, "no checks configured")
	case s.Failing > 0:
		add("ci", false, fmt.Sprintf("%d failing: %s", s.Failing, strings.Join(s.Failed, ", ")))
	case s.Pending > 0:
		add("ci", false, fmt.Sprintf("%d of %d still running", s.Pending, s.Total))
	default:
		add("ci", true, fmt.Sprintf("%d/%d green", s.Passed, s.Total))
	}

	switch strings.ToUpper(pr.ReviewDecision) {
	case "CHANGES_REQUESTED":
		add("review", false, "changes requested")
	case "REVIEW_REQUIRED":
		add("review", false, "branch protection requires an approving review")
	case "APPROVED":
		add("review", true, "approved")
	default:
		add("review", true, "no review required")
	}

	if opts.Repo != "" {
		n, err := gh.UnresolvedThreads(ctx, opts.Repo, pr.Number)
		switch {
		case err != nil:
			add("threads", false, "couldn't read review threads: "+err.Error())
		case n > 0:
			add("threads", false, fmt.Sprintf("%d unresolved review thread(s)", n))
		default:
			add("threads", true, "all review threads resolved")
		}
	}

	if rep.Housekeeping {
		add("verdict", true, "housekeeping PR ("+pr.HeadRefName+"): CI is the gate")
	} else {
		issue, ok := ClosesIssue(pr.Body)
		if !ok && len(pr.ClosingIssuesReferences) > 0 {
			issue, ok = pr.ClosingIssuesReferences[0].Number, true
		}
		if !ok {
			issue, ok = gitx.BranchIssue(pr.HeadRefName)
			add("closes", false, "PR body has no Closes #N")
		} else {
			add("closes", true, fmt.Sprintf("closes #%d", issue))
		}
		rep.Issue = issue
		switch {
		case !ok:
			add("verdict", false, "can't tell which issue this PR is for")
		case !g.RefExists(ctx, pr.HeadRefOid):
			if _, err := g.Run(ctx, "fetch", "--quiet", "origin", pr.HeadRefName); err != nil || !g.RefExists(ctx, pr.HeadRefOid) {
				add("verdict", false, "head "+gitx.Short(pr.HeadRefOid)+" isn't available locally (git fetch)")
				break
			}
			fallthrough
		default:
			v, err := verdict.Check(ctx, g, verdict.Options{
				Issue: issue, Rev: pr.HeadRefOid, Base: "origin/" + pr.BaseRefName, AcceptInconclusive: opts.AcceptInconclusive,
			})
			if err != nil {
				add("verdict", false, err.Error())
				break
			}
			rep.Verdict = &v
			if v.Fresh {
				add("verdict", true, verdictDetail(v))
			} else {
				add("verdict", false, strings.Join(v.Reasons, "; "))
			}
		}
	}

	rep.Ready = true
	for _, c := range rep.Checks {
		rep.Ready = rep.Ready && c.OK
	}
	return rep, nil
}

func verdictDetail(v verdict.Report) string {
	parts := []string{}
	if v.Verify != nil {
		parts = append(parts, fmt.Sprintf("%s on %s (%s)", v.Verify.Result, gitx.Short(v.Verify.SHA), v.Verify.How))
	}
	if v.Review != nil {
		parts = append(parts, fmt.Sprintf("review %s", v.Review.Result))
	}
	return strings.Join(parts, " · ")
}
