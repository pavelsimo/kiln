package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pavelsimo/kiln/internal/assets"
	"github.com/pavelsimo/kiln/internal/ghx"
	"github.com/pavelsimo/kiln/internal/lock"
	"github.com/pavelsimo/kiln/internal/run"
)

func initRepo(t *testing.T, opts Options) (string, Report) {
	t.Helper()
	if opts.Root == "" {
		opts.Root = t.TempDir()
	}
	opts.Version = "0.1.0"
	rep, err := Init(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return opts.Root, rep
}

func mustExist(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(root, p)); err != nil {
			t.Errorf("missing %s", p)
		}
	}
}

func TestInitTree(t *testing.T) {
	root, rep := initRepo(t, Options{})
	if rep.Skills != 21 {
		t.Errorf("want 21 skills, got %d", rep.Skills)
	}
	mustExist(
		t, root,
		".kiln/config.yaml", ".kiln/kiln.lock",
		".kiln/skills/kiln/SKILL.md", ".kiln/skills/kiln/playbooks/feature.md", ".kiln/skills/kiln-verify/SKILL.md",
		".kiln/memory/PROJECT.md", ".kiln/memory/lessons.md", ".kiln/memory/decisions/.gitkeep", ".kiln/memory/tasks/.gitkeep",
		".kiln/verify/checks.yaml", ".kiln/verify/features/.gitkeep", ".kiln/verify/evidence/.gitignore",
		".github/ISSUE_TEMPLATE/kiln-task.yml", ".github/pull_request_template.md", ".github/workflows/ci.yml",
		"AGENTS.md", "CLAUDE.md",
		".claude/skills/kiln-issue/SKILL.md", ".agents/skills/kiln-issue/SKILL.md", ".cursor/skills/kiln/SKILL.md",
	)
	target, err := os.Readlink(filepath.Join(root, ".claude/skills/kiln-issue"))
	if err != nil || target != "../../.kiln/skills/kiln-issue" {
		t.Errorf("link target = %q, %v", target, err)
	}
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(string(agents), MarkStart) || !strings.Contains(string(agents), ".kiln/skills/kiln/SKILL.md") {
		t.Errorf("AGENTS.md has no pointer block:\n%s", agents)
	}
	lk, err := lock.Load(root)
	if err != nil || lk.Version != "0.1.0" || len(lk.Files) != len(assets.SkillFiles()) {
		t.Errorf("lock = %d files, version %q, %v", len(lk.Files), lk.Version, err)
	}
}

func TestInitIsIdempotentAndKeepsYourFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# My rules\n\nBe nice.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("AGENTS.md", filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".github/workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".github/workflows/ci.yml"), []byte("name: mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	initRepo(t, Options{Root: root})
	_, rep := initRepo(t, Options{Root: root})
	if n := rep.Count("create", "replace", "link"); n != 0 {
		t.Errorf("second run changed %d things: %+v", n, rep.Actions)
	}
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if !strings.HasPrefix(string(agents), "# My rules") || strings.Count(string(agents), MarkStart) != 1 {
		t.Errorf("AGENTS.md lost content or got two blocks (CLAUDE.md is a symlink to it):\n%s", agents)
	}
	ci, _ := os.ReadFile(filepath.Join(root, ".github/workflows/ci.yml"))
	if string(ci) != "name: mine\n" {
		t.Error("init overwrote an existing ci.yml")
	}
}

func TestInitAgentsAndDryRun(t *testing.T) {
	root, rep := initRepo(t, Options{Agents: []string{"claude"}, DryRun: true})
	if _, err := os.Stat(filepath.Join(root, ".kiln")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote files")
	}
	if rep.Count("create") == 0 || rep.Count("link") != 21 {
		t.Fatalf("dry run should still report actions: %d creates, %d links", rep.Count("create"), rep.Count("link"))
	}
	root, _ = initRepo(t, Options{Agents: []string{"claude"}})
	if _, err := os.Stat(filepath.Join(root, ".cursor")); !os.IsNotExist(err) {
		t.Error("linked cursor without being asked")
	}
	cfg, _ := os.ReadFile(filepath.Join(root, ".kiln/config.yaml"))
	if !strings.Contains(string(cfg), "agents: [claude]\n") {
		t.Error("config.yaml should record the chosen agents")
	}
	if _, err := Init(context.Background(), Options{Root: t.TempDir(), Agents: []string{"vim"}}); err == nil {
		t.Error("want an error for an unknown agent")
	}
}

func TestInitCreatesLabels(t *testing.T) {
	fake := &run.Fake{Responses: map[string]run.Response{"gh label create *": {}}}
	_, rep := initRepo(t, Options{GH: ghx.New(fake, "")})
	if rep.Labels != 14 || len(fake.Calls) != 14 {
		t.Fatalf("labels = %d, calls = %d", rep.Labels, len(fake.Calls))
	}
	if !strings.Contains(fake.Calls[0], "gh label create kiln:needs-triage --color b48767") || !strings.HasSuffix(fake.Calls[0], "--force") {
		t.Fatalf("first call = %s", fake.Calls[0])
	}
}

func TestUpdate(t *testing.T) {
	root, _ := initRepo(t, Options{Agents: []string{"claude"}})
	ctx := context.Background()
	edited := filepath.Join(root, ".kiln/skills/kiln-commit/SKILL.md")
	pristine := filepath.Join(root, ".kiln/skills/kiln-pr/SKILL.md")

	// pretend the installed files came from an older kiln
	lk, _ := lock.Load(root)
	old := []byte("old version\n")
	for _, p := range []string{edited, pristine} {
		if err := os.WriteFile(p, old, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lk.Files["skills/kiln-commit/SKILL.md"] = lock.Hash(old)
	lk.Files["skills/kiln-pr/SKILL.md"] = lock.Hash(old)
	// and you edited one of them
	if err := os.WriteFile(edited, []byte("old version\nmy house rule\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// and a skill kiln no longer ships
	gone := filepath.Join(root, ".kiln/skills/kiln-gone/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(gone), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gone, old, 0o644); err != nil {
		t.Fatal(err)
	}
	lk.Files["skills/kiln-gone/SKILL.md"] = lock.Hash(old)
	if err := os.Symlink("../../.kiln/skills/kiln-gone", filepath.Join(root, ".claude/skills/kiln-gone")); err != nil {
		t.Fatal(err)
	}
	if err := lk.Save(root); err != nil {
		t.Fatal(err)
	}

	dry, err := Update(ctx, Options{Root: root, DryRun: true, Version: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(pristine); string(b) != string(old) {
		t.Fatal("dry run changed a file")
	}
	if dry.Count("conflict") != 1 || dry.Count("replace") != 1 {
		t.Fatalf("dry run: %+v", dry.Actions)
	}

	rep, err := Update(ctx, Options{Root: root, Version: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	want := assets.SkillFiles()
	if b, _ := os.ReadFile(pristine); string(b) != string(want["kiln-pr/SKILL.md"]) {
		t.Error("untouched file was not replaced")
	}
	if b, _ := os.ReadFile(edited); !strings.Contains(string(b), "my house rule") {
		t.Error("your edit was overwritten")
	}
	if b, err := os.ReadFile(edited + NewSuffix); err != nil || string(b) != string(want["kiln-commit/SKILL.md"]) {
		t.Error("no .kiln-new next to the edited file")
	}
	if _, err := os.Stat(gone); !os.IsNotExist(err) {
		t.Error("removed skill is still there")
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude/skills/kiln-gone")); !os.IsNotExist(err) {
		t.Error("dangling link to the removed skill is still there")
	}
	if rep.Count("remove") != 1 || rep.Count("unlink") != 1 {
		t.Errorf("actions: %+v", rep.Actions)
	}

	// a second update leaves your edit and its .kiln-new alone
	rep, err = Update(ctx, Options{Root: root, Version: "0.2.0"})
	if err != nil {
		t.Fatal(err)
	}
	if n := rep.Count("create", "replace", "conflict", "remove"); n != 0 {
		t.Errorf("second update changed %d things: %+v", n, rep.Actions)
	}
	lk, _ = lock.Load(root)
	if lk.Version != "0.2.0" {
		t.Errorf("lock version = %s", lk.Version)
	}
}

func TestUpsertBlock(t *testing.T) {
	block := MarkStart + "\nnew\n" + MarkEnd
	if got := UpsertBlock("", block); got != block+"\n" {
		t.Errorf("empty: %q", got)
	}
	doc := "# Mine\n\n" + MarkStart + "\nold\n" + MarkEnd + "\n\nafter\n"
	if got := UpsertBlock(doc, block); got != "# Mine\n\n"+block+"\n\nafter\n" {
		t.Errorf("replace: %q", got)
	}
}
