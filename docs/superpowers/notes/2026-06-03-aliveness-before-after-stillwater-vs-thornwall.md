# Aliveness authoring: Stillwater (6.1) → Thornwall (6.3) — before/after notes

Captured after the second zone pass, to script the XL broad rollout (6.5).
Two zones now share the "layered-by-fit" recipe: relationships → facts/knowledge
→ schedules → conversations, boot-validated layer by layer.

## What generalized cleanly (the repeatable recipe)

- **Layered-by-fit beats comprehensive.** Apply each substrate layer only where
  it earns its keep (schedules for anchor NPCs with a real work-arc, pairs only
  for co-located clusters). Trying to give every NPC every layer is wasted effort.
- **The conversation type-pools (employer/employee/family/friend/rival)** authored
  in 6.1 carried over with zero new pool work — pairs just extend them. Build the
  type-pools once; every future zone reuses them.
- **Sleep-in-place schedules avoid new rooms.** Anchors sleep in their work room
  (or an existing loft/barracks). This sidesteps the Cartesian room-overlap
  constraint entirely and keeps the schedule reachability-validator happy
  (same-room segments are trivially path-connected).
- **Role-gated `knows_facts`** (not universal) is the right default — it's what
  makes the knowledge model feel like knowledge rather than omniscience.
- **The roster-map-first approach** (dispatch an Explore agent to map ids, rooms,
  spawn points, dialogue lore, and implied relationships before authoring) is the
  single highest-leverage prep step. Do it per zone.

## The two hard-won rules (author these into every pair)

1. **Swap-safety.** The conversation engine randomizes which physical NPC plays
   speaker A vs B, so EVERY exchange line must work no matter who says it. The
   trap is authority/employer pairs (boss-only or guard-only lines). Both zones'
   reviews caught swap-breaks here; recast such lines as neutral observations
   either NPC could voice. Symmetric pairs (two old friends) are naturally safe.
2. **Co-location.** A pair-override only ever fires when the two NPCs share a room
   while both idle. Author pairs ONLY for clusters that co-locate via schedule
   (tavern staff, regulars at one table, a guard's beat overlapping a vendor's
   stall). If they never co-locate, leave the relationship as substrate only — no
   pair file. (6.1 shipped two dead pairs before a co-location fix; 6.3 avoided it
   by choosing co-located pairs from the start.)

## What was harder in a denser, quest-laden city (Thornwall vs Stillwater)

- **Quest-spoiler firewall.** Thornwall has active quests (a smuggling ring, a
  missing girl, a bribe ledger). Facts and conversations had to reference ONLY
  public troubles (the mayor's disgrace, road bandits, tax/"protection" grumble,
  the caravan) and keep quest secrets out of gossip. This is a per-line discipline
  the village (Stillwater) didn't need. For 6.5: list each zone's quest secrets
  up front and treat them as a denylist for authored gossip.
- **Travelling mobs are substrate-only.** The caravan crew (Ketil/Marta/Lars)
  has relationships but no pair-overrides — they travel the inter-zone route, so
  they're rarely idle in one room. Author their relationship edges (kin/crew) for
  the substrate, but don't waste pair content on mobs that won't co-locate idle.
- **Co-location is harder in a big room graph.** A village funnels NPCs through a
  square and a tavern; a city spreads them across a market, craft quarter, temple,
  bank, and barracks. Fewer natural co-locations → fewer viable pairs per NPC.
  Lean on deliberate schedule beats (e.g. the guard captain's midday market
  inspection) to manufacture one or two good co-locations.
- **A fact only gossips if a GOSSIPER knows it.** Seeding a fact onto a
  non-gossiper NPC means it never spreads. When authoring `knows_facts`, ensure
  each fact you want gossiped lands on at least one `gossiper`-tagged NPC.

## Engine fix surfaced by the pass (6.3 §E)

The 6.1 smoke noted some seeded facts never gossiped. Investigation showed it was
NOT a tag filter (the `fact-default` template catches every tag). The real cause:
`buildGossipLine` returned the generic fallback template whenever a zone had no
recent **world events**, *before* consulting the mob's known facts — so in a quiet
zone, seeded facts never gossiped at all. Fixed (6.3 Task 5): the no-events branch
now tries known facts first. This is a prerequisite for facts to matter in any
town that doesn't generate a steady stream of world events — relevant to every
6.5 zone.

## Recommendation for 6.5 (broad rollout)

Per zone, in order: (1) roster-map via Explore agent + list quest secrets;
(2) relationships (one side per edge); (3) facts (public-only) + role-gated
knowledge, ensuring gossipers cover them; (4) schedules for anchors (sleep-in-
place, no new rooms); (5) co-located, swap-safe pairs only. Boot-validate after
each layer. Most of this is delegable to content subagents once the per-zone
roster map + quest-secret denylist are in hand.
