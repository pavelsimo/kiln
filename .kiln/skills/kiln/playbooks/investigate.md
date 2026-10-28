# Playbook: investigate

Answer a question. No code change, no branch, no PR. The output is a written report and, when a finding should outlive the conversation, a decision record.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | light | issue optional |
| Shape | run | recall · explore → written report |
| Fire | skip | skip: read-only |
| Glaze | skip | skip: nothing to verify |
| Ship | skip | skip: no PR |
| Temper | run | findings → decisions/ |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1. kiln-recall · past tasks, decisions and lessons about <topic>
2. kiln-issue · only if the user wants the question tracked
3. kiln-explore · read-only map of <topic>
4. report · answer with file:line evidence and a ladder step per claim
5. skip: kiln-start · kiln-build · kiln-commit · read-only
6. skip: kiln-verify · nothing to verify
7. skip: kiln-pr · kiln-watch · kiln-merge · no PR
8. decision record · decisions/<date>-<slug>.md for findings that outlive this conversation
9. skip: kiln-log · no task worktree, rows go in the report
```

Steps 5, 6, 7 and 9 are skipped by definition. Keep them in the list so the run shows what didn't happen.

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 2 kiln-issue | the user just wants an answer | `skip: kiln-issue · question only` |
| 8 decision record | nothing found is worth keeping past today | `skip: decision record · nothing lasting` |

## Writing the report (step 4)

You write it yourself, in the conversation. Shape:

```
question   how does token refresh work?
answer     <two or three sentences>
evidence   src/auth/refresh.ts:22  refresh is scheduled 60s before expiry   (2 pointed at)
           pnpm test auth/refresh  passes, asserts the 60s window           (4 ran it)
gaps       <what you couldn't confirm, and why>
follow-up  <a bug or feature worth an issue, if any>
```

Label each claim with its ladder step (1 asserted, 2 pointed at, 3 reasoned, 4 ran it, 5 reproduced). If you found something broken, offer `/kiln-issue` for it. Don't fix it here.

## Writing the decision record (step 8)

You made the call, so you write it, as a **new** file in the current checkout using the format in `.kiln/skills/kiln-log/SKILL.md`. Never edit an existing decision record. There is no branch to commit it to, so leave it uncommitted and tell the user where it is; offer a docs chore to land it.
