---
name: kiln
description: The kiln router and the single entry point for work in a kiln repo. Restates the goal, classifies type and size, picks a playbook, copies its steps into the todo list, and chains kiln-* skills while respecting autonomy and gates. Use it for any task, bug, refactor, chore, question or release, or with no task to see what needs attention.
---

# kiln

You are the front door. Someone hands you a sentence, an issue number, or nothing at all. Your job is to understand what they want, size it, pick the right playbook, turn that playbook into the todo list, and walk the list by chaining kiln's action skills. You decide *which* skills run and *when to stop*. The skills do the work. The `kiln` binary supplies every fact.

You never move labels or merge yourself. Those go through `kiln move` and `kiln merge`, inside the skills that own them.

## Reads

- `.kiln/config.yaml`: `autonomy`, `gates`, `rigor`, `verify.live`, `worktrees.max_in_flight`
- `.kiln/memory/PROJECT.md`: commands, conventions, and **Do not touch without asking**
- `.kiln/memory/lessons.md`: `[strong]` lessons are rules for this run
- `.kiln/skills/kiln/playbooks/<playbook>.md`: the step list you copy
- `kiln status --json`: what is already in flight and waiting

If `PROJECT.md` is missing, stop and suggest `/kiln-onboard` first. Every check later depends on its commands.

## Steps

### 0. No task given

Run the floor report and suggest one next step:

```bash
kiln status --json
```

Read it and say, in this order of priority:
1. any `in_flight` item with `waiting: "merge"`: "PR #57 is waiting at ◆2. Run `/kiln-merge 57`?"
2. any item with `waiting: "plan"`: "The plan for #47 is waiting at ◆1. Want to see it?"
3. any item whose `next` names a skill: offer it (`kiln-watch #57`, `kiln-resume 52`)
4. `retro_queue` of 5 or more: suggest `/kiln-retro`
5. `backlog.ready` above zero and `in_flight` below `max_in_flight`: offer to start the top ready issue
6. `backlog.needs_triage` above zero: suggest `/kiln-triage`

Then stop. Don't start anything without an answer.

### 1. Restate the goal

Write one sentence: who gets what, observable. Show it before anything else.

```
goal   Users can switch Settings to a dark theme, and the choice sticks.
```

If you can't write that sentence, the request is vague. Say so and let kiln-issue run in interview mode. If the input is an existing issue (`#42`), read it first:

```bash
gh issue view 42 --json number,title,body,labels,state
```

If `kiln status --json` already lists that issue in `in_flight`, this is a resume, not a new task. Suggest `/kiln-resume 42` and stop.

### 2. Classify the type

| Type | Signal | Playbook |
|------|--------|----------|
| feature | new behavior a user can see | `feature` |
| bug | something is wrong today | `bug` |
| refactor | same behavior, better shape | `refactor` |
| chore / docs | deps, config, docs, cleanup | `chore` |
| investigate | a question, no code change wanted | `investigate` |
| release | ship what main already has | `release` |

An existing issue's `type:*` label wins over your guess. If two types fit, pick the one with the stronger verification (bug over chore, feature over refactor).

### 3. Size it

| Size | Rule |
|------|------|
| S | one area, about 100 changed lines or fewer, no public interface change |
| M | several modules, or a change to a public interface |
| L | cross-cutting work, migrations, security or data. Split into sub-issues |

When torn between two sizes, pick the larger. Anything touching a path under **Do not touch without asking** is at least M and needs a human before editing. An existing `size:*` label wins unless the code clearly says otherwise; then say why and fix it with `gh issue edit <n> --remove-label size:S --add-label size:M` (size and type labels are plain edits; `kiln:*` labels only move through `kiln move`).

### 4. Announce

```
goal   Users can switch Settings to a dark theme, and the choice sticks.
type   feature · size M · playbook feature · autonomy gated
```

### 5. Build the todo list

Open `.kiln/skills/kiln/playbooks/<playbook>.md`. Copy its **Todo list** into your agent's todo list **word for word**, one item per line, filling in the `<placeholders>`. Then apply the playbook's skip rules:

- A skipped step is never deleted. Rewrite it as `skip: <skill> · <reason>`, in the same position.
- The user may ask for a skip ("skip the plan"). Record their words as the reason.
- Size L never skips `kiln-plan` or its ◆1, whatever anyone asks.
- Every skip also goes to kiln-log, which writes it under `## Skipped`.

You can audit a run by reading its list. Keep it that way.

### 6. Walk the list

For each item, in order:

1. Read `.kiln/skills/<name>/SKILL.md` and follow it exactly. That file is the instruction, not your memory of it.
2. Mark the item done, or rewrite it as `skip: …` if it turned out not to apply (say why).
3. Take the journal row the skill hands back (`skill · action · ref · result`) and pass it to kiln-log. Rows produced before the worktree exists (kiln-recall, kiln-issue, kiln-plan) are held in the conversation and written when kiln-start creates the journal.
4. Decide whether to stop (step 7). If not, continue.

**Loops.** If kiln-verify returns NOT VERIFIED or kiln-review returns FIX, add new items right after the failed one: `kiln-build · fix P2 (loop 2)`, `kiln-commit`, `kiln-verify`, `kiln-review`. On the third loop for the same predicate or finding, stop and ask. If the human can't unblock it now: `kiln move <n> blocked --reason "<why>"`.

**Exit codes.** `1` means git or gh failed: report it. `2` means you called the binary wrong: fix the call. `3` means the answer is no: follow the reason it gives. Never work around a 3.

### 7. Stop where autonomy says

Gates are fixed places: ◆1 is inside kiln-plan, ◆2 inside kiln-merge, ◆ release inside kiln-release. The skill that owns the gate applies this table, and you honor it.

| | manual | gated (default) | autopilot |
|---|---|---|---|
| when it stops | after every step | at each gate listed in `gates:` | only forced gates and hard stops |
| S · self-verify · self-review | every step | ◆1 ◆2. The plan is a few lines; "ok" approves | no gates. Merges when `kiln ready` passes |
| M · fresh checker · independent reviewer | every step | ◆1 ◆2. Reviewer verdict shown at ◆2 | no gates. VERIFIED and PASS must hold for the head SHA (or an equal patch-id) |
| L · verifier swarm · adversarial review | every step | ◆1 ◆2, plus ◆2 for every sub-issue PR | ◆1 forced (`rigor.L.force_gates`). Merges follow |

A gate stops when **any** of these hold:
- `autonomy: manual`
- `autonomy: gated` and the gate is in `gates:`
- the gate is in `rigor.<size>.force_gates`
- it is ◆2 and any predicate is INCONCLUSIVE (only a human accepts that, in every mode)
- it is the release gate (always)

In **manual**, after each step report the result and the next item, then wait for "next" or "go". Nothing chains on its own.

At a stop, show what the gate needs to show and wait for an explicit answer. Silence, "hmm" or a question is not approval.

### 8. Always stop, in every mode

These are not settings. Before any of them, stop, say exactly what you are about to do, and wait for a yes:

- force-push to a shared branch
- delete unmerged work or data
- deploy or release, and before any tag
- merge with failing checks (`kiln merge` refuses anyway; never route around it)
- change secrets or repo settings
- message people outside the repo
- edit a path listed under **Do not touch without asking** in `PROJECT.md`

### 9. Finish

When every item is checked or skipped, give a short report: the issue, the PR, the merge SHA (or where it stopped and why), the skips, and any `kiln-retro` suggestion.

## Writes

- The todo list in your agent's session.
- Nothing on disk directly. Journal content goes through kiln-log. The one exception is the investigate playbook, where you write the report and any decision record yourself because you made the call.

## Rules

- Facts come from the binary: `kiln status|checks|verdict|ready|release plan --json`. Never rebuild them from raw `gh pr checks` or label lists.
- `kiln:*` labels change only through `kiln move`. Merges happen only through `kiln merge`.
- One door: action skills don't fire by accident. You chain them by reading `.kiln/skills/<name>/SKILL.md`; each one also works on its own.
- The playbook list is copied word for word. Skips are rewritten, never removed.
- Predicates agreed at ◆1 are never loosened later to make something pass.
- Kiln is standalone. Never call a skill or tool kiln didn't install.
- `[strong]` lessons in `lessons.md` are rules for planning, building and review.

## Done when

Every todo item is checked or written as `skip: <skill> · <reason>`, every gate that applied got an explicit answer, and you have reported where the task ended. Your own row for kiln-log, handed over with the kiln-start batch:

```
kiln · routed feature, size M, playbook feature · #42 · ok
```
