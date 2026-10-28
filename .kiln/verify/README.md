# verify

How kiln proves a change works on this project.

- `DRIVER.md` says how to launch the app, health-check it, drive it, capture
  evidence and clean up. kiln-verifier writes it and proves it with one full run.
- `features/` has one file per user-facing feature: how a user reaches it and
  which source files are behind it.
- `checks.yaml` lists standing checks that apply whenever a branch touches
  matching paths.
- `evidence/` is where screenshots, responses and logs land. It's gitignored.

None of these files hold pass or fail rules. Those come from each issue's
acceptance criteria.
