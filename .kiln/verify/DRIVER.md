# .kiln/verify/DRIVER.md · pavelsimo/kiln

kiln is a CLI, so "the running app" is the built binary driven inside a throwaway git repo.
Nothing here ever touches this checkout or a real GitHub repo.

## Launch

```bash
make build                                   # bin/kiln
export KILN="$PWD/bin/kiln"
export SANDBOX="$(mktemp -d)"
git -C "$SANDBOX" init -q -b main
git -C "$SANDBOX" -c user.email=v@kiln.test -c user.name=verify commit -q --allow-empty -m "🎉 init"
git -C "$SANDBOX" remote add origin git@github.com:acme/app.git
```

ready when  `"$KILN" version` prints `kiln <version>` and exits 0

## Doctor

```bash
"$KILN" -C "$SANDBOX" init --no-github && "$KILN" -C "$SANDBOX" doctor --no-github
```

Exits 0 with only warnings (no driver in the sandbox is expected).

## Drive

- Run commands with `"$KILN" -C "$SANDBOX" <command>`; add `--json` and read fields with `jq`.
- Anything that needs GitHub (status, ready, move, merge, labels) can't run in the sandbox. Drive those
  through the fake gh runner in their package tests (`internal/ready`, `internal/install`), and say so
  in the evidence. Never point them at pavelsimo/kiln.
- Make commits in the sandbox with `git -C "$SANDBOX" -c user.email=v@kiln.test -c user.name=verify commit`.

## Evidence

.kiln/verify/evidence/<task>/<feature>-<step>.txt: the command, its stdout and stderr, and the exit code.

## Cleanup

```bash
rm -rf "$SANDBOX" "$SANDBOX.kiln"
```

`ls "$SANDBOX"` fails afterwards.
