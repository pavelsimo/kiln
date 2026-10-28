---
name: kiln-watch
description: Takes an open kiln PR to green by asking `kiln ready <pr> --json` what is left, then handling merge conflicts, review threads and CI in that order, retrying a flaky job once and moving the issue to blocked if CI stays red. Use after kiln-pr opens a PR, or on its own to nudge a stuck PR.
disable-model-invocation: true
---

# kiln-watch

Gets a PR from "opened" to "kiln ready says yes". It works in a fixed order (conflicts, then review threads, then CI) because each step can change what the next one sees. This is a Ship stage skill. It never merges; kiln-merge does that at gate 2.

## Reads

- `kiln ready <pr> --json` (what is left, in order)
- `kiln verdict <n> --json` (after anything that changes the branch)
- Review threads and CI logs from gh
- `.kiln/memory/PROJECT.md` (test and lint commands), `.kiln/memory/lessons.md` (known flaky jobs)

## Steps

1. Locate the PR and its worktree. The issue number comes from the branch `kiln/<n>-<slug>`; release PRs (`kiln/release-v*`) have no issue.

   ```bash
   gh pr view <pr> --json headRefName,baseRefName,isDraft
   cd ../<repo>.kiln/<n>-<slug>
   kiln ready <pr> --json
   ```

   If the PR is a draft, `gh pr ready <pr>`; kiln opens ready PRs. If `ready` is true, you are done.

2. **Conflicts first** (the mergeable check is failing):

   ```bash
   git fetch origin
   git rebase origin/<default>
   # resolve each conflict keeping both sides' intent, then run PROJECT.md's test and lint commands
   git push --force-with-lease origin kiln/<n>-<slug>
   kiln verdict <n> --json
   ```

   If you cannot resolve a conflict confidently, stop and ask. If `kiln verdict` now says stale, the change set changed: run kiln-verify and then kiln-review before going on.

3. **Review threads next** (unresolved threads or CHANGES_REQUESTED):

   ```bash
   gh api graphql -F owner=<owner> -F repo=<repo> -F pr=<pr> -f query='
     query($owner:String!,$repo:String!,$pr:Int!){repository(owner:$owner,name:$repo){
       pullRequest(number:$pr){reviewThreads(first:100){nodes{
         id isResolved path line comments(first:20){nodes{author{login} body}}}}}}}'
   ```

   For each unresolved thread:
   - valid and small: fix it in a commit, reply with what changed and the SHA, then resolve the thread
   - valid but needs real rework: `kiln move <n> in-progress` and hand it to kiln-build
   - you disagree: reply with your reasoning and a `file:line` or command, and leave it for the reviewer to resolve
   - a question: answer it

   ```bash
   gh api graphql -F id=<thread-id> -f body='<reply>' -f query='
     mutation($id:ID!,$body:String!){addPullRequestReviewThreadReply(input:{pullRequestReviewThreadId:$id,body:$body}){comment{id}}}'
   gh api graphql -F id=<thread-id> -f query='
     mutation($id:ID!){resolveReviewThread(input:{threadId:$id}){thread{isResolved}}}'
   gh pr edit <pr> --add-reviewer <login>   # re-request review after CHANGES_REQUESTED
   ```

   Any code change makes the verdict stale: kiln-verify and kiln-review run again on the new SHA.

4. **CI last** (failing or pending checks). Find the runs for the head SHA and read only what failed:

   ```bash
   gh run list --branch kiln/<n>-<slug> --commit <head-sha> --json databaseId,workflowName,name,status,conclusion
   gh run view <run-id> --log-failed
   ```

   Sort each failure:
   - **real**: a deterministic assertion, compile, lint or type error in or near the diff, or it fails locally too. Reproduce it with PROJECT.md's command, fix it, commit, push, and rerun verify and review.
   - **flaky**: a timeout, network error, runner problem, a job lessons.md calls flaky, or a failure unrelated to the diff that passes locally. Retry it once:

   ```bash
   gh run rerun <run-id> --failed
   gh run watch <run-id> --exit-status
   ```

   Pending checks: wait with `gh pr checks <pr> --watch`, then ask `kiln ready` again.

5. **Still red after the one retry**: stop guessing and block the issue.

   ```bash
   kiln move <n> blocked --reason "CI job <name> red after one retry on <sha>: <one-line cause>"
   ```

   For a release PR there is no issue to block; report it to the person instead. Give kiln-log a Retro note naming the job, so the next retro can turn it into a lesson.

6. Loop: after each fix, run `kiln ready <pr> --json` again and take the first failing check. Checks that are not yours to fix: a stale verdict (run kiln-verify, then kiln-review), REVIEW_REQUIRED from a human (report and stop), a missing `Closes #N` (add it with `gh pr edit <pr> --body-file <file>`). After three rounds with no progress, block the issue with the reason.

## Writes

- Commits on the task's own branch, pushed (with `--force-with-lease` only after a rebase)
- Replies on review threads, resolved threads, re-requested reviews
- One retry per flaky job
- `kiln move <n> blocked --reason …` or `kiln move <n> in-progress` when needed

## Rules

- Order is fixed: conflicts, then threads, then CI.
- Never force-push anything except `--force-with-lease` on the task's own `kiln/` branch. Never push to the default branch.
- Retry a flaky job once per head SHA. A second red is not flaky.
- Never skip, disable or weaken a test or a CI job to get green, and never edit workflows unless the issue is about them.
- Never merge. "Ready" is the binary's answer, and the merge belongs to kiln-merge.
- Commit messages follow the gitmoji convention and never carry Co-authored-by trailers or AI attribution.

## Done when

`kiln ready <pr> --json` exits 0 (hand off to kiln-merge for gate 2), or the issue is blocked with a reason, or you stopped to ask about a conflict or a human review.

Journal row for kiln-log:

```
kiln-watch · <rebased, n threads, n retries> · #<pr> · ready|blocked
```
