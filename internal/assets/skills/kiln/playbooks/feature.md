# Playbook: feature

New behavior a user can see. Every stage runs in full. Review and verification depth follow the size label.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | run | issue with observable acceptance criteria |
| Shape | run | recall · explore · plan (◆1) |
| Fire | run | test-first build, small commits |
| Glaze | run | predicates proven · review tier by size |
| Ship | run | pr · watch · merge (◆2) |
| Temper | run | log · retro offer |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1.  kiln-recall · lessons, decisions and past tasks about <area>
2.  kiln-issue · <gitmoji> <title> with predicates P1..Pn
3.  kiln-explore · map the code in <area>
4.  ◆1 kiln-plan · plan comment on #<n>
5.  kiln-plan · decision record in the worktree, if the plan made a call
6.  kiln-start · ../<repo>.kiln/<n>-<slug>
7.  kiln-build · test-first, one predicate at a time
8.  kiln-commit · gitmoji commits, reordered before the PR
9.  kiln-verify · P1..Pn + standing checks · checker <self|fresh|swarm>
10. kiln-review · <self|independent|adversarial>
11. kiln-pr · ready PR, Closes #<n>
12. kiln-watch · conflicts, threads, CI
13. kiln-verify · swarm at the merge-ready head
14. kiln-log · final pass: skips, retro notes, pushed before ◆2
15. ◆2 kiln-merge · PR #<pr>
16. kiln-retro · offer if retro_queue is 5 or more
```

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 1 kiln-recall | never | |
| 2 kiln-issue | the task arrived as an existing issue | `skip: kiln-issue · #<n> already exists` |
| 3 kiln-explore | a task journal from the last month already maps this area | `skip: kiln-explore · <area> mapped in tasks/<file>.md` |
| 4 kiln-plan | never for features. The user may waive it for S only | `skip: kiln-plan · <user's words>` |
| 5 decision record | the plan made no lasting call | `skip: decision record · no call outlives the task` |
| 13 swarm | size S or M | `skip: kiln-verify swarm · size <S\|M>, verdict from step 9 covers it` |
| 16 kiln-retro | `retro_queue` below 5 | `skip: kiln-retro · retro_queue <k>` |

Size L: kiln-plan splits the work into sub-issues. Each sub-issue then runs this playbook from step 6 (kiln-start) on its own branch, worktree and ◆2; the parent's ◆1 covers them all. The parent issue stays at `kiln:planned` until the last sub-issue merges, then kiln-merge offers to close it.

## Notes

- Step 9 picks its checker from `rigor.<size>.verify`: S `self`, M `fresh`, L runs a normal pass here and the swarm at step 13.
- If kiln-watch pushes code after step 9, the verdict may go stale. kiln-merge finds out through `kiln ready` and sends you back to kiln-verify and kiln-review.
- Step 14 exists because `kiln merge` deletes the branch and worktree. Anything not pushed before ◆2 is lost.
