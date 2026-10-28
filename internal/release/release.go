// Package release infers the next version from gitmoji squash commits and drafts the changelog.
package release

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Bump levels, lowest to highest.
const (
	None  = "none"
	Patch = "patch"
	Minor = "minor"
	Major = "major"
)

var bumpRank = map[string]int{None: 0, Patch: 1, Minor: 2, Major: 3}

// Keep a Changelog section names.
const (
	Added      = "Added"
	Changed    = "Changed"
	Deprecated = "Deprecated"
	Removed    = "Removed"
	Fixed      = "Fixed"
	Security   = "Security"
)

// Sections lists Keep a Changelog sections in the order they are written.
var Sections = []string{Added, Changed, Deprecated, Removed, Fixed, Security}

type rule struct {
	bump    string
	section string
}

// emojiRules maps a gitmoji (without variation selectors) to its bump.
var emojiRules = map[string]rule{
	"💥": {Major, Changed},
	"✨": {Minor, Added},
	"🐛": {Patch, Fixed},
	"🚑": {Patch, Fixed},
	"⚡": {Patch, Changed},
	"⬆": {Patch, Changed},
	"⬇": {Patch, Changed},
	"♻": {Patch, Changed},
	"🗑": {Patch, Deprecated},
	"🔥": {Patch, Removed},
	"🔒": {Patch, Security},
	"🔧": {None, ""},
	"✅": {None, ""},
	"📝": {None, ""},
	"👷": {None, ""},
	"🎨": {None, ""},
	"🔖": {None, ""},
	"🎉": {None, ""},
	"💚": {None, ""},
	"🚨": {None, ""},
	"🙈": {None, ""},
	"🔨": {None, ""},
}

// typeRules is the fallback when a subject has no known gitmoji.
var typeRules = map[string]rule{
	"feat":     {Minor, Added},
	"fix":      {Patch, Fixed},
	"perf":     {Patch, Changed},
	"refactor": {Patch, Changed},
	"deps":     {Patch, Changed},
	"revert":   {Patch, Changed},
	"security": {Patch, Security},
}

// Commit is one parsed squash commit.
type Commit struct {
	SHA         string `json:"sha"`
	Subject     string `json:"subject"`
	Emoji       string `json:"emoji"`
	Type        string `json:"type"`
	Scope       string `json:"scope"`
	Breaking    bool   `json:"breaking"`
	PR          int    `json:"pr,omitempty"`
	Description string `json:"description"`
	Bump        string `json:"bump"`
	Section     string `json:"section,omitempty"`
}

var (
	subjectRe = regexp.MustCompile(`^(?:(\S+)\s+)?([a-z]+)(?:\(([^)]*)\))?(!)?:\s*(.+)$`)
	prRe      = regexp.MustCompile(`\s*\(#(\d+)\)\s*$`)
)

func stripVS(s string) string {
	return strings.NewReplacer("️", "", "︎", "").Replace(s)
}

// Parse classifies a commit subject and body.
func Parse(sha, subject, body string) Commit {
	c := Commit{SHA: sha, Subject: subject, Bump: None}
	rest := strings.TrimSpace(subject)
	if m := prRe.FindStringSubmatch(rest); m != nil {
		c.PR, _ = strconv.Atoi(m[1])
		rest = strings.TrimSpace(rest[:len(rest)-len(m[0])])
	}
	var r rule
	known := false
	if m := subjectRe.FindStringSubmatch(rest); m != nil {
		c.Emoji, c.Type, c.Scope, c.Breaking, c.Description = m[1], m[2], m[3], m[4] == "!", m[5]
	} else {
		// "✨ add thing" with no conventional type
		first, after, _ := strings.Cut(rest, " ")
		if _, ok := emojiRules[stripVS(first)]; ok {
			c.Emoji, c.Description = first, strings.TrimSpace(after)
		} else {
			c.Description = rest
		}
	}
	if c.Emoji != "" {
		r, known = emojiRules[stripVS(c.Emoji)]
	}
	if !known {
		r, known = typeRules[c.Type]
	}
	if known {
		c.Bump, c.Section = r.bump, r.section
	}
	if strings.Contains(body, "BREAKING CHANGE:") || strings.Contains(body, "BREAKING-CHANGE:") {
		c.Breaking = true
	}
	if c.Breaking {
		c.Bump = Major
		if c.Section == "" {
			c.Section = Changed
		}
	}
	return c
}

// Version is a parsed semantic version.
type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

var semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

// ParseVersion parses "1.2.3", "v1.2.3" or "1.2.3-rc.1".
func ParseVersion(s string) (Version, error) {
	m := semverRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Version{}, fmt.Errorf("%q is not a semantic version", s)
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	c, _ := strconv.Atoi(m[3])
	return Version{a, b, c, m[4]}, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Core drops the pre-release part.
func (v Version) Core() Version { return Version{v.Major, v.Minor, v.Patch, ""} }

// Less compares versions (pre-releases sort before their release).
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	if v.Patch != o.Patch {
		return v.Patch < o.Patch
	}
	if v.Pre == o.Pre {
		return false
	}
	if v.Pre == "" {
		return false
	}
	if o.Pre == "" {
		return true
	}
	return preLess(v.Pre, o.Pre)
}

func preLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(pa) && i < len(pb); i++ {
		na, ea := strconv.Atoi(pa[i])
		nb, eb := strconv.Atoi(pb[i])
		switch {
		case ea == nil && eb == nil && na != nb:
			return na < nb
		case ea != nil || eb != nil:
			if pa[i] != pb[i] {
				return pa[i] < pb[i]
			}
		}
	}
	return len(pa) < len(pb)
}

// Apply bumps v. Before 1.0, a major bump moves the minor version.
func Apply(v Version, bump string) Version {
	base := v.Core()
	if v.Pre != "" {
		// 1.4.0-rc.2 + minor → 1.4.0 when the rc already covers the bump
		return base
	}
	switch bump {
	case Major:
		if base.Major == 0 {
			return Version{0, base.Minor + 1, 0, ""}
		}
		return Version{base.Major + 1, 0, 0, ""}
	case Minor:
		return Version{base.Major, base.Minor + 1, 0, ""}
	case Patch:
		return Version{base.Major, base.Minor, base.Patch + 1, ""}
	}
	return base
}

// Highest returns the biggest bump among commits and a human rule describing it.
func Highest(cs []Commit) (string, string) {
	best := None
	count := map[string]int{}
	example := map[string]Commit{}
	for _, c := range cs {
		count[c.Bump]++
		if _, ok := example[c.Bump]; !ok {
			example[c.Bump] = c
		}
		if bumpRank[c.Bump] > bumpRank[best] {
			best = c.Bump
		}
	}
	if best == None {
		return None, "nothing user-visible since the last tag"
	}
	ex := example[best]
	label := strings.TrimSpace(ex.Emoji + " " + ex.Type)
	if best == Major {
		label = "💥 breaking"
	}
	n := count[best]
	unit := "PRs"
	if n == 1 {
		unit = "PR"
	}
	return best, fmt.Sprintf("%s in %d %s → %s", label, n, unit, best)
}
