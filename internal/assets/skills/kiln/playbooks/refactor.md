# Playbook: refactor

Same behavior, better shape. Behavior is pinned by tests before anything moves, and the proof is that observable output did not change.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | run | issue names the target shape |
| Shape | run | plan the smallest safe steps (◆1) |
| Fire | run | characterization tests pin behavior first |
| Glaze | run | pinned tests green · observable output unchanged |
| Ship | run | pr · watch · merge (◆2) |
| Temper | run | decision record for the new shape |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1.  kiln-recall · decisions and lessons about <area>
2.  kiln-issue · ♻️ <title> naming the target shape, predicates P1..Pn
3.  kiln-explore · map callers and tests of <area>
4.  ◆1 kiln-plan · smallest safe steps on #<n>
5.  kiln-start · ../<repo>.kiln/<n>-<slug>
6.  kiln-plan · decision record for the target shape, in the worktree
7.  kiln-build · characterization tests pin current behavior, green on the old code
8.  kiln-build · refactor in small steps, pinned tests green after each
9.  kiln-commit · ✅ test: pin … first, then ♻️ refactor: … steps
10. kiln-verify · pinned tests green · output diffed before and after
11. kiln-review · <self|independent|adversarial>
12. kiln-pr · ready PR, Closes #<n>
13. kiln-watch · conflicts, threads, CI
14. kiln-verify · swarm at the merge-ready head
15. kiln-log · final pass: skips, decisions, retro notes, pushed before ◆2
16. ◆2 kiln-merge · PR #<pr>
```

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 2 kiln-issue | the refactor arrived as an existing issue | `skip: kiln-issue · #<n> already exists` |
| 3 kiln-explore | never. A refactor that misses a caller breaks it | |
| 4 kiln-plan | never. The step order is the whole safety story | |
| 6 decision record | the new shape is a trivial rename with no lasting call | `skip: decision record · rename only` |
| 7 characterization | existing tests already pin every behavior the change touches (name them) | `skip: characterization · pinned by <test files>` |
| 14 swarm | size S or M | `skip: kiln-verify swarm · size <S\|M>, verdict from step 10 covers it` |

## Notes

- Most predicates in a refactor are "must not change" ones: public API, CLI output, HTTP responses, stored data.
- Step 10 follows the refactor row of the check table: existing checks pass, and observable output is captured on the old code and the new code and diffed. "Looks equivalent" is not evidence.
- If a step reveals a behavior change you want, stop. That is a separate feature or bug issue, not part of this refactor.
