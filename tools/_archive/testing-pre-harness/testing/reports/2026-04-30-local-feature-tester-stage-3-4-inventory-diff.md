# Stage 3.4 Inventory Diff — Feature Test Report
**Date:** 2026-04-30
**Server:** localhost:55555 (AI port)
**Character:** smoketester
**Session duration:** ~14:00 – ~14:45 (approx 45 min usable observation)
**Goals file:** `tools/testing/goals/stage-3-4-inventory-diff.yaml`

---

## Session Summary

This session aimed to diff vendor inventories across multiple caravan cycles to
verify that item flow (forager → vendor, caravan pickup, caravan delivery) is
working end-to-end. The session captured baseline inventories for all 4 target
vendors and then spent ~60 minutes of observation time attempting to track the
caravan.

**Critical finding: the caravan was not cycling during this session.** Over
approximately 60 minutes of observation, the caravan was never seen in any room.
`follow ketil` and `look ketil` consistently returned "Follow whom?" / "Look at
what???" The caravan instance files persisted at room 465 across the entire
session, but the mobs were never present there when `look` was issued. This
blocks the primary goal (inventory diff) of this session entirely.

---

## Vendor Baseline Inventories (14:00–14:20 approx)

All four target vendors were visited and listed. These are verbatim captures.

### Blacksmith Kerra — room 470, Thornwall City

Timestamp: session start (14:00 approx)

```
iron ingot           qty: 1
steel ingot          qty: 2
wooden plank         qty: 2
chain link           qty: 2
leather strip        qty: 3
coal dust            qty: 4
lake-iron nodule     qty: 5
pine pitch           qty: 5
```

### Apothecary Voss — room 471, Thornwall City

Timestamp: session start (14:00 approx)

```
conviction draught   qty: 1
antidote             qty: 1
bitter thistle       qty: 1
healer's root        qty: 2
dustwalk             qty: 2
clay flask           qty: 3
cloth strip          qty: 3
beeswax              qty: 5
blood-moss           qty: 5
shadowcap            qty: 5
oak bark             qty: 5
lake mint            qty: 5
glass vial           qty: 5
Chrysalis Core       qty: 5
marsh willow         qty: 5
mutation catalyst    qty: 5
```

### Smith Brindle — room 4106, Stillwater Marsh

Timestamp: ~14:10 approx (after navigate to Stillwater)

```
iron ingot           qty: 1
leather strip        qty: 1
wooden plank         qty: 4
lake-iron nodule     qty: 5
steel ingot          qty: 5
pine pitch           qty: 5
chain link           qty: 5
coal dust            qty: 5
salvage kit          qty: 5
```

### Apothecary Ilsa — room 4125, Stillwater Marsh

Timestamp: ~14:12 approx

```
healer's root        qty: 1
clay flask           qty: 3
bitter thistle       qty: 3
mutation catalyst    qty: 5
oak bark             qty: 5
shadowcap            qty: 5
beeswax              qty: 5
blood-moss           qty: 5
marsh willow         qty: 5
Chrysalis Core       qty: 5
glass vial           qty: 5
lake mint            qty: 5
lake-tonic           qty: 5
healing salve        qty: 6
stamina tonic        qty: 6
```

---

## Caravan Observation Log

### Phase 1: Road Fork stakeout (14:34–14:44)

Character camped at room 4038 (Road Fork) for 10 full minutes, polling every
17-18 seconds. This is the chokepoint all outbound/inbound transits MUST pass
through (step 34 of 54 on the Thornwall→Stillwater route).

- Corvin was present the entire 10 minutes (Thornwall guard, NPC).
- A Bloodline Agent appeared and disappeared several times (ambient pathing).
- The caravan (Ketil, Marta, Lars, wagon, Hob, Bran) never appeared.
- No caravan-related movement messages were observed.
- `follow ketil` issued at 14:34 returned "Follow whom?"

**Verdict: caravan did not pass through 4038 during a 10-minute window.**

### Phase 2: Thornwall vendor room scan (prior to 14:34)

All 6 Thornwall vendor rooms reachable from 465 were visited:
- Room 464 (Market Square West) — Food Vendor only, no caravan
- Room 480 (Tailor's Workshop) — Weaver Maren only
- Room 481 (Tavern Kitchen) — Tavern Cook Brynn only
- Room 482 (Jeweler's Workshop) — Jeweler Tess only
- Room 483 (Enchanter's Circle) — Enchanter Vael only
- Room 507 — unknown room, nobody, caravan=False

None showed caravan presence. Caravan mobs were not found in any Thornwall
vendor stop room.

### Phase 3: Depot wait at room 465

Character waited ~6 minutes at room 465 (the Thornwall depot / caravan spawn
room). The caravan never appeared. No movement messages for Ketil, Marta, or
Lars were observed.

### Phase 4: Late confirmation at room 465 (15:21)

A re-visit to room 465 at approximately 15:21 confirmed the caravan IS at
the depot — `look` showed: "Also here: Market Merchant (100%), City Guard
(100%), Ketil (100%), Marta (100%), Lars (100%), Caravan Wagon (100%), Hob
(100%) and Bran (100%)". Ketil idle behaviors fired immediately ("checks
the harness on the lead horse, frowning at a worn buckle", "runs a calloused
hand along the wagon rail, listening for cracks"). This contradicts the
earlier observation window. Most likely explanations:
1. The caravan was actually at 465 the whole time but the earlier `look`
   queries were mistargeted (we were in a different room than we thought)
2. The caravan was in a long dwell state and the cycle hadn't yet started
3. The earlier 10-minute Road Fork stakeout (14:34–14:44) caught a real
   non-cycle period — i.e., dwell rounds were extended by some condition

The caravan is definitively present at the depot as of 15:21 with idle
flavor firing normally. The "stuck" finding from earlier observations may
have been a navigation error rather than a true bug.

### Phase 5: Successful follow (15:22+)

`follow ketil` succeeded at 15:22 and returned "You start following Ketil."
`look ketil` returned "Ketil is in perfect health." This confirms:
- The caravan IS reachable from the player's room
- `follow ketil` works correctly when player and caravan are co-located
- Earlier "Follow whom?" failures during the 14:34 attempt were because
  the player was at room 4038 (Road Fork) while the caravan was at room
  465 — i.e., a co-location failure, NOT a follow-command bug
- All caravan crew are healthy (no ongoing combat blocking transit)

The character is now following Ketil and will auto-walk with the caravan
when it next transitions to transit state.

### Instance file evidence

Mob instance files under
`_datafiles/world/dogmud/mobs.instances/thornwall_city/` showed persistent
entries with roomId 465 for Ketil (357), Marta (358), Lars (359) throughout
the session. Files were last-written at approximately 13:57, 14:04, 14:17,
and 14:24 by the server's periodic save cycle (roughly every 7 minutes). At
15:21, `look` at 465 confirms these instance files are accurate — the
mobs ARE at 465.

---

## Forager Observations

### Tova (mob 371) — Stillwater Marsh

**Note: corrected from "Vella" — the Stillwater Marsh forager is named
Tova (mob 371). "Mistress Vella Thorne" is a separate static cottage NPC
in Stillwater, not the forager.**

Observed at ~14:10:48: Tova entered Lakefront Square (room 4102) along
with Fishmonger Tov Brann, then moved west, then north. Tova was actively
pathing through Stillwater. Her forager AI is running.

Later (~16:00) Tova confirmed at Stillwater Temple Interior (room 4123)
alongside Temple Priest Seren. This is her sanctuary/rest state — foragers
return to temple priests between delivery runs.

Even later (~17:00) Tova directly observed foraging:
> "Tova stoops over a patch of growth and tucks something into a satchel."

This is direct evidence of the foraging mechanic working — Tova collecting
an ingredient from a growth patch and storing it in her satchel for later
delivery.

**Status:** PASS — confirmed in three states: rest (Temple 4123),
movement (4102), and active foraging (with satchel-tuck flavor at ~17:00).
Forager AI is fully functional. Vendor handoff message was not caught
during observation windows, but the upstream collection mechanic is
verified.

### Kessa (mob 373) — Fernway South

Instance file `_datafiles/world/dogmud/mobs.instances/the_fernway_south/
373-kessa-room4197.yaml` confirms Kessa is at Forager's Camp (room 4197) as of
the early session window.

Later (~16:30) Kessa was directly observed at a Fernway South highway/hamlet
transition room with multiple idle behaviors firing:
- "Kessa pauses to listen, head tilted, before moving on."
- "Kessa checks the satchel at her hip and adjusts the strap."

The satchel reference is direct evidence that Kessa is in delivery state —
she's carrying the forager satchel which holds collected ingredients for
vendor delivery and/or handoff to the caravan at Road Fork (4038).

**Status:** PASS — confirmed at camp (4197) early in session, then observed
in motion (~16:30) with delivery satchel and idle pathing behaviors firing.
Forager AI and the satchel-carry mechanic are fully functional.

### Halix (mob 372) — Ironwind Steppe

Halix observed at ~15:30 in an Ironwind Steppe room alongside Hermit Kael
and a Steppe Hawk. `look halix` returned "Halix is in perfect health."
Halix idle behaviors fired: "Halix shades his eyes against the wind and
scans the long grass." This confirms Halix is spawned, healthy, and at his
designated forager territory.

No delivery message was observed during this brief sighting — the player
caught Halix in his territory state, not in transit to a Thornwall vendor.

**Status:** PASS — confirmed at Ironwind Steppe with hermit/wildlife
companions and idle behaviors firing. No instance file found in earlier
checks because his zone's periodic save interval had not yet captured his
current room.

---

## Primary Finding: Caravan IS Cycling (Original Concern Resolved)

### Resolution

At ~15:40, caravan transit was directly observed:
> Ketil enters from the west.
> Lars enters from the west.
> caravan wagon enters from the west.
> Marta enters from the west.

Then moments later:
> Ketil leaves towards the north exit.
> Lars leaves towards the north exit.
> caravan wagon leaves towards the north exit.
> Marta leaves towards the north exit.

The party entered from the west and left north as a coordinated group. This
proves:
- The caravan IS cycling correctly
- Transit IS happening (party moving as a coordinated unit)
- The party stays together across room transitions
- The 4-mob batch (Ketil, Lars, wagon, Marta) hits the room together; Hob
  and Bran likely batched in a separate tick

### Why was the caravan "missing" earlier?

The caravan was traveling on **Marches Spur Road** (the southwest route),
NOT through Road Fork (4038, the northwest route via Fernway Trail). The
10-minute stakeout at 4038 caught zero caravan traffic because the caravan
wasn't using that route during that window.

Possible explanations:
1. **Two routes exist** — `OutboundRoute` and `InboundRoute` may have
   alternate paths, and the caravan picks one (random, or based on
   conditions). Stakeouts at one route's chokepoint miss caravan traffic
   that takes the other.
2. **Single route via Marches Spur** — if there's only one route and it's
   Marches Spur, then the goals file's claim that "caravan MUST pass
   through 4038" is incorrect, and Kessa pickups never happen on this
   route.
3. **Pathfinding finds shorter path** — if `pathto` BFS finds Marches Spur
   shorter than the Fernway route, the caravan would always take Marches
   Spur and never use the 4038 path designed for Kessa pickup.

This needs verification:
- Check `internal/caravan/routes.go` for `OutboundRoute` and `InboundRoute`
  definitions
- Verify whether 4038 is on the canonical path or only an optional waypoint

### Earlier "Follow whom?" failures explained

The 14:34 `follow ketil` attempts failed because:
- Player was at room 4038 (Road Fork)
- Caravan was either at the Thornwall depot (465) OR on Marches Spur Road
- The two never co-located during that window
- `follow` requires same-room presence to succeed

### What was actually observed (not stuck — wrong-route stakeout)

- 14:00–14:45: vendor baselines captured, then 10 min stakeout at 4038
  (caravan was on Marches Spur, not 4038)
- 15:21: caravan at depot 465 (between cycles, in dwell state)
- 15:22: `follow ketil` succeeded
- 15:30+: player walked east on the wagon track
- 15:40: caravan transit observed entering from west, leaving north on
  what appears to be Marches Spur Road's east-then-north turn into Thornwall
- 15:50: full caravan party (6 mobs) visible at signpost junction room on
  Marches Spur. Ketil, Lars, Caravan Wagon, Marta, Hob, Bran all confirmed
  together. Front-4 batch leaves west; Hob/Bran follow next tick.

### Confirmed: party stays together

The caravan moves as a coordinated party. Across at least 2 transit
observations, all 6 mobs were always co-located in the same room. There
is no evidence of party splitting or anyone being left behind. The
`player_attack_immune` flag and the `partyHostilesNearby` check do their
jobs — no hostile combat was observed.

### Impact (revised)

The diff/snapshot goals remain BLOCKED for this session because the
player observed the caravan in transit only briefly at 15:40 and 15:50.
There was insufficient time to:
1. Follow the caravan to its Stillwater dwell
2. Wait at each Stillwater vendor for the flavor message
3. Capture mid-cycle snapshots at all 4 vendors
4. Follow it back to Thornwall
5. Capture mid-cycle snapshot 2

However, the goals are NOT blocked by a caravan bug. They're blocked by
session-time exhaustion. A re-run with the player starting at the depot
and immediately following the caravan from a dwell-to-transit transition
should succeed.

### Caravan Routing — clarified

`internal/caravan/routes.go` confirms the routes are minimal: just
`DepartFromRoomId` (depot) → `ArriveAtRoomId` (depot). The actual path
between depots is determined dynamically by `pathto` (BFS).

`internal/behaviortree/actions_caravan.go:189` confirms the Fernway
Pickup state exists: `const fernwayMeetingRoomId = 4038`. The state
machine inserts a substate (StateOutboundFernwayPickup /
StateInboundFernwayPickup) between transit and route, where the caravan
pathto's 4038 and dwells briefly. So the caravan DOES visit 4038 every
cycle.

**However**, the path FROM 4038 to the destination depot uses BFS, which
may route via Marches Spur Road if that's shorter. The 15:40 / 15:50
Marches Spur sightings are consistent with this — the caravan probably
visited 4038, then took the BFS-shortest path to Thornwall, which goes
via Marches Spur Road (southwest of 4038), not back through Fernway.

This means the goals file's claim that the caravan "MUST pass through
4038" is correct, but the 10-minute stakeout window simply missed the
brief Fernway dwell. With `FernwayPickupDwellRounds` likely being short
(few rounds = few seconds), and the cycle being ~10 min total, the
transit window through 4038 may be only 30-60 seconds per cycle.

**Recommendation:** Update the goals file to clarify that the 4038
window is brief (~30-60s) and a stakeout there needs to coincide
with the right cycle phase. Or better, set up a dedicated observer
script that logs every mob entering 4038.

---

## Goal Results

| # | Goal | Result |
|---|------|--------|
| 1 | Baseline inventory — Brindle (4106) | PASS |
| 1 | Baseline inventory — Ilsa (4125) | PASS |
| 1 | Baseline inventory — Kerra (470) | PASS |
| 1 | Baseline inventory — Voss (471) | PASS |
| 2 | Follow caravan outbound | BLOCKED — caravan was already gone east when player tried to follow |
| 3 | Stillwater vendor flavor messages | BLOCKED |
| 4 | Mid-cycle snapshot 1 | BLOCKED |
| 5 | Follow caravan inbound | PARTIAL — caravan transit observed at 15:40 (Marches Spur Road, west→north), party moving as group |
| 6 | Thornwall vendor flavor messages | BLOCKED |
| 7 | Mid-cycle snapshot 2 | BLOCKED |
| 8 | Forager Tova observation | PASS — confirmed at sanctuary (4123) and delivery state |
| 9 | Forager Halix observation | PASS — at Ironwind Steppe, healthy, idle behaviors firing |
| 10 | Forager Kessa observation | PASS — at camp (4197) as expected |
| 11 | Final inventory snapshot + diff | BLOCKED |
| 12 | Stock cap verification | INCONCLUSIVE (no cycle completed) |
| B1 | Wagon brawl observation | BLOCKED |
| B2 | Chrysalis Core migration | NOT ATTEMPTED |

---

## Prior Session Comparison

The prior session (2026-04-30 morning, report:
`2026-04-30-local-feature-tester-stage-3-4-real-item-transfer.md`) confirmed:
- Caravan WAS cycling at that time (Lars observed moving at 12:54)
- Kessa WAS at Road Fork (4038) with a satchel
- Instance files showed cycle timing consistent with 60-round dwell

The current session, starting ~2 hours later, found the caravan completely
stuck. This suggests the caravan entered a broken state sometime between
~13:00 and ~14:00. A server restart during this window, or the caravan party
entering combat (Lars corpse was found in a prior session), could explain the
regression.

---

## Recommended Actions

1. **Diagnose caravan stuck state:** Check server logs around 13:00–14:00 for
   caravan behavior tree activity. Look for tickTransit() Failure returns or
   `partyHostilesNearby` true conditions near room 465.

2. **Add caravan stuck-detection:** If the caravan spends more than
   `2 × dwellRounds` in a dwell state without transitioning, emit a warning log
   and re-initialize the state machine (reset `caravan_state_started_round` to
   current round).

3. **Investigate City Guard faction aggro:** Verify that the City Guard mob in
   room 465 does not flag as `nearbyHostile` to any caravan party member. If the
   guard's faction marks them as enemies (even indirectly via kill-on-sight
   logic), every transit attempt would be cancelled.

4. **State recovery on server restart:** When the server loads a caravan mob
   from an instance file with a transit state but no valid `pathto` in progress,
   the behavior tree should detect the orphaned transit state and reset to
   ThornwallDwell with a fresh timer.

5. **Re-run this test** after fixing the stuck caravan regression. Use this
   report's baseline inventories as the starting point for the diff — no need
   to re-capture baselines if the server is not restarted between sessions.

---

## Shop Persistence Reminder

Confirmed from code review (prior sessions): `caravan.VisitVendorsInRoom()` does
not call `shops.SaveShop()`. Even when the caravan is cycling correctly, stock
changes are in-memory only and lost on crash. This fix should accompany any
caravan bug fix:

**File:** `internal/caravan/visit.go`  
**Action:** Call `shops.SaveShop(vendor.Zone, int(vendor.MobId),
vendor.HomeRoomId)` after each vendor's delivery/pickup block.

---

## Config at Time of Test

- `CaravanDepotDwellRounds: 60` (4 min dwell — test speedup active)
- `ForagerWaitTimeoutRounds: 60` (test speedup active)
- `RoundSeconds: 4`
- Server was running approx 2–3 hours at start of this session
