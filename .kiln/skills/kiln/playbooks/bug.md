# Playbook: bug

Something is wrong today. The failing reproduction lands before the fix, and verification sees the bug on the old build and its absence on the new one.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | run | issue with repro steps, expected vs actual |
| Shape | light | recall · explore; plan only if M or larger |
| Fire | run | failing repro commit lands first, then the fix |
| Glaze | run | seen twice on the old build, gone twice on the new · root cause named |
| Ship | run | pr · watch · merge (◆2) |
| Temper | run | retro note if the bug class repeats |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1.  kiln-recall · past bugs and lessons about <area>
2.  kiln-issue · 🐛 <title> with repro steps and predicates P1..Pn
3.  kiln-explore · light: trace the failing path in <area>
4.  ◆1 kiln-plan · cause, fix and repro test on #<n>
5.  kiln-start · ../<repo>.kiln/<n>-<slug>
6.  kiln-build · failing reproduction test, committed alone
7.  kiln-build · smallest fix that turns the reproduction green
8.  kiln-commit · ✅ test: reproduce … then 🐛 fix: …, in that order
9.  kiln-verify · before/after: 2 of 2 on old, 0 of 2 on new · root cause at file:line
10. kiln-review · <self|independent|adversarial>
11. kiln-pr · ready PR, Closes #<n>
12. kiln-watch · conflicts, threads, CI
13. kiln-log · final pass: skips, retro notes, pushed before ◆2
14. ◆2 kiln-merge · PR #<pr>
```

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 2 kiln-issue | the bug arrived as an existing issue | `skip: kiln-issue · #<n> already exists` |
| 3 kiln-explore | the stack trace or repro already points at the file | `skip: kiln-explore · failure points at <file:line>` |
| 4 kiln-plan | size S | `skip: kiln-plan · S bug, the repro test is the plan` |

Never skip steps 6 or 9. A bug fix without a reproduction is a guess.

## Notes

- The issue's P1 is the symptom, stated so it can be watched: "following the repro steps lands on /login, not a redirect loop".
- At least one predicate says what must not change around the fix.
- In step 13, if `kiln-recall` found an earlier bug of the same class, kiln-log adds a Retro note naming both issues. That's how kiln-retro spots a pattern.
- A fix that only hides the symptom (a nil check, a swallowed error, a retry, a longer timeout) comes back NOT VERIFIED and loops to step 7.
