---
feature: install
sources: ["cmd/init.go", "cmd/update.go", "internal/install/**", "internal/lock/**", "internal/assets/**"]
---
# kiln init and kiln update

How a user reaches it: `kiln init` in a fresh repo, later `kiln update` after upgrading the binary.

Drive:
1. `"$KILN" -C "$SANDBOX" init --no-github` prints "21 skills" and "next: run /kiln-onboard".
2. `ls -l "$SANDBOX/.claude/skills/kiln-issue"` is a symlink to `../../.kiln/skills/kiln-issue`.
3. Edit a skill, then `"$KILN" -C "$SANDBOX" update --dry-run` keeps it and reports a `.kiln-new`
   only when the binary ships a different version.

Capture: init output, `find "$SANDBOX/.kiln" -maxdepth 2`, update output.
