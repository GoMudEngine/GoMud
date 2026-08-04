# DOGMud - Claude Code Project Memory

## Content Playtest-Review Gate (SOP)
Any plan or task that authors **player-facing content** (rooms, mobs, items,
quests, dialogue, tutorials, onboarding, room prose) MUST end with an **in-game
adversarial playtest-harness review before the work is handed to the user to
playtest**. This is a required final task on every content plan, not an
optional extra.

Boot-clean and "YAML parses" verify the *system*, never the *experience*.
Content defects — instructions buried in room `description:` prose, confusing or
double-rendered prompts, broken/mis-ordered lesson gates, dead-ends, awkward
pacing, wrong NPC voice — are invisible to a boot test and to code reasoning.
They only surface when something plays the content as a confused human would.

Procedure: run the playtest harness (`/playtest local bug-finder`, or a
route/feature-specific goals file) with an explicitly **critical, adversarial**
mandate — spawn a fresh character, drive the real player flow end to end, read
every line of in-game output, and report every usability problem bluntly. Fix
what it finds, re-run if needed, and only then turn it over to the user. Do NOT
claim content work "done" on the strength of a clean boot alone.

## Subagent Model Preference
Pick the model that fits the task — don't reflexively pin everything to haiku.

- **haiku** — trivial mechanical work: a single-file grep/glob, a one-shot
  symbol lookup, a fixed-recipe edit. Cheap and fine when there's no judgment
  involved.
- **sonnet / opus** — exploration or implementation that benefits from
  reasoning: tracing how a subsystem fits together, multi-file searches where
  the answer requires synthesis, refactoring, architectural decisions,
  multi-step code writing, or executing a plan task with real logic.

We added the **codegraph MCP** specifically to cut token use on code
intelligence (sub-millisecond symbol/caller/callee queries instead of grep +
many Reads). That headroom means using a stronger exploration agent is usually
the right call when the task warrants deeper reasoning — instruct those agents
to prefer codegraph tools for symbol verification so the stronger model spends
its budget on thinking, not file-spelunking. When in doubt for a non-trivial
task, default up (sonnet), not down.

## Git Workflow
Follow the branch strategy in `docs/guides/github_guide.md`:
- `master` is the main integration branch + production. `origin` = pruuk/DOGMud
- NEVER push to upstream (GoMudEngine/GoMud); cherry-pick from upstream only
- `development` is legacy from when the project still pulled from upstream — no longer used as the integration branch
- Feature branches: `feature/stage-X.Y-description`
- Merge features into master with `--no-ff`
- Use conventional commit messages (feat:, fix:, refactor:, docs:, chore:)

## Pre-Push SOP
Before pushing to prod (origin/master):
1. Update `docs/PATCH_NOTES.md` with dated entries describing the work.
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

## Instance Saves & Smoke-Test SOP (Important!)

The engine loads YAML templates first, then overwrites with instance
data from `_datafiles/world/dogmud/mobs.instances/` and
`_datafiles/world/dogmud/rooms.instances/` if present. **Stale instance
saves silently shadow template edits** — including new
`schedule_id:`, `patrol_id:`, `maxwander:`, idle commands, exits,
etc. This has been a recurring source of "my change isn't taking
effect" frustration.

**EXCEPTION — fields tagged `instance:"skip"` are NOT shadowed.**
`SaveRoomInstance` skips them when writing, and
`restoreSkipTaggedFields` (`internal/rooms/save_and_load.go`) copies
them back from the template after the instance overlay is applied,
so a stale save cannot override them. **Room spawn lists
(`Room.SpawnInfo`) are in this category** and were wrongly listed
above until 2026-07-25 — a spawn-list edit takes effect on the next
room load with no wipe needed. Check the struct tag before assuming
a field is shadowed.

**SOP: nuke instance saves before every local smoke test.** Mirror
the prod policy where these directories are not deployed. Run:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* \
       _datafiles/world/dogmud/rooms.instances/*
```

Then restart the server. The engine will re-spawn mobs and re-build
rooms from the (updated) templates. **Do NOT also wipe
`_datafiles/world/dogmud/shops/` or `_datafiles/world/dogmud/guilds/`** —
those are persistent living state (shop economy; player guilds), not
instance overrides (see Shop Persistence below). Guild files are
runtime-generated per-guild YAML (`guilds/<tag>.yaml`); a malformed one
logs+skips at boot rather than panicking (unlike authored content).

When making content changes you intend to smoke-test, run the wipe
as part of your pre-smoke ritual before the user is involved. When
the user reports "my change isn't taking effect," instance saves
should be your first guess.

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

## Moderation Persistence
Player-moderation state lives in `_datafiles/world/dogmud/moderation/` — `petitions.yaml`
(the `petition` queue: player→staff reports, open/resolved) and `bans.yaml`
(permanent account + IP bans). Like `shops/` and `guilds/`, this is persistent
living state: it is `.gitignore`d, kept on the prod droplet, and must **NOT** be
wiped by the instance-save smoke-test SOP. A malformed file logs + skips at boot
(does not panic), mirroring the guilds loader. Commands: `petition` (player),
`petitions`/`boot`/`ban`/`unban` (admin), and globally-targetable `mute`/`deafen`.
Account/IP ban rejection happens in `FinalizeLoginOrCreate`
(`internal/inputhandlers/login.go`). Config: `PetitionCooldownRounds`,
`PetitionMaxLen` (GamePlay block).

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

## NPC Patrols
Patrol routes (multi-room loops) are authored at
`_datafiles/world/dogmud/patrols/<zone>/<id>.yaml`. A mob can
reference one directly via `patrol_id:` (always-on patrol), or
a schedule segment can opt in via `activity: patrol` +
`patrol_id:` (patrol runs during the segment only). Two loop
shapes: `strict` (loop back to start) and `yo-yo` (flip
direction at endpoints). Per-waypoint `dwell_rounds`. Combat
interrupts patrols; the executor resumes to the same target
waypoint on the next idle tick. Path retries use the chunk 3.2
`ScheduleMaxPathRetries` knob, falling back to `pathto home`
after the threshold. See `docs/schemas/patrol.md`.

Inter-zone patrols and caravan unification onto the patrol
layer are deferred to chunk 3.7.

## NPC↔NPC Conversations
Townspeople with relationship edges (chunk 1.6) occasionally
exchange 2-4 line conversations drawn from a relationship-type-
keyed library at `_datafiles/world/dogmud/conversations/`.
Type pools (`types/<relationship-type>.yaml`) hold generic
exchanges per relationship type. Optional pair overrides
(`pairs/<lower>_<higher>.yaml`) add per-pair-specific
exchanges (extending the type pool). Optional subtype sub-pools
add flavor variation per relationship subtype string.

Triggers: low per-tick chance (`ConversationBaseChancePct`,
default 1%) per fully-idle NPC + a higher player-arrival boost
(`ConversationPlayerArrivalBoostPct`, default 25%). Pacing: one
line per round, shared `conversation_line_idx` MiscData counter
drives speaker alternation deterministically. Cooldown
(`ConversationCooldownRounds`, default 50) on both NPCs after
an exchange completes.

Gating: conversations only fire when both NPCs are fully idle
(no combat, no sleep, no patrol mid-walk, no existing
conversation, no cooldown). Mid-exchange interruption (partner
leaves the room / sleeps / enters combat) aborts gracefully
without applying a cooldown.

Script semantics: speaker "A" is the initiator-role, "B" is
the partner-role; the engine randomizes which physical NPC
plays "A" per conversation, so author role-agnostic scripts
(don't bake mob names into pair overrides). MobConversant
interface decouples this package from internal/mobs/ to keep
the import graph acyclic; `internal/conversationadapter` is
the bridge.

NPC↔NPC opinion store and "spoken about you" gossip are
deferred (see chunk 3.6 spec for rationale).

## Map Consistency & the `non_cartesian` / `oneway` Flags
The web mapper places rooms by crawling exit deltas (`internal/mapper`),
so the world must stay Cartesian-consistent. A startup pass
(`ValidateZoneConsistency`, gated by `GamePlay.MapConsistencyEnforce`:
`off|warn|panic`, default `warn`) and the `cartcheck [zone]` admin command
report coordinate collisions, non-reciprocal exits, wrap exits in
non-wrap zones, and long connectors crossing rooms.

Escape hatches: `oneway: true` on an exit (intentional one-way; skips the
reciprocity check, still collision-checked) and `non_cartesian: true` in a
zone's `zone-config.yaml` (intentionally toroidal/maze; skips the hard
checks and renders wrap exits as edge stubs). Portal/named (non-compass)
exits are automatically non-spatial and exempt. Flip the knob to `panic`
only after `cartcheck` is clean world-wide.

The web client renders an SVG map driven by the `Zone.Map` GMCP snapshot
(`modules/gmcp/gmcp.Zone.go`, sent on every move). Each room node is a
small biome-tinted circle with a faint glyph overlay ("hybrid" style);
connections are thin amber lines. Exit `kind` routes rendering: `normal`
and `long` exits draw a connector line (long exits render proportionally
longer), `wrap` exits draw a teal edge-stub with an outward chevron (used
for toroidal/maze zones declared `non_cartesian`), and `vertical` exits
draw a faint ▲ or ▼ tick on the room node. Fog of war is enforced by
`Character.VisitedRooms` (`map[string][]int`, zone→roomIds): `MarkRoomVisited`
is called on every successful move in `go.go`, and `(*mapper).Snapshot(visited)`
filters out unvisited rooms and their exits before sending the payload.
The client renderer (`RoomGridSVG` in `gmcp.js`) exposes `fit`, `centerOnRoom`,
`zoomIn`, and `zoomOut` controls.

The web mapper was subsequently restyled to an **antique tooled-leather**
aesthetic. A fixed leather-textured SVG surface (the "frame") holds a
nested pannable `worldSvg` containing the room grid. Connections are
styled per-exit-type, inferred from flags now carried on `SnapshotExit`:
biome roads/trails/water (color derived from room biome), locked, secret,
one-way, gate (from `exit.ExitMessage != ""`), stairs (▲/▼ ticks for
`vertical` exits), cross-zone boundary stubs (`ToZone` set), and fog
stubs for unvisited exits (`Stub: true`). Party-member positions arrive
via `Zone.Map.party` (a `[]int` of room IDs holding party members) and
are rendered as small figures on those room nodes; the current player's
room is given a raised (drop-shadow) treatment. Visual source of truth:
`docs/superpowers/specs/2026-06-06-mapper-leather-mockups/`. Client
renderer: `RoomGridSVG` in
`_datafiles/html/public/static/js/gmcp.js`. Connection-type styling and
party markers are web-only — the ASCII `map` command is unaffected.

## Project Context
- DOGMud (Delusions of Grandeur) is a MUD built on the GoMud engine
- World design document: `docs/world.md`
- Development roadmap: `docs/roadmaps/DEVELOPMENT_PLAN.md`
- Remote origin: https://github.com/pruuk/DOGMud
- Remote upstream: https://github.com/GoMudEngine/GoMud

## Stat & Progression System
- All stats (Strength, Dexterity, Perception, Vitality, Willpower, Charisma) are centered at **100 = human baseline**
- Stats improve via **use-based progression only** — `OnStatUse()` triggers probabilistic advancement. There is NO level-based or XP-based stat gain; levels and XP are being removed from the game entirely.
- **There is no soft cap on stat values.** `ValueAdj == Value` always; stats are
  used raw. Compression was removed 2026-08-02 — it was inherited from upstream,
  hid ~10 points from three veteran characters, and (because `HealthMax`,
  `StaminaMax`, `ConvictionMax` and `ActionPointsMax` are also `stats.StatInfo`
  and call the same `Recalculate()`) was silently shrinking every resource pool
  by roughly 40%. Do not reintroduce compression in `StatInfo.Recalculate()` —
  anything added there hits the pools too.
- `StatProgressionSoftCap` (default 150) is the *virtual rank* where progression
  slows sharply, plus the anti-exploit floor in `CheckStatProgression`. It is not
  a ceiling on stat values. This is the real brake on runaway stats: no
  production character has organically exceeded 195 under it.
- Resource pools: `HealthMax = 5 + Vit×3 + Str×1`, `StaminaMax = 5 + Vit×3 +
  Wil×1`, `ConvictionMax = 5 + Cha×3 + Wil×1`. One primary stat (×3) and one
  secondary (×1) each. Coefficients live in the balance config; note that a knob
  left *absent* from `config.yaml` falls back to its Go default, and `0` is a
  legal shipped value (`StaminaPerStrength: 0`).
- Skills (10 total) cap softly at 50 (`skillSoftCap`). They progress via
  `OnSkillUse()` → `CheckSkillProgression()`, probabilistically.
- **A progression roll happens on EVERY use, not every 25 uses.** Corrected
  2026-08-04; the old "every ~25 uses" wording here was wrong.
  `UsesPerRank` (25) is not a check cadence. It is the divisor that converts the
  use counter into a **virtual rank** (`progression.go:92`, `:163`), and that
  rank is what decays the odds. So it is "a roll every use, whose odds step down
  every 25 uses", not "a roll every 25 uses".
- Curve (`CalculateProgressionChance`, `internal/characters/progression.go:44-62`):
  below the soft cap `base × exp(-decayBelow × rank/softCap)`, above it the
  decay continues with `decayAbove` rather than reaching zero. Stat rolls also
  multiply by `StatProgressionRate`. With shipped config a fresh stat is roughly
  27% per use, falling to roughly 1.3% at virtual rank 150.
- Shipped config again differs from the Go defaults: `BaseProgressionChance`
  0.12 shipped against a 0.30 default, `StatProgressionRate` 2.25 against 1.0.
  Read `config.yaml`, not the defaults.
- `IncreaseStat` and `IncreaseSkill` contain **no bound check whatsoever** (there
  is a `TestIncreaseSkill_NoCap` regression test). The only hard ceilings in
  `CheckStatProgression` are `MobStatCap` / `MobSkillCap`, both gated on
  `c.IsMob`. Players have none.

## Dice & Rolling System
- **For all stat-based rolls use `dice.RollStat(mean)` or `dice.OpposedRollStat(atk, def)`** — no stdDev argument needed
- These wrappers automatically apply the global `RollSpread` factor: `stdDev = mean × RollSpread`
- `dice.Roll(mean, stdDev)` / `dice.OpposedRoll(atk, def, stdDev)` are low-level; only use them when variance is NOT stat-proportional (e.g., weapon damage variance from item specs)
- **`RollSpread`** is the single master randomness knob — set in `_datafiles/config.yaml` under `GamePlay.RollSpread` (default **0.15**). Changing it rescales every dice roll in the engine. See `internal/dice/README.md` for win-probability tables.
- Z-score thresholds: `ZScore >= 2.0` = crit; `ZScore <= -2.0` = fumble/backfire (~2.3% each, unaffected by `RollSpread`)
- `util.Rand` / `util.LogRoll` are NOT used for hit or attack checks; only `dice.*` functions

## Balance Lives in config.yaml, Not in Code

**Before hardcoding any balance number, check whether a knob already exists.**
There are **352 balance knobs** across 14 `internal/configs/config.balance*.go`
files (466 config fields in total), surfaced through a 1506 line
`_datafiles/config.yaml`. Damage scales, mitigation caps, regen percentages,
progression rates, resource penalty curves, shop pricing, toxicity, salvage
odds, conversation and schedule pacing are all tunable without a rebuild.

Three rules follow from this:

1. **Retuning is a config edit, not a code change.** If you find yourself
   editing a literal in `internal/` to change how something feels, stop and
   look for the knob. If there genuinely is not one, adding a knob is usually
   the better change than editing the literal.
2. **Never quote a Go default as a live value.** Defaults are fallbacks applied
   only when the key is absent from `config.yaml`. Several shipped values differ
   sharply from their defaults (`SpellDamageScale` ships at 3.12 against a
   default of 1.0). Read `config.yaml` for what the game actually does.
3. **Absence is meaningful.** A knob left out of `config.yaml` falls back to its
   Go default, and `0` is a legal shipped value (`StaminaPerStrength: 0`). Do
   not assume a missing key means "unset" or "zero".

This is also worth surfacing to readers of the public diagrams page: "the
combat model is data, not code" is a genuinely interesting property to an
engineering audience.

## Unified Damage & Mitigation Pipeline (Stage 34)
All damage flows through a three-channel pipeline in `internal/combat/damage_pipeline.go`:

### Damage Formula
All channels use the same unified formula, which has **five** factors, not four:
```
raw = stat × SkillMultiplier(rank) × itemMult × ChannelScale × GlobalDamageMultiplier
```
`GlobalDamageMultiplier` is a master knob applied to every channel
(`damage_pipeline.go:78`). It was missing from this table until 2026-08-04, so
any figure computed from the old four-factor version was wrong by whatever the
knob was set to.

**ChannelScale is a config value, not a constant.** `DamageScale()` reads it per
call from the balance config, so the scales below change whenever `config.yaml`
changes. Two sets of numbers matter and they are not the same:

| Channel    | Go default | Shipped in `config.yaml` (2026-08-04) | Knob |
|------------|-----------|----------------------------------------|------|
| Physical   | 0.30      | **0.52**                               | `MeleeDamageScale` |
| Magical    | 1.00      | **3.12**                               | `SpellDamageScale` |
| Conviction | 1.00      | **3.00**                               | `RhetoricDamageScale` |

`GlobalDamageMultiplier`: Go default 1.0, **shipped 0.5**.

Real math at stat=100, rank=0, itemMult=1.0, using the *shipped* values:

| Channel    | Calculation                          | Raw    |
|------------|--------------------------------------|--------|
| Physical   | 100 × 1.0 × 1.0 × 0.52 × 0.5         | **26** |
| Magical    | 100 × 1.0 × 1.0 × 3.12 × 0.5         | **156**|
| Conviction | 100 × 1.0 × 0.5 × 3.00 × 0.5         | **75** |

Do not quote the Go defaults as if they were live values. Read `config.yaml`.
Note also that a knob left *absent* from `config.yaml` falls back to its Go
default, so absence is meaningful.

Then `ApplyMitigation(raw, mitigation%, cap)` and `dice.RollStat(final)` for variance.

### Three Channels
| Channel | Stat | Skills | Item Field | Mitigation Method |
|---------|------|--------|-----------|------------------|
| Physical | Strength | weapon/unarmed/ranged-combat | `damage_multiplier` (weapon) | `GetPhysicalMitigation()` |
| Magical | Willpower | spellcasting | `damage_multiplier` (spell) | `GetMagicalMitigation()` |
| Conviction | Charisma | rhetoric | 0.5 (taunt base) | `GetConvictionMitigation()` |

> Note: the `shoot` command path (`ExecuteFire`) uses **Perception** for
> both hit and damage rolls — aimed shots are deliberate-move actions, not
> auto-attack swings. The Strength entry above applies to melee auto-attacks
> and mob basic attacks only.

### Skill Multiplier Curve
`mult = base + (max - base) × sqrt(rank / softCap)` — Config: `SkillMultiplierBase` (1.0), `SkillMultiplierMax` (3.0)

### Item Mitigation Fields (replaces old single DamageReduction)
Items use `physical_mitigation`, `magical_mitigation`, `conviction_mitigation` (integer percentages).
The legacy `DamageReduction` field was fully removed 2026-08-03 (its three
side-jobs migrated: shield classification keys on `physical_mitigation > 0 ||
subtype wearable`, the item-value formula prices the three mitigation fields,
and the dead upstream `Item.Enchant` that wrote it was deleted).

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

## Codegraph MCP — Code Intelligence

The `codegraph` MCP server indexes every Go symbol in the repo into a
local SQLite knowledge graph (~4.6k files, ~18k nodes, ~60k edges).
Sub-millisecond queries return signatures, sources, callers, callees,
and trails. Use it BEFORE writing code, not during.

**Use it for:**
- **Pre-dispatch verification.** Before sending a subagent off, run 2–3
  `codegraph_node` / `codegraph_search` calls to confirm the struct
  shapes, function signatures, and field names the plan references.
  Cheaper than letting a subagent waste turns rediscovering or, worse,
  shipping code against a stale plan. (Caught a real `Engine` field
  rename during 4.2 — plan said `mobTrees`/`noMobTree`, actual is
  `trees`/`noTree`.)
- **Symbol-trail navigation.** `codegraph_node Foo` with `includeCode:true`
  returns the source + callers/callees with file:line. Replaces
  grep + 5–10 Reads.
- **Disambiguation.** Code-base has many `Add` / `Remove` / `Clear` /
  `ClearCache` symbols across packages. `codegraph_node` lists all
  matches and shows the one you asked for, so you don't accidentally
  Read the wrong file.
- **Front-loading subagent prompts.** Paste the verified struct/signature
  into the prompt's "context I've already verified for you" block so
  the subagent skips exploration.

**Don't use it for:**
- File authoring that doesn't reference Go symbols — YAML data files,
  templates, prose docs.
- "Find me the test helper that looks similar to X" — codegraph models
  structure, not similarity. Glob + Read is right for that.
- Confirming code you JUST edited — the index lags ~1s; trust your edit
  + file state over the index for symbols you touched this turn.

**Tool selection by intent (lifted from the codegraph server docs):**
- "What's the deal with this task/feature/area?" → `codegraph_context`
  (composes search + node + callers + callees in one call).
- "What is/calls/triggers this symbol?" → `codegraph_node` (with
  `includeCode:true` for source).
- "Find a symbol by name" → `codegraph_search`.
- "Trace from X to Y" → `codegraph_trace`.

**Subagent guidance.** When dispatching a subagent that needs to touch
unfamiliar code, instruct it to prefer codegraph MCP tools over Read/Grep
for symbol verification — saves their context window and reduces
back-and-forth.

## Package `context.md` Convention

Every package under `internal/` and `modules/` carries a `context.md` — a
developer/agent-facing description of what the package is and how to use it
correctly. **Any work that creates a new package MUST ship one; any work that
reshapes an existing package's API, data model, or file list MUST update it.**
(This rule previously lived only in `docs/roadmaps/MOB_ALIVENESS_ROADMAP.md`,
which is why coverage drifted — 37 packages had none and several documented
functions that did not exist.)

**Verify before you document.** Every symbol you name must exist. Check it with
`codegraph_search` / `codegraph_node`, or extract the real surface with:

```powershell
Select-String -Path internal\<pkg>\*.go -Pattern '^(func|type|const|var)\s'
```

A `context.md` that describes an invented API is worse than no file at all — an
agent will code against it and the mistake surfaces at compile time or, worse,
at runtime.

**Structure** (adapt, don't pad):

- `## Purpose` — what it does and why it exists, 2–4 sentences. Say what it
  deliberately does *not* do.
- `## Files` — one line per file.
- Core types with real field names, in a `go` block.
- `## Public API` — verified signatures, grouped by job.
- `## Gotchas` — the things that bite. Nil-return contracts, panics,
  comparison hazards, ordering requirements, deliberate-looking-wrong code.
- `## Dependencies` and `## Consumers`.

**Do not write** "Future Enhancements," "Security Considerations,"
"Performance Characteristics," "Administrative Features," or "Scalability"
sections unless the package genuinely has something specific to say. The
upstream-generated files are full of that filler and it is being removed, not
copied.

Good exemplars (verified 2026-07-31): `internal/term/context.md` (small,
declarative), `internal/mutators/context.md` (medium, lifecycle-heavy),
`internal/mapper/context.md` (large, multi-subsystem).

## Data File Naming Convention
Before creating any new data file, verify the expected filename from the loader's `Filepath()` method:
- **Zone folder names must use underscores, not hyphens.** The engine derives the expected path by calling `ConvertForFilename()` on the zone's display name (e.g., `"Sanctum Basin"` → folder `sanctum_basin/`). A mismatch causes a startup panic: `filesystem path "..." did not end in Filepath() "..."`. This applies to both `rooms/` and `mobs/` subdirectories.
- Buffs: `{buffid}-{ConvertForFilename(name)}.yaml` — e.g., `name: Stunned` → `2-stunned.yaml`
- `ConvertForFilename()`: lowercase, keep a-z/0-9, drop apostrophes, all other chars → underscore
- Spells: use the `spellid` field value directly as the filename base (no conversion needed)
- Items/mobs follow the same `ConvertForFilename` pattern
- Mismatch between filesystem path and `Filepath()` output causes a startup panic

## Command Parsing & Multi-Word Input (`internal/parser`)

`internal/parser` is the shared target-resolution seam for **composition-heavy**
commands — the ones that must split input into roles (item vs. container, mob
vs. player). Use it instead of hand-rolling a per-command `strings.Fields`
ladder (that pattern is what hid the 2026-07-08 corpse-loot bug):

- **`SplitTrailingContainer(scope, input)`** — splits `<item> [from] <container|
  corpse|pet>`; `get.go` uses it. Room-scoped (`Scope{User, Room}`).
- **`SplitLeadingMatch(input, matches)`** — greedy longest-leading-span with a
  caller-injected validator; **scope-agnostic**, for globally-resolved slots like
  admin `<mob-template-name> <player>` (`knowledge`/`opinion` use it).
- **`Resolve` / `ResolveItem` / `ResolveActor`** — greedy multi-word resolution
  over requested `Kind`s. Gates (ownership, hidden-container discovery, exploding
  guards) stay in the **command**, not the parser — the parser only *finds*.

**Most single-token multi-word input already resolves** via the existing fuzzy
matchers (`room.FindByName`, `items.FindMatchIn`, `room.FindNoun`), which match
the whole phrase. So `attack bank clerk`, `get lake iron nodule`, and
`look hare paths` already work — do NOT add parser plumbing for cases that
already resolve. Reach for the parser only when a command must *split* input
into multiple slots.

**Authoring convention (un-hyphenated):** because the matchers handle multi-word
input, author multi-word room nouns, item names, and `component_tag`s with
**spaces**, not forced hyphens (`lake iron nodule`, not `lake-iron-nodule`). Use
a hyphen only where the term genuinely reads hyphenated to the player. Note the
mob-ident gotcha fixed in Stage 2: name lookups must `ConvertForFilename()` the
*input* too (so a space-form query matches the underscore filename form) — see
`knowledgeResolveMobIdent`/`opinionResolveMobIdent`. Full design +
divergences: `docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md`.

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
Use slash commands to generate new data files. Claude automatically loads docs/world.md,
the relevant schema, and existing examples before generating.

- `/new-mob "description"` — generate a mob YAML (+ optional JS stub)
- `/new-room "description"` — generate a room YAML
- `/new-item "description"` — generate an item YAML
- `/zone-sketch "concept"` — plan a new zone (room list + adjacency) before generating rooms
- `/sketch-quest "concept"` — plan a new quest (step chain, gating, files needed) for review
- `/new-quest <plan-file>` — generate all files from an approved `/sketch-quest` plan

Schema reference: `docs/schemas/` (room, mob, item, spell, buff, dialogue)
Full workflow: `docs/guides/CONTENT_GENERATION_GUIDE.md`

After generating any file: restart server. If editing an existing zone, check
`_datafiles/world/dogmud/rooms.instances/` for stale instance saves.

## AI Testing
Run autonomous AI testers against the MUD server via the **GoMud playtest
harness** (`mudagent` adapter + `/playtest` driver). The old `/test-mud` +
`tools/mud_bridge.py` + `tools/ai_player.py` stack was retired 2026-06-08
(archived under `tools/_archive/testing-pre-harness/`).

- `/playtest local feature-tester shop-economy.yaml` — test specific features locally
- `/playtest prod bug-finder` — exploratory bug hunting on production
- `/playtest local feel-tester` — natural play session for UX feedback

Usage: `/playtest <local|prod> <personality> [goals-file]`. The driver spawns
the harness's `mudagent` (located via `GOMUD_HARNESS_DIR`, default
`../gomud-playtest-harness`) and drives it over a JSON event protocol
(`output`/`gmcp`/`status`/`beacon`).

Overlay (DOGMud-specific): `tools/playtest/`
- `engine-profile.yaml` — DOGMud commands/world/mechanics (the one engine-specific file)
- `targets.yaml` — local/prod server creds
- `personalities/` (bug-finder, feature-tester, feel-tester)
- `goals/` (migrated session objectives), `reports/` (output, gitignored)

The vendored `playtest` server module (`modules/playtest/`) emits per-round
`Playtest.Round` GMCP **beacons** (`hp/sp/cp + max`, room) for reliable pacing +
structured verification, when enabled via `Modules.playtest.*`. Group/multi-agent
scenarios (`ptorch`) are Phase 2 (not yet wired).

Prerequisites: AI port enabled (`AIPort: 55555`); test character must be
AI-flagged (or the agent creates one via the new-player flow). Edit player YAML
directly for setup.

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
