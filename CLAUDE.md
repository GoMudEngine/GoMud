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

## Stat & Progression System
- All stats (Strength, Dexterity, Perception, Vitality, Willpower, Charisma) are centered at **100 = human baseline**
- Stats improve via **use-based progression only** — `OnStatUse()` triggers probabilistic advancement. There is NO level-based or XP-based stat gain; levels and XP are being removed from the game entirely.
- Soft cap: raw stat >= 105 applies `adjusted = 100 + sqrt(raw-100) * 2`; effective ceiling ~116 at raw 150
- Skills (9 total) cap softly at 50 (`skillSoftCap`). They progress via `OnSkillUse()` → `CheckSkillProgression()`, probabilistically, every ~25 uses.

## Dice & Rolling System
- All stat-based rolls use `dice.Roll(mean, stdDev)` or `dice.OpposedRoll(atk, def, stdDev)` from `internal/dice/dice.go`
- **stdDev must always be 15% of the mean**: use `dice.StdDevFor(mean)` — never hardcode `15.0`
  - `dice.StdDevFor(mean)` returns `mean * 0.15`, with a floor of 1.0
- Z-score thresholds: `ZScore >= 2.0` = crit success; `ZScore <= -2.0` = fumble/backfire
- For opposed rolls, use the attacker's score as the mean for `StdDevFor`: `dice.OpposedRoll(atkScore, defScore, dice.StdDevFor(atkScore))`
- `util.Rand` / `util.LogRoll` are NOT used for hit or attack checks; only `dice.*` functions

## Physical Armor Model
Physical defense comes from exactly **3 sources** — never use Vitality as a proxy for armor:
1. **Worn equipment** — sum of `DamageReduction` across all equipped slots (head, body, legs, feet, hands, neck, ring, offhand)
2. **Natural armor** — species traits or mutations (e.g., turtle shell, stone skin); stored as a character field or condition (TBD)
3. **Magical effects** — temporary conditions such as `ConditionShield` (Minor Shield spell)

Mental defense = `Willpower` stat only. Vitality governs hit points and physical endurance, not armor.

## Data File Naming Convention
Before creating any new data file, verify the expected filename from the loader's `Filepath()` method:
- Buffs: `{buffid}-{ConvertForFilename(name)}.yaml` — e.g., `name: Stunned` → `2-stunned.yaml`
- `ConvertForFilename()`: lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore
- Spells: use the `spellid` field value directly as the filename base (no conversion needed)
- Items/mobs follow the same `ConvertForFilename` pattern
- Always confirm the `.js` stub filename matches the `.yaml` filename (same base, different extension)
- Mismatch between filesystem path and `Filepath()` output causes a startup panic
