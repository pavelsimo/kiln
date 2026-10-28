---
name: kiln-review
description: Reviews a task branch with depth set by its size label (S self-review checklist, M fresh-context reviewer subagent, L adversarial reviewers whose findings need a reproducing command) and returns PASS or FIX for the exact HEAD SHA. Use in Glaze after kiln-verify, or on its own for a second opinion on a branch.
disable-model-invocation: true
---

# kiln-review

Asks whether the change is any good, with rigor proportional to its size. The verdict is tied to a commit SHA so the binary can tell later whether it still holds. This is a Glaze stage skill and runs after kiln-verify.

## Reads

- The issue (`gh issue view <n>`), especially the acceptance criteria
- The diff: `git diff origin/<default>...HEAD`
- `.kiln/memory/PROJECT.md`, and `.kiln/memory/lessons.md` for the S checklist
- The task journal frontmatter for `size` (fall back to the `size:*` label)
- `kiln verdict <n> --json`, `kiln status --json`

## Steps

1. Start clean and pin the SHA. Uncommitted changes are not reviewable; send them to kiln-commit first.

   ```bash
   git status --short            # must be empty
   git fetch origin
   git rev-parse --short=7 HEAD  # the reviewed SHA
   kiln verdict <n> --json
   ```

   If `review.fresh` is true and `review.result` is PASS, report that and stop; there is nothing new to review.

2. Prepare the inputs. Leave the journal out of the diff:

   ```bash
   mkdir -p .kiln/verify/evidence/<n>    # gitignored
   git diff origin/<default>...HEAD -- . ':(exclude).kiln/memory' > .kiln/verify/evidence/<n>/review.diff
   gh issue view <n> --json title,body
   ```

3. Review by size.

   **S: self-review checklist.** Answer every item with a `file:line` or "n/a":
   - the diff does what the acceptance criteria say, and nothing outside scope
   - new behavior has a test that would fail without the change
   - errors are handled, not swallowed; no silent retries or blanket nil checks
   - no debug output, commented-out code or stray files
   - no secrets, tokens or personal data
   - PROJECT.md conventions followed; no "Do not touch without asking" path changed without approval
   - every `[strong]` lesson in lessons.md respected
   - no public interface changed (if one did, the task is not S: review it as M)

   **M: fresh-context reviewer.** Start a new subagent with no conversation history. Give it only the diff, the issue and PROJECT.md. Not the plan, not the journal, not how it was built. Prompt:

   > Review this diff against the issue's acceptance criteria and the project conventions. Report each finding as: severity (blocker, major, minor, nit), `file:line`, what is wrong, and why it matters. Report a blocker or major only if you can point at the exact code. End with PASS or FIX.

   **L: adversarial reviewers.** Start 3 fresh subagents in parallel with the same inputs and one lens each: correctness and edge cases; security, data and migrations; regressions against the `must not change` predicates and existing callers. Tell each to try to break the change and to attach to every finding a command that reproduces it (a test, a `curl`, a script) with expected and actual output. Then rerun every command yourself on the reviewed SHA. A finding counts only if it reproduces. List the rest as "not counted: did not reproduce".

4. Decide:
   - **PASS**: no standing blocker or major finding. Minors and nits go in the report.
   - **FIX**: at least one blocker or major that you confirmed (and, at L, reproduced).

   You may disagree with a finding only by showing why with a `file:line` or a command. Write that reasoning in the report.

5. On FIX, send the work back:
   - hand the findings list to kiln-build
   - if `kiln status --json` shows the issue in state `review`, run `kiln move <n> in-progress` (if it is still `in-progress`, leave the label alone)
   - after the fix, the loop is kiln-build, kiln-commit, kiln-verify, kiln-review

## Writes

- A review report in the conversation (findings with severity and `file:line`, reproducing commands at L)
- The journal row (through kiln-log) with the verdict for the reviewed SHA
- On FIX only: `kiln move <n> in-progress` when the issue was in review

## Rules

- Never review uncommitted work. The verdict belongs to one SHA.
- The M and L reviewers get only the diff, the issue and PROJECT.md. Giving them the build conversation defeats the point.
- At L, a finding without a reproducing command that you reran does not count.
- Never change code inside kiln-review. Fixes belong to kiln-build, so verify and review run again on the new SHA.
- Never loosen acceptance criteria to reach PASS.

## Done when

You have a PASS or FIX for the HEAD SHA, with the findings shown, and on FIX the work is handed back to kiln-build.

Journal row for kiln-log (ref is the reviewed SHA, at least 7 characters):

```
kiln-review · <self-review checklist | reviewer subagent | 3 adversarial reviewers> · <sha> · PASS|FIX
```
