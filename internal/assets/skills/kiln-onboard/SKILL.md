---
name: kiln-onboard
description: Introduces kiln to a repo once, right after `kiln init`, by detecting the stack and the real test, lint, typecheck, build and e2e commands from manifests and CI, writing .kiln/memory/PROJECT.md, wiring the kiln CI workflow, seeding standing checks, building the app driver through kiln-verifier, and opening it all as one PR. Use once per repo after installing kiln.
disable-model-invocation: true
---

# kiln-onboard

Kiln knows how to work after `kiln init`; this skill teaches it your project. Everything it writes is a file you review in one PR, and PROJECT.md is the one every other skill reads, so getting the commands right here matters more than anything else. Runs once, outside the task loop.

## Reads

- Manifests and lockfiles: `package.json`, `pnpm-lock.yaml`, `yarn.lock`, `package-lock.json`, `bun.lockb`, `go.mod`, `Cargo.toml`, `pyproject.toml`, `uv.lock`, `requirements*.txt`, `Gemfile`, `pom.xml`, `build.gradle*`, `composer.json`, `mix.exs`, `Makefile`, `justfile`, `Taskfile.yml`, `mise.toml`, `Dockerfile`, `docker-compose*.yml`
- CI: `.github/workflows/*.yml` (and `.gitlab-ci.yml`, `.circleci/` if present)
- `CONTRIBUTING.md`, lint and formatter configs, `AGENTS.md` and `CLAUDE.md` outside the kiln block
- `.kiln/config.yaml`, `.kiln/kiln.lock`, `gh repo view`

## Steps

1. Preflight:

   ```bash
   test -f .kiln/config.yaml || echo "run kiln init first"
   gh repo view --json nameWithOwner,defaultBranchRef,primaryLanguage,languages
   kiln doctor
   ```

   Expect doctor to report "no verify driver yet". If `.kiln/memory/PROJECT.md` already has real content, stop: after onboarding, PROJECT.md changes only through kiln-retro PRs.

2. Branch from an up-to-date default branch. `kiln init`'s files are probably still uncommitted in this checkout; they ride along in this PR.

   ```bash
   git switch <default>
   git pull --ff-only origin <default>
   git switch -c kiln/onboard
   ```

3. Detect the stack: language, framework, package manager, test runner, e2e tool. Name only what the manifests show.

4. Find the real commands. CI is the source of truth, then manifest scripts, then Makefile-style targets. For each of test, lint, typecheck, build and e2e, run it once on this clean checkout:

   ```bash
   <command>; echo "exit $?"
   ```

   A command that fails on a clean checkout is written down with a note ("fails on main today: <reason>"), never silently dropped or replaced with a guess. Skip running e2e if it needs services you cannot start; note that.

5. Note conventions you can see: layout (`src/features/<area>/`), export style, test file naming, commit style from `git log --oneline -30`, lint rules that encode a convention. No aspirations, only what the code does.

6. Propose "Do not touch without asking" candidates (payments, auth, migrations, release workflows, infra, generated or vendored code) and ask the person once to confirm, trim or add.

7. Write `.kiln/memory/PROJECT.md`:

   ```markdown
   # <owner>/<repo>
   Written by kiln-onboard. Edited only via kiln-retro PRs.
   ## Stack
   TypeScript · React 19 · Vite · pnpm · Vitest · Playwright
   ## Commands
   test:      pnpm test
   lint:      pnpm lint
   typecheck: pnpm tsc --noEmit
   build:     pnpm build
   e2e:       pnpm test:e2e
   ## Conventions
   - components in src/features/<area>/, one folder per feature
   ## Do not touch without asking
   - src/billing/**   · payments, needs human review
   ```

8. Wire CI. Only touch `.github/workflows/ci.yml` if kiln wrote it: its path is in `.kiln/kiln.lock` with a sha256 that matches `sha256sum .github/workflows/ci.yml`, or it still has kiln's placeholder step. Replace the placeholder step with the real commands (setup, install, lint, typecheck, test, build). If another workflow already runs those commands on pull requests, ask whether to delete kiln's ci.yml instead of duplicating them. If ci.yml existed before kiln, leave it alone.

9. Seed `.kiln/verify/checks.yaml` from CI. Base commands (test, lint, build) are already run by kiln-verify; seed only path-scoped gates such as a migrations job, a schema or codegen diff, or a job with a `paths:` filter:

   ```yaml
   - id: migrations-reversible
     when: { paths: ["db/migrations/**"] }
     predicate: pnpm db:rollback && pnpm db:migrate both exit 0
     blocking: true
     from: .github/workflows/ci.yml · job migrations
   ```

   Check every glob matches real files (`git ls-files ':(glob)db/migrations/**'`). Zero seeded checks is fine.

10. Hand off to kiln-verifier in create mode: read `.kiln/skills/kiln-verifier/SKILL.md` and follow it on this branch. It writes `.kiln/verify/DRIVER.md` and `features/`, proves them with one full run, and commits here instead of opening its own PR.

11. Run `kiln doctor` again. Fix anything onboarding owns; report the rest.

12. Hand kiln-log the row below (it records onboarding in `.kiln/memory/tasks/onboard.md` on this branch), then commit by explicit path and open one ready PR:

    ```bash
    git add .kiln AGENTS.md CLAUDE.md .github/ISSUE_TEMPLATE/kiln-task.yml .github/pull_request_template.md .github/workflows/ci.yml
    git add .claude/skills/kiln* .agents/skills/kiln* .cursor/skills/kiln*   # only the ones that exist
    git status --short    # nothing unexpected staged
    git commit -m "🔧 chore(kiln): onboard <owner>/<repo>"
    git push -u origin kiln/onboard
    gh pr create --base <default> --title "🔧 chore(kiln): onboard <owner>/<repo>" --body-file <body>
    ```

    The body lists the stack, the commands and their exit codes, seeded checks, mapped features, and asks the reviewer to read "Do not touch without asking" first.

## Writes

- `.kiln/memory/PROJECT.md` (its only direct write; later edits go through kiln-retro)
- `.kiln/verify/checks.yaml` (seed only)
- `.github/workflows/ci.yml` placeholder step, only if kiln wrote the file
- `.kiln/verify/DRIVER.md` and `features/` through kiln-verifier
- One PR on `kiln/onboard`

## Rules

- Commands come from CI and manifests and are run once. Never write a command you did not see or run.
- Never overwrite a CI workflow kiln did not write.
- Stage explicit paths, never `git add -A`. Never commit `.kiln/verify/evidence/`.
- Ask once about "Do not touch without asking"; do not guess it alone.
- No Co-authored-by trailers or AI attribution in the commit or PR.
- The PR has no issue, so `kiln ready` treats it as a housekeeping PR: only GitHub's checks gate it. It changes what every future task reads, so it always waits for a human yes before `kiln merge <pr>`, in every autonomy mode.

## Done when

The PR is open, `kiln doctor` passes or its remaining findings are listed, and the person has been asked to review PROJECT.md.

Journal row for kiln-log:

```
kiln-onboard · PROJECT.md, <n> checks, <n> features · #<pr> · ok
```
