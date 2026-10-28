package release

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pavelsimo/kiln/internal/gittest"
	"github.com/pavelsimo/kiln/internal/gitx"
	"github.com/pavelsimo/kiln/internal/run"
)

func TestParse(t *testing.T) {
	tests := []struct {
		subject, body     string
		bump, section     string
		pr                int
		desc, scope, emoj string
		breaking          bool
	}{
		{"✨ feat: add dark mode to settings (#42)", "", Minor, Added, 42, "add dark mode to settings", "", "✨", false},
		{"🐛 fix(auth): stop the login redirect loop (#43)", "", Patch, Fixed, 43, "stop the login redirect loop", "auth", "🐛", false},
		{"♻️ refactor: move colors to css tokens", "", Patch, Changed, 0, "move colors to css tokens", "", "♻️", false},
		{"⚡️ perf: cache the theme", "", Patch, Changed, 0, "cache the theme", "", "⚡️", false},
		{"🔧 chore: bump eslint to v10 (#44)", "", None, "", 44, "bump eslint to v10", "", "🔧", false},
		{"📝 docs: explain gates", "", None, "", 0, "explain gates", "", "📝", false},
		{"✨ feat!: drop node 18", "", Major, Added, 0, "drop node 18", "", "✨", true},
		{"✨ feat: new config format", "BREAKING CHANGE: old keys are gone", Major, Added, 0, "new config format", "", "✨", true},
		{"💥 feat: remove v1 api", "", Major, Changed, 0, "remove v1 api", "", "💥", false},
		{"feat: plain conventional", "", Minor, Added, 0, "plain conventional", "", "", false},
		{"🔒️ security: escape html in notes", "", Patch, Security, 0, "escape html in notes", "", "🔒️", false},
		{"🔒 fix: escape html", "", Patch, Security, 0, "escape html", "", "🔒", false},
		{"🗑️ chore: deprecate --old flag", "", Patch, Deprecated, 0, "deprecate --old flag", "", "🗑️", false},
		{"✨ add a thing", "", Minor, Added, 0, "add a thing", "", "✨", false},
		{"Merge branch 'x'", "", None, "", 0, "Merge branch 'x'", "", "", false},
	}
	for _, tt := range tests {
		c := Parse("sha", tt.subject, tt.body)
		if c.Bump != tt.bump || c.Section != tt.section || c.PR != tt.pr || c.Description != tt.desc || c.Scope != tt.scope || c.Emoji != tt.emoj {
			t.Errorf("Parse(%q) = %+v", tt.subject, c)
		}
		if tt.breaking && !c.Breaking {
			t.Errorf("Parse(%q) should be breaking", tt.subject)
		}
	}
}

func TestApply(t *testing.T) {
	v := func(s string) Version {
		x, err := ParseVersion(s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	tests := []struct{ from, bump, want string }{
		{"1.3.0", Minor, "1.4.0"},
		{"1.3.2", Patch, "1.3.3"},
		{"1.3.2", Major, "2.0.0"},
		{"0.4.1", Major, "0.5.0"},
		{"0.4.1", Minor, "0.5.0"},
		{"1.4.0-rc.2", Minor, "1.4.0"},
	}
	for _, tt := range tests {
		if got := Apply(v(tt.from), tt.bump).String(); got != tt.want {
			t.Errorf("Apply(%s, %s) = %s, want %s", tt.from, tt.bump, got, tt.want)
		}
	}
}

func TestVersionLess(t *testing.T) {
	order := []string{"0.1.0", "1.0.0-rc.1", "1.0.0-rc.2", "1.0.0-rc.10", "1.0.0", "1.0.1", "1.10.0", "2.0.0"}
	for i := 0; i+1 < len(order); i++ {
		a, _ := ParseVersion(order[i])
		b, _ := ParseVersion(order[i+1])
		if !a.Less(b) || b.Less(a) {
			t.Errorf("want %s < %s", a, b)
		}
	}
}

func TestHighestRule(t *testing.T) {
	cs := []Commit{
		Parse("a", "✨ feat: one (#1)", ""),
		Parse("b", "🐛 fix: two (#2)", ""),
		Parse("c", "✨ feat: three (#3)", ""),
	}
	bump, rule := Highest(cs)
	if bump != Minor || rule != "✨ feat in 2 PRs → minor" {
		t.Fatalf("got %s, %q", bump, rule)
	}
}

func repoWith(t *testing.T, subjects ...string) (*gittest.Repo, *gitx.Git) {
	r := gittest.New(t)
	for i, s := range subjects {
		r.Commit(s, map[string]string{"f.txt": strings.Repeat("x", i+1)})
	}
	return r, gitx.New(run.Exec{}, r.Dir)
}

var today = time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)

func TestBuildPlan(t *testing.T) {
	r, g := repoWith(t, "✨ feat: first (#1)")
	r.Git("tag", "-a", "v1.3.0", "-m", "v1.3.0")
	r.Commit("✨ feat: dark mode in settings (#42)", map[string]string{"a": "1"})
	r.Commit("🐛 fix: login no longer loops (#43)", map[string]string{"b": "1"})
	r.Commit("🔧 chore: bump eslint (#44)", map[string]string{"c": "1"})
	r.Commit("✨ feat: export history as CSV (#51)", map[string]string{"d": "1"})

	p, err := Build(context.Background(), g, Options{Ref: "main", RepoURL: "https://github.com/acme/app", Today: today})
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != "1.4.0" || p.Tag != "v1.4.0" || p.LastTag != "v1.3.0" || p.Bump != Minor || p.LeftOut != 1 {
		t.Fatalf("plan = %+v", p)
	}
	want := "## [1.4.0] - 2026-10-28\n\n### Added\n\n- dark mode in settings (#42)\n- export history as CSV (#51)\n\n### Fixed\n\n- login no longer loops (#43)\n\n[1.4.0]: https://github.com/acme/app/compare/v1.3.0...v1.4.0\n"
	if p.Changelog != want {
		t.Fatalf("changelog:\n%s\nwant:\n%s", p.Changelog, want)
	}
}

func TestBuildFirstAndNothing(t *testing.T) {
	_, g := repoWith(t, "✨ feat: hello (#1)")
	p, err := Build(context.Background(), g, Options{Ref: "main", Today: today})
	if err != nil || p.Version != "0.1.0" {
		t.Fatalf("first release: %+v, %v", p, err)
	}

	r, g := repoWith(t, "✨ feat: hello (#1)")
	r.Git("tag", "v0.1.0")
	r.Commit("📝 docs: typo", map[string]string{"x": "1"})
	if _, err := Build(context.Background(), g, Options{Ref: "main", Today: today}); !errors.Is(err, ErrNothing) {
		t.Fatalf("want ErrNothing, got %v", err)
	}
	p, err = Build(context.Background(), g, Options{Ref: "main", Version: "0.1.1", Today: today})
	if err != nil || p.Version != "0.1.1" {
		t.Fatalf("explicit version: %+v, %v", p, err)
	}
	if _, err := Build(context.Background(), g, Options{Ref: "main", Version: "0.1.0", Today: today}); err == nil {
		t.Fatal("want an error for a version that isn't newer")
	}
}

func TestBuildPre(t *testing.T) {
	r, g := repoWith(t, "✨ feat: hello (#1)")
	r.Git("tag", "v1.0.0")
	r.Commit("✨ feat: more (#2)", map[string]string{"x": "1"})
	p, err := Build(context.Background(), g, Options{Ref: "main", Pre: "rc", Today: today})
	if err != nil || p.Version != "1.1.0-rc.1" {
		t.Fatalf("rc.1: %+v, %v", p, err)
	}
	r.Git("tag", "v1.1.0-rc.1")
	p, err = Build(context.Background(), g, Options{Ref: "main", Pre: "rc", Today: today})
	if err != nil || p.Version != "1.1.0-rc.2" {
		t.Fatalf("rc.2: %+v, %v", p, err)
	}
	p, err = Build(context.Background(), g, Options{Ref: "main", Today: today})
	if err != nil || p.Version != "1.1.0" || p.LastTag != "v1.0.0" {
		t.Fatalf("final after rc: %+v, %v", p, err)
	}
}

func TestBuildTagExists(t *testing.T) {
	r, g := repoWith(t, "✨ feat: hello (#1)")
	r.Git("tag", "v0.1.0")
	r.Commit("🐛 fix: oops (#2)", map[string]string{"x": "1"})
	// someone already pushed v0.1.1, but this clone never fetched it
	bare := t.TempDir()
	r.Git("init", "-q", "--bare", bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("tag", "v0.1.1")
	r.Git("push", "-q", "origin", "v0.1.1")
	r.Git("tag", "-d", "v0.1.1")
	if _, err := Build(context.Background(), g, Options{Ref: "main", Today: today}); !errors.Is(err, ErrTagExists) {
		t.Fatalf("want ErrTagExists, got %v", err)
	}
}
