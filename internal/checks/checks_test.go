package checks

import (
	"os"
	"path/filepath"
	"testing"
)

const file = `
- id: migrations-reversible
  when: { paths: ["db/migrations/**"] }
  predicate: rollback one step, then migrate again
  blocking: true
- id: tokens-only
  when: { paths: ["src/**/*.css", "src/**/*.tsx"] }
  predicate: pnpm lint:tokens exits 0
  from: lessons.md · strong · #31 #42 #49
- id: features-have-e2e
  when: { labels: ["type:feature"] }
  predicate: a new e2e spec covers the change
- id: always
  predicate: pnpm build exits 0
`

func load(t *testing.T, body string) ([]Check, error) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, File), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return Load(dir)
}

func TestApply(t *testing.T) {
	cs, err := load(t, file)
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ms []Match) []string {
		var out []string
		for _, m := range ms {
			out = append(out, m.ID)
		}
		return out
	}
	got := ids(Apply(cs, []string{"src/features/settings/Theme.tsx", "README.md"}, []string{"type:feature"}))
	want := []string{"tokens-only", "features-have-e2e", "always"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	m := Apply(cs, []string{"db/migrations/2026/001.sql"}, nil)
	if len(m) != 2 || m[0].ID != "migrations-reversible" || m[0].Matched[0] != "db/migrations/2026/001.sql" || !m[0].Blocking {
		t.Fatalf("got %+v", m)
	}
}

func TestLoadErrors(t *testing.T) {
	if cs, err := Load(t.TempDir()); err != nil || cs != nil {
		t.Fatalf("missing file should mean no checks, got %v %v", cs, err)
	}
	if _, err := load(t, "- predicate: x\n"); err == nil {
		t.Fatal("want error for a missing id")
	}
	if _, err := load(t, "- id: a\n- id: a\n"); err == nil {
		t.Fatal("want error for a duplicate id")
	}
	if _, err := load(t, "- id: a\n  when: { paths: ['src/[x'] }\n"); err == nil {
		t.Fatal("want error for a bad glob")
	}
	if cs, err := load(t, "# only comments\n[]\n"); err != nil || len(cs) != 0 {
		t.Fatalf("empty list: %v %v", cs, err)
	}
}
