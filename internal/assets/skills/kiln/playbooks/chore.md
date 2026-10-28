# Playbook: chore

Deps, config, docs, cleanup. There is nothing to design, so the plan is skipped and verification runs light. Use this playbook for `type:chore` and `type:docs`.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | run | short issue |
| Shape | skip | skip: no design decision |
| Fire | run | change + commit |
| Glaze | light | lint · build · matching standing checks |
| Ship | run | pr · watch · merge (◆2) |
| Temper | light | log only |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1.  kiln-recall · lessons about <area>
2.  kiln-issue · <gitmoji> <title>, short issue
3.  kiln-plan · only if the chore hides a design decision
4.  kiln-start · ../<repo>.kiln/<n>-<slug>
5.  kiln-build · light: make the change
6.  kiln-commit · <gitmoji> chore: <what>
7.  kiln-verify · light: lint · build · matching standing checks
8.  kiln-review · <self|independent>
9.  kiln-pr · ready PR, Closes #<n>
10. kiln-watch · conflicts, threads, CI
11. kiln-log · final pass: skips, pushed before ◆2
12. ◆2 kiln-merge · PR #<pr>
```

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 2 kiln-issue | the chore arrived as an existing issue | `skip: kiln-issue · #<n> already exists` |
| 3 kiln-plan | usually. Run it if the chore is size L or forces a choice (a major version with breaking changes, a config that changes behavior) | `skip: kiln-plan · chore with no design decision` |

## Notes

- Never skip kiln-review, even for a one-line chore. `kiln ready` needs a fresh PASS row before it lets anything merge. For a mechanical S diff the self-review checklist takes a minute; run it and log the row.

- Gitmoji: `⬆️` dependency bumps, `🔧` config, `📝` docs (`📝 docs: …`), `🔥` removal, `👷` CI. The emoji decides whether the change shows up in the next release, so pick it honestly.
- Step 7 still produces a real verdict. The predicates are short ("`pnpm lint` exits 0", "`pnpm build` exits 0", "docs site builds with no broken links") plus whatever `kiln checks --json` matches.
- A chore that turns out to change behavior is no longer a chore. Stop, tell the user, and re-route as feature or bug.
- Size L chores (a framework upgrade across the codebase) run step 3 with ◆1 forced.
