# Changelog

All notable changes to kiln are written here, once per release, by kiln-release.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and kiln uses [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-10-28

### Added

- `kiln init` installs kiln into a repo: 21 skills, config, memory and verify skeletons, agent links for Claude Code, Codex and Cursor, 14 labels, and issue, PR and CI templates
- `kiln update` upgrades installed skills and keeps your edits, writing new versions next to them as `.kiln-new`
- `kiln doctor` checks the whole setup, including whether the verify driver has fallen behind the code
- The `/kiln` router, six playbooks (feature, bug, refactor, chore, investigate, release) and 20 `kiln-*` skills that take a task from issue to merged PR
- Read-only fact commands with `--json`: `kiln status`, `kiln checks`, `kiln verdict`, `kiln ready` and `kiln release plan`
- Guarded writes: `kiln move` follows the label state machine, and `kiln merge` squash merges only when `kiln ready` passes
- Docs at [pavelsimo.github.io/kiln](https://pavelsimo.github.io/kiln)

[0.1.0]: https://github.com/pavelsimo/kiln/releases/tag/v0.1.0
