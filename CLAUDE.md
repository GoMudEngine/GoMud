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

## Pre-Push SOP
Before pushing to prod (origin/master):
1. Update `PATCH_NOTES.md` with dated entries describing the work.
2. Set `Logging.LogToFile: false` in `_datafiles/config.yaml` (prod
   droplet has limited disk space).
3. **Boot the server locally and confirm it starts cleanly past
   data-file loading.** `go build` only checks compilation; YAML
   data files (mobs, items, quests, dialogues, rooms) panic at
   server startup if there's a filename/name-field mismatch, an
   invalid trigger event, an ID collision, or any other load-time
   issue that the build can't see. Spinning the server up locally
   and watching for `mobs.LoadDataFiles() loadedCount=...`,
   `quests.LoadDataFiles() loadedCount=...`, etc. without panics
   is the only reliable check before promoting to prod.

## Room Instance Saves (Important!)
When editing room YAML templates in `_datafiles/world/dogmud/rooms/`, always check for **instance saves** in `_datafiles/world/dogmud/rooms.instances/` that may override your changes. The engine loads templates first, then overwrites with instance data if present. After editing room templates:
1. Check `rooms.instances/<zone>/` for matching room files
2. Delete any stale instance saves so the engine loads fresh templates
3. Remind the user to restart the server after cleanup

Known issue: The instance save system can silently override template edits, making it appear that file changes aren't taking effect.

## Shop Persistence (Living Economy)
Shop economic state (stock levels, NPC gold, restock timers) persists in
`_datafiles/world/dogmud/shops/{zone}/{mobid}-room{roomid}.yaml`. This
directory is completely separate from `rooms.instances/` and
`mobs.instances/` and is NOT cleaned by the instance save cleanup SOP.
Deleting a shop file resets that merchant to template defaults (500g
starting gold, base stock levels).

Dynamic pricing ranges from 0.25x (overstocked) to 5.0x (out of stock),
driven by the `ShopAbundanceThreshold` and normalized per item by restock
quantity. Config knobs: `ShopBuyRatio`, `ShopPriceFloor`, `ShopPriceCeiling`,
`ShopAbundanceThreshold`, `ShopMaterialReserve`, `ShopGoldReserveRatio`,
`BarterMaxDiscount`, `BarterMaxBonus`.

Non-combatant mobs (`non_combatant: true` in YAML) cannot be attacked,
stolen from, or targeted by harm spells.

## NPC Schedules
Townspeople NPCs can carry a `schedule_id:` field that references
a daily routine in
`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. Schedules
cover all 24 hours, swap the mob's idle command pool per segment,
steer the mob between rooms via the existing `pathto` plumbing,
and gate `TickMobCraft` via segment `activity:`. Schedule
validators panic at startup on coverage gaps, unreachable target
rooms, or unresolved `schedule_id` references — pre-push SOP
boot-test catches these. See `docs/schemas/schedule.md`.

## Sleep Mechanics
Players and NPCs can `sleep` (the verb — no slash). Sleepers gain
5× HP/SP/CP regen but the entire first round of attacks against
them auto-crits. Wake triggers: any damage, failed steal,
shout-in-room, light source entering room (via EmitsLight buff
flag), the `stand` command, or schedule segment end for scheduled
mobs. Scheduled NPCs sleep during segments with
`activity: sleeping` (see `docs/schemas/schedule.md`); a grace
cooldown (`ScheduleWakeGraceRounds`, default 50) prevents
immediate re-sleep after a wake event. Use
`actions.Sleep(actor, opts) SleepResult` for the actor-parity
entry point. State queryable via `HasBuffFlag(buffs.Sleeping)`.

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

### Damage Formula
All channels use the same unified formula:
```
raw = stat × SkillMultiplier(rank) × itemMult × ChannelScale
```
The per-channel scale absorbs any normalization:

| Channel    | ChannelScale | Math at stat=100, rank=0, itemMult=1.0 |
|------------|-------------|----------------------------------------|
| Physical   | **0.30**    | 100 × 1.0 × 1.0 × 0.30 = **30**      |
| Magical    | **1.00**    | 100 × 1.0 × 1.0 × 1.00 = **100**     |
| Conviction | **1.00**    | 100 × 1.0 × 0.5 × 1.00 = **50**      |

Then `ApplyMitigation(raw, mitigation%, cap)` and `dice.RollStat(final)` for variance.

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
- `combat.CalcRawDamage(stat, skillRank, itemMult, channel)` — compute raw damage
- `combat.ApplyMitigation(raw, pct, cap)` — apply percentage reduction
- `combat.SkillMultiplier(rank)` — sqrt curve from config
- `combat.ResourceMultiplier(current, max, penaltyMax)` — smooth resource depletion penalty
- `character.GetPhysicalMitigation()` / `GetMagicalMitigation()` / `GetConvictionMitigation()` — sum equipment

## Resource Depletion Penalties (Stage 35)
Smooth curve replaces old hard-cutoff stamina penalties. As any resource pool
drains, a multiplier reduces combat effectiveness gradually:
```
mult = 1 - maxPenalty × (1 - ratio)^curve
```
Config knobs: `ResourcePenaltyCurve` (default 2.0), per-pool `HealthPenaltyMax`,
`StaminaPenaltyMax`, `ConvictionPenaltyMax` (all default 0.28).

| Resource % | Multiplier | Penalty |
|-----------|------------|---------|
| 100%      | 1.000      | 0%      |
| 50%       | 0.930      | 7.0%    |
| 25%       | 0.843      | 15.7%   |
| 5%        | 0.747      | ~25%    |
| 0%        | 0.720      | 28%     |

Mapping: Stamina → attack count + hit rate, Health → melee damage,
Conviction → taunt hit/damage + spell damage.

## Defense Resolution: Best-of-All (Stage 35)
Defense is resolved by rolling **all** available defenses (dodge, parry, block)
and picking the one that won by the widest margin. This replaces the old
sequential short-circuit approach where dodge was always checked first.
Benefits: every defense type gets fair representation in combat text, and
having multiple defense types is always better (wider net).

**Defense Floor**: `MinDefenseChance` (default 0.15) ensures even massively
outclassed defenders have a 15% chance to avoid any swing. This prevents
fights from feeling like guaranteed hits when stat gaps are large.

## Combat Design Conventions
- **Prefer multipliers over flat bonuses/penalties.** Multipliers scale with
  character power and are easier to tune. Flat values create balance problems
  at different power levels (too strong at low stats, irrelevant at high stats).
- Prone effects use multipliers: `ProneAttackMultiplier` (default 0.80),
  `ProneVulnerabilityMultiplier` (default 1.15), `ProneDodge/Parry/BlockPenalty`.

## Regen System (Stage 29.5)
All HP/SP/CP regeneration is **percentage-of-max** — never flat values.
- Six config knobs in `Balance`: `PlayerHealthRegenPct`, `PlayerStaminaRegenPct`, `PlayerConvictionRegenPct`, `MobHealthRegenPct`, `MobStaminaRegenPct`, `MobConvictionRegenPct` (default 0.01 = 1% per tick)
- `HealthPerRound()` / `StaminaPerRound()` / `ConvictionPerRound()` compute `floor(poolMax * pct)`, min 1
- **Mutations** use multiplier effects (`health_regen_multiplier`, `health_regen_if_lit_multiplier`, `stamina_regen_multiplier`) — never flat `health_regen` effects
- **Heal spells** store a regen multiplier in `effect_magnitude` (e.g. 3 = 3x base regen); applied via `ConditionRegen`
- **Heal buffs** that heal should compute `floor(poolMax * fraction)` — never flat dice for healing
- NPCs regen health (out of combat), stamina (1/4 in combat), and conviction every tick

## ID Inventory & Collision Prevention
**Always run `python tools/id_inventory.py` before creating a new YAML.**
The script walks the world data tree and reports per-zone ID ranges,
gaps, and the next free ID per type (rooms / mobs / items / behaviors /
buffs / quests / dialogue). Filename-only parser, no YAML library
needed.

Common invocations:
- `python tools/id_inventory.py --zone stillwater` — focus one zone
- `python tools/id_inventory.py --type rooms` — focus one type
- `python tools/id_inventory.py --alloc rooms 20` — reserve a 20-ID
  block past the global max, for parallel subagent dispatch

**Parallel content-creation strategy.** When dispatching multiple
content-creation subagents in parallel (`/new-room`, `/new-mob`,
`/new-item`, etc.), they will otherwise scan the filesystem at the
same time, see the same "next free ID," and collide. Two options:

1. **Sequential dispatch (default).** Run content-creation subagents
   one at a time. Slower wall-clock but zero collision risk. Use this
   unless wall-time genuinely matters.

2. **Pre-allocated ID blocks.** When parallelism is worth the
   complexity:
   - For each parallel agent, run `id_inventory.py --alloc <type>
     <count>` to reserve a contiguous block.
   - Embed the assigned range in that agent's dispatch prompt
     verbatim ("use rooms IDs in 5101-5120").
   - Each agent picks IDs only from its assigned block. The blocks
     don't overlap by construction.
   - After merge, run the script once more as a detection pass.

Code-only subagents (no YAML creation) can always run in parallel —
this only matters for content tasks.

## Data File Naming Convention
Before creating any new data file, verify the expected filename from the loader's `Filepath()` method:
- **Zone folder names must use underscores, not hyphens.** The engine derives the expected path by calling `ConvertForFilename()` on the zone's display name (e.g., `"Sanctum Basin"` → folder `sanctum_basin/`). A mismatch causes a startup panic: `filesystem path "..." did not end in Filepath() "..."`. This applies to both `rooms/` and `mobs/` subdirectories.
- Buffs: `{buffid}-{ConvertForFilename(name)}.yaml` — e.g., `name: Stunned` → `2-stunned.yaml`
- `ConvertForFilename()`: lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore
- Spells: use the `spellid` field value directly as the filename base (no conversion needed)
- Items/mobs follow the same `ConvertForFilename` pattern
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

## Quest Re-Grant Prevention SOP
Every dialogue node or pattern with `grantsQuest` must include the quest's
**end token** (e.g., `{questid}-end`) in `questExcluded`, not just the token
being granted. Without this, a player who completed the quest can get it
re-offered. Example: `grantsQuest: "10-start"` requires
`questExcluded: ["10-start", "10-end"]`. The dialogue loader logs a warning
at runtime if this exclusion is missing.

## Quest NPC Dialogue SOP
Every quest-granting dialogue node (any tree node with `grantsQuest`) MUST include
`"quest"` and `"task"` in its `triggers` list. Similarly, quest-introducing
`patterns` entries must include `"quest"` and `"task"` in `keywords`. This ensures
`ask <npcname> quest` always works for discovering available quests.

## Dialogue Voice & Trigger Discoverability
- NPC `text` fields are spoken by the NPC — always first person ("I", "my", "me").
- `hints` are narrator text for the player — describe options from the player's
  perspective. **NEVER** write 3rd-person self-references like "Ask about why she
  left" when "she" is the speaking NPC. Write "You could ask why she left" or
  "You could ask about the marriage."
- Every trigger word MUST be discoverable — it must appear in a hint, NPC text,
  room description, or quest log. Undiscoverable triggers are broken triggers.
- Prefer `questRequired` over `requires` for quest-gated nodes. `requires` depends
  on per-player memory that can expire and brick quests.
- `expiryPeriod` should almost never be set. The ONLY valid use is quests
  where urgency is the design intent (e.g., timed delivery before an attack).
  For all other NPCs, leave it empty or omit entirely.

## Quest Item Delivery — give.go Gotcha
**CRITICAL:** `give.go` transfers the item from the player to the mob BEFORE
any handler fires. The handler cannot prevent or undo the transfer.
Consequences:
- Quest item delivery is handled by the quest engine's `item_give` triggers
  (in quest YAML) and/or behavior tree `player_give` handlers on the mob
- NPCs that should NOT keep the item (e.g., the quest giver who handed it
  out) need a behavior tree `player_give` handler that uses the `return_item`
  action to give the item back
- Quest givers who hand out physical items via `givesItem` must also have a
  recovery dialogue node that gives a replacement if the player lost the item

## Dialogue Engine: givesItem
Tree nodes and patterns support `givesItem: <itemId>`. When a node fires with
`givesItem` set, the player receives the item and sees "You receive a <itemname>."
Use this for NPCs handing quest items to the player during dialogue.

## Quest Flags System
Quest flags store arbitrary metadata about quest choices. Primary use case:
tracking which branch a player took in an opposed/branching quest.

### Flag Declaration (Quest YAML)
Quests declare expected flags with allowed values. **Undeclared flag
references cause a server panic at startup** — this catches typos before
they reach production.

```yaml
flags:
  - key: branch
    values: [sylara, rhett]
    description: "Which NPC the player sided with"
```

Flag key convention: `"{questId}-{flagName}"` (e.g., `"11-branch"`).

### Dialogue Integration
- `setsQuestFlag: {key: "11-branch", value: "rhett"}` — set a flag on
  node match
- `questFlagRequired: {"11-branch": "rhett"}` — gate on flag value
- `questFlagExcluded: {"11-branch": "sylara"}` — hide if flag matches

### Quest Engine Integration
- Conditions: `has_flag: {"11-branch": "rhett"}`, `missing_flag: ...`
- Action: `set_flag: {key: "11-branch", value: "rhett"}`

### Admin/Scripting
- `questtoken flags` — show all flags on your character
- `questtoken flag <key> [value]` — view or set a flag
- JS scripting: `user.GetQuestFlag(key)`, `user.SetQuestFlag(key, value)`,
  `user.HasQuestFlag(key)`

### Branching Quest SOP
Every branching quest MUST have:
1. Flag declaration in quest YAML with all valid values
2. `setsQuestFlag` on each branch NPC's quest-start dialogue node
3. `questFlagRequired` on followup quest offers to gate by branch
4. **Dismissal nodes** at the TOP of each NPC's tree nodes list for
   wrong-path players — without these, keyword patterns fire and
   players think there's a hidden quest
5. Root variants with `questFlagRequired` for path-specific greetings
6. Mid-quest root variants for cross-NPC visits during the OTHER quest

## Equipment Slots
Default slots: Weapon, Offhand, Head, Neck, Shoulders, Body, Back, Belt,
Wrist (x2), Gloves, Ring (x2), Legs, Feet, Component Bag.

Mutation-gated slots (Extra Arms mutation, levels 1-4):
- Each level unlocks one ExtraArm + one ExtraWrist slot
- Level 1: Arm 3 + Wrist 3. Level 2: Arm 4 + Wrist 4.
  Level 3: Arm 5 + Wrist 5. Level 4: Arm 6 + Wrist 6.
- Escalating penalties: charisma -28/-42/-56/-70, aggro 1.0/1.5/2.0/2.5x
- Combat hit penalty: +20 per arm beyond offhand

Back slot: Cloaks (stats) or backpacks (weight reduction on backpack
contents). Component Bag slot: Holds crafting materials. `is_component:
true` items auto-route on pickup. `sort` command migrates existing
materials. `bag_capacity` limits items. Weight reduction on component bag
contents (typical 30%).

ItemSpec fields: `is_component` (bool), `weight_reduction` (float64 0-1),
`bag_capacity` (int). New ItemTypes: `wrist`, `back`, `shoulders`,
`componentbag`.

Tail mutation: adds Tail slot, disables Legs slot. `tail` ItemType. Trip
reskins to tailsweep with enhanced damage/knockdown when mutation active.

## Spell Duration System
All spell durations use `calcSpellDuration(baseFolds, skill, willpower)`:
`duration = baseFolds × (10 + wil/20 + skill/2)`. Effect-specific scaling:
shield = full, heal = ÷2, DoT = ÷3.

## Buff/Ward Spell System
- Shield spells scale by `effect_magnitude` (100 = 1.0x baseline).
  Conviction Ward = 75, Chrysalis Cocoon = 125.
- Shield duration: via `calcSpellDuration`. Crits +50% strength.
- Buff statmods `magical_mitigation` and `conviction_mitigation` flow
  through `GetMagicalMitigation()` / `GetConvictionMitigation()`.
- Kick command auto-selects variant: kick (standing), stomp (prone),
  knee (grapple+control). Config: `KickDamagePercent` (0.80),
  `StompDamagePercent` (1.20), `KneeDamagePercent` (1.00).
- Hidden mob detection on room entry: Perception+Search vs Dex+Skullduggery
  opposed roll in `go.go`. Mobs can spawn hidden via `buffids: [9]`.

## Inventory & Item Disambiguation
- **Disambiguation formats:** Players can use `N.item` (diku-style) or `item#N`
  (hash-style) to target a specific item when multiples exist. `all.item` targets
  all matching items (supported by `get` and `drop`).
- **Unified FindItem:** `look` and `identify` search backpack + equipped items as
  a single pool for disambiguation. `dagger#2` can reach a wielded dagger if the
  first match is in backpack. Source is reported ("in your backpack" / "wielded").
- **Inventory stacking:** Display-only. Items with same ItemId + EnchantType +
  EnchantTier + Uses are grouped with `(xN)` count. Storage is unchanged.
- **Carry capacity:** `Strength × Balance.CarryCapacityMultiplier` (default 0.65).
  Displayed as colored encumbrance tiers (light/moderate/heavy/overburdened/crushed),
  never raw numbers. `{enc}` prompt token available.
- **Encumbrance penalties:** Movement stamina 1-5x multiplier when over capacity
  (`go.go`). Combat swings reduced up to 50% when over capacity (`combat_helpers.go`).
- **Multi-buy:** `buy 5 iron ingot` purchases N copies, stops early on insufficient
  funds or carry capacity.
- **Enchanting targeting:** `craft <recipe> <item-name>` targets a specific item.
  Searches both backpack and equipped items. Shows numbered list when ambiguous.

## Content Generation Commands
Use slash commands to generate new data files. Claude automatically loads world.md,
the relevant schema, and existing examples before generating.

- `/new-mob "description"` — generate a mob YAML (+ optional JS stub)
- `/new-room "description"` — generate a room YAML
- `/new-item "description"` — generate an item YAML
- `/zone-sketch "concept"` — plan a new zone (room list + adjacency) before generating rooms
- `/sketch-quest "concept"` — plan a new quest (step chain, gating, files needed) for review
- `/new-quest <plan-file>` — generate all files from an approved `/sketch-quest` plan

Schema reference: `docs/schemas/` (room, mob, item, spell, buff, dialogue)
Full workflow: `docs/CONTENT_GENERATION_GUIDE.md`

After generating any file: restart server. If editing an existing zone, check
`_datafiles/world/dogmud/rooms.instances/` for stale instance saves.

## AI Testing
Run autonomous AI testers against the MUD server:
- `/test-mud local feature-tester phase2-summons.yaml` — test specific features locally
- `/test-mud prod bug-finder` — exploratory bug hunting on production
- `/test-mud local feel-tester` — natural play session for UX feedback

Config: `tools/testing/targets.yaml` (server credentials)
Roles: `tools/testing/roles/` (bug-finder, feature-tester, feel-tester)
Goals: `tools/testing/goals/` (session-specific test objectives)
Reports: `tools/testing/reports/` (output, gitignored)

Prerequisites: test character must exist, be AI-flagged, and have
tutorial completed. Edit player YAML directly for setup.

## Mob Stat Archetypes
Mobs have an optional `archetype` field that controls stat pool distribution:
- `"fighting"` — 80% physical (Str/Dex/Vit), 20% mental (Per/Wil/Cha)
- `"casting"` — 20% physical (Str/Dex/Vit), 80% mental (Per/Wil/Cha)
- `""` (default) — uniform random across all 6 stats

Set in mob YAML: `archetype: fighting` or `archetype: casting`.

## Caster Weapon Types
Three weapon subtypes designed for spellcasters: `wand`, `sceptre`, `staff`.
Each has a `spell_damage_multiplier` field on ItemSpec that multiplies spell
damage when the weapon is equipped. This is independent of `damage_multiplier`
(melee). Caster weapons use `weapon-combat` skill for melee (same as swords).

| Subtype  | Hands | Melee Mult | Spell Mult | Speed | Parry | Notes |
|----------|-------|-----------|------------|-------|-------|-------|
| wand     | 1     | 0.40      | 1.30       | 1.2   | 2     | Light, fast |
| sceptre  | 1     | 0.55      | 1.25       | 0.9   | 4     | Moderate |
| staff    | 2     | 0.80      | 1.60       | 0.7   | 12    | Defensive, high spell boost |

`spell_damage_multiplier` is applied in `calcSpellDamage()` and
`calcMobSpellDamage()` in `internal/hooks/spell_resolution.go`.

## Alchemy & Potions System
Potions use a witcher-style design with aging, toxicity, and craft-skill scaling.

### Potion Aging
- Five phases: Fresh (1.0x) → Fermented (1.15x) → Peak (1.30x) → Declining (1.30→0.5x) → Spoiled (harmful)
- Thresholds defined per-potion in `aging:` YAML field (ferment/peak/decay/spoil rounds)
- Aging speed = `bottleMultiplier × (1.0 - craftSkill/200)` — higher = faster aging
- `items.GetAgingPhase()` and `items.CalcEffectiveAgingSpeed()` in `internal/items/aging.go`

### Bottle Tiers
| Bottle | ItemID | Aging Multiplier | component_tag |
|--------|--------|-----------------|---------------|
| Clay Flask | 40043 | 3.0x (fastest) | bottle |
| Glass Vial | 40006 | 1.0x (baseline) | bottle |
| Sealed Phial | 40044 | 0.5x | bottle |
| Crystalline Decanter | 40045 | 0.25x (slowest) | bottle |

All share `component_tag: bottle`. Crafting consumes the first match. The bottle's `BottleAgingMultiplier` is stamped on the output item's `BottleMultiplier` field.

### Toxicity
- Each potion has a `toxicity` field (int) on ItemSpec
- `Character.Toxicity` accumulates; decays by `ToxicityDecayPerTick` per regen tick
- `GetToxicityMax() = ToxicityBaseMax + Vitality/ToxicityVitalityScale`
- Threshold penalties via `GetToxicityPenalties()`: regen/Per/Dex penalties at 50/75/90%
- Spoiled potions apply 3x toxicity + nausea debuff (buff 75)

### Craft Skill Scaling
- Duration: `baseDuration × (1.0 + craftSkill/100) × agingPotencyMultiplier`
- Aging speed reduction: skill 30 = 15% slower aging
- Applied in `drink.go` via `AddBuffScaled()`

### Potion Bandolier
- Belt-slot item with `is_bandolier: true` and `bandolier_capacity` field
- Auto-routes potions in `StoreItem()`, consumed first by `drink` (oldest first)
- Removal spills to backpack. Weight reduction applies to contents.
- `Character.PotionItems` slice, displayed in inventory "Potions:" section

### Buff IDs
- 54-60: Pool regen potions (healing salve through elixir of renewal)
- 61-70: Combat/utility potions (ironhide through purging draught)
- 71-74: Progression potions (essence of growth through chrysalis catalyst)
- 75: Spoiled potion nausea debuff
- 76: Purging draught weakness debuff

### Item IDs
- 30036-30056: New potion items
- 40043-40049: New alchemy materials (bottles + forage/drop ingredients)

## Salvage System
Players can break down crafted items (or items with `salvage_returns` on
their ItemSpec) to recover materials. New standalone skill: `salvage`,
primary stat: Perception, progression multiplier 2.0.

### How It Works
- `salvage <item>` starts a multi-round activity (1-5 rounds based on
  ingredient gold value).
- Each ingredient is rolled independently. Chance scales with skill:
  `chance = min + (max - min) * sqrt(skill / softCap)`.
- Config: `SalvageMinChance` (0.15), `SalvageMaxChance` (0.85),
  `SalvageSoftCap` (50).
- Item is always consumed, even if no materials recovered.

### Stations
- Salvage works anywhere; no tool required as of 2026-05-01.
- Skill rank gates yield rate (Perception-based, see formula above).

### ItemSpec Fields
- `salvage_returns`: list of `{item_tag, quantity}` for non-crafted items.
  Every `item_tag` must match a valid `component_tag` on an existing item.
