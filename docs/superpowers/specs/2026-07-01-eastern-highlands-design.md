# Eastern Highlands (#21) — design

*Date: 2026-07-01. Phase 7 — The Eastern Road (Endgame Approach), leg 2.*
*Canonical plan: `docs/ZONE_EXPANSION.md` §Zone 7.2; lore `world.md` §The Crash Site.*

## Purpose

The second leg of the endgame arc: the broken, cursed highland east of
Cascade Pass — Reth's old survey territory — where the buried, overgrown
**hull of the crashed colony ship** first breaks the surface, and the road
ends at the **disc-door**, the locked threshold into the crash-site
interior (#22). It does three things:

1. **Reveal the hull** — escalate the world's central mystery from "hinted"
   to "undeniably a made thing," while keeping the *answer* locked inside
   #22.
2. **Deliver the hardest overworld content** — the true endgame ramp,
   peaking in a boss set-piece calibrated head-to-head with a ~300g
   Elemental Oasis royal encounter, dropping BIS-competitive gear.
3. **Stand at the door** — build the exterior up to the locked disc-door
   threshold; the disc, the opening mechanic, and the interior are all #22.

## Geography & attach

- **Opens Cascade Pass 6342's east.** 6342 ("The Threshold of the Reach")
  was deliberately built as a not-yet-passable terminus; #21 makes it the
  way in (add 6342 `east` → 6343, both cross-zone annotated).
- **30 rooms, 6343–6372**, folder **`eastern_highlands`**
  (= `ConvertForFilename("Eastern Highlands")`), region **The Eastern
  Reach**. Biome `mountains`/`cliffs`/`land` (highland plateau, erosion
  gullies, exposed basalt, heavy scrub).
- The land curves around **"a large, oddly-shaped hill"** — the buried,
  overgrown hull (world.md). The zone is shaped so the player circles
  toward the southeastern curve where the hull enters the ground (the
  door). Cartesian-clean; lay out east/southeast with z used for the
  broken elevation.

## Three mini-stages (canon)

### 7.2a — the approach (6343–6352)
Highland scrub, erosion gullies, basalt formations. **Reth's old weathered
survey markers** (the orbital symbol density ramps up — his territory). A
**lightning-split cairn** landmark (Reth named it in his Greenford Q75
testimony — a payoff room for players who did that quest). An abandoned
survey camp. Genuinely hostile terrain: the hardest natural mobs so far +
the first environmental menace. **No hull yet** — just the wrongness, and
the sense the landscape itself knows something.

### 7.2b — the hull (6353–6362)
The reveal. A **line where vegetation stops — too clean, too precise.**
Then the exposed surface: **smooth, metallic, no grain, no seam.** Rooms
walking *along* a buried made-thing of impossible scale, the light catching
it wrong at certain angles. Undeniably artificial, vast, ancient, and
profoundly WRONG. **The degraded defenses concentrate here** — the hazards
wake as you near the hull. Awe + dread. (Lore boundary below.)

### 7.2c — the entrance (6363–6372)
The southeastern curve where the hull goes into the ground. **The cleared
section — Maren's father was here** (environmental only: his weathered
camp, his tools, the labor of one man who got this far and stopped — ties
to the Ashwick/Maren lore, no NPC, no required quest). The recesses in the
hull surface (some trapped). **The Sentinel** — the last functioning
defense — guards the final approach. Beyond it: **the disc-shaped
depression bearing the symbol — the locked door.** The endgame threshold:
examine it, understand you need *the disc*, cannot open it. A not-yet
terminus toward #22.

## Lore boundary (CRITICAL — protects the whole game's central mystery)

At the hull the player may perceive that it is: **artificial, made,
metallic, seamless, vast, buried, ancient, cursed, and wrong.** Awe and
dread are the register. **NEVER** stated or implied anywhere in #21:
- that it is a *ship / vessel / craft*, that it came *from the sky / above
  / fell / crashed*, anything about *the moons / stars / Earth / another
  world*, or the revelation itself (the Chrysalis is native, the colonists
  were infected, the bloodline's immunity was chance not divine).
- "Metallic / no seam / made" is allowed (undeniable at the hull surface).
  "Machine / technology / builders / who made it" edges too far — keep to
  the *effect* (it is made and it is wrong), never the *category*.
- No NPC decodes anything. The symbol stays uninterpreted. The answer is
  #22's records, discovered, not told.

## The degraded defenses — hazard mechanics (data-driven, no engine work)

Three escalating layers, all built from existing machinery + new buff
content:

1. **Hazard rooms** (7.2b/7.2c). A room carries a `mutators:` entry with
   `playerbuffids: [<new discharge buff>]`. `Room.RoundTick()` applies the
   buff to every player in the room each round (verified:
   `rooms.go:2306 RoundTick` → `ApplyBuffIdToPlayers(spec.PlayerBuffIds)`).
   The buff is a harmful DoT ("the air over the smooth surface crawls and
   stings; something buried bites up through the boots"). You can pass
   through but not *linger*; a few rooms are effectively impassable without
   mitigation/haste. Mutators are declared always-on (force `enabled`, no
   decay) for a permanent hazard.
2. **Trapped passages.** `lock.trapbuffids` on select exits (the sealed
   recesses) — crossing fires a harmful buff, and they are **`defuse`-able**
   (the defuse skill disarms them — "you still the ward before it wakes";
   see `internal/actions/defuse.go`). Rewards a prepared player with the
   defuse skill; punishes the careless.
3. **New harmful buffs** (buff IDs 94+): 1–2 discharge / DoT buffs as the
   payload (an "energy discharge" DoT; optionally a lingering "seared"
   debuff). Described non-techie (glare, burn, a wrongness in the blood).

The defenses are **degraded** — intermittent, not a flawless kill-field —
so a strong, careful player threads them; but they make the approach
genuinely dangerous (world.md: "dangerous even to approach").

## Difficulty — the true endgame ramp

Well above Cascade Pass (which tested LOW — statpool 275 was trivial to a
mid char). #21 ramps from Cascade's level up toward **Oasis-300g parity**:

- **General mobs:** hostile highland fauna climbing from ~300 (trash) to
  higher tough/apex tiers as the player nears the hull. Hazards stack on
  top of combat.
- **The set-piece (the gate for BIS loot):** calibrated **head-to-head
  with a ~300g Oasis king/queen/prince/princess encounter + the random
  ellies that wander in.** Oasis royals are `statpool: 4` × gold, so 300g =
  ~1200 effective. The Sentinel is the primary boss (~1200) with
  **converging defense/fauna adds (~300 each)** that activate as it wakes.
- **Calibration target:** the user's actual prod char (~3800–4000
  leaderboard, clears ~325g Oasis comfortably with rare deaths) — NOT the
  low smoketester. All arc combat numbers (Cascade #20 + Highlands #21) are
  validated together in **one geared-char calibration pass** (owed;
  Cascade's 275/550 re-tune folds into it), using an Oasis A/B.

## The Sentinel boss & loot (economy-grounded)

**The Sentinel** — the one automated defense still functioning, guarding
the disc-door. A real endgame set-piece (~statpool 1200 + adds), described
from a non-techie POV (a great socketed watcher, a cold unblinking eye, a
ward that hums awake). For **solo players it is bypassable via `defuse` /
skill** (disarm rather than destroy) so it gates without hard-walling solo;
killing it is the group path and the one that yields the full loot.

**Loot (drops from the Sentinel / the set-piece):**
- **2 pieces of gear competitive head-to-head with ~300g Oasis BIS, in
  slots currently UNSERVED by instance BIS.** Slot sanity-check (2026-07-01):
  Oasis covers weapon/body/head/shoulders/ring/neck/wrist; Arena covers
  weapon/offhand/gloves/legs. **Feet and ranged weapon have NO BIS.** So the
  two drops are:
  - **Endgame boots (feet slot)** — BIS-competitive defensive boots;
    optionally a small hazard/discharge-resist flavor tying to the zone's
    degraded defenses (thematically apt — footing on the biting ground).
  - **Endgame ranged weapon (`subtype: shooting`)** — the first BIS-tier
    ranged weapon in the game (a masterwork heavy bow/arbalest). **Frame it
    as mundane masterwork (dark horn/steel, ancient make), NOT an energy
    weapon** — the lore boundary forbids overt tech; ambiguous-origin
    masterwork is fine.
  Authored as FIXED items (overworld has no gold buy-in) whose stats are
  matched to the ~300g-affixed Oasis templates (Volcanic Plate 20072,
  Earthshaker Warhammer 10027, etc. + their affix scaling). **Build task:
  read the instance affix-scaling to compute the 300g-equivalent stat
  target, then author the fixed items to match.** These are real BIS-tier
  rewards, justified by a real BIS-tier fight, and they fill genuine slot
  gaps rather than duplicating existing instance BIS.
- **A tech-relic trophy** — the Sentinel's "core" (a still-warm heartstone
  / oracle-shard), high-value sellable + a lore/quest hook, non-techie
  description.
- **A NEW ultra-rare crafting material, ~3% drop** — earmarked for the
  **future pinnacle / legendary-BIS crafts** (see
  `project_legendary_bis_craft_items`). It is currently just a
  vendor-sellable material (no recipes consume it yet) — a deliberate seed
  for the legendary-reagent economy and a genuine chase item. Give it a
  `component_tag` so future recipes can reference it.

Rarity: the game's current gear ceiling is ~rarity 60–80 (no legendary
tier exists yet). The BIS pieces + relic sit at ~rarity 80–85 (at the
ceiling, below the future legendary tier). Natural mobs top out below the
Sentinel.

## The disc-door terminus (locked; disc + interior deferred to #22)

The door at **6372** is the dramatic locked threshold: the disc-shaped
depression, the symbol pressed deep, recesses that do not yield. The player
examines it and **learns they need the disc** (which they may recall from
Pothole Coulee 5343, where "the disc" is a look-only hidden_noun today). It
does **not** open — a not-yet terminus, exactly like Cascade's 6342. No
east exit past it. Frame it as the awe/dread endgame threshold, not a
"come back later" gate.

## Deferred to #22 (Crash Site Interior) — flagged, not designed here

Brainstorm these when we design the interior:
- **The disc-acquisition mechanism** — how the player obtains a usable
  disc (Maren? a craft? a find? the Pothole Coulee disc becoming takeable?).
- **Acquisition frequency / cadence** — one-time vs repeatable, and how
  that interacts with re-runnable endgame content.
- **Whether the interior is instanced** — (the Oasis/Arena are instanced;
  the interior may want to be, for scaling + the rare legendary reagents).
- The disc→door opening mechanic; the interior itself; the revelation.

## NPCs & the lore layer

Cursed, taboo, rarely-penetrated country — **near-zero living NPCs by
design.** The Cascade Pass survivor is the last living person; beyond him
the only "voices" are the dead's. Human anchors are environmental:
- **Maren's father's cleared section** (7.2c) — his old camp/tools/work.
- **Reth's weathered survey markers + the lightning-split cairn** (7.2a) —
  Q75 payoff; the symbol density peaks on his markers.
- **Symbol density peaks here** (Reth's markers, the hull recesses, the
  door) — still threshold-only, never decoded.
- Mobs = hostile highland fauna + degraded-defense hazards + the Sentinel.
  No friendly questgivers, no shops.

## ID allocations

- Rooms **6343–6372** (30). Mobs **9543+** (fauna + Sentinel + adds).
  Items **40165+** (1–2 BIS pieces + trophy + rare material + any fauna
  loot). Buffs **94+** (1–2 discharge/DoT hazard buffs). Run
  `id_inventory.py` at build start to confirm.

## Build method & banked gotchas (from Cascade Pass / Greenford)

- Full **spec → plan → subagent-driven** (parallel room-authoring subagents
  with exact roomid/coord/exits; two-stage review; world-critic + geared
  feel-test at the end).
- Zone folder = `eastern_highlands`. Stage explicit git pathspecs, never
  `git add -A`.
- Prose: no colon-space `": "` in values; no semicolons in NPC/dialogue
  text (there's little/no dialogue here); `|` literal blocks for any long
  text; don't split hyphenated compounds across folded lines; hard-wrap
  ~78 cols; no hard numbers in player-visible text.
- Biomes: `mountains`/`cliffs`/`land` only (`wilderness` invalid).
- Overworld `statpool` is applied DIRECTLY (no instance scaling).
- Overworld combat mobs pair `behavior_archetype` + `aiprofile`.
- Terminus (6372) must not invite `go east`.
- Boot test: `ValidateZoneConsistency errors=0 warnings=0 mode=panic`,
  mob/item/buff loadedCount clean, 0 panics; nuke instance saves first.
- Hazard verification: the mutator-hazard rooms and trapped exits need an
  in-game feel-test (does the discharge buff tick, is the defuse path
  real) — part of the geared feel-test.

## What #21 deliberately is NOT

- **No revelation** (artificial + wrong, never the answer).
- **No disc, no working door, no interior** (all #22).
- **No finished-BIS-from-nothing** — BIS loot is gated behind an
  Oasis-300g-parity fight; the ultra-rare material is a future-craft seed.
- **No friendly NPCs / shops / quest.** (Q75 is referenced, not extended.)
- **No new engine code** — hazards are data-driven (mutator playerbuffids +
  trapped exits + new buffs).
