---
name: kiln-merge
description: Holds gate 2. Shows what kiln ready reports for a PR, waits for the human according to autonomy, then squash merges only through kiln merge and reads back the merged state and cleanup. Use at the end of the Ship stage once kiln-watch has the PR green; INCONCLUSIVE verdicts need explicit human acceptance.
disable-model-invocation: true
---

# kiln-merge

This is gate ◆2, the last stop before main. Every fact on the gate comes from `kiln ready`, and the merge itself goes only through `kiln merge`, which refuses a red PR or a stale verdict no matter how sure anyone is. After the merge the issue closes through `Closes #N`, the binary clears its `kiln:*` labels, and the worktree and branch are removed.

## Reads

- `kiln ready <pr> --json`
- `.kiln/config.yaml`: `autonomy`, `gates`, `rigor.<size>.force_gates`
- the task journal's verify report, for any INCONCLUSIVE predicate
- `kiln status --json` afterwards, for `retro_queue`

## Steps

1. **Make sure the journal is pushed.** `kiln merge` deletes the branch and removes the worktree. The playbook's final kiln-log pass comes before this skill; confirm `git status --porcelain` is empty and `git log @{u}..` shows nothing unpushed.

2. **Ask the binary.**
   ```bash
   kiln ready <pr> --json
   ```

3. **If it's not ready** (exit 3), list each check with `ok: false` and route it. Don't merge.

   | Failing check | Go to |
   |---------------|-------|
   | CI pending only | `gh pr checks <pr> --watch`, then ask `kiln ready` again |
   | CI failing, conflicts, unresolved threads, changes requested | kiln-watch |
   | verdict missing or stale | kiln-verify, then kiln-review, on the current HEAD |
   | no `Closes #N` | fix the body with `gh pr edit <pr> --body-file` |
   | draft | `gh pr ready <pr>` |
   | INCONCLUSIVE | step 4 |

4. **INCONCLUSIVE needs a human.** Show which predicates are INCONCLUSIVE, their `why`, and what would unblock them. Only an explicit acceptance from the human ("accept P3", "merge anyway") counts. Then:
   ```bash
   kiln ready <pr> --accept-inconclusive --json
   ```
   Autopilot never passes `--accept-inconclusive` on its own. It stops and asks.

5. **Hold gate ◆2.** Stop when any of these hold:
   - `autonomy: manual`
   - `autonomy: gated` and `merge` is in `gates:`
   - `merge` is in `rigor.<size>.force_gates`
   - any predicate is INCONCLUSIVE

   Show the gate from the `kiln ready` JSON:
   ```
   ◆ gate 2 · merge PR #57?
     ✨ feat: add dark mode to settings (#42)
     ✓ checks 6/6 · ✓ threads resolved · ✓ 4/4 VERIFIED · ✓ review PASS on a1c9e0f
     next  squash merge · close #42 · remove worktree and branch
   ```
   Wait for an explicit yes. In autopilot with no forced stop and `ready: true`, go on.

6. **Merge from the main checkout.** The worktree is about to be removed, so don't run this from inside it:
   ```bash
   kiln -C <main checkout> merge <pr> --json
   # add --accept-inconclusive only if the human accepted in step 4
   ```
   Exit 3 means it refused. Show the reason and go back to step 3. Never fall back to `gh pr merge`.

7. **Read back the result.** From the JSON: `merged` must be `true`; report `merge_sha` and each `cleanup` line. Confirm the issue closed:
   ```bash
   gh issue view <n> --json state
   ```
   If the worktree wasn't removed (it had changes), say so and leave it. Never force-remove unmerged work.

8. **Suggest what's next.**
   ```bash
   kiln status --json
   ```
   - `retro_queue` of 5 or more: suggest `/kiln-retro`.
   - an L sub-issue: if every sibling listed in the parent's plan is closed, offer to close the parent (`gh issue close <parent>`).

## Writes

- the squash merge, through `kiln merge` only
- nothing in the journal: the branch and worktree are gone, and merged state lives on GitHub

## Rules

- Merges go only through `kiln merge`. Never `gh pr merge`, never a manual push to the default branch.
- Never merge with failing checks. That's a hard stop in every mode.
- INCONCLUSIVE merges only with the human's explicit acceptance, in every mode.
- Say "merged" only after `kiln merge` reports `merged: true`.
- Never force-remove a worktree or delete a branch the binary chose to keep.

## Done when

`kiln merge` returned `merged: true`, the issue is closed, and you've reported the merge SHA, the cleanup and any retro suggestion. Hand kiln-log the row; with no worktree left it is reported, not written:

```
kiln-merge · squash merged PR #57 · 3f2b7d1 · merged
```
