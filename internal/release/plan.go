package release

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pavelsimo/kiln/internal/gitx"
)

// ErrNothing means no user-visible change merged since the last tag.
var ErrNothing = errors.New("nothing to release")

// ErrTagExists means the computed tag is already taken.
var ErrTagExists = errors.New("tag already exists")

// Options control Plan.
type Options struct {
	Ref       string // the branch to release, usually the default branch
	Version   string // explicit version override
	Pre       string // pre-release label, e.g. "rc"
	TagFormat string // e.g. "v{version}"
	RepoURL   string // https://github.com/owner/name, for compare links
	Today     time.Time
}

// Plan is the answer to "what version comes next?".
type Plan struct {
	LastTag    string              `json:"last_tag"`
	Version    string              `json:"version"`
	Tag        string              `json:"tag"`
	Bump       string              `json:"bump"`
	Rule       string              `json:"rule"`
	Ref        string              `json:"ref"`
	Commits    []Commit            `json:"commits"`
	Sections   map[string][]string `json:"sections"`
	LeftOut    int                 `json:"left_out"`
	Changelog  string              `json:"changelog"`
	CompareURL string              `json:"compare_url,omitempty"`
}

func tagFor(format, version string) string { return strings.ReplaceAll(format, "{version}", version) }

func versionFromTag(format, tag string) (Version, bool) {
	pre, post, _ := strings.Cut(format, "{version}")
	if !strings.HasPrefix(tag, pre) || !strings.HasSuffix(tag, post) {
		return Version{}, false
	}
	v, err := ParseVersion(strings.TrimSuffix(strings.TrimPrefix(tag, pre), post))
	return v, err == nil
}

// Build computes the release plan.
func Build(ctx context.Context, g *gitx.Git, opts Options) (Plan, error) {
	if opts.TagFormat == "" {
		opts.TagFormat = "v{version}"
	}
	if opts.Today.IsZero() {
		opts.Today = time.Now()
	}
	pattern := strings.ReplaceAll(opts.TagFormat, "{version}", "*")
	tags, err := g.Tags(ctx, pattern)
	if err != nil {
		return Plan{}, err
	}
	var stable, latest *Version
	stableTag := ""
	for _, t := range tags {
		v, ok := versionFromTag(opts.TagFormat, t)
		if !ok {
			continue
		}
		if latest == nil || latest.Less(v) {
			vv := v
			latest = &vv
		}
		if v.Pre == "" && (stable == nil || stable.Less(v)) {
			vv := v
			stable, stableTag = &vv, t
		}
	}
	p := Plan{LastTag: stableTag, Ref: opts.Ref, Sections: map[string][]string{}}
	rangeSpec := opts.Ref
	if stableTag != "" {
		rangeSpec = stableTag + ".." + opts.Ref
	}
	raw, err := g.Log(ctx, rangeSpec)
	if err != nil {
		return p, err
	}
	for _, c := range raw {
		pc := Parse(c.SHA, c.Subject, c.Body)
		if pc.Emoji != "" && stripVS(pc.Emoji) == "🔖" {
			continue // earlier release commits
		}
		p.Commits = append(p.Commits, pc)
	}
	p.Bump, p.Rule = Highest(p.Commits)

	var next Version
	switch {
	case opts.Version != "":
		v, err := ParseVersion(opts.Version)
		if err != nil {
			return p, err
		}
		if stable != nil && !stable.Less(v) {
			return p, fmt.Errorf("%s is not newer than %s", v, stableTag)
		}
		next = v
		p.Rule = "version set by hand"
	case stable == nil:
		if len(p.Commits) == 0 {
			return p, ErrNothing
		}
		next = Version{0, 1, 0, ""}
		p.Rule = "first release → 0.1.0"
	case p.Bump == None:
		return p, ErrNothing
	default:
		next = Apply(*stable, p.Bump)
	}
	if opts.Pre != "" && next.Pre == "" {
		n := 1
		if latest != nil && latest.Pre != "" && latest.Core() == next.Core() {
			parts := strings.Split(latest.Pre, ".")
			if len(parts) == 2 && parts[0] == opts.Pre {
				var k int
				if _, err := fmt.Sscanf(parts[1], "%d", &k); err == nil {
					n = k + 1
				}
			}
		}
		next.Pre = fmt.Sprintf("%s.%d", opts.Pre, n)
	}
	p.Version = next.String()
	p.Tag = tagFor(opts.TagFormat, p.Version)
	for _, t := range tags {
		if t == p.Tag {
			return p, fmt.Errorf("%w: %s", ErrTagExists, p.Tag)
		}
	}
	if g.RemoteTagExists(ctx, p.Tag) {
		return p, fmt.Errorf("%w on origin: %s", ErrTagExists, p.Tag)
	}

	for _, c := range p.Commits {
		if c.Section == "" {
			p.LeftOut++
			continue
		}
		line := c.Description
		if c.Breaking {
			line = "**BREAKING** " + line
		}
		if c.PR > 0 {
			line += fmt.Sprintf(" (#%d)", c.PR)
		}
		p.Sections[c.Section] = append(p.Sections[c.Section], line)
	}
	if opts.RepoURL != "" {
		if stableTag != "" {
			p.CompareURL = fmt.Sprintf("%s/compare/%s...%s", opts.RepoURL, stableTag, p.Tag)
		} else {
			p.CompareURL = fmt.Sprintf("%s/releases/tag/%s", opts.RepoURL, p.Tag)
		}
	}
	p.Changelog = p.render(opts.Today)
	return p, nil
}

func (p Plan) render(today time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## [%s] - %s\n", p.Version, today.Format("2006-01-02"))
	for _, s := range Sections {
		items := p.Sections[s]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\n", s)
		for _, it := range items {
			fmt.Fprintf(&b, "- %s\n", it)
		}
	}
	if p.CompareURL != "" {
		fmt.Fprintf(&b, "\n[%s]: %s\n", p.Version, p.CompareURL)
	}
	return b.String()
}

// SectionNames returns the sections that have entries, in changelog order.
func (p Plan) SectionNames() []string {
	var out []string
	for _, s := range Sections {
		if len(p.Sections[s]) > 0 {
			out = append(out, s)
		}
	}
	return out
}
