---
name: kiln-release
description: Cuts a release of what the default branch already has. It takes the version and grouped commits from `kiln release plan --json`, edits CHANGELOG.md for readers, lands it through a release PR, stops at the release gate in every autonomy mode, then merges with `kiln merge`, tags the merge commit and publishes. Use when you want to ship; `/kiln-release 2.0.0` sets the version and `/kiln-release --pre rc` cuts a release candidate.
disable-model-invocation: true
---

# kiln-release

A release has no issue of its own. It gathers every task merged since the last tag, writes CHANGELOG.md once, lands it through a normal PR, and tags the merge commit. Until you say yes at the gate, closing the PR undoes everything. This is a Ship stage skill that runs outside the per-task loop.

## Reads

- `kiln release plan --json` (version, rule, grouped commits, draft changelog, compare URL)
- `.kiln/config.yaml` `release:` (`tag`, `changelog`, `version_files`, `publish`)
- The changelog file (default `CHANGELOG.md`) and each file in `version_files`
- `.github/workflows/*.yml` (to find a tag-triggered publish workflow)
- `kiln ready <pr> --json`, `kiln merge <pr> --json`

## Arguments

| You type | Maps to |
|----------|---------|
| `/kiln-release` | `kiln release plan --json` |
| `/kiln-release 2.0.0` (or `v2.0.0`) | `kiln release plan --json --version 2.0.0` |
| `/kiln-release --pre rc` | `kiln release plan --json --pre rc` (for example `1.4.0-rc.1`) |

Strip a leading `v` before passing `--version`.

## Steps

1. **Preflight.** Fresh default branch, clean tree, gh auth:

   ```bash
   gh auth status
   default=$(gh repo view --json defaultBranchRef -q .defaultBranchRef.name)
   git fetch origin --tags
   git status --short        # must be empty
   ```

   Stop on a dirty tree or failed auth. If a PR already exists for this release (`gh pr list --head kiln/release-v<version> --state open`), continue from it instead of starting over.

2. **Collect and version.** Ask the binary, with any override:

   ```bash
   kiln release plan --json --ref origin/$default [--version X.Y.Z] [--pre rc]
   ```

   Exit 3 means the tag already exists or nothing releasable merged since the last tag: report which and stop. Otherwise keep `version`, `tag`, `last_tag`, `rule`, `commits`, `sections`, `left_out`, `changelog` and `compare_url`. Show the rule line and the counts:

   ```
   KILN release · acme/app · v1.3.0 → v1.4.0
   rule   ✨ feat in 2 PRs → minor
   since  v1.3.0 · 6 merged PRs · 3 left out
   ```

   The version is the binary's answer. If the person disagrees, rerun with `--version`; do not edit the number by hand.

3. **Changelog.** Work in a release worktree:

   ```bash
   git worktree add ../<repo>.kiln/release-v<version> -b kiln/release-v<version> origin/$default
   cd ../<repo>.kiln/release-v<version>
   ```

   Start from the plan's `changelog` section and edit only the wording, for readers: say what changed for a user, keep `(#N)`, drop ⚡️ ⬆️ ♻️ entries nobody can see, mark breaking changes **BREAKING**. Keep a Changelog form:

   ```markdown
   ## [Unreleased]

   ## [1.4.0] - 2026-10-28
   ### Added
   - dark mode in settings (#42)
   ### Fixed
   - login no longer loops after the session expires (#43)

   [Unreleased]: https://github.com/acme/app/compare/v1.4.0...HEAD
   [1.4.0]: https://github.com/acme/app/compare/v1.3.0...v1.4.0
   ```

   Insert the section below `## [Unreleased]` (create the file with the standard Keep a Changelog header if it is missing). Rewrite the link block: `[Unreleased]` compares the new tag to HEAD, the new version uses `compare_url`, older links stay. A first release links `releases/tag/<tag>` instead. Bump the version string in each `version_files` entry and nothing else in those files.

4. **Release PR.** Commit by explicit path and open a ready PR:

   ```bash
   git add CHANGELOG.md <version_files>
   git commit -m "🔖 chore(release): v<version>"
   git push -u origin kiln/release-v<version>
   gh pr create --base $default --title "🔖 chore(release): v<version>" --body-file <body>
   ```

   The body holds the rule, the changelog section and the publish route. No `Closes #N`: `kiln ready` treats `kiln/` branches without an issue number as housekeeping PRs and skips the closes and verdict checks. Then read `.kiln/skills/kiln-watch/SKILL.md` and take the PR to green.

5. **Publish route.** Decide it now so the gate can show it. With `publish: auto`, read `.github/workflows/*.yml`:
   - a workflow with `on: push: tags:` matching the tag: route **tag** (push the tag, watch that workflow)
   - a workflow on `release: types: [published]`: route **gh** (creating the release is what triggers it)
   - neither: route **gh**

   `publish: tag` and `publish: gh` force the route.

6. **◆ Release gate, in every autonomy mode, autopilot included.** Show the version, the changelog section, `kiln ready <pr> --json`, and the publish route, then wait:

   ```
   PR #61  kiln/release-v1.4.0  ✓ CI green
   publish  release.yml runs on tags → watch it
   ◆ merge #61, tag v1.4.0, publish?  (yes / cancel)
   ```

   Only an explicit yes continues. Cancel leaves the PR open; close it only if the person asks.

7. **Merge.**

   ```bash
   kiln merge <pr> --json
   ```

   Exit 3 means not ready: show the failing checks and stop. Keep `merge_sha`; never tag anything else.

8. **Tag.** An annotated tag on the merge commit, then push only that tag:

   ```bash
   git fetch origin
   git tag -a v<version> <merge_sha> -m "release v<version>"
   test "$(git rev-list -n 1 v<version>)" = "<merge_sha>"
   git push origin v<version>
   ```

   Use the plan's `tag` if config changes the template. If the push is rejected because the tag exists, stop and report. Never force.

9. **Publish.**

   Route tag: find the run for the tag and watch it, then read the release back:

   ```bash
   gh run list --workflow <file> --branch v<version> --json databaseId,status,conclusion --limit 1
   gh run watch <run-id> --exit-status
   gh release view v<version> --json url,tagName,isDraft,isPrerelease
   ```

   A failed run that looks flaky may be rerun once (`gh run rerun <run-id> --failed`). If it still fails, report it; the tag stays.

   Route gh: write the changelog section to a notes file and create the release:

   ```bash
   gh release create v<version> --verify-tag --title "v<version>" --notes-file <notes> [--prerelease]
   gh release view v<version> --json url,tagName,isDraft,isPrerelease
   ```

   Pass `--prerelease` for `--pre` versions.

10. **Log.** The release journal lives on the release branch, and `kiln merge` deletes that branch, so log *before* the gate. Hand kiln-log the rows up to the open PR (plan, changelog, PR opened) so they land in `.kiln/memory/tasks/release-v<version>.md` and merge with the release. What happens after the merge (merge SHA, tag, publish) lives on GitHub: the tag, the Release and the workflow run are the record. Report them in your final message.

## Writes

- CHANGELOG.md (this skill is its only writer) and the `version_files`
- The release PR on `kiln/release-v<version>`, merged through `kiln merge`
- The annotated tag `v<version>` on the merge commit, and the GitHub Release (directly or through the tag workflow)

## Rules

- The release gate stops in every autonomy mode. No yes, no merge, no tag.
- The version comes from `kiln release plan`. Overrides go through `--version` and `--pre`, never a hand edit.
- Merge only with `kiln merge`. Tag only the `merge_sha` it returns.
- Never force-push, never move, delete or re-push a tag, never `git push --tags`. A broken release is fixed with a patch release.
- The changelog is written once, here. No PR appends to `[Unreleased]` along the way.
- No Co-authored-by trailers or AI attribution in the commit, tag, PR or release notes.

## Done when

`gh release view v<version>` returns the release (or the tag workflow finished and the failure is reported), the tag points at the merge SHA, and your final message names the merge SHA, the tag and the release URL.

Journal rows for kiln-log, written before the release gate:

```
kiln-release · planned v<version> (<rule>) · v<version> · ok
kiln-release · opened release PR · #<pr> · ok
```
