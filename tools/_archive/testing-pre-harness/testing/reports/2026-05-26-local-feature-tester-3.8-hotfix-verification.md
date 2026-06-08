# Test Report: 3.8 Hotfix Verification

**Date:** 2026-05-26
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester
**Goals file:** 3.8-hotfix-verification.yaml
**Duration:** ~30 minutes, ~60 commands sent

## Session Summary

Tested the 3.8 hotfix changes by following Lars on his runner circuits in
both Stillwater and Thornwall, capturing pre/post vendor list state at
multiple shops, observing forager state at sanctuary/territory, and
querying the economy dashboard JSON API. Strongly mixed results: vendor
trades clearly happen (multiple shops showed stock decreases consistent
with Lars's bucketed pickups), but the FormatVisitMessage flavor line was
never captured verbatim in the bridge output, and the dashboard
FromForagers column is STILL 0 for all caravan-served zones despite
on-disk + 1w-delta evidence of forager deliveries. **The H3/H4 hotfix
does not appear to be effective on the currently-running build.**

Important context: I noticed during investigation that the working
checkout was on `feature/aliveness-4.1-goal-representation`, not
`feature/aliveness-3.8-hotfix`. The running server's source for
`internal/economy/health/scoring.go::territoryMatchesZone()` does NOT
contain the case-normalization fix from hotfix commit `f1e0d63c`. If the
server was started against this checkout, that explains the FromForagers
failure — the hotfix code is not actually live. Worth confirming with
the user before treating the H3/H4 result as a regression.

## Goal Results

- [ ] **Find Ketil + Lars at depot** — PARTIAL: Located both. Initial
  session: Lars at 4101 (South Approach) on a mid-circuit step,
  Ketil at 4105 (Tinder & Tackle). Found wagon parked at 4109
  (Stillwater depot) and later at 465 (Thornwall depot) during the
  caravan return leg.

- [ ] **BUG-01 (first vendor stop produces trade message)** — BLOCKED:
  Trade clearly happened at Voss (first stop on Thornwall circuit) —
  Voss's healer's root went 23→21, bitter thistle 38→37, dustwalk
  herb 79→78, glass vial 99→98 between Lars's arrival and departure.
  **However**, the verbatim "Lars unloads/loads up/trades a satchel"
  message from `FormatVisitMessage` was NOT observed in the bridge
  output. The only emote I caught was the generic restock message
  "A runner drops off a bundle of materials. apothecary Voss checks
  the contents and nods." — this is from
  `internal/hooks/MobIdle_HandleIdleMobs.go` (TickMobCraft restock
  branch), not from `arrival_listener.handleVendorArrival`. Either
  the FormatVisitMessage broadcast didn't fire, OR I missed it in
  the 4-second window between teleport and Lars's departure (the
  patrol dwell is 3 rounds so the trade emote can pass quickly).

- [ ] **BUG-02 (material pickup expansion at Brindle/Wulf)** — FAIL:
  Wulf (Storekeeper, Tinder & Tackle, 4105) wasn't found by
  `locate` — no Wulf mob in the running world; the patrol still
  lists 4105 as a vendor stop but it's now Ketil's idle location
  rather than Wulf's shop. Brindle was checked twice (4106): Lars
  visited and the after-list showed iron ingot 96→**100**
  (increased!) and steel ingot/coal dust UNCHANGED at 80 each.
  Brindle's lake-iron nodule (zone-source) was also unchanged at 6.
  This is the OPPOSITE of the expected behavior per goal — the
  base/overlap-bucket pickup expansion is not visibly working on
  Brindle's iron/steel/coal stock. Iron ingot's increase from 96→100
  is consistent with a natural restock cycle running between visits.

- [x] **H7 (waypoint reorder polish)** — PASS: Lars's Thornwall
  circuit (471 → ? → 475 → 481 → 465) and Stillwater circuit
  (4102 → 4103 → 4106 → 4125 → 4126 → 4143 → 4109) both walked
  through distinct rooms with no obvious revisit of any vendor
  room mid-circuit. Specifically did NOT see him traverse 4102
  (Lakefront Square) more than once on a single Stillwater circuit
  even though the underlying graph would make that easy. Transit
  felt reasonably efficient.

- [x] **H8 (foragers stop hoarding)** — PARTIAL PASS:
  - Tova (371) was found at her sanctuary room 4198 (Reedwoven Hut)
    with the locked ironbound lockbox present and "Carrying: a few
    objects" — this is consistent with her completing a deliver→
    sanctuary cycle (not stuck Foraging indefinitely).
  - Halix (372) was actively walking through her steppe territory
    (observed 3001 → 3003 movement during the session), not stuck
    at sanctuary or stuck in one room.
  - Could not directly observe a state-transition event in flight,
    but neither forager presented as cargo-pinned or hoarding.

- [ ] **H3 + H4 (FromForagers column non-zero)** — FAIL: Hit
  `/admin/api/economy/` JSON. `scores.InputRateByZone` shows:
  ```
  Stillwater:     from_foragers: 0  from_restock: 36685
  Thornwall City: from_foragers: 0  from_restock: 45295
  ```
  Both still 0. Cross-checked the deltas: in the 1w window,
  forager 371 has `DeliveriesByTierDelta: {"30": 972}` and forager
  372 has `DeliveriesByTierDelta: {"40": 106}`. So deliveries ARE
  recorded; they're just not being attributed to the correct zone
  in `buildInputRateRow`. Source confirms the case-mismatch bug:
  `territoryMatchesZone("stillwater_marsh", "Stillwater")` returns
  false because `"stillwater_marsh"[:10] == "stillwater" != "Stillwater"`.
  The hotfix's `f1e0d63c` "normalize display<->snake case" change
  appears not to be present in the running build (see Session Summary).

## Findings

### FAIL: H3/H4 dashboard FromForagers still zero

Verified directly via `/admin/api/economy/` JSON: both Stillwater and
Thornwall City zones report `from_foragers: 0` despite 972 + 106
deliveries logged in the corresponding forager YAML files
(`_datafiles/world/dogmud/foragers/stillwater_marsh/371.yaml` and
`ironwind_steppe/372.yaml`) and despite a 1w `DeliveriesByTierDelta`
of 972 and 106 in the dashboard's own deltas window. The
case-sensitivity bug in `territoryMatchesZone` is still active on the
running server. Note: source inspection shows the file
`internal/economy/health/scoring.go` does NOT have the case-normalization
fix from hotfix commit `f1e0d63c`, so the running build likely does
not include the hotfix. The user should confirm which branch the
running server binary was built from.

### CONCERN: BUG-01 trade message format never observed

Lars's first-vendor-stop trade at Voss (Apothecary, room 471, Thornwall
circuit) clearly executed — inventory deltas (healer's root, bitter
thistle, dustwalk herb, glass vial) are consistent with bucketed
pickup. But the `FormatVisitMessage` flavor line
("Lars unloads/loads up/trades a satchel of...") was NOT seen in
the bridge output. Possible explanations:
1. The message DID fire but I missed it in the brief window between
   Lars's arrival and my `list` command (dwell is 3 rounds, teleport
   eats one).
2. The message fires under a `messaging.CategoryMobEmote` category
   that the bridge or my output capture filtered out.
3. The arrival_listener never reached the `FormatVisitMessage` branch
   because the trade was triggered by some other code path
   (e.g., a separate vendor restock heuristic, not the caravan_vendor
   arrival event).

Recommend a code-side verification that the arrival_listener fires
the message broadcast in the `handleVendorArrival` path before
treating BUG-01 as fixed. The MEMORY note for chunk 3.8 says messages
fire correctly in unit tests; observation gap in the live world is
the open question.

### CONCERN: BUG-02 Brindle iron/steel/coal unchanged after Lars visit

Brindle's stock at room 4106 between Lars's pre-arrival and
post-departure snapshots:

```
                  pre   post  delta
iron ingot         96    100    +4   (restock, not pickup)
steel ingot        80     80     0
coal dust          80     80     0
lake-iron nodule    6      6     0   (zone-source bucket)
leather strip      20     23    +3   (restock)
```

This is the exact failure the BUG-02 hotfix was supposed to address.
Iron ingot's increase is a normal restock (max_stock 100, was 96).
The hotfix `28b02054` "expand pickup filter to crafting materials,
not just zone-source" should have made iron ingot/steel/coal show
up in Lars's pickup bucket, with current/10 ≈ 8-10 of each item
removed during his visit. None of that visibly happened. Combined
with the H3/H4 source mismatch, this strongly suggests the running
server is NOT executing the hotfix code.

### OBSERVATION: Lars circuit timing

A full Thornwall circuit (depot → 5-6 vendor stops → depot) takes
roughly 12-15 game-hours of real time (observed time advanced from
~9:05 AM to ~9:45 AM during one Stillwater circuit; Thornwall
circuit from caravan-arrives-at-depot ~2:00 PM to Lars-back-at-depot
~2:30 PM). Per the patrol YAML each waypoint dwells 3 rounds plus
travel time, so this matches expectations. Wagon parks at the depot
between circuits and doesn't follow Lars — that part of chunk 3.8 is
clearly working.

### OBSERVATION: Greedy nearest-neighbor reordering working

Lars's Stillwater circuit visited 4102 → 4103 → 4106 → 4125 → 4126 →
4143 → 4109. The patrol YAML lists waypoints in declared order
4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143, 4109. The actual
walk skipped 4105 (Tinder & Tackle / Ketil's idle, no shop right
now) and 4135 (Miller, also no shop active). The path was
reasonably direct — no obvious thrashing back through Market
Square mid-circuit. H7 fix is visibly working.

### OBSERVATION: Foragers Tova and Halix both alive and active

- Tova at sanctuary 4198 carrying "a few objects" + locked lockbox
  present. State `foraging` per dashboard but physically present at
  sanctuary, which is consistent with the post-fix state-transition
  flow (she's mid-cycle, not stuck).
- Halix observed actively moving through her steppe territory rooms
  (3001 → 3003 transition seen in real time). Dashboard shows her
  state as `foraging`, which makes sense.
- The pre-3.8 hoarding behavior (foragers stuck in StateForaging
  with full inventory) was not observed for either.

The MEMORY note about "Tova despawn post-3.8" was investigated — she
was found alive at her sanctuary, so either the despawn issue is
intermittent or the user's local server didn't reproduce it in this
session window.

### PASS: Caravan wagon parks at depot

Confirmed both at room 4109 (Stillwater depot) and room 465 (Thornwall
depot, when caravan returned during session): "Caravan Wagon" present
in room with Ketil, Marta, Bran, Hob, Lars, and Fenwick. Wagon does
not follow Lars on his runner circuit — exactly the chunk 3.8
architecture.

### PASS: Voss first-stop trade fired (despite unobserved message)

Whether or not the flavor message was captured, the underlying bucket
trade executed at the FIRST vendor stop of Lars's Thornwall circuit
(Voss at 471). Pre-fix behavior was reportedly a silent first stop;
post-fix the inventory delta is visible. Counts BUG-01 as
functionally fixed if you're willing to take inventory deltas as
proof — but the goal language asks for the verbatim trade message,
which I could not capture.

## Raw Stats

- Commands sent: ~60
- Fights: 0
- Deaths: 0
- Spells cast: 1 (chrysalis-glow for dark room at 4198)
- Items used: 0
- Bugs found: 0 new (verifying existing hotfix)
- Concerns: 2 (BUG-01 message capture gap, BUG-02 Brindle stock unchanged)
- Observations: 4 (circuit timing, waypoint reorder, forager state,
  wagon parking)
- Confirmed running source matches hotfix: NO (scoring.go territoryMatchesZone
  still has case-sensitive comparison — see Session Summary)

## Pre/Post Vendor List Excerpts (verbatim)

### Thornwall: Voss (room 471, FIRST vendor stop on Thornwall circuit)

Pre-trade (right after Lars arrives):
```
| 100 | clay flask         | object | 1     |
| 23  | healer's root      | object | 2     |
| 24  | warrior's brew     | potion | 30    |
| 37  | healing salve      | potion | 5     |
| 37  | stamina tonic      | potion | 5     |
| 38  | bitter thistle     | object | 2     |
| 39  | minor antidote     | potion | 5     |
| 40  | conviction draught | potion | 7     |
| ...
| 79  | dustwalk herb      | object | 2     |
| 80  | cloth strip        | object | 2     |
| 99  | glass vial         | object | 1     |
```

Post-trade (after Lars leaves north):
```
| 100 | clay flask         | object | 1     |
| 21  | healer's root      | object | 2     |   <- -2
| ...
| 37  | bitter thistle     | object | 2     |   <- -1
| ...
| 78  | dustwalk herb      | object | 2     |   <- -1
| 80  | cloth strip        | object | 2     |
| 98  | glass vial         | object | 1     |   <- -1
```

Emote captured immediately after (NOT the satchel emote):
```
A runner drops off a bundle of materials. apothecary Voss checks
the contents and nods.
```
(This is the generic TickMobCraft restock emote, not
FormatVisitMessage. The actual satchel message was not captured.)

### Stillwater: Brindle (room 4106)

Pre Lars visit:
```
| 100 | wooden plank     | object  | 1     |
| 100 | chain link       | object  | 1     |
| 20  | leather strip    | object  | 2     |
| ...
| 6   | lake-iron nodule | object  | 118   |   (zone-source, capped low)
| 80  | steel ingot      | object  | 2     |
| 80  | coal dust        | object  | 2     |
| 96  | iron ingot       | object  | 1     |
```

Post Lars visit:
```
| 100 | iron ingot       | object  | 1     |   <- +4 (restock)
| 100 | wooden plank     | object  | 1     |
| 100 | chain link       | object  | 1     |
| 23  | leather strip    | object  | 2     |   <- +3 (restock)
| ...
| 6   | lake-iron nodule | object  | 118   |   <- unchanged
| 80  | steel ingot      | object  | 2     |   <- unchanged
| 80  | coal dust        | object  | 2     |   <- unchanged
```

No emote captured during Lars's window in the room. None of the
base-bucket materials (iron/steel/coal) moved.

### Stillwater: Edda (room 4143)

Pre Lars visit:
```
| 10  | cattail down  | object | 3     |
| 12  | cloth strip   | object | 2     |
| ...
| 50  | thread spool  | object | 1     |
| 50  | bone needle   | object | 1     |
```

Post Lars visit:
```
| 6   | cattail down  | object | 8     |   <- -4 (and price 3->8)
| 12  | cloth strip   | object | 2     |
| ...
| 48  | thread spool  | object | 1     |   <- -2
| 50  | bone needle   | object | 1     |
```

Cattail down (-4) and thread spool (-2) consistent with Lars pickup
bucket including base materials. No verbatim trade message captured.

## Dashboard JSON excerpt

`/admin/api/economy/` → `scores.InputRateByZone`:
```json
"Stillwater": {
  "score": 100,
  "items_per_day": 1483,
  "from_foragers": 0,           <- FAIL
  "from_restock": 36685,
  ...
},
"Thornwall City": {
  "score": 100,
  "items_per_day": 2193,
  "from_foragers": 0,           <- FAIL
  "from_restock": 45295,
  ...
}
```

`/admin/api/economy/` → `deltas["1w"].foragers`:
```json
"371": { "DeliveriesByTierDelta": {"30": 972}, ... },   (Tova)
"372": { "DeliveriesByTierDelta": {"40": 106}, ... },   (Halix)
```

So the deliveries are visible in the deltas pipeline but never
attributed to a zone in `from_foragers`. Case-sensitivity bug at
`territoryMatchesZone("stillwater_marsh", "Stillwater") == false`.
