# 2026-05-01 Room-Pin + Salvage Validation — Feature Test Report

**Date:** 2026-05-01
**Server:** localhost:55555
**Character:** smoketester
**Session duration:** ~10:00 – ~11:00 (approx. 60 min total, ~50 min usable observation)
**Goals file:** `tools/testing/goals/room-pin-and-salvage-validation.yaml`

---

## Headline Findings

- **Vendor inventory growth:** PARTIAL — Brindle saw stock DECLINE (sold through, no replenishment
  observed); Ilsa stable at cap. Kerra and Voss (Thornwall) were unreachable — see navigation
  notes. Growth could not be confirmed or denied at Thornwall vendors.
- **Corpse salvage anywhere:** PASS — salvage ran to completion in Stillwater Marsh with no
  kit-required error on both attempts. Second attempt yielded explicit "1x leather-strip" message.
- **Forager Tova room-pin:** PASS — Tova was present in temple at session start, then observed
  actively foraging throughout Stillwater Marsh (multiple rooms, multiple sighting events) across
  the middle portion of the session, and was absent from the temple at session end. Clear state
  change confirmed.

---

## Navigation Notes (Thornwall Vendor Gap)

Thornwall is not reachable within a single 60-min session by a low-gold character from Stillwater:
- The Old Bridge (room 422, Watchers Crossing) has a 5 gold toll.
- smoketester has 0 gold carried (bank: 90, no withdraw command available on road).
- Even without the toll, the route covers ~30+ rooms across North Road + Marches Spur Road +
  Watchers Crossing + Thornwall Outskirts before entering Thornwall City.
- A brief attempt to walk the road resulted in a bandit encounter (Crossroads Village Square)
  that took smoketester to near-death, requiring ~5 minutes of regen.

Thornwall vendor data (Kerra/Voss) is absent from this report. If future sessions need that
data, either give smoketester travel funds (at least 10g) or add a starting location closer
to Thornwall.

---

## Vendor Inventory Captures

### Baseline (~10:05 session time)

**Smith Brindle (room 4106, Brindle's Smithy)**
```
+-----+------------------+--------+-------+
| Qty | Name             | Type   | Price |
+-----+------------------+--------+-------+
| 3   | leather strip    | object | 19    |
| 5   | lake-iron nodule | object | 142   |
| 5   | steel ingot      | object | 19    |
| 5   | pine pitch       | object | 83    |
| 5   | iron ingot       | object | 5     |
| 5   | wooden plank     | object | 3     |
| 5   | chain link       | object | 5     |
| 5   | coal dust        | object | 12    |
| 5   | salvage kit      | object | 3     |
+-----+------------------+--------+-------+
```

**Apothecary Ilsa (room 4125, Healer's Alcove)**
```
+-----+---------------------------+--------+-------+
| Qty | Name                      | Type   | Price |
+-----+---------------------------+--------+-------+
| 4   | healing salve             | potion | 15    |
| 5   | beeswax                   | object | 71    |
| 5   | healer's root             | object | 12    |
| 5   | mutation catalyst         | object | 237   |
| 5   | oak bark                  | object | 83    |
| 5   | shadowcap mushroom        | object | 95    |
| 5   | marsh willow bark         | object | 60    |
| 5   | blood-moss                | object | 107   |
| 5   | Chrysalis Core            | object | 1181  |
| 5   | bitter thistle            | object | 12    |
| 5   | glass vial                | object | 8     |
| 5   | clay flask                | object | 3     |
| 5   | lake mint                 | object | 60    |
| 5   | stamina tonic             | potion | 12    |
| 5   | lake-tonic of steady hand | potion | 43    |
+-----+---------------------------+--------+-------+
```

**Blacksmith Kerra (room 470) / Apothecary Voss (room 471):** NOT REACHED — see Navigation Notes.

---

### Mid-session (~10:35 session time — after salvage test)

**Smith Brindle (room 4106)**
```
+-----+------------------+--------+-------+
| Qty | Name             | Type   | Price |
+-----+------------------+--------+-------+
| 1   | iron ingot       | object | 9     |
| 1   | leather strip    | object | 33    |
| 4   | wooden plank     | object | 3     |
| 5   | lake-iron nodule | object | 142   |
| 5   | steel ingot      | object | 19    |
| 5   | pine pitch       | object | 83    |
| 5   | chain link       | object | 5     |
| 5   | coal dust        | object | 12    |
| 5   | salvage kit      | object | 3     |
+-----+------------------+--------+-------+
```

**Apothecary Ilsa (room 4125)**
```
+-----+---------------------------+--------+-------+
| Qty | Name                      | Type   | Price |
+-----+---------------------------+--------+-------+
| 4   | healing salve             | potion | 15    |
| 5   | beeswax                   | object | 71    |
| 5   | healer's root             | object | 12    |
| 5   | mutation catalyst         | object | 237   |
| 5   | oak bark                  | object | 83    |
| 5   | shadowcap mushroom        | object | 95    |
| 5   | marsh willow bark         | object | 60    |
| 5   | blood-moss                | object | 107   |
| 5   | Chrysalis Core            | object | 1181  |
| 5   | bitter thistle            | object | 12    |
| 5   | glass vial                | object | 8     |
| 5   | clay flask                | object | 3     |
| 5   | lake mint                 | object | 60    |
| 5   | stamina tonic             | potion | 12    |
| 5   | lake-tonic of steady hand | potion | 43    |
+-----+---------------------------+--------+-------+
```

**Mid-session delta (Brindle only — Ilsa unchanged):**

| Item          | Baseline | Mid | Change |
|---------------|----------|-----|--------|
| iron ingot    | 5        | 1   | -4     |
| leather strip | 3        | 1   | -2     |
| wooden plank  | 5        | 4   | -1     |

Interpretation: items were sold/consumed between baseline and mid. No new items
appeared. Price changes confirm dynamic pricing: iron ingot rose from 5 to 9
(shortage pricing), leather strip from 19 to 33. This is the dynamic economy
working as designed — scarcity drives price up.

---

### Final (~10:55 session time)

**Smith Brindle (room 4106)**
```
+-----+------------------+--------+-------+
| Qty | Name             | Type   | Price |
+-----+------------------+--------+-------+
| 1   | iron ingot       | object | 9     |
| 1   | leather strip    | object | 33    |
| 4   | wooden plank     | object | 3     |
| 5   | lake-iron nodule | object | 142   |
| 5   | steel ingot      | object | 19    |
| 5   | pine pitch       | object | 83    |
| 5   | chain link       | object | 5     |
| 5   | coal dust        | object | 12    |
| 5   | salvage kit      | object | 3     |
+-----+------------------+--------+-------+
```

**Apothecary Ilsa (room 4125)**
*(Identical to baseline and mid-session — no change across all 3 passes.)*
```
+-----+---------------------------+--------+-------+
| Qty | Name                      | Type   | Price |
+-----+---------------------------+--------+-------+
| 4   | healing salve             | potion | 15    |
| 5   | beeswax                   | object | 71    |
| 5   | healer's root             | object | 12    |
| 5   | mutation catalyst         | object | 237   |
| 5   | oak bark                  | object | 83    |
| 5   | shadowcap mushroom        | object | 95    |
| 5   | marsh willow bark         | object | 60    |
| 5   | blood-moss                | object | 107   |
| 5   | Chrysalis Core            | object | 1181  |
| 5   | bitter thistle            | object | 12    |
| 5   | glass vial                | object | 8     |
| 5   | clay flask                | object | 3     |
| 5   | lake mint                 | object | 60    |
| 5   | stamina tonic             | potion | 12    |
| 5   | lake-tonic of steady hand | potion | 43    |
+-----+---------------------------+--------+-------+
```

**Final delta (Brindle, from mid to final):**

| Item          | Mid | Final | Change |
|---------------|-----|-------|--------|
| iron ingot    | 1   | 1     | 0      |
| leather strip | 1   | 1     | 0      |
| wooden plank  | 4   | 4     | 0      |

No change from mid to final at Brindle. No forager delivery was observed
during this session window.

**Full session delta (Brindle, baseline to final):**

| Item          | Baseline | Final | Change |
|---------------|----------|-------|--------|
| iron ingot    | 5        | 1     | -4     |
| leather strip | 3        | 1     | -2     |
| wooden plank  | 5        | 4     | -1     |

---

## Salvage Anywhere Test

### Attempt 1 — Marsh Rat

- **Kill location:** Mossy Hummock (Stillwater Marsh, approx. room 4184 area)
- **Mob killed:** Marsh Rat (mob 367)
- **Salvage command:** `salvage marsh rat corpse`
- **Server response:** `You begin carefully working over the marsh rat corpse...`
  *(completion message was obscured by Tova movement noise — see Forager section)*
- **Mats yielded:** sinew (1x) — inferred from inventory change. Pre-salvage inventory
  had no sinew; post-salvage inventory showed sinew present. Leather strip count
  unchanged at this point (x2 from pre-existing items).
- **Kit in inventory?** No. No "you need a salvage kit" error was produced.
- **Corpse state after salvage:** Gone from room (confirmed via `look`).
- **Result:** PASS — salvage ran to completion with no kit-required error.

### Attempt 2 — Dragonfly Swarm

- **Kill location:** Dragonfly Glade (Stillwater Marsh)
- **Mob killed:** Dragonfly Swarm (mob 368)
- **Salvage command:** `salvage dragonfly swarm corpse`
- **Server response (full sequence):**
  ```
  You begin carefully working over the dragonfly swarm corpse...
  You continue working on salvaging... (1/4)
  You continue working on salvaging... (3/4)
  You finish working over the dragonfly swarm corpse and recover: 1x leather-strip.
  *** You feel your salvage skills sharpening! ***
  ```
- **Mats yielded:** 1x leather-strip (explicit server message)
- **Kit in inventory?** No. No kit-required error.
- **Result:** PASS — explicit success message, no kit required, skill progression fired.

---

## Forager Activity Check

### Tova (room 4123, Temple of Stillwater)

- **Session start:** Present in Temple of Stillwater (100% health), along with Temple
  Priest Seren. Observed immediately on login.
- **During marsh exploration (~10:20–10:45):** Tova was NOT in the temple. She was
  observed in Stillwater Marsh, actively moving through multiple rooms:
  - Entered/left Mossy Hummock multiple times (north and south exits)
  - Entered/left Dragonfly Glade (east and west exits)
  - Fired forager idle action: "stoops over a patch of growth and tucks something
    into a satchel"
  - Also fired: "checks the satchel at her hip, eyes scanning the reedline"
- **Session end (~10:55):** NOT in temple. Temple check showed only Temple Priest Seren.
- **Conclusion:** Tova was out foraging for the majority of the session. Room-pin fix
  is confirmed working — Tova is not stuck in her spawn room.

### Halix (room 3040, Sheltered Ridge Alcove, Ironwind Steppe)

- **State:** NOT CHECKED — Ironwind Steppe (x=16, y=3) is geographically far from
  Stillwater (x=-18 to -14, y=1 to 4). Could not reach in session budget.

### Kessa (room 4197, Fernway South)

- **State:** NOT CHECKED — similarly out of range for this session.

---

## Findings

### PASS — Salvage kit removal confirmed

Salvage ran to completion twice in Stillwater Marsh without requiring a salvage kit.
No "you need a salvage kit" or "you must be at a crafting station" message appeared.
Explicit recovery message on second attempt: "1x leather-strip". Skill progression
fired. This is the intended behavior per the 2026-05-01 fix.

### PASS — Forager Tova room-pin fix confirmed

Tova's state changed from "resting in temple" at session start to "actively foraging
in Stillwater Marsh" within the session. She was observed in at least 4 different
Stillwater Marsh rooms over a ~25 minute window. This is direct evidence the room-pin
fix is working — she is no longer stuck at her home room indefinitely.

### OBSERVATION — Vendor replenishment not observed in 60-min window

No stock growth was observed at either Stillwater vendor (Brindle or Ilsa) during the
session. Ilsa was already at max stock (5 of most items) at baseline, so any new
deliveries from Tova would not be visible unless Ilsa's stock was depleted first.
Brindle's depleted items (iron ingot 5→1, leather strip 3→1) were not replenished
during the session. The ForagerWaitTimeoutRounds=150 config means Tova's foraging
cycle can take up to 150 game rounds before she returns home with materials and
triggers a shop delivery. At roughly 6 rounds/minute that is ~25 minutes minimum
from when she went out. Given that Tova appeared to be mid-route when we observed her
(already in the marsh), and the session did not observe her returning, it is plausible
that a full delivery cycle takes longer than this single session could capture.

This is NOT a confirmed failure of the vendor growth mechanic — it is an inconclusive
observation given session constraints. The recommended follow-up is a longer session
(90-120 min) with the test character starting at the moment a forager departs, or
adding server-side logging of forager delivery events to verify the pipeline.

### OBSERVATION — Tova "confused (salvage corpse)" idle message

Tova repeatedly triggered the message "looks a little confused (salvage corpse)" when
entering a room with a salvage corpse present. This appears to be a behavior-tree
reaction from Tova's NPC AI encountering a state it doesn't have a handler for
(a corpse in her foraging zone). The message is player-visible and reads oddly — it
exposes internal behavior-tree state names ("salvage corpse") to the player. This
should be investigated: either the behavior tree node should be silenced, or Tova
should have a proper player-visible idle action for this case (e.g., "Tova glances
at the remains and moves on").

### CONCERN — Thornwall vendors unreachable from Stillwater without gold

The bridge toll (5g) at Watchers Crossing (room 421/422) blocks a zero-gold character
from reaching Thornwall. The test character smoketester starts with 0 carried gold
(bank: 90g but no bank access on the road). Future test sessions needing Thornwall
vendor data should either:
- Give the test character carried gold (at least 10g for toll + margin)
- Or use a separate test character that starts in Thornwall

### CONCERN — Combat on the road (Bandit Lookout encounter)

While navigating south toward Thornwall via North Road, smoketester was attacked by a
Bandit Lookout at the Crossroads Village Square (approx. room 4042). The character was
taken to ~40% HP before the bandit fled. This was recoverable (full regen in ~5 min)
but cost significant time and is a navigation hazard for unarmored test characters.
smoketester has no armor equipped. If future test plans involve road travel, equipping
the test character with basic armor would be prudent.

### OBSERVATION — Dynamic pricing confirmed working

Iron ingot price rose from 5 to 9 and leather strip from 19 to 33 as stock depleted
at Brindle's Smithy. This is the dynamic economy working as designed (scarcity drives
price up toward the ceiling). Not a bug.

---

## Raw Stats

- Commands sent: ~65
- Vendor list captures: 6 (3 timepoints × 2 vendors — Kerra/Voss not reached)
- Combat encounters: 2 (Bandit Lookout on road — survived; Marsh Rat — killed; Dragonfly
  Swarm — killed)
- Salvage attempts: 2 (both PASS — no kit required, no station required)
- Bugs / Concerns / Observations: 0 / 2 / 4
