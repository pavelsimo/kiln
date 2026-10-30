---
issue: 3
title: ✨ add hooks on kiln move and kiln merge
type: feature
size: M
playbook: feature
branch: kiln/3-hooks-move-merge
pr:
---
## Goal
kiln users can attach their own shell commands to task transitions (kiln move, kiln merge), so tools like herdr, tmux or notify-send react to kiln without kiln knowing about them.
## Acceptance criteria
- P1 with a user-level `hooks.yaml` holding an `on: move, to: in-progress` hook, `kiln move <n> in-progress` runs that command once after the label changes, with `KILN_EVENT`, `KILN_ISSUE`, `KILN_TITLE`, `KILN_FROM`, `KILN_TO`, `KILN_REASON`, `KILN_REPO`, `KILN_BRANCH` and `KILN_WORKTREE` set, and the command's `--json` report on stdin
- P2 hooks in `.kiln/hooks.local.yaml` run too, after the user-level ones, and `kiln init` adds that file to `.gitignore`
- P3 `kiln merge <pr>` runs `on: merge` hooks after the merge is confirmed and before the worktree is cleaned up, with `KILN_PR` and `KILN_MERGE_SHA` also set
- P4 a hook that exits non-zero or runs past its timeout (10s default, `timeout:` per hook) prints a warning on stderr; kiln's exit code stays what it would have been, and the `--json` report carries `hooks: [{run, ok, error}]`
- P5 a hook's own output never reaches kiln's stdout: `kiln move ... --json` stays valid JSON when a hook prints text
- P6 hooks don't run on `--dry-run`, on a refused move or merge (exit 3), with `--no-hooks`, or with `KILN_NO_HOOKS=1`
- P7 `kiln doctor` lists the hook files it found and how many hooks each has, and fails on a hook file it can't parse
- P8 must not change: with no hook files, `kiln move` and `kiln merge` print exactly what they print today, in both human and `--json` output
- P9 must not change: `.kiln/config.yaml` keeps its current format, so older kiln binaries still read it
- P10 `docs/cli.html` documents hooks, with recipes for herdr, tmux and notify-send
## Journal
| ts | skill | action | ref | result |
|----|-------|--------|-----|--------|
| 2026-10-30 23:30 | kiln | routed feature, size M, playbook feature | #3 | ok |
| 2026-10-30 23:31 | kiln-recall | 0 lessons, 0 decisions, 0 tasks | #3 | ok |
| 2026-10-30 23:34 | kiln-explore | mapped move, merge, config, doctor, install · 9 files | #3 | ok |
| 2026-10-30 23:36 | kiln-issue | opened issue | #3 | ok |
| 2026-10-30 23:40 | kiln-plan | posted plan | #3 | approved |
| 2026-10-30 23:44 | kiln-start | worktree ../kiln.kiln/3-hooks-move-merge | kiln/3-hooks-move-merge | ok |
## Verify report
## Decisions
- hooks live in personal files only and never veto a transition → decisions/2026-10-30-hooks-personal-no-veto.md
## Skipped
- skip: kiln-verify swarm · size M, verdict from step 9 covers it
- skip: kiln-retro · retro_queue 0
## Retro notes
- `git worktree add -b ... origin/main` set the new branch to track origin/main, so a bare `git push` would have targeted main. kiln-start could pass `--no-track`.
