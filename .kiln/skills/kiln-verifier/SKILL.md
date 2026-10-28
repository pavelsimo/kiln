---
name: kiln-verifier
description: Builds and maintains the project's app driver in .kiln/verify/ (DRIVER.md plus a feature map). Create mode learns how to launch, health-check, drive and clean up the app, writes the driver and 3 to 5 feature files, and proves them with one full run; maintain mode checks every feature against the source and the live app, fixes only the map, and ships at most one PR. Use from kiln-onboard (create), from kiln-retro when kiln doctor reports drift (maintain), or on its own to rebuild the driver.
disable-model-invocation: true
---

# kiln-verifier

Teaches kiln how to use your app the way a person would, so kiln-verify can prove predicates on the real surface. The driver says how to run things. It never says what counts as passing; that comes from each issue's predicates. This is a Glaze support skill with two modes: create and maintain.

## Reads

- `.kiln/memory/PROJECT.md`, `.kiln/config.yaml` (`verify.dir`, `verify.live`)
- The repo: run scripts, entry points, routes, CLI commands, env examples, seed scripts, docker-compose files
- `kiln doctor` output (maintain: which features drifted and by how many commits)
- `git log` since the driver last changed (maintain)

## The files

`.kiln/verify/DRIVER.md` has exactly these sections:

```markdown
# .kiln/verify/DRIVER.md · <owner>/<repo>
## Launch
pnpm dev --port 5173
ready when  curl -fsS localhost:5173/health → 200
## Doctor
curl -fsS localhost:5173/api/me?as=seed-user → 200
## Drive
harness  Playwright · http://localhost:5173
sign in  seed-user@acme.test · code in .dev/mail.log
## Evidence
.kiln/verify/evidence/<task>/<feature>-<step>.png
## Cleanup
kill $APP_PID · lsof -i :5173 is empty
<!-- maintained 2026-10-28 at a1c9e0f -->
```

Each `.kiln/verify/features/<feature>.md` uses this frontmatter, exactly:

```markdown
---
feature: appearance-settings
sources: ["src/features/settings/**", "src/theme/**"]
---
# Appearance settings
How a user reaches it, steps to drive it, what to capture.
```

What "using the app" means depends on the project: a browser harness for a web UI, `curl` with status and body for an API, the real binary with output and exit code for a CLI.

## Steps: create

1. Find, from the code, the five things the driver needs:
   - run command and port (package scripts, Procfile, Makefile, `cmd/`, docker-compose)
   - ready signal (a health route, a log line, an open port)
   - drive mechanism (Playwright or another harness already in the repo, `curl`, the CLI itself)
   - auth and seed data (seed scripts, fixtures, test users, `.env.example`)
   - isolation (a separate port, a temp database or data dir, so a run never touches real data)

2. Ask the person only what the code cannot answer (a test account, a local API key), in one batch. Never write a secret into the driver; reference the env var name.

3. Pick 3 to 5 features: user-facing entry points (pages, endpoint groups, top-level commands), favoring areas with recent churn:

   ```bash
   git log --since=90.days --name-only --format= | sort | uniq -c | sort -rn | head -30
   ```

4. Write DRIVER.md and the feature files. Check every `sources` glob matches real files and that evidence is ignored:

   ```bash
   git ls-files ':(glob)src/features/settings/**' | head
   git check-ignore -q .kiln/verify/evidence/probe && echo ignored
   ```

5. Prove it with one full run, exactly as written: Launch and wait for ready, run Doctor, Drive one feature end to end, save Evidence where the driver says, then Cleanup and confirm it is clean (process gone, port free, temp data removed). Also open each other feature's entry point once. When a step fails, fix the driver and rerun from Launch. A driver that has not run is a guess.

6. If something cannot be made to work (a missing credential, a paid service), keep the step and mark it `unresolved: <what is needed>`, and say so in the report. Do not fake the run.

7. Commit by explicit path:

   ```bash
   git add .kiln/verify/DRIVER.md .kiln/verify/features/
   git commit -m "🔧 chore(kiln): add verify driver"
   ```

   Called from kiln-onboard: commit on its branch and return. Standalone: work on `kiln/verifier-<date>` from `origin/<default>`, push, and open one ready PR.

## Steps: maintain

1. Read what drifted and since when:

   ```bash
   kiln doctor
   since=$(git log -1 --format=%H -- .kiln/verify/DRIVER.md .kiln/verify/features/)
   git log --oneline "$since"..HEAD -- <feature sources>
   ```

2. Source pass: start one fresh read-only subagent per feature, in parallel. Give each the feature file, DRIVER.md, `git diff "$since"..HEAD -- <its sources>` and the current source. Ask it to report every step, route, selector, flag or command in the map that no longer matches the code, renamed or moved paths, new user-facing sub-flows, and `sources` globs that match nothing.

3. Live pass: one run over every feature, not only the flagged ones, following DRIVER.md (Launch, Doctor, Drive each feature, Evidence, Cleanup). One app instance, one agent.

4. Sort every mismatch into exactly one bucket:

   | Bucket | Means | Action |
   |--------|-------|--------|
   | doc drift | the map is wrong, the app is right (renamed route, moved button) | fix the feature file or DRIVER.md |
   | harness gap | the driver cannot do something it needs (new login step, missing seed) | fix DRIVER.md |
   | product gap | the app is broken or contradicts its own code | file an issue, leave the map alone |
   | unreachable | needs what kiln cannot get (paid account, hardware) | note it under `## Unreachable` in the feature file |

5. Report each product gap as an issue with the commands and output you saw (evidence files are gitignored, so paste the text):

   ```bash
   gh issue create --title "🐛 <what is broken>" --label type:bug --body-file <body>
   kiln move <new-issue> needs-triage
   ```

6. Update `sources` globs, add a feature file for a new user-facing area, delete one for a removed area. Rerun the live pass for everything you changed. Update the `<!-- maintained <date> at <sha> -->` line at the end of DRIVER.md, so `kiln doctor`'s drift count resets even when nothing else changed.

7. Ship at most one PR. Called from kiln-retro: commit on its branch and return. Standalone: commit on `kiln/verifier-<date>`, push, and open one ready PR titled `🔧 chore(kiln): maintain verify driver`.

## Writes

- `.kiln/verify/DRIVER.md` and `.kiln/verify/features/*.md` (this skill is their only writer)
- Evidence under `.kiln/verify/evidence/` (gitignored, never committed)
- Issues for product gaps (maintain); at most one PR

## Rules

- Never write a pass/fail rule into DRIVER.md or a feature file. "Click theme-toggle, capture the page" is fine; "the page should turn dark" is not.
- Fix only the map. Never change product code to match the driver, and never rewrite the map to match broken behavior.
- Every driver change is proven by a live run before it is committed.
- No secrets in the driver. Runs are isolated from real data.
- Stage explicit paths. No Co-authored-by trailers or AI attribution in commits or PRs.

## Done when

Create: DRIVER.md and 3 to 5 feature files exist, one full run passed (or its unresolved steps are named), and the work is committed. Maintain: every feature was read against source and driven live, every mismatch is bucketed, product gaps are issues, and at most one PR carries the map fixes.

Journal row for kiln-log:

```
kiln-verifier · create: <n> features, proven once · <sha> · ok
kiln-verifier · maintain: <n> drift, <n> harness, <n> product (#..), <n> unreachable · #<pr> · ok
```
