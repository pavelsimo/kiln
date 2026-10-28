---
name: kiln-build
description: Implements the approved plan test-first inside the task's worktree, one predicate at a time, committing small; bug fixes commit a failing reproduction before the fix and refactors pin behavior with characterization tests first. Use at the Fire stage after kiln-start, and again when kiln-verify or kiln-review sends work back.
disable-model-invocation: true
---

# kiln-build

This is the Fire stage. You turn the plan into code: a test that fails for the right reason, the smallest change that makes it pass, then a tidy up. Each loop ends in a small commit through kiln-commit. The issue stays at `kiln:in-progress`.

## Reads

- the issue predicates and the plan comment: `gh issue view <n> --json body,comments`
- `.kiln/memory/PROJECT.md`: test, lint, typecheck and build commands, conventions, **Do not touch without asking**
- `.kiln/memory/lessons.md`: `[strong]` lessons are rules here
- the task journal, for findings sent back by kiln-verify or kiln-review

## Steps

1. **Confirm where you are.**
   ```bash
   git rev-parse --abbrev-ref HEAD    # must print kiln/<n>-<slug>
   ```
   If not, `cd ../<repo>.kiln/<n>-<slug>` first. Never build in the main checkout.

2. **Take a baseline.** Run the project's test command once before changing anything. If it's red on a fresh branch, the failure isn't yours. Stop, report it, and note it for kiln-log's Retro notes. If it blocks the task, `kiln move <n> blocked --reason "<test> red on <default>"`.

3. **Work in loops, one predicate or plan step at a time.**
   1. **Red.** Write the test that proves this step. Run it and read the failure. It must fail on an assertion about the behavior, not on an import error or a typo.
   2. **Green.** Make the smallest change that passes it. Run it again.
   3. **Clean.** Tidy names and duplication with the tests green.
   4. **Commit** through `.kiln/skills/kiln-commit/SKILL.md`. One loop, one commit.

   Write tests the way kiln-verify will audit them: call the real code the way a user does, assert literal expected values, and make sure the test would fail if the feature returned nothing.

4. **Follow the type's first move.**
   - **Bug.** The first commit is the reproduction alone: a test that follows the repro steps and fails today. Run it and confirm it fails *because of the bug*. Commit it (`✅ test: reproduce <symptom>`) before you touch the fix. Then the fix (`🐛 fix: <what>`). If a pre-commit hook refuses the failing commit, stop and ask; never pass `--no-verify` on your own.
   - **Refactor.** First commit characterization tests that pin today's observable behavior, green on the old code (`✅ test: pin <behavior> before refactor`). Then move code in small steps, with the pinned tests green after every step.
   - **Feature.** Test-first per predicate, in the plan's order.
   - **Chore (light).** Make the change. Add a test only if behavior moves (and if behavior moves, it isn't a chore: tell the router).

5. **Stay inside the lines.**
   - Anything listed under Out of scope stays out. Note it as a follow-up instead.
   - A path under **Do not touch without asking**: stop and ask before editing.
   - If the approach has to change materially (new files, a different design), post a short update on the issue and, unless autonomy is autopilot, ask before going on.

6. **Fix-loop mode.** When kiln-verify (NOT VERIFIED) or kiln-review (FIX) sends you back, fix only the named predicates or findings. Add a test that fails without each fix. Don't expand scope while you're there.

7. **Finish green.** Run the full set from `PROJECT.md`: test, lint, typecheck, build. All must pass before you hand off to kiln-verify.

8. **Record lasting calls.** If you made an implementation choice that will outlive this task (a library, a data shape, a convention), write a **new** file `.kiln/memory/decisions/<YYYY-MM-DD>-<slug>.md` in the worktree using the format in `.kiln/skills/kiln-log/SKILL.md`, commit it with the code it explains, and hand kiln-log the path.

## Writes

- code and tests in the task's worktree, committed through kiln-commit
- optionally new decision records

## Rules

- No production code without a failing test first, except light chores.
- The bug reproduction commit always lands before the fix commit, and is never squashed into it.
- Never weaken, skip or delete a test to get green. Never edit a predicate.
- Never commit with `--no-verify` or disable a check without the human's yes.
- Small commits. If a diff is hard to describe in one subject line, it's two commits.

## Done when

All plan steps are done, the full test, lint, typecheck and build pass, and the working tree is clean. Your row (ref is HEAD):

```
kiln-build · 3 commits, 11 tests · a1c9e0f · ok
```

In fix-loop mode the action reads `fix P2 (loop 2)`.
