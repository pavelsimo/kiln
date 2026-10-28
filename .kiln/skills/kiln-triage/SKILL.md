---
name: kiln-triage
description: Sweeps issues labeled kiln:needs-triage, merges duplicates, fills in missing acceptance criteria, sets type and size labels, promotes finished issues with `kiln move <n> ready`, and orders the ready queue by priority. Use when the backlog has untriaged issues or feels messy.
disable-model-invocation: true
---

# kiln-triage

Turns a pile of rough issues into a queue kiln can work from. Every issue that leaves triage has observable acceptance criteria, a type, a size and a gitmoji title. This is a Gather stage skill.

## Reads

- Open issues labeled `kiln:needs-triage`, plus open `kiln:*` issues and recently closed ones (for duplicates)
- `.kiln/memory/PROJECT.md` and `.kiln/memory/lessons.md` (to size and scope well)
- `.kiln/config.yaml` (autonomy, and the size definitions under `rigor`)

## Steps

1. Collect the sweep and the context around it:

   ```bash
   gh issue list --label kiln:needs-triage --state open --limit 100 --json number,title,body,labels,author,createdAt,comments
   gh issue list --label kiln:ready --state open --limit 100 --json number,title,body,labels
   gh issue list --state closed --search "<keywords> sort:updated-desc" --limit 20 --json number,title,stateReason
   ```

2. Find duplicates among the sweep and against open and recently closed issues. Keep the older or better specified issue as canonical.

3. For each remaining issue, check that the body has:
   - a problem statement (what is wrong or missing, for whom)
   - acceptance criteria as numbered predicates `P1 … Pn`, each an outcome you could watch pass or fail
   - at least one `must not change:` predicate
   - an out of scope line

   Rewrite anything unobservable ("feels faster" becomes "p95 of `GET /search` under 200 ms on the seed data"). Keep the author's words under `## Original request`. If you cannot infer criteria without guessing, do not invent them: the issue stays in needs-triage with a comment asking the questions.

4. Set type and size:
   - type: `type:feature`, `type:bug`, `type:refactor`, `type:chore` or `type:docs`
   - size: `S` one area, about 100 changed lines or fewer, no public interface change · `M` several modules or a public interface · `L` cross-cutting, migrations, security or data
   - title: `<gitmoji> <lowercase summary>`, for example `🐛 login redirect loop after session expiry`

5. Show the triage plan as one table before writing anything:

   ```
   #   action                         type     size  →
   61  duplicate of #44 (close)        -        -     -
   62  criteria added (P1-P3, P4 mnc)  feature  M     ready
   63  needs info: asked 2 questions   bug      ?     stays
   64  title + size only               chore    S     ready
   ```

   In manual and gated mode, wait for an ok. In autopilot, apply it.

6. Apply, issue by issue:

   ```bash
   # duplicates: carry over anything unique, then close
   gh issue comment <keep> --body "Merged from #<dup>: <details worth keeping>"
   gh issue close <dup> --reason "not planned" --comment "Duplicate of #<keep>."

   # body, title, type and size
   gh issue edit <n> --title "<gitmoji> <summary>" --body-file <file>
   gh issue edit <n> --add-label type:<type>,size:<S|M|L> --remove-label <old type/size label>

   # questions for issues that cannot be finished yet
   gh issue comment <n> --body "<the questions>"

   # promote only complete issues
   kiln move <n> ready
   ```

   `kiln move` exits 3 on an illegal move: report it and leave the issue alone.

7. Order the ready queue by priority: bugs hurting users today, then anything blocking another issue, then features, then refactors, chores and docs. Within a group, smaller and older first. Show the ordered list with a one-line reason per issue.

## Writes

- Issue titles, bodies and comments; closed duplicates
- `type:*` and `size:*` labels through `gh issue edit`
- `kiln:ready` only through `kiln move <n> ready`

## Rules

- `kiln:*` state labels change only through `kiln move`. Never `gh issue edit --add-label kiln:…`.
- Never invent acceptance criteria to get an issue promoted. Ask instead.
- Never loosen criteria the author wrote; tighten or clarify them.
- Close duplicates, never delete issues. Always link the canonical issue.
- Issues touching PROJECT.md's "Do not touch without asking" areas get a note saying so in the body.

## Done when

Every swept issue is promoted, closed as a duplicate, or waiting on a question, and the ordered ready queue is shown.

Journal row for kiln-log:

```
kiln-triage · <n> triaged: <r> ready, <d> duplicate, <q> need info · kiln:needs-triage · ok
```
