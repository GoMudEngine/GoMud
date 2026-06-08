# Test Report: Aliveness sanity pass (pre-push, today's work)

**Date:** 2026-06-05
**Target:** local (port 33333)
**Role:** feel-tester (focused sanity, not a full session)
**Character:** smoketester (admin)
**Goals file:** none (ad-hoc: factions, town schedules, gossip, shops, Warren)
**Duration:** ~5 min, ~10 commands (quick pre-push sanity, not a 30-min run)

## Session Summary

Quick admin-driven sanity pass over today's merged work (6.4 perf instrumentation,
templates crash fix, the full 6.5 content pass, revenge-witness, general-store
restock) before a prod push. Teleported through Thornwall (Temple), Ashwick, and
Watcher's Crossing; confirmed the faction graph, NPC schedules, idle behavior, and
shop stocking are live and sane. The deeper behavioral systems (gossip, conversations,
kin-revenge, crime-witness reactions, restock-over-time) are real but need time/setup
to observe — noted below as "checkable but not in a 5-min pass."

## Findings

### PASS: Faction graph live + Warren corrected
`faction list` shows all **13** factions with the intended graph — the law-bloc
clique (guards/wardens/citizens/caravans/shopkeepers/villages all mutually allied),
bandits ↔ ironwind_tribe as the outlaw cluster (enemies of all 9 law-bloc), and
`bloodline_agents` as a neutral placeholder. **The Warren shows `allies=[] enemies=[]`**
— the 1.2 guard-enemy edge is gone, confirming the reframing landed. default_reps
correct (bandits −35, ironwind −25, caravans +10).

### PASS: NPC schedules steering by time-of-day
Mid-work-hours, Ashwick's Central Green (the evening social hub) held only a Village
Chicken — the townsfolk were at their *work* stations: **Delia found at her cottage**
(4027), her morning-work segment. At Watcher's Crossing, **Innkeeper Tolva + the
Traveling Merchant were at the Inn** (Tolva's all-day anchor) while Brecca/Harn were
away at their day stations. NPCs are distributed by schedule, not clustered.

### PASS: Idle emotes + shops
Idle commands fire (the chicken's "struts across the road" emote). Brecca's trading
post `list` renders a full, well-stocked buy list (100 iron ingot, 100 clay flask,
20× consumables, 80 leather strip) — shops are stocked and the buy UI works.

### CONCERN: `map-room">` artifact in the ASCII map box
At the Crossing Inn, the rendered map box contained a leaked `map-room">` fragment.
Likely a bridge ANSI/charset-strip artifact (the skill warns these are bridge
limitations, not game bugs), but it reads like a leaked template/markup tag, so
worth a quick eyeball in the map renderer before assuming it's harmless. Cosmetic;
not push-blocking.

### OBSERVATION: what a tester CAN vs CAN'T verify quickly
**Checkable fast (verified above):** the faction graph (`faction list`), NPC
schedule placement (visit work rooms vs the social hub at different game hours),
idle emotes, shop stock + `buy`/`list`.
**Checkable but needs time/setup (NOT done in this pass — for a longer session or
your in-game check):**
- **Gossip / NPC↔NPC conversations** — probabilistic per-tick; needs two co-located
  idle gossiper NPCs (e.g. evening at the hub when schedules converge) and patience.
- **General-store restock-over-time** — deplete Wulf (Stillwater, caravan-served)
  and idle ~37 rounds (~2.5 min) to watch the baseline refill fire.
- **Kin-revenge** — kill one wolf/goblin/warren member, watch surviving kin react.
- **Crime-witness split** — steal/attack in a guarded room; expect the guard to
  enforce (warn/arrest), a noncombatant to recoil + flee, a combat thug to pursue.
- **Warren no-auto-attack** — visit the labyrinth; a low-rep but non-enemy Warren
  should be wary, not attack on sight.

## Raw Stats
- Commands sent: ~10
- Fights: 0 · Deaths: 0 · Spells: 0 · Items used: 0
- Bugs: 0 · Concerns: 1 (map-room artifact) · Passes: 3
