---
name: kiln-commit
description: Makes atomic commits in gitmoji plus Conventional Commits form (for example "✨ feat: add theme toggle to settings"), staging explicit paths only, and reorders the branch into small logical commits before the PR. Use at the Fire stage whenever kiln-build finishes a step, and once more before kiln-pr.
disable-model-invocation: true
---

# kiln-commit

This is the Fire stage's bookkeeping. You turn working changes into commits a reviewer can read one by one and that squash into a clean line on main. The PR title becomes the squash subject, but the branch history still matters: it shows the reproduction before the fix and the pin before the refactor.

## Reads

- the working tree: `git status --short`, `git diff`
- the branch so far: `git log --oneline origin/<default>..HEAD`
- `.kiln/config.yaml` `conventions.commits` (always `gitmoji-conventional`)

## Steps

1. **See what changed.**
   ```bash
   git status --short
   git diff
   ```
   Group the changes into units that each do one thing.

2. **Stage explicit paths.**
   ```bash
   git add src/theme/tokens.css src/features/settings/AppearanceSection.tsx
   git diff --cached --stat
   ```
   Check the staged list: no stray files, no secrets or `.env`, no build output, nothing from `.kiln/verify/evidence/`.

3. **Write the subject.** `<gitmoji> <type>[(scope)][!]: <lowercase imperative>`, at most 72 characters, no trailing period.

   | Gitmoji | Type | Use for | Release effect |
   |---------|------|---------|----------------|
   | ✨ | feat | new user-visible behavior | minor |
   | 🐛 | fix | a bug fix | patch |
   | 🚑️ | fix | an urgent hotfix | patch |
   | ♻️ | refactor | same behavior, new shape | patch |
   | ⚡️ | perf | faster, same behavior | patch |
   | ⬆️ | chore | dependency bump | patch |
   | 🔥 | refactor or chore | removing code or features | patch |
   | 🗑️ | chore | deprecating something | patch |
   | 🔒️ | fix | security fix | patch |
   | ✅ | test | tests only | none |
   | 📝 | docs | docs only | none |
   | 🔧 | chore | config | none |
   | 👷 | ci | CI | none |
   | 🎨 | style | formatting | none |
   | 💥 | feat! or fix! | breaking change | major |

   `kiln release plan` reads these emoji to pick the next version, so choose the one that's true. Add `!` after the type for breaking changes.

   Examples: `✨ feat: add theme toggle to settings` · `✅ test: reproduce login loop on expired session` · `♻️ refactor(settings): move colors to css tokens`

4. **Commit.**
   ```bash
   git commit -m "✨ feat: add theme toggle to settings"
   ```
   Add a body (`-m "<why>"`) only when the why isn't obvious from the diff.

5. **Reorder before the PR.** Once, when kiln-build is done and before kiln-verify runs (so the verdict covers the history you will push), read the branch:
   ```bash
   git log --oneline origin/<default>..HEAD
   ```
   If there are fixups or the order tells a muddled story, rewrite it while it's still unpushed:
   - fold small fixes in with `git commit --fixup=<sha>` as you go, then
     ```bash
     GIT_SEQUENCE_EDITOR=: git rebase -i --autosquash origin/<default>
     ```
   - or, for a bigger reshuffle, `git reset --soft origin/<default>` and re-commit in logical groups with explicit paths.

   Keep these orders intact: reproduction test before the bug fix, characterization tests before the refactor. Run the tests again after any rewrite.

## Writes

- commits on `kiln/<n>-<slug>`

## Rules

- Stage explicit paths. Never `git add -A`, `git add .`, `git add -u` or `git commit -a`.
- **Never add `Co-authored-by` trailers or any AI attribution** ("Generated with…", "Co-Authored-By: Claude", robot emoji credits) in the subject, body or trailers. The commit belongs to the human.
- Lowercase imperative subject: "add", not "Added" or "adds".
- Rewrite history only before the first push. After kiln-pr pushes, add commits; the squash merge cleans up. Force-pushing a shared branch is a hard stop.
- Never `--no-verify`. If a hook fails, fix the cause or ask.
- Don't rewrite history after kiln-verify has run. Journal commits from kiln-log (`📝 docs(kiln): log …`) stay as separate commits.

## Done when

`git status --short` shows nothing left that belongs to this step, and every new commit subject matches the format. Your row (ref is HEAD):

```
kiln-commit · 3 commits, reordered · a1c9e0f · ok
```
