---
name: kiln-start
description: Claims an issue by creating its git worktree and kiln/<n>-<slug> branch from a fresh default branch, assigning you, moving it to kiln:in-progress, and creating the task journal through kiln-log. Use at the start of the Fire stage, once the plan is approved or skipped, and only while in-flight work is below max_in_flight.
disable-model-invocation: true
---

# kiln-start

This is the first step of the Fire stage. You give the issue its own checkout next to the repo, so the human's checkout is never touched and several tasks can build side by side. The issue moves to `kiln:in-progress`, and its task journal is born.

## Reads

- `kiln status --json`: `in_flight` (how many tasks are running, and whether this one already is)
- `.kiln/config.yaml`: `worktrees.dir` (default `../{repo}.kiln`), `worktrees.max_in_flight`, `conventions.branch`
- the issue title, for the slug
- `.kiln/memory/PROJECT.md`: an install or setup command, if listed

## Steps

1. **Check capacity.**
   ```bash
   kiln status --json
   ```
   - If this issue is already in `in_flight`, it has a worktree. Stop and suggest `/kiln-resume <n>`.
   - If `in_flight` has `max_in_flight` or more entries, stop. Report what's running and what's waiting on the human, and suggest finishing one first. Never raise the limit yourself.

2. **Find the names.**
   ```bash
   gh repo view --json name,defaultBranchRef -q '.name + " " + .defaultBranchRef.name'
   ```
   The slug comes from the issue title: drop the gitmoji, lowercase, keep three to five meaningful words, join with `-`, at most 40 characters. `✨ add dark mode to settings` becomes `dark-mode-settings`.

3. **Fetch and create the worktree.** Run from the main checkout's root:
   ```bash
   git fetch origin --prune
   git worktree add ../<repo>.kiln/<n>-<slug> -b kiln/<n>-<slug> origin/<default>
   ```
   If the branch already exists locally or on `origin`, stop. That's earlier work: suggest `/kiln-resume <n>` instead of creating a second branch.

4. **Claim the issue.**
   ```bash
   gh issue edit <n> --add-assignee @me
   kiln move <n> in-progress
   ```
   `kiln move` accepts this from `ready` or `planned`. Exit 3 means the issue is somewhere else (still in triage, or already in review): report it and stop. Don't undo the worktree; tell the human.

5. **Move in.** Every later step for this task runs inside `../<repo>.kiln/<n>-<slug>`. If `PROJECT.md` lists an install command, run it there now so the first test run isn't a setup failure.

6. **Create the journal.** Hand kiln-log everything it needs to create `.kiln/memory/tasks/<n>-<slug>.md`:
   - frontmatter: `issue`, `title`, `type`, `size`, `playbook`, `branch` (no `pr` yet)
   - the router's goal sentence for `## Goal`
   - the predicates from the issue for `## Acceptance criteria`
   - the rows held so far (kiln, kiln-recall, kiln-issue, kiln-plan), then your own row
   - every `skip:` line in the todo list so far

7. **Pending decision.** If kiln-plan flagged a decision record, the next todo item writes it now, in this worktree.

## Writes

- the worktree `../<repo>.kiln/<n>-<slug>` and the branch `kiln/<n>-<slug>`
- the assignee and the `kiln:in-progress` label (through `kiln move`)
- the journal, through kiln-log only

## Rules

- Always branch from a freshly fetched `origin/<default>`, never from whatever is checked out.
- One issue, one branch, one worktree. Never reuse another task's worktree.
- Respect `max_in_flight`.
- Never touch the human's main checkout beyond `git fetch` and `git worktree add`.

## Done when

`git -C ../<repo>.kiln/<n>-<slug> rev-parse --abbrev-ref HEAD` prints `kiln/<n>-<slug>`, the issue shows `kiln:in-progress` and your assignment, and the journal exists. Your row:

```
kiln-start · worktree ../app.kiln/42-dark-mode-settings · kiln/42-dark-mode-settings · ok
```
