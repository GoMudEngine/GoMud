# Mob Aliveness 6.5 — Content Pass Decomposition

**Status:** Decomposition approved — sub-chunk specs to be written when each is picked up
**Roadmap chunk:** 6.5 (Phase 6 / Polish), Size XL → decomposed into 6.5a + three category batches
**Date:** 2026-06-05
**Depends on:** 6.3 (Done). Performance gate: 6.6 re-review runs after, against the 6.4 baseline.

## Why this doc

6.5 ("apply the aliveness framework to the rest of the game") is XL and cannot be
a single spec. This decomposes it into one faction precursor (6.5a) plus three
**category batches**, each its own roadmap sub-chunk with its own spec/plan when
picked up, and each independently boot- and smoke-tested before the next.

The core insight driving the decomposition: **"the framework" means different
things per zone category.** Towns get the full 6.1/6.3 treatment; wilderness gets
a lighter touch (no townsfolk → no schedules/conversations); roads are mostly
already covered by 3.7 inter-zone patrols. So the batches differ by recipe, not
just by zone list.

## Scope

**In scope** (3 categories):
- **Towns/settlements** — full framework
- **Wilderness/dungeon** — light treatment
- **Roads/travel** — caravan/patrol flavor

**Excluded:**
- Instance/system zones (instance_arena, instance_jail_cell, instance_planar_oasis,
  test_arena, shadow_realm) — ephemeral/combat, no standing population.
- sanctum_basin — superseded by the newbie-area rework.

## Sequence (run in this order)

### ① 6.5a — Faction definitions (precursor — MUST land first)

Author the remaining world factions on the 1.2/1.3 substrate so every content
batch tags mobs (`groups: [<faction_id>]`) against a settled roster + ally/enemy
graph. Doing this first avoids each batch inventing factions ad hoc and a later
reconciliation pass.

**Already authored (5):** `warren`, `thornwall_guards`, `thornwall_citizens`,
`stillwater_guards` (the Stillwater militia), `stillwater_citizens`. So the 6.5a
brief's "Stillwater militia & citizens" is effectively done — 6.5a is the
*remaining* roster.

**To author in 6.5a:**
- **bandits** — wilderness/road raider faction; enemy of guards + citizens.
- **ironwind shaman / Ironwind tribe** — the ironwind_steppe pack/tribe faction.
- **Dustwalk caravans** — the caravan/merchant-train faction (dustwalk_road).
- **road wardens** — road-patrol/law faction (north_road, marches_spur_road,
  world_road); ally of guards, enemy of bandits.
- **Bloodline Agents** — **first-class faction, author now.** Only one member in
  game today (`north_road/287-bloodline_agent.yaml`), but this is intended to be
  a **major force later**, so stand up the faction definition + ally/enemy graph
  now and tag the lone agent. Authoring early means future Bloodline content slots
  into an existing faction rather than retrofitting one.

**Candidate factions (optional — author if they earn their keep, not strictly
required):**
- **bounty hunters** — would group the 5.2 bounty-hunter NPCs into a faction so
  rep/relations apply uniformly. Useful but 5.2 already functions without it.
- **shopkeepers / chamber of commerce** — merchant-solidarity faction so that
  wronging one merchant (theft, assault) ripples to others' disposition/pricing.
  Nice emergent flavor; defer unless the towns batch reveals a concrete need.

**6.5a deliverables:** faction YAMLs under `_datafiles/world/dogmud/factions/`,
the full ally/enemy graph across the complete set, `groups:` tags on the
faction-relevant mobs world-wide, and any schema gaps surfaced. (6.5a gets its
own brainstorm + spec when picked up; this doc only fixes its roster + ordering.)

### ② Towns batch — full framework

Zones: **ashwick, thornwall_outskirts, watchers_crossing**.

Per town, mirror the 6.1 (Stillwater) / 6.3 (Thornwall) recipe:
- **Relationships** — family/friend/rival/employer/employee edges among the
  town's NPCs (`relationships:` field on mob templates; engine auto-mirrors).
- **Facts** — public troubles only, **no quest spoilers** (the 6.3 rule). New
  facts in `facts.yaml` + `knows_facts:` on relevant mobs + gossiper-group
  tagging.
- **Schedules** — reuse the 6 archetype templates authored in 6.3 where they fit
  (market merchant / food vendor / apothecary / jeweler / weaver / guard captain);
  author zone-specific ones as needed. Schedules cover all 24h or the boot-time
  validator panics.
- **Conversations** — pair overrides (`conversations/pairs/<lo>_<hi>.yaml`) +
  reuse the existing type pools; author role-agnostic, swap-safe scripts.
- **Faction tags** — from 6.5a.

### ③ Wilderness/dungeon batch — light

Zones: **a_dark_forest, ironwind_steppe, stillwater_marsh, the_fernway,
the_fernway_south, endless_trashheap, labyrinth_of_low_tunnels**.

- **No schedules, no conversations** (no townsfolk).
- **Relationships** where they fit — pack/kin edges (e.g. ironwind wolf packs;
  predator/leader archetypes already partly present).
- **Facts** — zone-level world facts (e.g. "bandits moved camp", "the marsh is
  flooding") + `knows_facts:` on relevant mobs so they gossip to passing players.
- **Behavior tuning** — predator/forager/patrol archetype tuning; light touch,
  most archetypes already exist.
- **Faction tags** — bandits, ironwind tribe, Bloodline Agents, etc.

### ④ Roads batch — caravan/patrol (lightest)

Zones: **dustwalk_road, marches_spur_road, north_road, world_road**.

- Largely covered already by 3.7 inter-zone patrols + caravans — **verify routes**
  rather than build from scratch.
- Traveler/caravan flavor; road-condition / danger facts; faction tags
  (Dustwalk caravans, road wardens, bandits, Bloodline Agents — the lone agent
  lives on north_road).

## Delegation (within a batch)

Use the CLAUDE.md parallel content-creation strategy:
- Dispatch **one content-creation subagent per zone** in the batch.
- Hand each a **pre-allocated ID block** (`python tools/id_inventory.py --alloc
  <type> <count>`) embedded verbatim in its prompt, so concurrent agents don't
  collide on "next free ID."
- Give each the per-category recipe above + `world.md` + the relevant
  `docs/schemas/` + the 6.1/6.3 zones as worked examples.
- Sequential dispatch is the default (zero collision risk); pre-allocated blocks
  enable parallelism when wall-time matters.
- Run `id_inventory.py` once more after merge as a collision-detection pass.

## Validation

- **Per zone:** boot test — the schedule / patrol / relationship / fact
  validators panic at startup on coverage gaps, unreachable target rooms, or
  unresolved references, so a clean boot past data-file load is the first gate
  (pre-push SOP). Then in-game smoke (walk the zone, observe schedules firing,
  conversations triggering, gossip).
- **Per batch:** all zones in the batch boot clean + smoke before the batch is
  marked done.
- **After all batches:** **6.6 performance re-review** against the 6.4 baseline
  (`docs/perf/aliveness-perf-baseline.md`). This is the reason 6.4 ran first —
  the content pass scales mob/zone counts and per-tick aliveness work, and 6.6
  compares the post-pass numbers against the captured idle + under-load floors to
  catch regressions while there's still headroom.

## Roadmap impact

Add sub-chunk rows to `MOB_ALIVENESS_ROADMAP.md` when 6.5 is picked up:
6.5a (faction definitions), 6.5b (towns batch), 6.5c (wilderness batch),
6.5d (roads batch) — or keep 6.5a as-is and track the three batches under 6.5.
Each gets its own `docs/superpowers/specs/` + `plans/` when started.
