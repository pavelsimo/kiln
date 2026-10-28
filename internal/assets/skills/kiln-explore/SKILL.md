---
name: kiln-explore
description: Builds a read-only map of the code a task will touch (where things live, how they connect, which tests cover them, and the gotchas), using parallel read-only subagents for size L. Use in Shape before planning, or on its own to understand an area without changing anything.
disable-model-invocation: true
---

# kiln-explore

Maps the code before anyone changes it. The output is a short, cited map that kiln-plan builds on. This is a Shape stage skill and it never edits code, config or memory.

## Reads

- The issue (`gh issue view <n>`) and the kiln-recall brief, if one exists
- `.kiln/memory/PROJECT.md` (layout conventions and the "Do not touch without asking" list)
- The source tree, through `git grep`, `git ls-files` and `git log`

## Steps

1. Restate the question in one line: what does the task need to know? ("Where are theme colors defined, and who reads them?")

2. Find the entry points:

   ```bash
   git ls-files | grep -iE '<area>|<feature>'
   git grep -n -E '<symbol>|<route>|<flag>'
   ```

3. Follow the connections from each entry point: callers, callees, data flow, config and env reads. Read the code; do not guess from file names.

4. Find the tests that cover the area, and whether they call the real code or mock it:

   ```bash
   git grep -ln '<symbol>' -- '*test*' '*spec*'
   ```

5. Read recent history for the hot spots:

   ```bash
   git log --oneline -10 --follow -- <file>
   ```

   Recent churn, reverts and "fix fix" commits are gotchas worth naming.

6. Check the area against PROJECT.md's "Do not touch without asking" list. Any overlap goes at the top of the map.

7. Size L: split the area into 2 to 4 slices (for example API, storage, UI, migrations) and give each slice to a fresh read-only subagent at the same time. Tell each one:
   - the question and its slice's paths
   - "read only: do not edit, stage, commit or run anything that writes"
   - return the map sections below, with `file:line` for every claim

   Merge the answers, drop duplicates, and resolve contradictions by reading the code yourself.

8. Write the map:

   ```
   explore · #42 · theme colors
   lives in     src/theme/colors.ts:1-40 (14 hex values) · src/styles/global.css:12
   connects     Button.tsx:8 → colors.primary · 13 more components import colors.ts
   tests        Button.test.tsx asserts className only (mocked theme)
   gotchas      e2e snapshots cover every page; any color change breaks them
   no-go        none touched
   open         is colors.ts imported by the marketing site too? (git grep found 1 hit in site/)
   ```

## Writes

Nothing in the repo. The map goes into the conversation, and its summary (plus any gotchas worth keeping) goes to kiln-log for the task journal.

## Rules

- Read-only, including subagents. No edits, no formatting runs, no installs, no commits.
- Every claim gets a `file:line` or a command that shows it. "Probably" is an open question, not a finding.
- Prefer `git grep` and `git ls-files` so ignored and generated files stay out of the map.
- Stay on the task's question. A tour of the whole repo is not a map.

## Done when

The map is shown with lives in, connects, tests, gotchas, no-go and open questions, each backed by a reference.

Journal row for kiln-log:

```
kiln-explore · mapped <area> · <n> files · #<n> · ok
```
