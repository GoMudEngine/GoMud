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
- Soft cap: stats are linear up to `StatSoftCap` (default 150), then diminishing returns: `adjusted = softCap + (raw - softCap)^0.75 * multiplier` (default multiplier 2.0). `StatSoftCapThreshold` (105) is the floor below which no adjustment applies.
- Skills (9 total) cap softly at 50 (`skillSoftCap`). They progress via `OnSkillUse()` → `CheckSkillProgression()`, probabilistically, every ~25 uses.

## Dice & Rolling System
- **For all stat-based rolls use `dice.RollStat(mean)` or `dice.OpposedRollStat(atk, def)`** — no stdDev argument needed
- These wrappers automatically apply the global `RollSpread` factor: `stdDev = mean × RollSpread`
- `dice.Roll(mean, stdDev)` / `dice.OpposedRoll(atk, def, stdDev)` are low-level; only use them when variance is NOT stat-proportional (e.g., weapon damage variance from item specs)
- **`RollSpread`** is the single master randomness knob — set in `_datafiles/config.yaml` under `GamePlay.RollSpread` (default **0.15**). Changing it rescales every dice roll in the engine. See `internal/dice/README.md` for win-probability tables.
- Z-score thresholds: `ZScore >= 2.0` = crit; `ZScore <= -2.0` = fumble/backfire (~2.3% each, unaffected by `RollSpread`)
- `util.Rand` / `util.LogRoll` are NOT used for hit or attack checks; only `dice.*` functions

## Unified Damage & Mitigation Pipeline (Stage 34)
All damage flows through a three-channel pipeline in `internal/combat/damage_pipeline.go`:

### Damage Formula (all channels)
```
raw_damage = stat × SkillMultiplier(rank) × item_multiplier
final_damage = ApplyMitigation(raw, mitigation%, cap)
```
Then `dice.RollStat(final_damage)` for variance.

### Three Channels
| Channel | Stat | Skills | Item Field | Mitigation Method |
|---------|------|--------|-----------|------------------|
| Physical | Strength | weapon/unarmed/ranged-combat | `damage_multiplier` (weapon) | `GetPhysicalMitigation()` |
| Magical | Willpower | spellcasting | `damage_multiplier` (spell) | `GetMagicalMitigation()` |
| Conviction | Charisma | rhetoric | 0.5 (taunt base) | `GetConvictionMitigation()` |

### Skill Multiplier Curve
`mult = base + (max - base) × sqrt(rank / softCap)` — Config: `SkillMultiplierBase` (1.0), `SkillMultiplierMax` (3.0)

### Item Mitigation Fields (replaces old single DamageReduction)
Items use `physical_mitigation`, `magical_mitigation`, `conviction_mitigation` (integer percentages).
Old `DamageReduction` field is kept for legacy compatibility but no longer used by the pipeline.

### Mitigation Caps
Default 75% each: `PhysicalMitigationCap`, `MagicalMitigationCap`, `ConvictionMitigationCap`

### Key Functions
- `combat.CalcRawDamage(stat, skillRank, itemMult)` — compute raw damage
- `combat.ApplyMitigation(raw, pct, cap)` — apply percentage reduction
- `combat.SkillMultiplier(rank)` — sqrt curve from config
- `character.GetPhysicalMitigation()` / `GetMagicalMitigation()` / `GetConvictionMitigation()` — sum equipment

## Regen System (Stage 29.5)
All HP/SP/CP regeneration is **percentage-of-max** — never flat values.
- Six config knobs in `Balance`: `PlayerHealthRegenPct`, `PlayerStaminaRegenPct`, `PlayerConvictionRegenPct`, `MobHealthRegenPct`, `MobStaminaRegenPct`, `MobConvictionRegenPct` (default 0.01 = 1% per tick)
- `HealthPerRound()` / `StaminaPerRound()` / `ConvictionPerRound()` compute `floor(poolMax * pct)`, min 1
- **Mutations** use multiplier effects (`health_regen_multiplier`, `health_regen_if_lit_multiplier`, `stamina_regen_multiplier`) — never flat `health_regen` effects
- **Heal spells** store a regen multiplier in `effect_magnitude` (e.g. 3 = 3x base regen); applied via `ConditionRegen`
- **Buff scripts** that heal should use `Math.floor(actor.GetHealthMax() * fraction)` — never flat dice for healing
- NPCs regen health (out of combat), stamina (1/4 in combat), and conviction every tick

## Data File Naming Convention
Before creating any new data file, verify the expected filename from the loader's `Filepath()` method:
- **Zone folder names must use underscores, not hyphens.** The engine derives the expected path by calling `ConvertForFilename()` on the zone's display name (e.g., `"Sanctum Basin"` → folder `sanctum_basin/`). A mismatch causes a startup panic: `filesystem path "..." did not end in Filepath() "..."`. This applies to both `rooms/` and `mobs/` subdirectories.
- Buffs: `{buffid}-{ConvertForFilename(name)}.yaml` — e.g., `name: Stunned` → `2-stunned.yaml`
- `ConvertForFilename()`: lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore
- Spells: use the `spellid` field value directly as the filename base (no conversion needed)
- Items/mobs follow the same `ConvertForFilename` pattern
- Always confirm the `.js` stub filename matches the `.yaml` filename (same base, different extension)
- Mismatch between filesystem path and `Filepath()` output causes a startup panic

## MUD Line Width
All player-visible text (descriptions, help files, templates, ANSI-formatted tables) must wrap at **80 characters per line**. MUD clients render in fixed-width columns — long lines get cut off or wrap uglily. When writing multi-line `description:` fields, room descriptions, or help templates, hard-wrap prose at ~78–80 chars.

## Player-Facing Messages — No Hard Numbers
Never display raw numeric values (damage, healing, armor points, round counts, etc.) directly to the player in combat or spell messages. Use descriptive language instead:
- **Damage**: use `combat.GetDamageDescription(amount, targetMaxHP)` → "light wounds", "serious wounds", etc.
- **Healing**: use `combat.GetHealDescription(amount, targetMaxHP)` → "light mending", "moderate restoration", etc.
- **Durations / other numbers**: describe the effect, not the mechanics ("A barrier forms around you." not "A barrier forms for 10 rounds.")
- **Armor / stat bonuses**: describe the feel ("bolsters your defenses" not "+33 armor")

Displaying raw numbers breaks immersion and leaks internal balance values to players. The exception is the `status` command's stat sheet — that is a deliberate mechanical display.

## Content Generation Commands
Use slash commands to generate new data files. Claude automatically loads world.md,
the relevant schema, and existing examples before generating.

- `/new-mob "description"` — generate a mob YAML (+ optional JS stub)
- `/new-room "description"` — generate a room YAML
- `/new-item "description"` — generate an item YAML
- `/zone-sketch "concept"` — plan a new zone (room list + adjacency) before generating rooms

Schema reference: `docs/schemas/` (room, mob, item, spell, buff, dialogue)
Full workflow: `docs/CONTENT_GENERATION_GUIDE.md`

After generating any file: restart server. If editing an existing zone, check
`_datafiles/world/dogmud/rooms.instances/` for stale instance saves.
