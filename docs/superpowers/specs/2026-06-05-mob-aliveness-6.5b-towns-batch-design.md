# Mob Aliveness 6.5b — Towns Batch (Ashwick + Watcher's Crossing)

**Status:** Design approved — pending spec review → implementation plan
**Roadmap chunk:** 6.5b (Phase 6 / Polish, towns batch of the 6.5 content pass), Size M
**Date:** 2026-06-05
**Depends on:** 6.5a (faction definitions — done), 6.1/6.3 (the framework + schedule patterns)

## Goal

Apply the full aliveness framework (relationships, schedules, conversations,
facts/gossip) to the two micro-settlements — **Ashwick** (farming village) and
**Watcher's Crossing** (waystation) — at a scale matched to their size (~4 NPCs
each), mirroring the 6.1 (Stillwater) / 6.3 (Thornwall) treatment.

thornwall_outskirts is NOT in this batch — it was handled as light faction-tagging
in 6.5a (1 farmer + gate guard + highwayman).

## NPC rosters (townsfolk only)

**Ashwick** (zone rooms 4015–4034):
- `259-delia` — herbalist; home Delia's Cottage/Garden (4027/4028)
- `260-deacon_ferris` — rural deacon; Deacon's Chapel (4022), Ritual Circle (4021)
- `261-farmer_hesta` — farmer; Farmstead (4020), South Fields (4025)
- `262-the_forager` — trained herbalist, wary outsider; Forager's Camp (4031),
  Herb Clearing (4032). **Is a `noncombat_questgiver`** — verify its quest
  dialogue before writing the backstory gossip (keep gossip public-level).
- Social hub: Central Green (4017). Wildlife (fox/timber_wolf/hawk/chicken/mouse)
  are not townsfolk.

**Watcher's Crossing** (zone rooms 420–427):
- `84-innkeeper_tolva` — innkeeper; The Crossing Inn (423) — the social anchor
- `85-merchant_brecca` — trader (`noncombat_shopkeeper`); Trading Post (424)
- `86-toll_collector_harn` — toll-keeper; The Tollhouse (421)
- `88-traveling_merchant` — transient trader; sits at The Crossing Inn (423)
- Social hub: The Crossing Inn (423).

## 1. Relationships

Author `relationships:` edges on the mob templates (engine auto-mirrors symmetric
types; employer↔employee asymmetric). Approved graph:

**Ashwick:**
- Delia → The Forager: **employer** (Forager is **employee**) — herb-work
  mentor; the Forager is the wary outsider Delia took in.
- Delia ↔ Farmer Hesta: **friend**
- Deacon Ferris ↔ Farmer Hesta: **friend** (and Ferris ↔ Delia: **friend**)

**Watcher's Crossing:**
- Tolva ↔ Brecca: **friend**
- Harn ↔ Tolva: **friend**
- Harn ↔ Brecca: **rival** (the toll friction)
- Traveling Merchant ↔ Brecca: **rival** (competing merchants)

## 2. Schedules (medium depth — work / social / home-sleep)

New schedule YAMLs at `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`,
referenced by each mob's `schedule_id:`. **Schedules must cover all 24 hours or
the validator panics; every `pathto` target room must exist and be reachable.**
The 6.3 thornwall schedule files are structural references only — they target
thornwall rooms, so 6.5b authors fresh schedules with Ashwick/Watcher's room IDs.

Medium pattern per NPC: a work segment (at station), a midday or evening **social
segment** at the hub (co-locates NPCs so conversations can fire), an evening/home
segment, and a night **sleep** segment (`activity: sleeping`).

**Ashwick:**
- Delia: Cottage/Garden (4027/4028) work → Central Green (4017) social → Cottage sleep
- Deacon Ferris: Chapel (4022) work → Ritual Circle (4021) beat → Green social → Chapel/home sleep
- Farmer Hesta: Farmstead/South Fields (4020/4025) work → Green social → Farmstead sleep
- The Forager: Camp/Herb Clearing (4031/4032) work → **Delia's Garden (4028)** herb-delivery beat (co-locates with Delia) → Camp sleep. Stays wary — does NOT join the Green social beat.

**Watcher's Crossing:**
- Tolva: The Crossing Inn (423) throughout (the social anchor) → back room sleep
- Brecca: Trading Post (424) work → Inn (423) evening social → Trading Post/quarters sleep
- Harn: Tollhouse (421) work → Inn (423) evening social → home sleep
- Traveling Merchant: sits at the Inn (423) — **minimal/no schedule** (transient; left at the Inn). If a schedule is given, keep it Inn-bound.

## 3. Conversations

Bespoke per-pair scripts (`conversations/pairs/<lowerMobId>_<higherMobId>.yaml`,
role-agnostic & swap-safe per the 3.6 convention) for the two signature pairs:
- **Delia ↔ The Forager** (`pairs/259_262.yaml`) — warmth/mentorship; the
  "newcomer I took in" beat, public-level (no quest spoiler).
- **Harn ↔ Brecca** (`pairs/85_86.yaml`) — the unpopular-toll friction; light,
  comedic, Harn apologetic.

All other approved relationships rely on the generic relationship-**type** pools
(`conversations/types/{friend,rival,...}.yaml`) — no per-pair override needed.
Conversations fire only when both NPCs are co-located and fully idle, so the
schedules' shared hub beats (Central Green / the Inn; Delia's Garden for the
Forager pair) are what give them the opportunity.

## 4. Facts / gossip

Author standing facts in `_datafiles/world/dogmud/facts.yaml`, add the fact ids to
the relevant NPCs' `knows_facts:`, and tag the gossipers. **Public troubles only —
no quest spoilers** (the 6.3 rule). Four seeds:

**Ashwick:**
- (a) **Wolves prowling from the Deep Woods** — livestock/safety worry (ties to the
  timber_wolf near the village). Known by Hesta, Delia, Ferris.
- (b) **The wary newcomer Delia took in** — village curiosity about The Forager,
  kept to public-knowledge level (verify against the Forager's quest dialogue so
  it doesn't pre-empt a reveal). Known by Delia, Hesta, Ferris.

**Watcher's Crossing:**
- (c) **Bandits raiding the trade roads** — a deliberate cross-tie to the 6.5a
  `bandits` faction (travelers nervous, caravans wary). Known by Tolva, Brecca,
  Harn, the Traveling Merchant.
- (d) **Grumbling over Harn's toll** — light social fact. Known by Brecca, Tolva
  (and reflected in the Harn↔Brecca conversation).

## Validation

- **Boot test (hard gate):** the schedule validator panics at startup on coverage
  gaps, unreachable target rooms, or unresolved `schedule_id`/relationship/fact
  references; the conversation loader and facts loader validate their refs too. A
  clean boot past data-file load is the primary gate.
- **In-game smoke** (may be deferred to user per chunk 2.8/2.9 precedent): visit
  each settlement across the day; confirm NPCs move per schedule, gather at the hub,
  fire the bespoke conversations, sleep at night, and surface the facts as gossip.

## Out of scope

- thornwall_outskirts (done in 6.5a, light).
- Wilderness/roads zones (6.5c/6.5d batches).
- New quests / quest changes for these NPCs (separate content).
- Behavior-archetype changes (these stay noncombat_questgiver/shopkeeper).
- `278-haral` shopkeepers tagging (followup for the roads batch once it gets a
  `shop:` block — logged in 6.5a).

## Files touched (anticipated)

- Edit: the 8 NPC mob YAMLs (`relationships:`, `schedule_id:`, `knows_facts:`,
  gossiper tagging).
- New: ~7 schedule YAMLs under `schedules/ashwick/` + `schedules/watchers_crossing/`.
- New: 2 conversation pair YAMLs (`pairs/259_262.yaml`, `pairs/85_86.yaml`).
- Edit: `facts.yaml` (+4 facts).
- Edit: roadmap (6.5b status); per-zone context if any.
