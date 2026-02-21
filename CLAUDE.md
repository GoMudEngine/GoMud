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
- **For all stat-based rolls use `dice.RollStat(mean)` or `dice.OpposedRollStat(atk, def)`** — no stdDev argument needed
- These wrappers automatically apply the global `RollSpread` factor: `stdDev = mean × RollSpread`
- `dice.Roll(mean, stdDev)` / `dice.OpposedRoll(atk, def, stdDev)` are low-level; only use them when variance is NOT stat-proportional (e.g., weapon damage variance from item specs)
- **`RollSpread`** is the single master randomness knob — set in `_datafiles/config.yaml` under `GamePlay.RollSpread` (default **0.15**). Changing it rescales every dice roll in the engine. See `internal/dice/README.md` for win-probability tables.
- Z-score thresholds: `ZScore >= 2.0` = crit; `ZScore <= -2.0` = fumble/backfire (~2.3% each, unaffected by `RollSpread`)
- `util.Rand` / `util.LogRoll` are NOT used for hit or attack checks; only `dice.*` functions

## Physical Armor Model
Physical defense comes from exactly **3 sources** — never use Vitality as a proxy for armor:
1. **Worn equipment** — sum of `DamageReduction` across all equipped slots (head, body, legs, feet, hands, neck, ring, offhand)
2. **Natural armor** — species traits or mutations (e.g., turtle shell, stone skin); stored as a character field or condition (TBD)
3. **Magical effects** — temporary conditions such as `ConditionShield` (Minor Shield spell)

Mental defense = `Willpower` stat only. Vitality governs hit points and physical endurance, not armor.

## Data File Naming Convention
Before creating any new data file, verify the expected filename from the loader's `Filepath()` method:
- **Zone folder names must use underscores, not hyphens.** The engine derives the expected path by calling `ConvertForFilename()` on the zone's display name (e.g., `"Sanctum Basin"` → folder `sanctum_basin/`). A mismatch causes a startup panic: `filesystem path "..." did not end in Filepath() "..."`. This applies to both `rooms/` and `mobs/` subdirectories.
- Buffs: `{buffid}-{ConvertForFilename(name)}.yaml` — e.g., `name: Stunned` → `2-stunned.yaml`
- `ConvertForFilename()`: lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore
- Spells: use the `spellid` field value directly as the filename base (no conversion needed)
- Items/mobs follow the same `ConvertForFilename` pattern
- Always confirm the `.js` stub filename matches the `.yaml` filename (same base, different extension)
- Mismatch between filesystem path and `Filepath()` output causes a startup panic

## Player-Facing Messages — No Hard Numbers
Never display raw numeric values (damage, healing, armor points, round counts, etc.) directly to the player in combat or spell messages. Use descriptive language instead:
- **Damage**: use `combat.GetDamageDescription(amount, targetMaxHP)` → "light wounds", "serious wounds", etc.
- **Healing**: use `combat.GetHealDescription(amount, targetMaxHP)` → "light mending", "moderate restoration", etc.
- **Durations / other numbers**: describe the effect, not the mechanics ("A barrier forms around you." not "A barrier forms for 10 rounds.")
- **Armor / stat bonuses**: describe the feel ("bolsters your defenses" not "+33 armor")

Displaying raw numbers breaks immersion and leaks internal balance values to players. The exception is the `status` command's stat sheet — that is a deliberate mechanical display.
