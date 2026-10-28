# pavelsimo/kiln

Written by kiln-onboard. Edited only via kiln-retro PRs.

## Stack

Go 1.25 · Cobra · yaml.v3 · doublestar · golangci-lint v2 · gofumpt + goimports · goreleaser · GitHub Pages (hand-written HTML)

## Commands

test:      make test
lint:      make lint
format:    make fmt-check
build:     make build
all:       make ci
docs:      make docs        (preview at http://localhost:8000)
e2e:       go test ./cmd    (builds the binary and drives it in temp repos)

## Conventions

- Facts in code, judgment in skills: anything git or gh can answer exactly belongs in the binary.
- One file per command in cmd/, logic in internal/<pkg>/, table-driven tests next to the code.
- Every query takes --json. stdout is data, stderr is progress. Exit codes 0 ok, 1 error, 2 usage, 3 "no".
- Skills ship from internal/assets/skills/. .kiln/skills/ is this repo's own install of them (kiln update refreshes it).
- Docs and skills are conversational and never use em-dashes or en-dashes.
- When a command changes, update docs/cli.html and every skill that calls it.

## Do not touch without asking

- .github/workflows/release.yml and .goreleaser.yaml (they publish binaries and the Homebrew formula)
- internal/assets/templates/kiln/config.yaml defaults (every new install inherits them)
