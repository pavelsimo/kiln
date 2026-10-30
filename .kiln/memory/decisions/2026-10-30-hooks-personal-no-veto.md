---
date: 2026-10-30
status: accepted
source: #3
---
# Hooks are personal and never veto
## Context
People want kiln to drive their own tools (herdr, tmux, notifications) when tasks change state, without kiln depending on any of them. A hook is a shell command, so where it's configured decides who gets to run code on your machine.
## Decision
Hooks live only in `~/.config/kiln/hooks.yaml` (or `$XDG_CONFIG_HOME`) and an untracked `.kiln/hooks.local.yaml`, never in the committed `.kiln/config.yaml`. They run after a transition succeeds and can't stop it: a failing hook is a warning, never an exit code. Kiln ships recipes for tools in the docs, not adapters in code.
## Consequences
+ cloning a repo and running `kiln move` never runs commands someone else committed
+ each person picks their own tools; supporting a new one is a docs recipe, not a release
- a team can't share hooks through the repo; they share recipes instead
- gates stay inside kiln, so a hook can't be used as a pre-merge check
