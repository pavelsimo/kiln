package journal

import "testing"

const sample = `---
issue: 42
title: ✨ add dark mode to settings
type: feature
size: M
playbook: feature
branch: kiln/42-dark-mode
pr: 57
---
## Goal
Users can switch settings to a dark theme.

## Journal
| ts               | skill       | action              | ref     | result   |
|------------------|-------------|---------------------|---------|----------|
| 2026-10-27 14:02 | kiln-issue  | opened issue        | #42     | ok       |
| 2026-10-27 14:37 | kiln-verify | 4 predicates        | a1c9e0f | NOT VERIFIED |
| 2026-10-27 14:52 | kiln-verify | 4 predicates, again | ` + "`b2d0f1a`" + ` | verified |
| 2026-10-27 14:55 | kiln-review | reviewer subagent   | b2d0f1a | PASS     |

## Skipped
- skip: kiln-explore · settings area already mapped

## Retro notes
- the settings mock lives in test/fixtures
- e2e snapshots need --update after token changes
`

func TestParse(t *testing.T) {
	j, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if j.Meta.Issue != 42 || j.Meta.Size != "M" || j.Meta.PR != 57 || j.Meta.Branch != "kiln/42-dark-mode" {
		t.Fatalf("meta = %+v", j.Meta)
	}
	if len(j.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(j.Rows))
	}
	v, ok := j.Latest("kiln-verify")
	if !ok || v.Ref != "b2d0f1a" || Normalize(v.Result) != "VERIFIED" {
		t.Fatalf("latest verify = %+v", v)
	}
	if len(j.Skips) != 1 || len(j.RetroNotes) != 2 {
		t.Fatalf("skips %v, notes %v", j.Skips, j.RetroNotes)
	}
	last, ok := j.LastActivity()
	if !ok || last.Format("15:04") != "14:55" {
		t.Fatalf("last activity = %v", last)
	}
}

func TestParseNoFrontmatter(t *testing.T) {
	j, err := Parse("## Journal\n| 2026-10-28 | kiln-release | opened PR | #61 | ok |\n")
	if err != nil || len(j.Rows) != 1 || j.Meta.Issue != 0 {
		t.Fatalf("%+v %v", j, err)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"verified": "VERIFIED", " not_verified ": "NOT VERIFIED", "Inconclusive": "INCONCLUSIVE", "NOT  VERIFIED": "NOT VERIFIED"} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
