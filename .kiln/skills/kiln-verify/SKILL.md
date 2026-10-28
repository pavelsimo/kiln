---
name: kiln-verify
description: Proves a change on the real surface it describes. Turns the issue's acceptance criteria and matching standing checks into predicates, picks the check that fits each surface, drives the running app through .kiln/verify/DRIVER.md when needed, audits every test it cites, and gives each predicate VERIFIED, NOT VERIFIED or INCONCLUSIVE with an evidence-ladder step, tied to a commit SHA. Use at the Glaze stage before review, or on its own to prove any branch.
disable-model-invocation: true
---

# kiln-verify

This is the Glaze stage. Done means every predicate was checked on the thing it describes: the running app, the real command, a value read back. A green build is where you start, not where you stop. You produce one verdict per predicate, an overall verdict that is the weakest of them, and a report tied to the exact commit you checked. The issue stays at `kiln:in-progress`; you move no labels and change no code.

## Reads

- the predicates: `gh issue view <n> --json body` → `P1..Pn` under Acceptance criteria
- the plan comment's test plan
- standing checks: `kiln checks --json`
- the diff: `git diff origin/<default>...HEAD`
- `.kiln/verify/DRIVER.md` and `.kiln/verify/features/*.md`
- `.kiln/config.yaml`: `rigor.<size>.verify`, `verify.live` (default `[ui, api]`)
- `.kiln/memory/PROJECT.md`: test, lint, typecheck, build commands

## Steps

### 1. Pin what you're checking

```bash
git rev-parse HEAD
git status --porcelain
```

The tree must be clean. You verify a commit, not a working copy. Uncommitted changes go through kiln-commit first. Every verdict you give belongs to this SHA.

### 2. Collect the predicates

- `P1..Pn` copied from the issue **verbatim**.
- Standing checks matched by the binary, passing the issue's type label:
  ```bash
  kiln checks --json --labels type:feature
  ```
  Each entry in `checks` becomes `C1..Cn` with its `predicate` text. `blocking: true` checks count toward the overall verdict; non-blocking ones are reported as advisory.

Never loosen, merge, reinterpret or drop a predicate to make it pass. If one is wrong or impossible to check, report it as INCONCLUSIVE with that reason. Changing it is the human's call, made on the issue.

### 3. Decide who checks, by size

| Size | `rigor.<size>.verify` | Who |
|------|------|-----|
| S | self | you, the author, in this session |
| M | fresh | a fresh-context checker |
| L | swarm | three lanes at the merge-ready head |

- **M, fresh checker.** Start a subagent with a clean context and give it **only**: the predicate list (P and C, verbatim), the diff, the worktree path and SHA, `DRIVER.md` plus the matching feature files, and this file. Not the plan discussion, not the build conversation, not your opinion of the code. It runs steps 4 to 9 and returns the report. You don't edit its verdicts. If your agent can't start a subagent, say so, run it yourself, and write `checker: self (no subagent)` in the report so the gap is visible.
- **L, swarm.** Three independent checkers in parallel, each with the same inputs as the M checker:
  - **gates lane:** reruns every gate and standing check from a clean state
  - **live lane:** proves the key behavior on the real surface at ladder 5, through the driver
  - **audit lane:** reads the diff and the receipts: runs the test audit, opens the evidence files and confirms they show what the report claims, flags anything out of scope

  For each predicate take the weakest verdict across lanes. The live lane is the floor: if it can't run, the overall verdict can't be VERIFIED.

### 4. Run the gates

Run test, lint, typecheck and build from `PROJECT.md`. Record each exit code. A red gate makes the overall verdict NOT VERIFIED, whatever the predicates say.

### 5. Match the check to the change

For each predicate, name the surface it describes and pick the check that reaches it.

| Change | Check | Doesn't count |
|--------|-------|---------------|
| UI | walk the changed flow in the running app | a component unit test |
| HTTP API | call the real endpoint, keep status and body | a handler test with mocked I/O |
| CLI | run the real command, keep output and exit code | "it compiles" |
| Storage | write through the app, read it back from a second view | a "saved" toast or a 200 |
| Migration · parser | replay a real input, diff the output | a passing type check |
| Refactor | existing checks pass, observable output diffed before and after | "looks equivalent" |
| Config · infra | apply to a disposable environment and observe | the file parses |

A change that touches two surfaces needs a check on each.

### 6. Drive the app when the check needs it

If the check needs the running app, follow `.kiln/verify/DRIVER.md` step by step:

1. **Launch**, and wait for its ready signal.
2. **Doctor**: its health check must pass before you trust anything you see.
3. **Drive** the feature using `.kiln/verify/features/<feature>.md`.
4. **Evidence**: save what you saw where DRIVER.md says.
5. **Cleanup**, always, even when a step failed. Confirm it (port free, process gone).

**No driver, or the driver can't reach the feature?** Drive it by hand if you can (start the app with the `PROJECT.md` command, `curl` the endpoint, use a browser tool your agent has) and write `by hand` in the report. If you can't, mark the predicate INCONCLUSIVE, say what was missing, and suggest `/kiln-verifier` (create, or maintain if the driver has drifted). **Never swap in a proxy**: a unit test can't stand in for the screen. Don't edit DRIVER.md yourself; kiln-verifier owns it.

### 7. Climb the evidence ladder

| Step | Name | Meaning | Earns |
|------|------|---------|-------|
| 5 | reproduced | seen in the running app, on the changed surface | VERIFIED |
| 4 | ran it | a test or script calls the real code and fails loudly if wrong | VERIFIED |
| 3 | reasoned | the failure path was walked step by step | supports only |
| 2 | pointed at | a real file:line that shows the code | supports only |
| 1 | asserted | someone said so, including the agent that wrote it | supports only |

- Only 4 and 5 can earn VERIFIED.
- At **M and L**, a predicate on a surface listed in `verify.live` (UI and API by default) needs **5**. Reaching only 4 there is INCONCLUSIVE.
- Libraries and CLIs are fine at 4. At S, 4 is enough everywhere.
- Every `ran` line must be a command or driver step you actually executed at this SHA, in this session.

### 8. Audit every test you cite

A test counts as evidence only if it:
1. calls the real code the way users do (not a mock of the thing under test),
2. asserts a literal expected value (not "truthy", "not null", "has length", or a snapshot written in the same change without review), and
3. would fail if every import returned `undefined`.

Rejected tests don't count. Name each one in the report with its reason so it can be fixed. If a predicate's only evidence was a rejected test, it falls back to a lower ladder step and needs another check.

### 9. Bugs: before and after

For `type:bug`, prove the bug existed and is gone, on the same surface:

```bash
base=$(git merge-base origin/<default> HEAD)
git worktree add --detach ../<repo>.kiln/<n>-before "$base"
```

1. In `<n>-before`, build and run the old code (on a separate port or data dir if DRIVER.md describes isolation). Follow the repro steps. **See the symptom twice.** Save `old-1`, `old-2`.
2. On HEAD, same steps, same surface. **See it gone twice.** Save `new-1`, `new-2`.
3. Name the root cause at a `file:line`, and say how the fix addresses it.
4. Clean up: `git worktree remove ../<repo>.kiln/<n>-before` (add `--force` only for untracked build output; it holds no commits).

Verdicts:
- symptom hidden instead of fixed (a nil check, a swallowed error, a retry, a longer timeout): **NOT VERIFIED**, even with every test green
- symptom never reproduced on the old build: **INCONCLUSIVE**. You can't prove you fixed what you never saw break.

### 10. Give verdicts

- **VERIFIED**: the right check ran at the required ladder step and what you saw matches the predicate.
- **NOT VERIFIED**: the check ran and contradicts the predicate, or the fix only hides the symptom.
- **INCONCLUSIVE**: the right check couldn't run, or only reached a step below what's required. Always say why and what would unblock it.

The overall verdict is the **weakest** one: any NOT VERIFIED (or red gate) makes it NOT VERIFIED; otherwise any INCONCLUSIVE makes it INCONCLUSIVE. INCONCLUSIVE is never rounded up to a pass.

### 11. Save evidence

Everything goes under `.kiln/verify/evidence/<issue>/` (gitignored): screenshots, response bodies, command output with exit codes. Name files `<P#>-<what>.<ext>` unless DRIVER.md says otherwise. Every `saw` line in the report points at a file there or quotes the literal output.

### 12. Write the report

```
kiln-verify · #42 · size M · checker: fresh
gates          pnpm test · lint · tsc --noEmit · build   all exit 0
P1  toggle in Appearance settings     VERIFIED  5
    ran  driver · settings/appearance · click theme-toggle
    saw  <html data-theme="dark"> · P1-settings-dark.png
P2  choice survives reload            VERIFIED  5
    ran  reload · read localStorage.theme → "dark"
P3  first visit follows the OS theme  VERIFIED  4
    ran  pnpm test:e2e theme.spec.ts --project os-dark
P4  light theme unchanged             VERIFIED  5
    ran  e2e snapshots, light · 0 diffs
C1  colors only via tokens            VERIFIED  4
    ran  pnpm lint:tokens · exit 0   (standing check)
tests audited  11 used · 1 rejected (settings.test.ts:40 asserts only "not null")
overall        VERIFIED on a1c9e0f
evidence       .kiln/verify/evidence/42/   (gitignored)
```

For an INCONCLUSIVE predicate add `why` and `unblock` lines. For a bug add:

```
    old build  loop seen 2 of 2 · old-1.png, old-2.png
    new build  lands on /login 2 of 2 · new-1.png, new-2.png
root cause     src/auth/middleware.ts:41 trusts a stale session cookie
```

### 13. Hand off and route

Give kiln-log the row and the full report (it goes into `## Verify report`). kiln-log commits it after you; the binary ignores `.kiln/memory/` when judging freshness, so that commit doesn't make your verdict stale. Once it's committed, confirm the binary sees it:

```bash
kiln verdict <n> --only verify --json     # result, sha, fresh
```

(`--only verify` ignores the review row, which doesn't exist yet. Exit 0 means your VERIFIED is on record and fresh.)

Then:
- **VERIFIED**: on to kiln-review.
- **NOT VERIFIED**: back to kiln-build with the failing predicates as its fix list.
- **INCONCLUSIVE**: report why and what would unblock it. The playbook continues, but merging will need the human's explicit acceptance at ◆2.

## Writes

- evidence files in `.kiln/verify/evidence/<issue>/` (gitignored)
- the report and row, through kiln-log only
- nothing else: no code, no tests, no driver edits, no labels

## Rules

- Check the surface that changed, never a proxy.
- Only ladder 4 and 5 earn VERIFIED; live surfaces at M and L need 5.
- Never loosen, reword or drop a predicate. Never mark a predicate VERIFIED on a step you didn't run at this SHA.
- INCONCLUSIVE is honest, not a failure, and never a pass.
- A checker doesn't fix what it finds. Fixes go through kiln-build, then you verify again.
- If HEAD moves (outside `.kiln/memory/`), your verdict may be stale. `kiln verdict` decides, not you.

## Done when

Every predicate has a verdict, ladder step and evidence, the report is written, and kiln-log has the row. The ref is the HEAD SHA you verified, and the result is one of the three exact tokens:

```
kiln-verify · 5 predicates, fresh checker · a1c9e0f · VERIFIED
```
