// Package checks matches standing checks in .kiln/verify/checks.yaml against a branch's changes.
package checks

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

// File is the standing checks file, relative to the verify dir.
const File = "checks.yaml"

// Check is one standing check.
type Check struct {
	ID   string `yaml:"id" json:"id"`
	When struct {
		Paths  []string `yaml:"paths" json:"paths,omitempty"`
		Labels []string `yaml:"labels" json:"labels,omitempty"`
	} `yaml:"when" json:"when"`
	Predicate string `yaml:"predicate" json:"predicate"`
	Blocking  bool   `yaml:"blocking" json:"blocking"`
	From      string `yaml:"from" json:"from,omitempty"`
}

// Load reads checks.yaml from the verify dir. A missing file means no checks.
func Load(verifyDir string) ([]Check, error) {
	b, err := os.ReadFile(filepath.Join(verifyDir, File))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cs []Check
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, fmt.Errorf("%s: %w", File, err)
	}
	seen := map[string]bool{}
	for i, c := range cs {
		if c.ID == "" {
			return nil, fmt.Errorf("%s: check %d has no id", File, i+1)
		}
		if seen[c.ID] {
			return nil, fmt.Errorf("%s: duplicate id %q", File, c.ID)
		}
		seen[c.ID] = true
		for _, p := range c.When.Paths {
			if !doublestar.ValidatePattern(p) {
				return nil, fmt.Errorf("%s: %s: bad glob %q", File, c.ID, p)
			}
		}
	}
	return cs, nil
}

// Match is a check that applies, with the files that triggered it.
type Match struct {
	Check
	Matched []string `json:"matched"`
}

// Apply returns the checks that apply to changed files and issue labels.
// A check with paths needs at least one matching file; a check with labels needs one
// matching label; a check with neither always applies.
func Apply(cs []Check, changed, labels []string) []Match {
	var out []Match
	for _, c := range cs {
		var hit []string
		if len(c.When.Paths) > 0 {
			for _, f := range changed {
				for _, p := range c.When.Paths {
					if ok, _ := doublestar.Match(p, f); ok {
						hit = append(hit, f)
						break
					}
				}
			}
			if len(hit) == 0 {
				continue
			}
		}
		if len(c.When.Labels) > 0 && !slices.ContainsFunc(c.When.Labels, func(l string) bool { return slices.Contains(labels, l) }) {
			continue
		}
		if hit == nil {
			hit = []string{}
		}
		out = append(out, Match{Check: c, Matched: hit})
	}
	return out
}
