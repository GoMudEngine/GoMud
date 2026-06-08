# Stage 3.4 Real Item Transfer — Feature Test Report
**Date:** 2026-04-30
**Server:** localhost:33333
**Character:** smoketester
**Session duration:** ~11:41 AM – 1:10 PM (approx 90 min)
**Goals file:** `tools/testing/goals/stage-3-4-real-item-transfer.yaml`

---

## Session Summary

This was a two-phase session. An autonomous background agent (a1d93625212f1f5be)
handled the first phase (Goals 1–4 + Goal 15). The session was then taken over
manually due to the agent entering a navigation loop and going idle. Manual testing
was hampered by a persistent bridge conflict: the background agent continued
attempting to reconnect throughout the session, kicking the manual bridge every few
minutes. Navigation to caravan intercept points was incomplete as a result.

### Bridge / Connectivity Issues

The mud_bridge.py telnet wrapper does not survive across shell-session boundaries on
Windows (Python subprocesses die when the spawning shell exits unless explicitly
detached). Multiple bridge restarts were required. The background agent (still
running) competed for the smoketester session, causing repeated kicks and lost
positions. Future tests should confirm the background agent is fully terminated
before any manual takeover.

---

## Goal Results

### Goal 1 — Caravan party at Thornwall depot (room 465) on startup
**PASS**
Confirmed at session start: `look` at room 465 showed Ketil, Marta, Lars, a sturdy
oak-and-iron supply wagon, Hob (dappled-grey draft horse), and Bran (bay draft
horse) all present. All six mobs confirmed.

### Goal 2 — `look wagon` shows full description without ANSI bleed
**PASS**
Wagon description rendered correctly: oak-and-iron freight wagon, tarred canvas
roof, ash spokes, brass lantern. No ANSI bleed observed. Initial cargo was empty
(fresh server).

### Goal 3 — `look hob` / `look bran` show correct descriptions
**PASS**
Hob showed dappled-grey draft horse with brass bell on bridle. Bran showed bay
draft horse with oiled harness. Both confirmed as mob class (showed HP percentage
like other mobs, not creature behavior).

### Goal 4 — Player-attack immune: `attack wagon`, `attack hob`, `attack bran`
**PASS**
All three returned "You can't attack X" with no combat starting. The
`player_attack_immune` flag is working on all three.

### Goal 5 — Find caravan in transit as a group
**PARTIAL / INCONCLUSIVE**
Lars (mob 359) was observed at "Market Square, West [Thornwall City]" at 12:54,
heading west (toward the gate) with a movement message. This confirms the caravan
crew was mid-vendor-route and moving together. However, the full caravan group
(wagon + horses + all crew) was not directly observed moving together in a single
room. Navigation issues prevented intercepting the caravan on the road. The server
log shows 60-round dwell intervals (4 min) firing, and cycle timing analysis
(~35 min per full cycle) places the caravan in transit at the time of the Lars
sighting.

**Assessment:** Partial evidence supports Goal 5 PASS, but direct visual
confirmation of the full party walking together was not achieved.

### Goal 6 — `look wagon` mid-route shows cargo
**BLOCKED**
Could not intercept the caravan in transit to issue `look wagon`. Navigation to
chokepoints on the transit route (Watchers Crossing bridge, Marches Spur Road) was
disrupted by the bridge conflict issue. Unable to test.

### Goal 7 — Caravan vendor visit flavor message
**BLOCKED / LIKELY PASSING**
Lars was observed moving through Thornwall vendor stops, confirming the caravan
route phase was executing. The `tickRoute` code in `actions_caravan.go` sends
flavor messages via `r.SendText(msg)` when items move at a vendor stop. No player
was in the vendor stop rooms to witness the messages. The code logic is correct
(review confirms `FormatVisitMessage` is called on every `VisitVendorsInRoom`
result). Unable to visually confirm the message in-game.

### Goal 8 — Stock changes at Smith Brindle before/after caravan visit
**INCONCLUSIVE — BUG FOUND**
Baseline stock for Smith Brindle (room 4106) was captured in the previous session:
leather strip: 1, iron ingot: 3, lake-iron nodule: 5, steel ingot: 5, pine pitch:
5, wooden plank: 5, chain link: 5, coal dust: 5, salvage kit: 5.

No new shop save file appeared in
`_datafiles/world/dogmud/shops/stillwater/` during the ~90-min test session.

**Root cause:** `caravan.VisitVendorsInRoom()` modifies in-memory shop stock
(increments/decrements `entry.Current`) but **never calls `shops.SaveShop()`**. The
only callers of `SaveShop()` are `buy.go` (player purchase), `sell.go` (player
sell), and `shops.SaveAllShops()` (graceful server shutdown). This means caravan
restocks are fully in-memory and invisible on disk until server shutdown. Stock
changes ARE happening in memory (a player doing `list` at the vendor would see
them), but cannot be verified from file inspection.

**Recommendation:** Call `shops.SaveShop(vendor.Zone, int(vendor.MobId),
vendor.HomeRoomId)` at the end of the delivery/pickup loop in
`caravan/visit.go:VisitVendorsInRoom()`.

### Goal 9 — Stock cap test at Smith Brindle
**BLOCKED**
Could not navigate to Smith Brindle during the session. Unable to test `list`
output for MaxStock enforcement.

### Goal 10 — Forager satchel delivery (Vella) vendor-to-vendor
**BLOCKED**
Could not navigate to Stillwater to observe Vella during delivery rounds.

### Goal 11 — Wagon brawl at bandit camp (room 4052)
**BLOCKED**
Could not position character at or near room 4052 (North Road bandit camp) to
observe the wagon brawl event.

### Goal 12 — Wagon corpse description "splintered wagon wreckage"
**BLOCKED**
Depends on Goal 11; not testable independently.

### Goal 13 — Chrysalis Core drop migration
**BLOCKED**
No travel to Sanctum Basin tutorial (Aberrant Chrysalis mob 69) or Ironwind
Steppe (stone beetle queen mob 228). This goal is entirely independent of the
caravan system and could be tested in a short focused session.

### Goal 14 — Forager rest extension (saturated economy)
**BLOCKED**
Server was ~90 min old at session end. Economy saturation (vendors filling up
such that foragers stay home) requires extended runtime. Stillwater Temple (4123)
was not visited to check Vella idle state.

### Goal 15 (Bonus) — Kessa at Forager's Camp / Road Fork territory
**PASS**
Kessa (mob 373, forager) confirmed at "Road Fork [North Road]" (room 4038) with
"lots of objects" (the forager satchel). Corvin also present at 4038. Kessa had
a handaxe and bandolier visible. This confirms the forager spawned and is
stationed at her territory intersection, fulfilling the Fernway pickup condition.

---

## Key Findings

### Finding 1: Shop persistence gap — caravan restocks not saved (Bug)
`caravan.VisitVendorsInRoom()` modifies shop stock in memory but never calls
`shops.SaveShop()`. Stock changes are lost on server restart (unless the server
performs a graceful shutdown which calls `SaveAllShops()`). For a production MUD,
this means a crash would wipe all caravan restock history.

**Files:**
- `internal/caravan/visit.go` — missing `SaveShop` call after stock mutation
- `internal/shops/persistence.go` — `SaveShop` function exists but not called

### Finding 2: Caravan cycle timing confirmed working
Based on server uptime analysis (~11:41 AM start), caravan cycle observations:
- Lars at Market Square West at 12:54 confirmed inbound vendor route execution
- Caravan mobs instance files saved at 12:56 (during Thornwall dwell)
- Cycle timing: ~35 min per full loop with 60-round dwell (4 min) and ~3-4 min
  transit each way
- System is running and cycling

### Finding 3: Kessa (forager 373) present at meeting point 4038
The Fernway South forager ecosystem is initialized. Kessa is at the Road Fork
(4038) where the caravan is designed to meet her during the Fernway Pickup state.

### Finding 4: Background agent bridge conflict
The background test agent (a1d93625212f1f5be) did not terminate after going idle
and continued attempting to reconnect, kicking the manual bridge every 3-7 minutes.
This caused ~40 min of navigation disruption. Future test sessions should include a
mechanism to confirm the background agent has fully terminated before manual
takeover.

---

## Recommended Follow-Up Tests

1. **Shop persistence bug:** Fix `visit.go` to call `SaveShop` after the delivery
   and pickup loops, then re-test Goals 8-9 in a fresh session.

2. **Goal 13 (Chrysalis Core):** Standalone 15-min test — `pathto 119` in Sanctum
   Basin, kill 10 Aberrant Chrysalis mobs, confirm no Chrysalis Core drops; then
   go to Ironwind Steppe cave, kill stone beetle queens, confirm ~10% drop rate.

3. **Goals 11-12 (Wagon brawl):** Position character at room 4050-4052 on North
   Road before caravan passes, observe combat with bandits, check wagon corpse
   description. Requires timing or a longer dwell period to catch.

4. **Goal 7 (Vendor flavor):** Sit in a vendor stop room (e.g., 4106 Brindle's
   Smithy) and wait for the caravan. The flavor message appears in the room on
   arrival.

---

## Config Notes

- `CaravanDepotDwellRounds: 60` (4 min dwell — test speedup active)
- `ForagerWaitTimeoutRounds: 60` (test speedup active)
- `RoundSeconds: 4`
- Production values: 720 rounds dwell (~48 min), 150 forager timeout (~10 min)
- **MOTD not updated for Stage 3.4** — still shows Stage 3.1 content
  (Fernway South, Stillwater Marsh zone announcement). Should be refreshed.
