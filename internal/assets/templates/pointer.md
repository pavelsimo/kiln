<!-- kiln:start -->
## Kiln

This repo uses [kiln](https://github.com/pavelsimo/kiln), a software factory for coding agents.
Start every task at `.kiln/skills/kiln/SKILL.md` (the `/kiln` router). It picks a playbook
and chains the `kiln-*` skills in `.kiln/skills/`.

- Facts come from the `kiln` binary: `kiln status`, `kiln checks`, `kiln verdict`, `kiln ready`,
  `kiln release plan`, all with `--json`. Don't re-derive them from raw git or gh.
- Labels move only through `kiln move`, and merges happen only through `kiln merge`.
- Project memory lives in `.kiln/memory/`. Read `PROJECT.md` and `lessons.md` before you work.

This block is managed by `kiln init` and `kiln update`. Everything outside the markers is yours.
<!-- kiln:end -->
