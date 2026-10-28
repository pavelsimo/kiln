# Playbook: release

Ship what main already has. A release has no issue of its own: it gathers every task merged since the last tag. The version and grouping come from `kiln release plan`, and the release gate stops in every autonomy mode.

| Stage | Mode | What runs |
|-------|------|-----------|
| Gather | skip | skip: collects closed issues |
| Shape | light | version + changelog draft from `kiln release plan --json` |
| Fire | light | 🔖 release commit on `kiln/release-v<version>` |
| Glaze | light | CI on the release PR |
| Ship | run | pr · ◆ release gate · merge · tag · publish |
| Temper | light | log only |

## Todo list

Copy these lines into the todo list word for word, filling the `<placeholders>`.

```
1. skip: kiln-issue · a release collects closed issues
2. kiln-release · preflight and kiln release plan --json → v<version>
3. kiln-release · changelog section and version_files edited for readers
4. kiln-release · 🔖 chore(release): v<version> on kiln/release-v<version>
5. kiln-release · ready release PR, CI green through kiln-watch
6. kiln-log · tasks/release-v<version>.md on the release branch, pushed before the gate
7. ◆ release gate · version, changelog, CI and publish route shown · always stops
8. kiln-release · kiln merge, tag the merge SHA, push the tag, publish
```

All the release mechanics live in `.kiln/skills/kiln-release/SKILL.md`. Follow it for steps 2 to 8; this list only keeps the stages visible.

## Skip rules

| Step | Skip when | Write |
|------|-----------|-------|
| 1 kiln-issue | always | `skip: kiln-issue · a release collects closed issues` |

Nothing else may be skipped. If `kiln release plan` exits 3 (nothing releasable, or the tag already exists), stop the playbook and report its reason. Don't force a version unless the user gave one.

## Notes

- The release gate is the "deploy or release" hard stop. Autopilot stops here too.
- A broken release is fixed by cutting a patch release. Never move, delete or force-push a tag.
- The release journal is written before the gate because `kiln merge` deletes the release branch. The merge, tag and publish results are read back from GitHub and reported, not written to the journal.
