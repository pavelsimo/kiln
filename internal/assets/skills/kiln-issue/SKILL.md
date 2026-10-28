---
name: kiln-issue
description: Turns a rough request into a well-formed GitHub issue with a gitmoji title, a problem statement, observable acceptance criteria P1..Pn (including at least one "must not change"), out of scope, and type and size labels. Use at the Gather stage, or on its own to write an idea down for later; switches to interview mode when the request is vague.
disable-model-invocation: true
---

# kiln-issue

This is the Gather stage. You turn a sentence into the contract for the whole task: the acceptance criteria written here become the predicates kiln-verify must prove later, and they are never loosened after ◆1. The issue is created at `kiln:ready` when the criteria are complete, or at `kiln:needs-triage` when they aren't.

## Reads

- the request, and the goal line the router restated
- `.kiln/memory/PROJECT.md` (areas, conventions, no-go paths) and `.kiln/memory/lessons.md`
- `.github/ISSUE_TEMPLATE/kiln-task.yml`: the body uses its fields, **Problem**, **Acceptance criteria**, **Out of scope**
- open issues, to avoid duplicates

## Steps

1. **Look for a duplicate.**
   ```bash
   gh issue list --state open --search "<two or three keywords>" --json number,title,labels
   ```
   If one already covers the request, stop and offer to use it instead.

2. **Decide whether to interview.** If you can't write the problem in two sentences, or can't name one observable outcome, switch to interview mode. Ask in one batch, at most five questions:
   - Who notices this, and where (which screen, endpoint, command)?
   - What do they see when it's done that they don't see today?
   - What must keep working exactly as it does now?
   - What is explicitly not part of this?
   - For bugs: the exact steps, what you expected, what happened.

   Never invent criteria to fill a gap. Rough ideas are welcome; they just cost a question round.

3. **Write the title.** `<gitmoji> <lowercase imperative>`, no type prefix, no period, under 60 characters.

   | Type | Gitmoji | Example |
   |------|---------|---------|
   | feature | ✨ | `✨ add dark mode to settings` |
   | bug | 🐛 | `🐛 fix login loop after session expiry` |
   | refactor | ♻️ | `♻️ split rate limiter store from policy` |
   | chore | 🔧 or ⬆️ | `⬆️ bump eslint to v10` |
   | docs | 📝 | `📝 document the export api` |

4. **Write the predicates.** Each acceptance criterion is a predicate `P1..Pn`: one outcome that passes or fails when you look at the real surface (a screen, a response, an exit code, a stored value).
   - Name the surface: "a toggle in Settings → Appearance switches the page to dark".
   - At least one starts with `must not change:`, for the thing most likely to break ("must not change: the light theme looks exactly as before").
   - Rewrite anything unobservable, or ask:

     | Unobservable | Rewritten |
     |--------------|-----------|
     | feels faster | p95 of `GET /history` under 200 ms on the seed data |
     | cleaner settings page | (ask: what should a user see differently?) |
     | works with big files | importing a 500 MB CSV completes and shows 1,000,000 rows |
     | better errors | an expired token returns 401 with `{"error":"token_expired"}` |

   - Bugs: P1 is the symptom gone, stated against the repro steps. Refactors: name the target shape in the Problem, and most predicates are `must not change:` ones.

5. **Pick labels.** One `type:*` (feature, bug, refactor, chore, docs) and one `size:*` using the router's rules: S is one area, about 100 lines, no public interface change. M is several modules or a public interface. L is cross-cutting, migrations, security or data.

6. **Pick the starting state.** `kiln:ready` only if every predicate is observable, at least one is a `must not change`, the scope is clear, and the size is known. Otherwise `kiln:needs-triage`, and say what's missing.

7. **Create the issue.** Write the body to a temp file using the template's fields:
   ```markdown
   ### Problem
   Settings only has a light theme. Several users have asked for a dark one.

   ### Acceptance criteria
   - P1 a toggle in Settings → Appearance switches the page to dark
   - P2 the choice survives a reload
   - P3 a first visit follows the OS theme
   - P4 must not change: the light theme looks exactly as before

   ### Out of scope
   Other pages. They can pick up the theme in a follow-up.
   ```
   ```bash
   gh issue create --title "✨ add dark mode to settings" --body-file "$BODY" \
     --label type:feature --label size:M --label kiln:ready
   ```
   For bugs, add `Steps to reproduce`, `Expected` and `Actual` lines under Problem.

8. **Read it back.**
   ```bash
   gh issue view <n> --json number,title,labels,url
   ```

## Writes

- One GitHub issue with a type label, a size label and one `kiln:*` label.

## Rules

- The `kiln:*` label set at creation is the only one not set through `kiln move`. Every later state change goes through `kiln move`.
- Never mark an issue `kiln:ready` with an unobservable criterion or without a `must not change`.
- Never invent requirements the user didn't give. Ask.
- Out of scope is written down, even if it's one line.

## Done when

`gh issue view` shows the issue with its labels, and you've shown the user the title and predicates. Hand kiln-log the row (held until kiln-start creates the journal), plus the predicates for the journal's `## Acceptance criteria`:

```
kiln-issue · opened issue · #42 · ok
```

Use `needs-triage` as the result when it started there.
