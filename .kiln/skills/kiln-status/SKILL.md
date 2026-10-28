---
name: kiln-status
description: Shows the kiln factory floor (tasks in flight, backlog, retro queue) by running `kiln status --json`, and suggests the next action for every item waiting on you. Use when you want to know what kiln is doing or what needs you; it never changes anything.
disable-model-invocation: true
---

# kiln-status

A thin wrapper around the binary's `kiln status`. Status is a fact, so the binary owns it: you run it, render it, and add one suggestion per item that needs a human. This skill works in any stage and from any checkout. It is strictly read-only.

## Reads

- `kiln status --json` (the only source of truth for the floor)
- `.kiln/config.yaml` only to name the autonomy mode in the header, if the JSON lacks it

## Steps

1. Ask the binary:

   ```bash
   kiln status --json
   ```

   Exit 1 means git or gh failed: show the stderr and suggest `kiln doctor`. Do not fall back to `gh issue list` or `gh pr checks` to rebuild the table yourself.

2. Render the floor. One line per `in_flight` item, then backlog and memory:

   ```
   KILN · acme/app · 3 in flight · 2 waiting on you
   #42  ✨ dark mode settings    M  glaze  PR #57  ● CI 4/6
   #43  🐛 login redirect loop   S  ship   PR #58  ✓ green · ◆2 you
   #47  ♻️ rate limiter refactor L  shape  no PR   ◆1 you · plan ready
   backlog  6 ready · 2 need triage · 1 blocked (#39)
   memory   3 retro notes queued for kiln-retro
   ```

   CI comes from `pr.checks` (`passed/total`, with failing or pending called out). "Waiting on you" is the count of items whose `waiting` is not empty.

3. Suggest the next action for each item waiting on you, using `waiting`:

   | `waiting` | Suggestion |
   |-----------|------------|
   | `plan` (◆1) | Read the plan comment (`gh issue view <n> --comments`), then approve or correct it. |
   | `merge` (◆2) | Run `/kiln-merge <pr>`; it shows `kiln ready` and waits for your yes. |

4. For items not waiting on you, show the binary's `next` field as is (for example `kiln-watch #57`). Do not invent a different next step.

5. Add backlog and memory suggestions only when they apply:
   - `backlog.needs_triage > 0`: suggest `/kiln-triage`.
   - `backlog.blocked` not empty: suggest `gh issue view <n> --comments` to read why each one is blocked.
   - `retro_queue` of 5 or more: suggest `/kiln-retro`.
   - Nothing in flight and `backlog.ready > 0`: suggest `/kiln` to start the next ready issue.

6. If the person answers with an action ("merge #43", "show me the plan for #47"), hand it to the skill that owns it (`kiln-merge`, `kiln-plan`, `kiln-resume`). Status itself does nothing more.

## Writes

Nothing. No files, no labels, no comments, no commits.

## Rules

- Read-only, always. Never call `kiln move`, `kiln merge`, `gh issue edit` or `git` writes from this skill.
- The binary's JSON is the floor. Never re-derive CI, labels or merge state from raw gh.
- Never copy the floor into a file. Live state is read live.
- Keep suggestions to one line per item. The table is the point.

## Done when

The floor is shown and every item with a non-empty `waiting` has a suggestion.

Journal row: a standalone `/kiln-status` writes nothing. When the router runs it inside a task, hand kiln-log:

```
kiln-status · read floor · <n in flight> in flight · ok
```
