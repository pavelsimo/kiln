---
name: kiln-plan
description: Posts the implementation plan as an issue comment (approach, files, test plan, risks, predicates restated), splits size L work into sub-issues, moves the issue to kiln:planned, and holds gate 1 according to autonomy. Use at the Shape stage, after recall and explore, before any code is written.
disable-model-invocation: true
---

# kiln-plan

This is the Shape stage and the home of gate ◆1, the last cheap moment to change what "done" means. You post a plan the human can review on the issue, move it from `kiln:ready` to `kiln:planned`, and stop if autonomy says so. No code is written until the plan is approved or autonomy allows going on.

## Reads

- the issue: `gh issue view <n> --json number,title,body,labels,comments`
- kiln-recall and kiln-explore output from this run
- `.kiln/memory/decisions/` (calls already made) and `.kiln/memory/lessons.md` (`[strong]` lessons are rules)
- `.kiln/memory/PROJECT.md`: commands, conventions, **Do not touch without asking**
- `.kiln/config.yaml`: `autonomy`, `gates`, `rigor.<size>.force_gates`
- `.kiln/verify/DRIVER.md` and `features/`, to know which predicates can be proven live

## Steps

1. **Draft the plan.** Keep S plans to a few lines. M and L get each section in full.
   ```markdown
   ## Kiln plan · #42
   **Approach** Move colors to CSS custom properties on :root and swap them under
   [data-theme="dark"]. Toggle in Settings → Appearance, saved in localStorage.
   **Files** styles/tokens.css (new) · 14 components swap hex for tokens ·
   src/features/settings/AppearanceSection.tsx
   **Test plan**
   - P1 e2e: driver clicks theme-toggle, read data-theme (ladder 5)
   - P2 e2e: reload, read localStorage.theme (ladder 5)
   - P3 e2e with prefers-color-scheme: dark (ladder 4)
   - P4 light-theme visual snapshots, 0 diffs (ladder 5)
   **Risks** every visual snapshot changes once
   **Predicates** P1..P4 exactly as in the issue
   **Decision** CSS tokens over a React theme context (record after kiln-start)
   ```
   - Restate the predicates verbatim. If one is wrong or can't be proven, say so here and propose the change. Never quietly reword one.
   - For each predicate, name the check from kiln-verify's "match the check to the change" table. A unit test can't prove a UI predicate.
   - If a file is under **Do not touch without asking**, flag it in Risks. That forces a stop at ◆1.
   - If the work is bigger than its size label, say so and propose the new size.

2. **Split L work.** Each sub-issue is an M-or-smaller slice with its own predicates. Create each one by following `.kiln/skills/kiln-issue/SKILL.md`, adding `Part of #<parent>` to its Problem. List them in the plan as a task list (`- [ ] #52 limiter store`) with their order. Each sub-issue later gets its own branch, PR and ◆2.

3. **Post it.**
   ```bash
   gh issue comment <n> --body-file "$PLAN"
   ```
   A revision is a new comment headed `## Kiln plan · #<n> · rev 2` that starts with what changed. Keep the history visible.

4. **Move the issue.**
   ```bash
   kiln move <n> planned
   ```
   Exit 3 means the issue isn't `kiln:ready` (often still `needs-triage`). Fix the criteria first; don't force it.

5. **Hold gate ◆1.** Stop when any of these hold:
   - `autonomy: manual`
   - `autonomy: gated` and `plan` is in `gates:`
   - `plan` is in `rigor.<size>.force_gates` (always true for L)
   - a **Do not touch** path is in the plan

   At the stop, show the approach, files, tests and risks in a few lines, link the comment, and ask "Approve?". Then:
   - **approved** ("ok", "yes", "approved"): continue.
   - **changes requested**: revise, post rev N, ask again. If the human changed a predicate, edit the issue body with `gh issue edit <n> --body-file` and note it in the revision.
   - **rejected**: `kiln move <n> ready` and stop.

   In autopilot with no forced gate, continue and record `auto` as the result.

6. **Decision record, later.** If the plan made a call that will outlive the task, you write it, but in the task's worktree, so the playbook has a step for it right after kiln-start. Create `.kiln/memory/decisions/<YYYY-MM-DD>-<slug>.md` as a **new** file, using the format in `.kiln/skills/kiln-log/SKILL.md`, and hand kiln-log the path.

## Writes

- a plan comment on the issue (and sub-issues for L)
- the `kiln:planned` label, through `kiln move`
- at most one new decision record, in the worktree, after kiln-start

## Rules

- No code before ◆1 is answered or autonomy waives it. L plans always need a human.
- Predicates are restated, never loosened. Changing one needs the human's explicit yes at this gate.
- One plan per issue, revised by new comments, never by deleting old ones.
- Never edit an existing decision record. Supersede it with a new one that links back.

## Done when

The plan comment is on the issue, `kiln move` succeeded, and ◆1 has an answer (or autonomy skipped it). Hand kiln-log:

```
kiln-plan · posted plan · #42 · approved
```

The result is `approved`, `auto`, `changes requested`, or `rejected`. For L, add one row per sub-issue: `kiln-plan · opened sub-issue · #52 · ok`.
