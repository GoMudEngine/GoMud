# DOGMud - Claude Code Project Memory

## Subagent Model Preference
Always use `model: "haiku"` when spawning subagents via the Task tool, unless the task clearly requires deeper reasoning (complex refactoring, architectural decisions, multi-step code writing). Default to haiku for exploration, search, and simple research tasks.

## Git Workflow
Follow the branch strategy in `github_guide.md`:
- `master` tracks upstream GoMud exactly — never commit directly
- `development` is the main integration branch
- Feature branches: `feature/stage-X.Y-description`
- Merge features into development with `--no-ff`
- Use conventional commit messages (feat:, fix:, refactor:, docs:, chore:)

## Stage Completion SOP
After completing each substage (e.g., 3.6, 3.7, etc.):
1. Update `DEVELOPMENT_PLAN.md` — mark the stage as ✅ COMPLETED with merge commit hash
2. Update the timeline table status (e.g., "3.1–3.7 Complete")
3. Update the "Current Stage" line at the bottom to reflect the next stage

## Room Instance Saves (Important!)
When editing room YAML templates in `_datafiles/world/dogmud/rooms/`, always check for **instance saves** in `_datafiles/world/dogmud/rooms.instances/` that may override your changes. The engine loads templates first, then overwrites with instance data if present. After editing room templates:
1. Check `rooms.instances/<zone>/` for matching room files
2. Delete any stale instance saves so the engine loads fresh templates
3. Remind the user to restart the server after cleanup

Known issue: The instance save system can silently override template edits, making it appear that file changes aren't taking effect.

## Project Context
- DOGMud (Delusions of Grandeur) is a MUD built on the GoMud engine
- World design document: `world.md`
- Development roadmap: `DEVELOPMENT_PLAN.md`
- Remote origin: https://github.com/pruuk/DOGMud
- Remote upstream: https://github.com/GoMudEngine/GoMud
