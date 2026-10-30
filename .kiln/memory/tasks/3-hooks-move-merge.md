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
| 2026-10-30 23:52 | kiln-build | 6 commits, 22 tests | 53adefa | ok |
| 2026-10-30 23:52 | kiln-commit | 6 commits, lint fixes folded in | 53adefa | ok |
| 2026-10-31 00:03 | kiln-verify | 13 predicates, fresh checker | f64bb7a | NOT VERIFIED |
| 2026-10-31 00:04 | kiln-build | fix P2 (loop 2) | 6221580 | ok |
| 2026-10-31 00:09 | kiln-verify | 13 predicates, fresh checker, after fix loop 2 | 2b765c7 | NOT VERIFIED |
## Verify report
```
kiln-verify · #3 · size M · checker: fresh · re-verify after fix loop 2
gates          make fmt-check · lint · test · build   all exit 0
P1  move hook runs once after the label change, KILN_* set, report on stdin   VERIFIED  5
P2  local hooks run after user ones; kiln init gitignores the file           NOT VERIFIED  5
    saw  main checkout: user then local · task worktree: fixed, local file found, doctor lists it
         separate-git-dir and submodule checkouts: local hooks silently skipped (worked at f64bb7a)
    why  gitx.MainWorktree is filepath.Dir(--git-common-dir), which isn't the work tree in those layouts
    unblock  resolve the main checkout from the first `git worktree list` entry, add a submodule test; or rule those layouts out of scope
P3  merge hooks after merge, before cleanup, KILN_PR and KILN_MERGE_SHA    VERIFIED  5
P4  failing and timed-out hooks warn, exit code unchanged, hooks in JSON    VERIFIED  5
P5  hook output never reaches stdout                                        VERIFIED  5
P6  no hooks on dry-run, exit 3, --no-hooks, KILN_NO_HOOKS                  VERIFIED  5
P7  doctor lists hook files, fails on a broken one                          VERIFIED  5
P8  no hook files: move and merge identical to origin/main (18/18 + worktree) VERIFIED  5
P9  config.yaml format unchanged, base binary reads a HEAD install          VERIFIED  5
P10 hooks documented, tmux recipe run live, background advice checked       VERIFIED  5
C1  ci-gate · C2 docs-no-dashes · C3 skills-match-cli                       VERIFIED  4
tests audited  21 used · 0 rejected · no test covers submodule or separate-git-dir layouts
overall        NOT VERIFIED on 2b765c7
evidence       .kiln/verify/evidence/3/r2/   (gitignored)
```
## Decisions
- hooks live in personal files only and never veto a transition → decisions/2026-10-30-hooks-personal-no-veto.md
## Skipped
- skip: kiln-verify swarm · size M, verdict from step 9 covers it
- skip: kiln-retro · retro_queue 0
## Retro notes
- gitx.MainWorktree (dirname of --git-common-dir) is wrong for submodules and --separate-git-dir checkouts. setup() uses it for the config fallback too, so that path has the same latent bug.
- Gitignored per-repo files only exist in the main checkout, but kiln commands mostly run from task worktrees. Anything that reads a local, untracked file needs the MainWorktree fallback that setup() uses for config. The fresh checker caught it; the author's tests all ran from the main checkout.
- The fake `gh` on PATH drove real `kiln move` and `kiln merge` end to end (merge pushes to a local bare repo via `remote.origin.pushurl`). DRIVER.md says GitHub commands can't run in the sandbox; this pattern says they can.
- `git worktree add -b ... origin/main` set the new branch to track origin/main, so a bare `git push` would have targeted main. kiln-start could pass `--no-track`.
