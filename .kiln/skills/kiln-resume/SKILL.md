---
name: kiln-resume
description: Picks up an interrupted kiln task from its task journal, branch, worktree and PR, and continues from the first playbook step that is not done. Use after a crash, a closed laptop, an agent restart, or when moving a task to a different agent.
disable-model-invocation: true
---

# kiln-resume

Continues a task that stopped partway. The journal is the record of what happened, so you trust it over your own guesses, check it against git and GitHub, and carry on from the first unchecked playbook step. Works at any stage.

## Reads

- `kiln status --json` (worktree, PR, state and `next` for the issue)
- `.kiln/memory/tasks/<n>-<slug>.md` in the task's worktree (frontmatter, journal rows, Skipped)
- `.kiln/skills/kiln/playbooks/<playbook>.md` (the step list)
- `.kiln/config.yaml` (autonomy and gates)
- git state of the branch, and the PR if there is one

## Steps

1. Find the task. With an issue number, look it up in the floor:

   ```bash
   kiln status --json
   ```

   Take `worktree`, `pr`, `state` and `next` from the matching `in_flight` entry. Without a number, list the in-flight items and ask which one.

2. Get into the worktree. If the folder exists, `cd` into it. If it does not (new machine, or it was removed) but the branch exists on the remote, recreate it:

   ```bash
   git fetch origin
   git worktree add ../<repo>.kiln/<n>-<slug> kiln/<n>-<slug>
   ```

   No branch anywhere means the task never got past kiln-start: resume from the playbook step before kiln-start.

3. Read the journal `.kiln/memory/tasks/<n>-<slug>.md`: the `playbook` in the frontmatter, every journal row, and `## Skipped`.

4. Rebuild the todo list. Copy the playbook's steps from `.kiln/skills/kiln/playbooks/<playbook>.md` word for word, then mark:
   - ☑ a step with a journal row whose result is ok, approved, VERIFIED or PASS
   - ☒ a step listed under `## Skipped` (keep its `skip: <reason>`)
   - ☐ everything else

   The first ☐ is where you resume. A FIX or NOT VERIFIED row means the work went back to kiln-build, so resume there.

5. Check the journal against reality, without overwriting it:

   ```bash
   git status --short
   git log --oneline origin/<default>..HEAD
   kiln verdict <n> --json
   gh pr view <pr> --json state,headRefOid,isDraft
   ```

   - Uncommitted changes are unfinished kiln-build work. Keep them; never `git reset` or `git checkout .` them away.
   - Commits newer than the last journal row: the build got further than the journal says. Resume at kiln-commit or kiln-verify, not at the start of kiln-build.
   - `kiln verdict` says stale: the verify or review row no longer counts, so mark that step ☐ again.
   - The PR is MERGED: nothing to build. Resume at the playbook's Temper steps.

6. Show what you found before doing anything:

   ```
   reading tasks/52-limiter-store.md · branch kiln/52-limiter-store · no PR yet
   ✓ last finished step: kiln-build · 2 commits · 9 of 12 tests green
   → continuing kiln-build: the 3 failing tests in limiter.store.test.ts
   ```

7. Continue by reading `.kiln/skills/<next-skill>/SKILL.md` and following it. Respect autonomy: if the next step is a gate (◆1 plan, ◆2 merge), stop there and wait. In manual mode, report and stop after the one step.

## Writes

Nothing directly. The skill you resume writes its own outputs, and kiln-log records the resume row.

## Rules

- The journal wins over memory of the conversation. If the journal and git disagree, say so and follow git for code state and the journal for decisions.
- Never discard uncommitted or unpushed work to "start clean".
- Never re-run a step that has a fresh, successful journal row just to be safe.
- Labels change only through `kiln move`. If the label is behind the journal (for example still `kiln:ready` after kiln-start ran), fix it with the legal `kiln move` and note it.

## Done when

The todo list is rebuilt, the resume point is shown, and the next skill has started (or you stopped at a gate).

Journal row for kiln-log:

```
kiln-resume · resumed at <skill> · #<n> · ok
```
