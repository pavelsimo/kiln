// Package config loads and validates .kiln/config.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/pavelsimo/kiln/internal/assets"
)

// Path is where the config lives, relative to the repo root.
const Path = ".kiln/config.yaml"

// Rigor is the verify and review depth for one size.
type Rigor struct {
	Verify     string   `yaml:"verify" json:"verify"`
	Review     string   `yaml:"review" json:"review"`
	ForceGates []string `yaml:"force_gates,omitempty" json:"force_gates,omitempty"`
}

// Config mirrors .kiln/config.yaml.
type Config struct {
	Version  int              `yaml:"version" json:"version"`
	Autonomy string           `yaml:"autonomy" json:"autonomy"`
	Gates    []string         `yaml:"gates" json:"gates"`
	Rigor    map[string]Rigor `yaml:"rigor" json:"rigor"`
	Verify   struct {
		Dir  string   `yaml:"dir" json:"dir"`
		Live []string `yaml:"live" json:"live"`
	} `yaml:"verify" json:"verify"`
	Conventions struct {
		Commits string `yaml:"commits" json:"commits"`
		Branch  string `yaml:"branch" json:"branch"`
		Merge   string `yaml:"merge" json:"merge"`
		PR      string `yaml:"pr" json:"pr"`
	} `yaml:"conventions" json:"conventions"`
	Worktrees struct {
		Dir         string `yaml:"dir" json:"dir"`
		MaxInFlight int    `yaml:"max_in_flight" json:"max_in_flight"`
	} `yaml:"worktrees" json:"worktrees"`
	Release struct {
		Tag          string   `yaml:"tag" json:"tag"`
		Changelog    string   `yaml:"changelog" json:"changelog"`
		VersionFiles []string `yaml:"version_files" json:"version_files"`
		Publish      string   `yaml:"publish" json:"publish"`
	} `yaml:"release" json:"release"`
	Agents []string `yaml:"agents" json:"agents"`
	GitHub struct {
		Labels    bool `yaml:"labels" json:"labels"`
		Templates bool `yaml:"templates" json:"templates"`
		CI        bool `yaml:"ci" json:"ci"`
	} `yaml:"github" json:"github"`
}

// Default returns the config kiln init writes.
func Default() Config {
	var c Config
	if err := decode(assets.Template("kiln/config.yaml"), &c); err != nil {
		panic(err)
	}
	return c
}

// Parse decodes config YAML on top of the defaults, so missing keys keep their default.
func Parse(b []byte) (Config, error) {
	c := Default()
	err := decode(b, &c)
	return c, err
}

func decode(b []byte, c *Config) error {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("config: %w", err)
	}
	return nil
}

// Load reads the config under root. A missing file returns the defaults and os.ErrNotExist.
func Load(root string) (Config, error) {
	b, err := os.ReadFile(filepath.Join(root, Path))
	if err != nil {
		return Default(), err
	}
	return Parse(b)
}

// Known values for validation.
var (
	Autonomies = []string{"manual", "gated", "autopilot"}
	GateNames  = []string{"plan", "merge"}
	Sizes      = []string{"S", "M", "L"}
	VerifyBy   = []string{"self", "fresh", "swarm"}
	ReviewBy   = []string{"self", "independent", "adversarial"}
	AgentNames = []string{"claude", "codex", "cursor"}
	Publishes  = []string{"auto", "gh", "tag"}
)

// Validate returns every problem found, or nil.
func (c Config) Validate() []error {
	var errs []error
	bad := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	if c.Version != 1 {
		bad("version: want 1, got %d", c.Version)
	}
	if !slices.Contains(Autonomies, c.Autonomy) {
		bad("autonomy: %q is not one of %s", c.Autonomy, strings.Join(Autonomies, ", "))
	}
	for _, g := range c.Gates {
		if !slices.Contains(GateNames, g) {
			bad("gates: unknown gate %q", g)
		}
	}
	for _, s := range Sizes {
		r, ok := c.Rigor[s]
		if !ok {
			bad("rigor.%s: missing", s)
			continue
		}
		if !slices.Contains(VerifyBy, r.Verify) {
			bad("rigor.%s.verify: %q is not one of %s", s, r.Verify, strings.Join(VerifyBy, ", "))
		}
		if !slices.Contains(ReviewBy, r.Review) {
			bad("rigor.%s.review: %q is not one of %s", s, r.Review, strings.Join(ReviewBy, ", "))
		}
		for _, g := range r.ForceGates {
			if !slices.Contains(GateNames, g) {
				bad("rigor.%s.force_gates: unknown gate %q", s, g)
			}
		}
	}
	for k := range c.Rigor {
		if !slices.Contains(Sizes, k) {
			bad("rigor: unknown size %q", k)
		}
	}
	if !strings.Contains(c.Conventions.Branch, "{issue}") {
		bad("conventions.branch: must contain {issue}")
	}
	if c.Conventions.Merge != "" && c.Conventions.Merge != "squash" {
		bad("conventions.merge: only squash is supported")
	}
	if c.Worktrees.MaxInFlight < 1 {
		bad("worktrees.max_in_flight: must be at least 1")
	}
	if !strings.Contains(c.Release.Tag, "{version}") {
		bad("release.tag: must contain {version}")
	}
	if !slices.Contains(Publishes, c.Release.Publish) {
		bad("release.publish: %q is not one of %s", c.Release.Publish, strings.Join(Publishes, ", "))
	}
	for _, a := range c.Agents {
		if !slices.Contains(AgentNames, a) {
			bad("agents: unknown agent %q", a)
		}
	}
	return errs
}

// GateActive reports whether kiln stops at gate for a task of size.
func (c Config) GateActive(gate, size string) bool {
	if slices.Contains(c.Rigor[size].ForceGates, gate) {
		return true
	}
	switch c.Autonomy {
	case "manual":
		return true
	case "autopilot":
		return false
	default:
		return slices.Contains(c.Gates, gate)
	}
}

// WorktreeDir resolves worktrees.dir for a repo at root.
func (c Config) WorktreeDir(root string) string {
	d := strings.ReplaceAll(c.Worktrees.Dir, "{repo}", filepath.Base(root))
	if !filepath.IsAbs(d) {
		d = filepath.Join(root, d)
	}
	return filepath.Clean(d)
}

// TagFor renders release.tag for a version.
func (c Config) TagFor(version string) string {
	return strings.ReplaceAll(c.Release.Tag, "{version}", version)
}

// ErrMissing reports a missing config file.
var ErrMissing = errors.New(Path + " not found (run kiln init)")
