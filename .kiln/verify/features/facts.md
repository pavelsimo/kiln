---
feature: facts
sources: ["cmd/checks.go", "cmd/verdict.go", "cmd/release.go", "internal/checks/**", "internal/verdict/**", "internal/journal/**", "internal/release/**", "internal/gitx/**"]
---
# kiln checks, kiln verdict, kiln release plan

How a user reaches it: skills call these with `--json`; people run them in a terminal.

Drive:
1. Add a check to `$SANDBOX/.kiln/verify/checks.yaml`, branch `kiln/42-x`, commit a matching file,
   then `"$KILN" -C "$SANDBOX" checks --base main --json` lists it with the matched file.
2. Commit a journal with a kiln-verify row for that SHA, then `verdict --only verify` exits 0, and
   plain `verdict` exits 3 (no review yet).
3. Squash the branch onto main as `✨ feat: … (#42)`, then `release plan --ref main --json` gives 0.1.0.

Capture: each command's output and exit code.
