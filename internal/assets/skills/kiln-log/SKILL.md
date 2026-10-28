---
name: kiln-log
description: The only writer of a task journal at .kiln/memory/tasks/<n>-<slug>.md. Creates it at kiln-start, appends the row every other skill hands over, records the verify report, decisions, skips and retro notes, and commits it with "📝 docs(kiln): log <skill> for #<n>". Use at the end of every kiln skill, and for the final pass before gate 2.
disable-model-invocation: true
---

# kiln-log

This is the Temper stage, running a little after every other skill. Memory follows one writer per file, and the task journal's writer is you, only in that task's worktree. That is why parallel tasks never conflict on memory. You record what each skill did and why; live state (labels, CI, merged or not) is never copied here, because GitHub already holds it.

## Reads

- the row handed over: `skill · action · ref · result`, plus any report, decision path, skip or retro note
- the journal, if it exists
- the current branch: `git rev-parse --abbrev-ref HEAD`

## Steps

1. **Find the right place.** The journal is `.kiln/memory/tasks/<n>-<slug>.md` inside `../<repo>.kiln/<n>-<slug>`, and the branch there must be `kiln/<n>-<slug>`. Never write it from the main checkout or another task's worktree.
   - **No worktree yet** (kiln, kiln-recall, kiln-issue, kiln-plan): hold the rows. kiln-start's call creates the file with them.
   - **No worktree at all** (investigate playbook, or after `kiln merge` removed it): write nothing. Echo the row back so it appears in the final report.

2. **Create it** on kiln-start's call, exactly in this format:
   ```markdown
   ---
   issue: 42
   title: ✨ add dark mode to settings
   type: feature
   size: M
   playbook: feature
   branch: kiln/42-dark-mode-settings
   pr:
   ---
   ## Goal
   Users can switch Settings to a dark theme, and the choice sticks.
   ## Acceptance criteria
   - P1 a toggle in Settings → Appearance switches the page to dark
   - P2 the choice survives a reload
   - P3 a first visit follows the OS theme
   - P4 must not change: the light theme looks exactly as before
   ## Journal
   | ts | skill | action | ref | result |
   |----|-------|--------|-----|--------|
   | 2026-10-27 14:02 | kiln-issue | opened issue | #42 | ok |
   ## Verify report
   ## Decisions
   ## Skipped
   ## Retro notes
   ```
   For a release journal (`tasks/release-v<version>.md`), use `title`, `type: release`, `playbook: release`, `branch` and `pr`, and leave out `issue`.

3. **Append rows.** One table row per handed row, at the end of `## Journal`:
   ```
   | 2026-10-27 14:37 | kiln-verify | 5 predicates · fresh | a1c9e0f | VERIFIED |
   ```
   - `ts` is local time, `YYYY-MM-DD HH:MM`.
   - `ref`: issues and PRs as `#N`, commits as a 7+ character SHA, otherwise a branch or path.
   - kiln-verify rows: ref is the SHA that was verified, result exactly `VERIFIED`, `NOT VERIFIED` or `INCONCLUSIVE`. kiln-review rows: ref is the SHA, result exactly `PASS` or `FIX`. The binary reads these, and the latest row wins.
   - Replace any `|` inside a cell with `/`.
   - Append only. Never edit or delete an earlier row, even a wrong one; add a correcting row.

4. **Keep the frontmatter.** Never drop or rename a key. Fill `pr:` when kiln-pr hands you the number. Change `size` or `playbook` only when the router re-sized or re-routed, and add a row saying so.

5. **Fill the sections.**
   - `## Verify report`: replace the contents with the latest full report from kiln-verify, verbatim. Older reports live in git history; the rows keep every result.
   - `## Decisions`: append `- <one line> → decisions/<file>.md` for each record a skill wrote. You link them; you don't write them.
   - `## Skipped`: append `- skip: <skill> · <reason>` for every skip in the todo list, in the router's words.
   - `## Retro notes`: append one bullet per piece of friction worth a lesson: something that cost a loop, a surprise in the code, a flaky check, a bug class seen before (name both issues). kiln-retro reads these.

6. **Commit it.**
   ```bash
   git add .kiln/memory/tasks/<n>-<slug>.md
   git commit -m "📝 docs(kiln): log kiln-verify for #42"
   ```
   If the calling skill is about to commit code anyway (a kiln-build loop), stage the journal with that commit instead. After kiln-verify, always commit separately and never amend. The binary ignores `.kiln/memory/` when judging freshness, so logging after verify never makes the verdict stale. Stage decision files only when the skill that wrote them asks you to.

7. **Push once there's a PR.** If the branch has an upstream, `git push`. Before ◆2, everything must be pushed: `kiln merge` deletes the branch and worktree, and the squash takes the journal to main.

## Decision record format

Written as a **new** file by the skill that made the call, at `.kiln/memory/decisions/<YYYY-MM-DD>-<slug>.md`. Never edited afterwards; a later call supersedes it with a new file that links back.

```markdown
---
date: 2026-10-27
status: accepted
source: #42
---
# Theme via CSS custom properties
## Context
Why a choice was needed, in two or three lines.
## Decision
What was chosen, and what was not.
## Consequences
+ what gets easier
- what it costs, and any rule that now follows
```

## Writes

- `.kiln/memory/tasks/<n>-<slug>.md`, and nothing else

## Rules

- You are the only writer of task journals, and you write only in that task's worktree.
- Append-only rows. Frontmatter keys are kept.
- No live state: never record CI status, labels or merged state as facts. Rows say what a skill did and what it concluded.
- Commit messages are exactly `📝 docs(kiln): log <skill> for #<n>`, with no AI attribution.

## Done when

The row is in the table, the sections are updated, and the journal is committed (or staged into the caller's next commit). `git status --porcelain .kiln/memory/tasks/` prints nothing. You don't hand yourself a row.
