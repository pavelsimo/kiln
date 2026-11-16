---
issue: 3
title: ✨ add hooks on kiln move and kiln merge
type: feature
size: M
playbook: feature
branch: kiln/3-hooks-move-merge
pr: 5
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
| 2026-11-16 23:23 | kiln-resume | resumed at kiln-build, rebased on main after #4 | #3 | ok |
| 2026-11-16 23:23 | kiln-build | fix P2 (loop 3): MainWorktree asks git when the checkout is the main one | 9811beb | ok |
| 2026-11-16 23:23 | kiln-verify | 12 predicates · self | 9811beb | VERIFIED |
| 2026-11-16 23:32 | kiln-review | independent reviewer · 1 major (hook timeout left children running), 7 minor or nit | 9811beb | FIX |
| 2026-11-16 23:32 | kiln-build | fix review finding (loop 2): kill the hook's process group on timeout, reject multi-document hook files | 9f24a1a | ok |
| 2026-11-16 23:32 | kiln-verify | 12 predicates · self · after review fix | 9f24a1a | VERIFIED |
| 2026-11-16 23:32 | kiln-review | independent reviewer · re-review, 0 blockers, 0 majors | 9f24a1a | PASS |
| 2026-11-16 23:33 | kiln-pr | opened ready PR, moved to review | #5 | ok |
## Verify report
```
kiln-verify · #3 · size M · checker: self · after the review fix, rebased on main
gates          make ci (fmt-check · lint · test · build)   all exit 0
P1  move hook runs once after the label change, KILN_* set, report on stdin   VERIFIED  4
    ran  go test ./cmd -run TestMoveRunsHooks (real binary, fake gh on PATH)
P2  local hooks run after user ones; kiln init gitignores the file           VERIFIED  5
    ran  real kiln move 9 in-progress with a user hook and a local hook, in four layouts
    saw  main checkout, task worktree, separate git dir, submodule: order "user local", 2 hooks ok, exit 0
         evidence/3/r3/P2-layouts.txt · TestInitGitignoresLocalHooks · TestMainWorktree fails without the fix
P3  merge hooks after merge, before cleanup, KILN_PR and KILN_MERGE_SHA    VERIFIED  4
    ran  go test ./cmd -run TestMergeRunsHooksBeforeCleanup
P4  failing and timed-out hooks warn, exit code unchanged, hooks in JSON    VERIFIED  4
    ran  go test ./cmd -run TestMoveSurvivesFailingHooks · go test ./internal/hooks
    ran  real kiln move, hook "sleep 7.3141; true", timeout 1s, stderr piped · 1006ms, no child left
         evidence/3/r3/P4-timeout-kills-children.txt · TestTimeoutKillsWhatTheHookStarted fails without the fix
P5  hook output never reaches stdout                                        VERIFIED  4
    ran  go test ./cmd -run TestMoveSurvivesFailingHooks (stdout decoded as JSON)
P6  no hooks on dry-run, exit 3, --no-hooks, KILN_NO_HOOKS                  VERIFIED  4
    ran  go test ./cmd -run 'TestMoveSkipsHooks|TestMergeSkipsHooks'
P7  doctor lists hook files, fails on a broken one                          VERIFIED  4
    ran  go test ./cmd -run TestDoctorListsHookFiles
P8  no hook files: move and merge output unchanged                          VERIFIED  4
    ran  go test ./cmd -run 'TestMoveWithoutHooksIsUnchanged|TestMergeWithoutHooksIsUnchanged'
P9  config.yaml format unchanged                                            VERIFIED  4
    ran  git diff origin/main --stat -- internal/assets/templates/kiln/config.yaml internal/config · empty
P10 hooks documented with herdr, tmux and notify-send recipes               VERIFIED  4
    ran  grep docs/cli.html · herdr 2, tmux 5, notify-send 4 · tmux recipe rerun under a pty on a private tmux server, window created, ok true
C1  ci-gate (now includes TestSkillsMatchCLI) · C2 docs-no-dashes           VERIFIED  4
tests audited  35 passing tests in cmd, hooks, install, gitx · the cmd tests drive the real binary
findings       advisory, not part of the verdict
    F1  from a linked worktree of a separate-git-dir or submodule repo, the main checkout is still
        taken as the parent of the git dir, so a local hook file there would not be found.
        git worktree list reports the same wrong path, so git offers no direct answer.
        follow-up  🐛 fix: find local hooks from a worktree of a submodule or separate-git-dir repo
    F2  a hook that reads or reconfigures the terminal (sudo or ssh prompt, fzf) is stopped by the
        kernel and reports a timeout, because hooks now run in their own process group. From the review.
        follow-up  📝 docs: say hooks can't use the terminal, or ignore SIGTTOU and SIGTTIN in the hook
    F3  KILN_NO_HOOKS=0 also disables hooks · timeout 0s means the default · an existing wider
        .gitignore pattern still gets the exact line appended. Review nits, none block.
overall        VERIFIED on 9f24a1a
evidence       .kiln/verify/evidence/3/r3/   (gitignored)
```
## Decisions
- hooks live in personal files only and never veto a transition → decisions/2026-10-30-hooks-personal-no-veto.md
## Skipped
- skip: kiln-verify swarm · size M, verdict from step 9 covers it
- skip: kiln-retro · retro_queue 0
## Retro notes
- The independent reviewer found a real major the verify rounds never looked for: a hook timeout killed only the shell, so a piped caller waited for the child. Both timeout tests used a bare `sleep`, which the shell execs in place. A timeout test needs a command the shell has to fork for.
- Loop 2's checker suggested resolving the main checkout from the first `git worktree list` entry. Probing git 2.56 showed that entry is wrong in exactly the layouts it was meant to fix (it prints the git dir). Check a suggested fix against real git before building on it.
- The loop 2 fix regressed layouts that worked before it. The smallest repair was to ask git for the top level when git-dir equals git-common-dir.
- gitx.MainWorktree (dirname of --git-common-dir) is wrong for submodules and --separate-git-dir checkouts. setup() uses it for the config fallback too, so that path has the same latent bug.
- Gitignored per-repo files only exist in the main checkout, but kiln commands mostly run from task worktrees. Anything that reads a local, untracked file needs the MainWorktree fallback that setup() uses for config. The fresh checker caught it; the author's tests all ran from the main checkout.
- The fake `gh` on PATH drove real `kiln move` and `kiln merge` end to end (merge pushes to a local bare repo via `remote.origin.pushurl`). DRIVER.md says GitHub commands can't run in the sandbox; this pattern says they can.
- `git worktree add -b ... origin/main` set the new branch to track origin/main, so a bare `git push` would have targeted main. kiln-start could pass `--no-track`.
