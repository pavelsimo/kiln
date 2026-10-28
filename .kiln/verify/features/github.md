---
feature: github
sources: ["cmd/status.go", "cmd/ready.go", "cmd/move.go", "cmd/merge.go", "internal/ready/**", "internal/status/**", "internal/state/**", "internal/ghx/**"]
---
# kiln status, ready, move, merge

How a user reaches it: `/kiln-status`, kiln-pr, kiln-watch and kiln-merge, or a terminal.

Drive: these need a real GitHub repo, so drive them through tests with the fake gh runner:
`go test ./internal/ready ./internal/status ./internal/state ./internal/ghx -run . -v`.
For argument handling, run the binary in the sandbox: `"$KILN" -C "$SANDBOX" move 42 done` exits 2.

## Unreachable

Live runs against GitHub. A disposable test repo would make these reachable; none exists yet.
