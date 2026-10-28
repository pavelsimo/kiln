# Working on kiln

kiln is a Go CLI (Cobra) plus a set of agent skills that the CLI embeds and installs.

## Layout

```
cmd/                    Cobra commands, one file per command; human output lives here
internal/
  assets/skills/        the 21 skills kiln installs (source of truth, embedded)
  assets/templates/     config.yaml, memory and verify skeletons, GitHub templates, pointer block
  install/              kiln init and kiln update
  doctor/ status/ checks/ verdict/ ready/ release/ state/ journal/ lock/ config/
  gitx/ ghx/ run/       thin wrappers over git and gh, with a fake runner for tests
docs/                   the GitHub Pages site, hand-written HTML (no build step)
```

## Commands

```bash
make build      # bin/kiln
make test       # go test with coverage
make lint       # golangci-lint
make fmt        # gofumpt + goimports
make ci         # fmt-check, lint, test, build
make docs       # preview the docs site at http://localhost:8000
```

## Rules

- Facts in code, judgment in skills. If git or gh can answer a question exactly, it belongs in the binary, never in a skill. The binary never calls a model and never edits code.
- Every query command supports `--json`. stdout is data, stderr is progress and errors.
- Exit codes: 0 ok, 1 runtime error, 2 bad usage, 3 the answer is no.
- When you change a command's behavior, update `docs/cli.html` and any skill that calls it.
- Docs and skills use a conversational tone and never use em-dashes or en-dashes.
- Commits use gitmoji plus Conventional Commits, for example `✨ feat: add kiln verdict --only`. No AI co-author trailers.

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
