# Crash Site (#22) — Difficulty Knob Table

Compiled 2026-07-05 for a geared 3-player party test run. Zone folder:
`_datafiles/world/dogmud/rooms/crash_site_interior/` (rooms 6373-6402) +
`_datafiles/world/dogmud/mobs/crash_site_interior/` (mobs 9554-9562).

## 0. Entry / instance mechanics (read this first — it drives everything else)

| Item | Value | File:line |
|---|---|---|
| Instance type | `instanced: true`, `non_cartesian: true`, per-party (owner + party members are the authorized user list) | `_datafiles/world/dogmud/rooms/crash_site_interior/zone-config.yaml:2-3` |
| Entry room | 6373 (The Breach) | `zone-config.yaml:6` |
| Death policy | `rejoin` | `zone-config.yaml:4` |
| Portal duration | 30 real minutes | `zone-config.yaml:5` |
| Recall | disabled inside (`allow_recall: false`) | `zone-config.yaml:7` |
| Entry gate | requires item **40168** (Attuned Disc, from Quest 76) in inventory — reusable key, not consumed | `_datafiles/world/dogmud/behaviors/eastern_highlands/9553-the_threshold_keeper.yaml:66-67` |
| **Min buy-in gold** | **`min_gold: 200`** — asked as `ask keeper crash <gold>`; below this the keeper refuses | `9553-the_threshold_keeper.yaml:70` |
| **THE difficulty knob** | **Whatever gold the party pays scales every mob's total stat pool linearly.** `scaled = goldPaid × spawninfo.statpool_multiplier`, capped by `InstanceStatPoolCap` (default 50000, uncapped if 0). This value **replaces** (not adds to) the mob's own template `statpool:` field entirely. | `internal/rooms/instances.go:262-276` (`ScaleSpawnStatPools`), `internal/rooms/rooms.go:796-805`, applied in `internal/mobs/mobs.go:470-473` (`forceStatPool` overrides `mob.StatPool`) |
| Config for the cap | `InstanceStatPoolCap: 50000` (default, no override in `_datafiles/config.yaml`) | `internal/configs/config.balance.go:511`, default set `internal/configs/config.balance.misc.go:280-281` |

**Practical read for the playtest:** paying exactly the 200g minimum makes every "×1" trash mob a 200-point statpool build (below their own template default of 300-400!), while Warden-Prime (×5 multiplier) becomes 1000 and the Core Guardian (×7 multiplier) becomes 1400. Paying more scales linearly — e.g. at 300g (the amount the NPC's own hint text suggests: `ask keeper crash 300`), trash = 300 (matches template), Warden-Prime = 1500, Core Guardian = 2100. **Decide the buy-in amount deliberately before the run — it is the single biggest lever in this fight**, bigger than any individual mob stat.

> **Correction to prior project-memory notes:** MEMORY.md's "Core Guardian ×7 / Warden-Prime ×5 (spawned)" language refers to this **per-mob statpool multiplier coefficient**, not a spawn headcount. Verified against the actual shipped room files: room 6392 (Warden-Prime's Hold) and room 6395 (Core Guardian's Vault) each have exactly **one** `spawninfo` entry — one boss each. The design plan originally spec'd the coefficients at ×4/×6 and they were bumped to ×5/×7 post-playtest (`docs/superpowers/plans/2026-07-01-crash-site-B2-interior-content.md:549,593,605`).

## 1. Mob roster — trash constructs (species 37, group `construct`)

All eight non-boss mob templates share: `behavior_archetype: generic_fighter`, `aiprofile: brute`, `archetype: fighting` (80% physical stat weighting: Str/Dex/Vit), `maxwander: 0`, `hostile: true`, `itemdropchance: 60`. Base template `statpool` is irrelevant in-instance (see §0 — it's overridden by `goldPaid × multiplier`); it only matters if the mob is ever spawned outside this instance (it isn't, per the room spawn lists below).

| Mob ID | Name | File | Template statpool (unused in-instance) | speciesid | Rooms it spawns in (× multiplier) | Notes |
|---|---|---|---|---|---|---|
| 9554 | Hull Warden | `mobs/crash_site_interior/9554-hull_warden.yaml` | 300 | 37 | 6374(×1), 6376(×1), 6380(×1,×1), 6382(×1), 6389(×1,×2) | drops item 40171 (20%), loot_pool 20091 |
| 9555 | Sentry Drift | `9555-sentry_drift.yaml` | 300 | 37 | 6379(×1), 6388(×1,×2), 6393(×1,×1) | drops item 40172 (20%), loot_pool 10047 |
| 9556 | Maintenance Frame | `9556-maintenance_frame.yaml` | 300 | 37 | 6380(×1), 6389(×1), 6393(×1) | drops item 40171 (25%), loot_pool 20092 |
| 9557 | Cold-Light Sentinel | `9557-cold_light_sentinel.yaml` | 300 | 37 | 6384(×1) | anchor room ("Medical Bay"); drops item 40172 (25%), loot_pool 20093 |
| 9558 | Grey Automaton | `9558-grey_automaton.yaml` | 300 | 37 | 6387(×1) | drops item 40171 (20%), loot_pool 20094 |
| 9559 | Silent Warden | `9559-silent_warden.yaml` | **400** | **35 (not 37 — see gap below)** | 6390(×1), 6401(×1) | denser/slower "optional risk-reward" mob; drops item 40172 (25%), loot_pool 20095 |
| 9560 | Deep Drifter | `9560-deep_drifter.yaml` | 300 | 37 | 6402(×1) | secret/optional room; drops item 40171 (20%), loot_pool 10048 |

**Gap flagged:** mob 9559 "Silent Warden" is documented in project memory/plan as a construct (species 37) but its YAML has `speciesid: 35`, not 37. Worth a quick look before the run if species drives any damage-type or resistance logic — it may be an authoring typo (the description explicitly calls it "denser and slower than the lesser guardians," i.e. intended as the same construct family).

**Trash headcount by room (each `spawninfo` list entry = one spawned mob instance, `statpool:` under it is the multiplier, not a count):**

| Room | ID | Entries (mobid×mult) | Instance count |
|---|---|---|---|
| The Breach approach | 6374 | 9554×1 | 1 |
| — | 6376 | 9554×1 | 1 |
| Warden Post | 6380 | 9554×1, 9554×1, 9556×1 | 3 |
| — | 6379 | 9555×1 | 1 |
| Warden Nest | 6389 | 9554×1, 9554×2, 9556×1 | 3 |
| Optional: Silent Vault | 6390 | 9559×1 | 1 |
| Collapsed Junction | 6388 | 9555×1, 9555×2 | 2 |
| Lower Landing | 6382 | 9554×1 | 1 |
| Fabrication Bay | 6387 | 9558×1 | 1 |
| Medical Bay | 6384 | 9557×1 | 1 |
| Command Approach | 6393 | 9555×1, 9555×1, 9556×1 | 3 |
| Optional: Drift Vault | 6401 | 9559×1 | 1 |
| Optional: The Long Dark | 6402 | 9560×1 | 1 |
| **Total trash** | | | **20** |

## 2. Bosses

Both share: `behavior_archetype: leader`, `aiprofile: brute`, `archetype: fighting`, `submission_policy: lethal`, `surrender_policy: never` (cannot be fled/talked down), `groups: [construct, guardian]`, `spawnmutations: [large]`, `itemdropchance: 100`, `speciesid: 37`. Both have `bash` in their combat command pool (knockdown proc via the standard bash mechanic, not a scripted special).

| | Warden-Prime (9561) | The Core Guardian (9562) |
|---|---|---|
| File | `mobs/crash_site_interior/9561-warden_prime.yaml` | `9562-the_core_guardian.yaml` |
| Room | 6392 "The Warden-Prime's Hold" (gates stage 7.3c) | 6395 "The Core Guardian's Vault" (final boss) |
| Template statpool | 500 (unused — see §0) | 500 (unused — see §0) |
| **In-instance statpool multiplier** | **×5** (design plan originally ×4, bumped post-playtest) | **×7** (design plan originally ×6, bumped post-playtest) |
| Effective statpool at 200g buy-in | 1000 | 1400 |
| Effective statpool at 300g buy-in | 1500 | 2100 |
| Bonus stat training | +30 Strength, +30 Vitality (`character.stats.strength/vitality.training: 30`) on top of the distributed statpool | +30 Strength, +30 Vitality |
| `spawnmutations` | `[large]` — check `internal/mutations` for the `large` mutation's stat/size effect if you want the exact multiplier it applies | `[large]` |
| Adds / phase mechanics | **None found** — no scripted add-summon, no phase/enrage trigger in the YAML or behavior tree. It is a stat-scaled single combatant with `bash` in its swing pool. | **None found** — same; single combatant, no adds, no phases. |
| Guaranteed drops | itemid 40169 (100%), 40174 (45%), 40171 (60%), 40195 (5%), 40196 (5%) | itemid 40169 (100%), 40175 (100%), **30067 Catalyst of Unmaking (100%, guaranteed mutation-scour)**, 40170 (60%), 40190-40194 (4% each, the 18-reagent pool) |
| `loot_pool` | 10047, 20092, 20093 | 10047, 10048, 10049, 20091, 20094, 20095 |

## 3. The suppression aura (Chrysalis dampening)

| Item | Value | File:line |
|---|---|---|
| Config knob | **`CrashSiteSuppressionFactor: 0.35`** (default; no override in `_datafiles/config.yaml`) — "spell power and mutation combat bonuses are scaled to this fraction; 0=fully suppressed, 1=no effect" | Declared `internal/configs/config.balance.go:514`; default clamp/set `internal/configs/config.balance.misc.go:287-289` (clamps to 0.35 if outside `(0, 1.0]`) |
| Room application | Every one of the 30 rooms (6373-6402) carries `mutators: [{mutatorid: hull_suppression}]` | verified via grep — all 30 room files match |
| Mutator definition | `hull_suppression`: applies `playerbuffids: [95]` every room-tick, plus `lightmod: 2` (fixes the cave biome's default darkness) and a description-append | `_datafiles/world/dogmud/mutators/hull_suppression.yaml` |
| Buff applied | Buff 95 "Hull Dampening": sets flag `dampened`, `statmods: {willpower: -30, charisma: -30}`, `triggerrate: 1 round`, `triggercount: 3` (re-applied every room tick so it never lapses while inside) | `_datafiles/world/dogmud/buffs/95-hull_dampening.yaml` |
| Flag consumed | `buffs.Dampened` (`= "dampened"`) | `internal/buffs/buffspec.go:88` |
| **Chokepoint 1 — spell damage** | If caster `HasBuffFlag(buffs.Dampened)`, `rawDmg *= CrashSiteSuppressionFactor` (floor 1) — applies to spellcasting damage only | `internal/hooks/combat_shared_helpers.go:39-48` |
| **Chokepoint 2 — physical mutation damage bonus** | If source `HasBuffFlag(buffs.Dampened)`, the mutation damage-multiplier *bonus* (the part above 1.0) is pulled toward 1.0 by the same factor via `mutations.DampenBonus(1.0+mutDmgMult, factor) - 1.0`; multiplier ≤1.0 (a penalty, not a bonus) passes through untouched | `internal/combat/combat_helpers.go:356-365`, `DampenBonus` at `internal/mutations/mutations.go:356-364` |
| Net effect | -30 Willpower / -30 Charisma flat on every player + spell damage scaled to 35% of normal + mutation-bonus damage scaled to 35% of its bonus portion. **Plain physical melee (Strength-based) is NOT touched by this aura at all.** A martial-heavy party is largely unaffected by the aura; a caster/mutation-heavy party will feel it hard. | — |

## 4. Hazard rooms

| Room(s) | Mutator | Buff applied | Effect | File:line |
|---|---|---|---|---|
| 6379, 6383, 6388, 6391, 6393, 6402 | `hull_discharge_deep` (in addition to `hull_suppression`) | Buff 96 "Hull Discharge" (`playerbuffids: [96]`) | `tick_pool: health`, `tick_percent: -0.09`, `tick_variance: 0.03`, `tick_min: 8` per tick, `triggercount: 3` — passive room-tick HP drain (~9% max HP per tick, min 8 flat, for 3 ticks) just from standing in these 6 corridor rooms | `_datafiles/world/dogmud/mutators/hull_discharge_deep.yaml`, buff `_datafiles/world/dogmud/buffs/96-hull_discharge.yaml` |
| 6386 → 6387 (east exit only) | **Optional arc-trap**: locked door, `lock.difficulty: 12`, `trapbuffids: [97]` on a failed pick attempt. The west route around it is safe/open — this is purely optional risk. | Buff 97 "Arc Trap" (`tick_pool: health`, `tick_percent: -0.11`, `tick_variance: 0.04`, `tick_min: 10`, `triggercount: 2`) | Slightly harder single-hit-then-decay variant of buff 96 (~11%/tick, min 10, 2 ticks) triggered only on a failed lockpick of the trapped door | `_datafiles/world/dogmud/rooms/crash_site_interior/6386.yaml:22-24`, buff `_datafiles/world/dogmud/buffs/97-arc_trap.yaml` |

No other room-based damage mutators were found in 6373-6402 (checked every room file for `mutatorid:`; only `hull_suppression` and `hull_discharge_deep` appear).

## 5. Global combat knobs that also apply here

All values below are live defaults from `_datafiles/config.yaml` (none are overridden specifically for the Crash Site — they apply to every fight in the game, listed here because they still shape this one).

| Knob | Value | File:line | Effect of raising/lowering |
|---|---|---|---|
| `RollSpread` | 0.15 | `_datafiles/config.yaml:358`; struct `internal/configs/config.balance.go:11` | Master variance knob (`stdDev = stat × RollSpread`). Higher = swingier fights (more crits/fumbles/upset outcomes), lower = more deterministic, stat-gap-dominated fights. |
| `MinDefenseChance` | 0.15 | `_datafiles/config.yaml:391` | Floor chance to dodge/parry/block even when badly outclassed. Lower = geared party's gear-gap advantage matters more (bosses land more guaranteed hits on a weak defender); higher = softens worst-case swings. |
| `PhysicalMitigationCap` / `MagicalMitigationCap` / `ConvictionMitigationCap` | 0.75 / 0.75 / 0.75 | `_datafiles/config.yaml:573-575` | Max % damage reduction from mitigation stacking per channel. Raising lets a heavily-armored party trivialize a channel; the bosses here hit primarily physical (Strength-based `generic_fighter`/`brute`), so `PhysicalMitigationCap` is the one that matters most for this fight. |
| `SkillMultiplierBase` / `SkillMultiplierMax` | 1.0 / 3.0 | `_datafiles/config.yaml:565-566` | Skill-rank damage curve ceiling. Affects both the party's weapon-combat output and the mobs' own combat-skill scaling (mobs have no explicit skill training set in these YAMLs, so they likely sit near skill 0, i.e. near the 1.0x base — the party's skill training is the variable here). |
| `ResourcePenaltyCurve` / `HealthPenaltyMax` / `StaminaPenaltyMax` / `ConvictionPenaltyMax` | 2.0 / 0.28 / 0.28 / 0.28 | `_datafiles/config.yaml:591,597-599` | Smooth resource-depletion combat penalty. Relevant on a long boss fight (Core Guardian at high statpool = high HP = long fight) where the party's own HP/Stamina will sag over many rounds and soften their output near the end. |
| `InstanceStatPoolCap` | 50000 (0 = uncapped) | `_datafiles/config.yaml` (not present — default applies), `internal/configs/config.balance.go:511` | Ceiling on `goldPaid × multiplier`. Irrelevant at realistic buy-ins (200-300g × 7 = 1400-2100, nowhere near 50000) — only matters if someone pays an absurd amount of gold at the keeper. |

## Headline numbers (quick reference)

- **Entry:** item 40168 (Attuned Disc) + **200g minimum** buy-in to the keeper (mob 9553), instanced per-party, 30 real-minute portal.
- **Total combatants across the whole dungeon if every room is cleared:** **22** (20 trash constructs + Warden-Prime + Core Guardian). Two rooms (6401, 6402) and one room (6390) are explicitly optional/secret; one locked door (6386→6387, arc-trapped) is also optional. Skipping all optional content still means fighting ~17-18 trash + both bosses.
- **Warden-Prime statpool:** 1000 at 200g buy-in / 1500 at 300g (×5 multiplier), no adds, no phases, `bash` in its swing pool.
- **Core Guardian statpool:** 1400 at 200g buy-in / 2100 at 300g (×7 multiplier), no adds, no phases, guaranteed Catalyst of Unmaking (30067) drop.
- **Suppression aura:** flat -30 Wil/-30 Cha everywhere in the dungeon, spell damage × 0.35, mutation-bonus damage × 0.35 (physical melee untouched). Applies zone-wide, all 30 rooms.
- **Hazard tax:** 6 corridor rooms passively drain ~9%/tick HP (3 ticks) just for standing in them; 1 optional trapped door adds ~11%/tick (2 ticks) on a failed pick.
