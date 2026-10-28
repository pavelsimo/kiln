// Package ghx is a thin wrapper over the gh command line.
package ghx

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pavelsimo/kiln/internal/run"
)

// GH runs gh in Dir.
type GH struct {
	R   run.Runner
	Dir string
}

// New returns a GH for dir.
func New(r run.Runner, dir string) *GH { return &GH{R: r, Dir: dir} }

// Run runs gh and returns stdout.
func (g *GH) Run(ctx context.Context, args ...string) ([]byte, error) {
	return g.R.Run(ctx, g.Dir, nil, "gh", args...)
}

func (g *GH) json(ctx context.Context, v any, args ...string) error {
	out, err := g.Run(ctx, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("gh %s: decode: %w", strings.Join(args, " "), err)
	}
	return nil
}

// Login returns the authenticated user's login.
func (g *GH) Login(ctx context.Context) (string, error) {
	out, err := g.Run(ctx, "api", "user", "--jq", ".login")
	if err != nil {
		return "", fmt.Errorf("gh is not authenticated (run gh auth login): %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Repo describes the current repository on GitHub.
type Repo struct {
	NameWithOwner    string `json:"nameWithOwner"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
	URL string `json:"url"`
}

// Repo returns the repository for Dir.
func (g *GH) Repo(ctx context.Context) (Repo, error) {
	var r Repo
	err := g.json(ctx, &r, "repo", "view", "--json", "nameWithOwner,defaultBranchRef,url")
	return r, err
}

// Label is a repository label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// Labels lists repository labels.
func (g *GH) Labels(ctx context.Context) ([]Label, error) {
	var ls []Label
	err := g.json(ctx, &ls, "label", "list", "--limit", "500", "--json", "name,color,description")
	return ls, err
}

// CreateLabel creates or updates a label.
func (g *GH) CreateLabel(ctx context.Context, l Label) error {
	_, err := g.Run(ctx, "label", "create", l.Name, "--color", l.Color, "--description", l.Description, "--force")
	return err
}

// Issue is the subset of issue fields kiln reads.
type Issue struct {
	Number int     `json:"number"`
	Title  string  `json:"title"`
	State  string  `json:"state"`
	Labels []Label `json:"labels"`
	URL    string  `json:"url"`
}

// LabelNames returns the issue's label names.
func (i Issue) LabelNames() []string {
	out := make([]string, 0, len(i.Labels))
	for _, l := range i.Labels {
		out = append(out, l.Name)
	}
	return out
}

// OpenIssues lists open issues.
func (g *GH) OpenIssues(ctx context.Context) ([]Issue, error) {
	var is []Issue
	err := g.json(ctx, &is, "issue", "list", "--state", "open", "--limit", "500", "--json", "number,title,state,labels,url")
	return is, err
}

// Issue fetches one issue.
func (g *GH) Issue(ctx context.Context, n int) (Issue, error) {
	var i Issue
	err := g.json(ctx, &i, "issue", "view", strconv.Itoa(n), "--json", "number,title,state,labels,url")
	return i, err
}

// Comment is an issue comment.
type Comment struct {
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// IssueComments lists an issue's comments, oldest first.
func (g *GH) IssueComments(ctx context.Context, n int) ([]Comment, error) {
	var v struct {
		Comments []Comment `json:"comments"`
	}
	err := g.json(ctx, &v, "issue", "view", strconv.Itoa(n), "--json", "comments")
	return v.Comments, err
}

// EditLabels adds and removes labels on an issue.
func (g *GH) EditLabels(ctx context.Context, n int, add, remove []string) error {
	args := make([]string, 0, 3+2*len(add)+2*len(remove))
	args = append(args, "issue", "edit", strconv.Itoa(n))
	for _, a := range add {
		args = append(args, "--add-label", a)
	}
	for _, r := range remove {
		args = append(args, "--remove-label", r)
	}
	if len(add)+len(remove) == 0 {
		return nil
	}
	_, err := g.Run(ctx, args...)
	return err
}

// CommentIssue posts a comment on an issue.
func (g *GH) CommentIssue(ctx context.Context, n int, body string) error {
	_, err := g.Run(ctx, "issue", "comment", strconv.Itoa(n), "--body", body)
	return err
}

// Check is one entry of a PR's statusCheckRollup (a CheckRun or a StatusContext).
type Check struct {
	Typename   string `json:"__typename"`
	Name       string `json:"name"`
	Context    string `json:"context"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	State      string `json:"state"`
}

// Label returns the check's display name.
func (c Check) Label() string {
	if c.Name != "" {
		return c.Name
	}
	return c.Context
}

// Check outcomes.
const (
	OutcomePass    = "pass"
	OutcomeFail    = "fail"
	OutcomePending = "pending"
)

// Outcome classifies a check as pass, fail or pending.
func (c Check) Outcome() string {
	if c.Typename == "StatusContext" || (c.State != "" && c.Status == "") {
		switch strings.ToUpper(c.State) {
		case "SUCCESS":
			return OutcomePass
		case "PENDING", "EXPECTED":
			return OutcomePending
		default:
			return OutcomeFail
		}
	}
	if strings.ToUpper(c.Status) != "COMPLETED" {
		return OutcomePending
	}
	switch strings.ToUpper(c.Conclusion) {
	case "SUCCESS", "NEUTRAL", "SKIPPED":
		return OutcomePass
	default:
		return OutcomeFail
	}
}

// CheckSummary counts check outcomes.
type CheckSummary struct {
	Passed  int      `json:"passed"`
	Total   int      `json:"total"`
	Failing int      `json:"failing"`
	Pending int      `json:"pending"`
	Failed  []string `json:"failed,omitempty"`
}

// Summarize counts outcomes across checks.
func Summarize(cs []Check) CheckSummary {
	s := CheckSummary{Total: len(cs)}
	for _, c := range cs {
		switch c.Outcome() {
		case OutcomePass:
			s.Passed++
		case OutcomePending:
			s.Pending++
		default:
			s.Failing++
			s.Failed = append(s.Failed, c.Label())
		}
	}
	return s
}

// PR is the subset of pull request fields kiln reads.
type PR struct {
	Number            int     `json:"number"`
	Title             string  `json:"title"`
	State             string  `json:"state"`
	IsDraft           bool    `json:"isDraft"`
	Mergeable         string  `json:"mergeable"`
	ReviewDecision    string  `json:"reviewDecision"`
	HeadRefName       string  `json:"headRefName"`
	HeadRefOid        string  `json:"headRefOid"`
	BaseRefName       string  `json:"baseRefName"`
	Body              string  `json:"body"`
	URL               string  `json:"url"`
	StatusCheckRollup []Check `json:"statusCheckRollup"`
	MergeCommit       *struct {
		Oid string `json:"oid"`
	} `json:"mergeCommit"`
	ClosingIssuesReferences []struct {
		Number int `json:"number"`
	} `json:"closingIssuesReferences"`
}

const prFields = "number,title,state,isDraft,mergeable,reviewDecision,headRefName,headRefOid,baseRefName,body,url,statusCheckRollup,mergeCommit,closingIssuesReferences"

// PR fetches one pull request.
func (g *GH) PR(ctx context.Context, n int) (PR, error) {
	var p PR
	err := g.json(ctx, &p, "pr", "view", strconv.Itoa(n), "--json", prFields)
	return p, err
}

// OpenPRs lists open pull requests.
func (g *GH) OpenPRs(ctx context.Context) ([]PR, error) {
	var ps []PR
	err := g.json(ctx, &ps, "pr", "list", "--state", "open", "--limit", "200", "--json",
		"number,title,state,isDraft,mergeable,reviewDecision,headRefName,headRefOid,statusCheckRollup")
	return ps, err
}

// UnresolvedThreads counts unresolved review threads on a PR.
func (g *GH) UnresolvedThreads(ctx context.Context, nameWithOwner string, n int) (int, error) {
	owner, name, ok := strings.Cut(nameWithOwner, "/")
	if !ok {
		return 0, fmt.Errorf("bad repo %q", nameWithOwner)
	}
	const q = `query($owner:String!,$name:String!,$n:Int!){repository(owner:$owner,name:$name){pullRequest(number:$n){reviewThreads(first:100){nodes{isResolved}}}}}`
	var v struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	err := g.json(ctx, &v, "api", "graphql", "-f", "query="+q, "-F", "owner="+owner, "-F", "name="+name, "-F", "n="+strconv.Itoa(n))
	if err != nil {
		return 0, err
	}
	c := 0
	for _, t := range v.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if !t.IsResolved {
			c++
		}
	}
	return c, nil
}

// SquashMerge squash merges a PR, but only if its head is still sha.
func (g *GH) SquashMerge(ctx context.Context, n int, sha string) error {
	_, err := g.Run(ctx, "pr", "merge", strconv.Itoa(n), "--squash", "--match-head-commit", sha)
	return err
}
