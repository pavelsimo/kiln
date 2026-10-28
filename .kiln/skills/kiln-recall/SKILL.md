---
name: kiln-recall
description: Rebuilds context before work by searching kiln's memory (past task journals, decision records, lessons) for anything related to the current task, then checking the live state of everything it found on GitHub. Use at the start of Shape, or whenever you need to know what the project already learned about an area.
disable-model-invocation: true
---

# kiln-recall

Finds what the repo already knows about the task at hand, so the plan starts warm. Memory says why; GitHub says where. You read old notes for reasoning and read live state for everything that moves. Runs in the Shape stage, usually first.

## Reads

- `.kiln/memory/lessons.md`, `.kiln/memory/decisions/*.md`, `.kiln/memory/tasks/*.md`, `.kiln/memory/PROJECT.md`
- Journals of other in-flight tasks in `../<repo>.kiln/*/.kiln/memory/tasks/` (not merged yet)
- The issue (`gh issue view <n>`), and live state of every issue and PR your notes mention

## Steps

1. Pull keywords from the issue: the area, file and module names, the feature, the error text for a bug.

   ```bash
   gh issue view <n> --json title,body,labels
   ```

2. Search merged memory on the default branch:

   ```bash
   grep -rniE '<kw1>|<kw2>|<kw3>' .kiln/memory/lessons.md .kiln/memory/decisions .kiln/memory/tasks
   ```

   Then search in-flight journals, which live only in their worktrees until they merge:

   ```bash
   grep -rliE '<kw1>|<kw2>' ../<repo>.kiln/*/.kiln/memory/tasks/ 2>/dev/null
   ```

3. Read what matched:
   - lessons: keep the confidence (`[strong]`, `[recurring]`, `[candidate]`) and the issue refs.
   - decisions: keep the title, `status` and the Decision paragraph.
   - task journals: the goal, Decisions, Skipped and Retro notes, plus any FIX or NOT VERIFIED rows.

4. Check live state for everything you plan to cite. Never trust an old note about something that moves:

   ```bash
   gh issue view <ref> --json state,title,labels
   gh pr view <ref> --json state,mergedAt,title
   git log -1 --format='%h %cs' -- <path-a-note-mentions>
   ```

   A path that no longer exists, a decision with `status: superseded`, or an issue reopened since: flag the note as stale rather than repeating it.

5. Check overlap with work in flight. If another in-flight task touches the same area, name it (issue, worktree, stage) so kiln-plan can avoid a conflict.

6. Write a short brief in the conversation:

   ```
   recall · #42 dark mode settings
   lessons    [strong] run pnpm test:e2e --update after any token change (#31 #42 #49)
   decisions  2026-10-27 css-tokens · accepted · colors only via tokens
   tasks      #31 settings page · settings mock lives in test/fixtures
   in flight  #48 touches src/theme/** (fire, ../app.kiln/48-theme-pages)
   stale      tasks/12-…: mentions src/legacy/theme.ts, deleted in 3f2b7d1
   ```

   Nothing found is a valid result. Say so in one line.

## Writes

No memory files. The brief goes into the conversation, and its one-line summary goes to kiln-log for the journal.

## Rules

- Read-only. Never edit lessons.md, decisions or another task's journal; each has its own writer.
- Live state (labels, CI, merged, open or closed) always comes from GitHub, never from a note.
- Strong lessons are rules for this task. Recurring and candidate lessons are context, so say which is which.
- Keep the brief short. Cite files and refs; do not paste whole journals.

## Done when

The brief is shown, with every cited item checked against live state and every stale note flagged.

Journal row for kiln-log:

```
kiln-recall · <n> lessons, <n> decisions, <n> tasks · #<n> · ok
```
