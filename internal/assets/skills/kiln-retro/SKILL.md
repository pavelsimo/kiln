---
name: kiln-retro
description: Reads the task journals merged since the last retro, turns recurring friction into lessons with confidence levels, graduates strong checkable lessons into standing checks, proposes edits to PROJECT.md and kiln's own skills, runs kiln-verifier maintain when kiln doctor reports driver drift, and ships everything as one PR. Use after a batch of merges, when kiln status shows lessons queued, or on request.
disable-model-invocation: true
---

# kiln-retro

Closes the loop. Each task leaves Retro notes and a trail of skips, FIX rows and retries; the retro reads them together and changes what kiln does next time. Nothing in memory changes behind anyone's back: every edit lands in one PR you review. This is the Temper stage.

## Reads

- `kiln status --json` (`retro_queue`)
- `.kiln/memory/lessons.md` (its `last_retro` frontmatter and current lessons)
- `.kiln/memory/tasks/*.md` on the default branch whose latest journal row is after `last_retro`
- `.kiln/memory/PROJECT.md`, `.kiln/verify/checks.yaml`, `.kiln/skills/*/SKILL.md`
- `kiln doctor` output

## Steps

1. Check there is something to do, and set up a worktree from a fresh default branch:

   ```bash
   kiln status --json          # retro_queue
   git fetch origin
   git worktree add ../<repo>.kiln/retro-<date> -b kiln/retro-<date> origin/<default>
   cd ../<repo>.kiln/retro-<date>
   ```

   If `retro_queue` is 0 and no journal has rows after `last_retro`, say so and stop. No empty PRs.

2. Read each journal since `last_retro`. Collect:
   - every bullet under `## Retro notes`
   - every `## Skipped` line (a skip that keeps recurring may mean a playbook step is wrong, or that it was skipped when it should not have been)
   - friction rows: FIX, NOT VERIFIED, INCONCLUSIVE, blocked, CI retries

3. Group the friction by cause across tasks, and propose lessons in lessons.md's format:

   ```markdown
   - [strong] Run `pnpm test:e2e --update` after any token change; visual snapshots fail otherwise. (#31, #42, #49)
   - [recurring] The `settings` API mock lives in test/fixtures, not __mocks__. (#31, #42)
   - [candidate] CI's Node 20 job is flaky on the websocket test. Retry once before treating it as real. (#47)
   ```

   Confidence: `candidate` seen once · `recurring` seen in 2 to 3 tasks · `strong` a rule (seen 3 or more times, or confirmed by a person). Promote an existing lesson when it shows up again and append the new issue refs. Propose removing a lesson the journals now contradict. Every lesson keeps its refs.

4. Graduate strong lessons that can be checked (a command with an exit code, or an observable you can read back) into `.kiln/verify/checks.yaml`:

   ```yaml
   - id: tokens-only
     when: { paths: ["src/**/*.css", "src/**/*.tsx"] }
     predicate: pnpm lint:tokens exits 0
     blocking: true
     from: lessons.md · strong · #31 #42 #49
   ```

   Run the predicate once on this branch (it should pass on the default branch) and confirm each glob matches files with `git ls-files ':(glob)<glob>'`. A lesson that cannot be checked stays a lesson.

5. Propose PROJECT.md edits the journals justify: a command that changed, a new convention, a new "Do not touch without asking" area.

6. Propose edits to kiln's own skills in `.kiln/skills/<name>/SKILL.md` when the friction is in kiln's procedure (a step that keeps being skipped, a check that keeps missing the same class of bug). Keep each edit small and say which journals motivated it. Note in the PR body that `kiln update` keeps an edited skill as is and writes the new upstream version next to it as `SKILL.md.kiln-new` for a manual merge.

7. Check the driver:

   ```bash
   kiln doctor
   ```

   If it reports the verify driver is behind, read `.kiln/skills/kiln-verifier/SKILL.md` and run it in maintain mode on this branch. Its map fixes join this PR; product gaps become their own issues.

8. Set `last_retro: <today>` in lessons.md's frontmatter. Do this even when no lesson changed, so the queue clears.

9. Hand kiln-log the row below (it records the retro in `.kiln/memory/tasks/retro-<date>.md` on this branch), then commit by explicit path and open one ready PR:

   ```bash
   git add .kiln/memory/lessons.md .kiln/memory/PROJECT.md .kiln/verify/checks.yaml .kiln/verify/DRIVER.md .kiln/verify/features .kiln/skills .kiln/memory/tasks/retro-<date>.md
   git status --short   # only what you meant to change; drop paths you did not touch
   git commit -m "📝 docs(kiln): retro after #<first> to #<last>"
   git push -u origin kiln/retro-<date>
   gh pr create --base <default> --title "📝 docs(kiln): retro after #<first> to #<last>" --body-file <body>
   ```

   The body has one section per kind of change (lessons, checks, PROJECT.md, skills, driver), each change with the journals behind it, plus the list of journals read.

## Writes

- `.kiln/memory/lessons.md` (this skill is its only writer, always through this PR)
- `.kiln/verify/checks.yaml` and `.kiln/memory/PROJECT.md` (after onboarding, only through this PR)
- `.kiln/skills/*/SKILL.md` edits, proposed in this PR
- DRIVER.md and features/ only through kiln-verifier maintain
- One PR on `kiln/retro-<date>`

## Rules

- One PR per retro, everything in it. Never commit memory changes straight to the default branch.
- Never write a task journal; that is kiln-log's file. Read only.
- A lesson without issue refs does not go in. A strong lesson needs the evidence to back it.
- A standing check must run green on the default branch before it is proposed.
- Stage explicit paths. No Co-authored-by trailers or AI attribution in the commit or PR.
- The PR has no issue, so `kiln ready` treats it as a housekeeping PR: only GitHub's checks gate it. It changes what every future task reads, so it always waits for a human yes before `kiln merge <pr>`, in every autonomy mode.

## Done when

The retro PR is open (or you reported there was nothing to retro), `last_retro` is today in the PR, and driver drift from `kiln doctor` was handled.

Journal row for kiln-log:

```
kiln-retro · <n> journals: <l> lessons, <c> checks, <s> skill edits · #<pr> · ok
```
