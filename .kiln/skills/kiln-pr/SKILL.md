---
name: kiln-pr
description: Pushes the task branch and opens a ready (never draft) pull request with a gitmoji title and the repo's PR template sections (Why, What changed, Scope, Verification, Kiln, Closes #N), then moves the issue to kiln:review. Use at the Ship stage once kiln-verify and kiln-review have a fresh verdict for HEAD.
disable-model-invocation: true
---

# kiln-pr

This is the start of the Ship stage. You put the work in front of reviewers as a real, ready PR. Its title becomes the one commit that lands on main, so it has to read well on its own. The issue moves from `kiln:in-progress` to `kiln:review`.

## Reads

- `kiln verdict <n> --json`: the verify and review results and whether they're fresh for HEAD
- the task journal: verify report, review verdict, skips, decisions
- the branch: `git log --oneline origin/<default>..HEAD`
- `.github/pull_request_template.md`
- the issue title and predicates

## Steps

1. **Check the verdict.**
   ```bash
   kiln verdict <n> --json
   ```
   - `verify.result` VERIFIED and `review.result` PASS, both `fresh`: go on.
   - `verify.result` INCONCLUSIVE with review PASS, both fresh: go on, but flag it at the top of Verification. ◆2 will need the human's acceptance.
   - NOT VERIFIED, FIX, missing, or stale: don't open the PR. Go back to kiln-build, or rerun kiln-verify and kiln-review on the current HEAD.

2. **Check the branch is final.** Commits reordered by kiln-commit, journal rows committed by kiln-log, `git status --porcelain` empty.

3. **Push.**
   ```bash
   git push -u origin kiln/<n>-<slug>
   ```

4. **Write the title.** Gitmoji plus Conventional Commits, lowercase imperative, no `(#N)` (the squash adds it):
   ```
   ✨ feat: add dark mode to settings
   ```
   Pick the emoji for the most significant change in the PR. `kiln release plan` reads it later to choose the version bump.

5. **Write the body** from the template's sections, with real content:
   ```markdown
   ## Why
   Settings had no dark theme; users asked for one (#42).

   ## What changed
   - colors moved to css tokens
   - toggle in Settings → Appearance, saved in localStorage

   ## Scope
   Settings page only. Other pages pick up tokens in a follow-up.

   ## Verification
   - kiln-verify: VERIFIED on a1c9e0f · checker fresh · 4 predicates + 1 standing check
   - P1 toggle switches to dark · VERIFIED 5 · P2 survives reload · VERIFIED 5
   - P3 follows OS theme · VERIFIED 4 · P4 light unchanged · VERIFIED 5
   - tests audited: 11 used, 1 rejected
   - kiln-review: PASS on a1c9e0f (independent reviewer)

   ## Kiln
   playbook feature · size M · .kiln/memory/tasks/42-dark-mode-settings.md

   Closes #42
   ```
   Any INCONCLUSIVE predicate goes first under Verification, with its `why` and `unblock`. Skips worth a reviewer's attention go under Kiln.

6. **Open it ready.**
   ```bash
   gh pr create --base <default> --head kiln/<n>-<slug> \
     --title "✨ feat: add dark mode to settings" --body-file "$BODY" --assignee @me
   ```

7. **Read it back.**
   ```bash
   gh pr view <pr> --json number,url,isDraft,title,body
   ```
   `isDraft` must be false and the body must contain `Closes #<n>`. If either is wrong, fix it (`gh pr ready <pr>`, `gh pr edit <pr> --body-file`).

8. **Move the issue.**
   ```bash
   kiln move <n> review
   ```

## Writes

- the pushed branch and one ready PR
- the `kiln:review` label, through `kiln move`

## Rules

- Never `--draft`. Kiln PRs are ready from the first push.
- One PR per issue, and it says `Closes #<n>`. `kiln ready` refuses a PR without it.
- Verification summarizes the actual report. Never claim a check that isn't in the journal.
- No AI attribution anywhere in the title or body.
- Don't paste evidence files into the PR; they stay local and gitignored.

## Done when

`gh pr view` shows a ready PR with `Closes #<n>`, and the issue is at `kiln:review`. Hand kiln-log the row and the PR number (it fills `pr:` in the journal frontmatter):

```
kiln-pr · opened PR · #57 · ok
```

Next comes kiln-watch to take it to green.
