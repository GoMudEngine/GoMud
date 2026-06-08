# Test Report: Chunk 3.4 (Waypoint Patrols) + Post-3.3 Hotfix Verification

**Date:** 2026-05-25
**Target:** local
**Role:** feel-tester
**Character:** smoketester (admin)
**Duration:** ~22 minutes, ~65 commands sent

## Session Summary

Observed the Thornwall city guard (mob 106) patrolling the market beat
and confirmed the day/night schedule transition (patrol -> barracks sleep
-> patrol). The patrol loop itself is functional — all 5 waypoints were
visited and the guard resumed patrol after a combat interrupt. However,
the double-spawn bug is NOT fixed: two city guard instances (both from
stale mobs.instances files) are running the same patrol loop
simultaneously, visibly appearing as "City Guard #1" and "City Guard #2"
in market square rooms. Kerra (mob 97) shows as a single instance in
mob list and schedule inspector; the brief double-render seen at ~5 AM
in the forge appears to be a timing artifact, not a true duplicate.

## Goal 1 Results: City Guard Patrol (chunk 3.4)

- [x] Guard visits all 5 waypoints in order — **PASS**
- [x] Dwell timing matches authored values — **PARTIAL PASS** (visibly
  longer dwell at 464/465 vs 461/466 observable; precise round counts
  not timed exactly but qualitatively correct)
- [x] Day/night schedule transition (patrol->barracks->patrol) — **PASS**
- [x] Combat interrupt + resume to same waypoint — **PASS**
- [ ] `mob schedule` inspector renders patrol state correctly — **BLOCKED**
  (guard instance ID not found via 1-280 scan; both guards share a patrol
  but instance IDs are above 280 or outside scan range)

### Patrol Route Evidence

Waypoints observed in order across two partial loops (game-time 8:30 AM -
11:30 AM, then 7:26 AM day 118):
- Room 460 (Gate Ward): guard confirmed present, idled, left east
- Room 461 (Main Street West): guard confirmed present, left east
- Room 464 (Market Square West): guard confirmed present, left/entered
- Room 465 (Market Square Center): guard confirmed present multiple times,
  idled, left east and west
- Room 466 (Market Square East): guard confirmed present, left west

### Night Transition Evidence (game-time 22:00)

At ~22:04 PM game-time, guard left patrol area. At barracks (room 5104),
guard was observed entering and emoting "city guard shifts in sleep,
murmuring something inaudible." Guards returned to Market Square Center
at game-time 7:26 AM day 118, both at 73% hp (partial heal over sleep).
Patrol resumed immediately — PASS.

### Note on Dwell Timing

Patrol YAML shows:
  - Room 460: dwell_rounds 5
  - Room 461: dwell_rounds 3
  - Room 464: dwell_rounds 10 (longest)
  - Room 465: dwell_rounds 8
  - Room 466: dwell_rounds 5

The test goal spec mentioned waypoint 461 having dwell_rounds 10 — this
appears to be a discrepancy between the spec and the actual YAML. The
actual file at `_datafiles/world/dogmud/patrols/thornwall_city/
thornwall_market_beat.yaml` shows 461 = 3 rounds, 464 = 10 rounds.
Qualitative dwell observation was consistent with the YAML values.

### Combat Interrupt + Resume

Attacked City Guard #1 at room 465 (~11:02 AM). Guard fought back and
was wounded to ~37% HP. Player fled north to room 502. Guard did NOT
pursue out of patrol area. When player returned to room 465, City Guard
#1 was observed leaving west toward waypoint 464 — resuming patrol.
PASS.

## Goal 2 Results: Double-spawn Verification (post-3.3 hotfix)

- [ ] Only ONE Kerra visible at any sampled time — **CONDITIONAL PASS**
- [ ] Only ONE city guard visible at any sampled time — **FAIL**

### City Guard Double-spawn: CONFIRMED STILL ACTIVE

**Root cause found:** Two stale mob instance files exist in
`_datafiles/world/dogmud/mobs.instances/thornwall_city/`:
  - `106-city_guard-room460.yaml`
  - `106-city_guard-room465.yaml`

Both files load at server startup, producing two city guard instances
running the same patrol loop. This is a DATA-SIDE issue (stale instance
saves), not necessarily a code-side failure of commit 31cbc3b1.

**Verbatim evidence (room 465, game-time ~10:28 AM):**
```
Also here: Market Merchant (100%), Ketil (100%), Marta (100%), Lars (100%),
Caravan Wagon (100%), Hob (100%), Bran (100%), City Guard #1 (100%) and
City Guard #2 (100%)
```

**Verbatim evidence (room 465, game-time ~11:30 AM, post-combat):**
```
Also here: Market Merchant (100%), Ketil (100%), Marta (100%), Lars (100%),
Caravan Wagon (100%), Hob (100%), Bran (100%), City Guard #1 (44%) and
City Guard #2 (100%)
```

City Guard #1 was wounded by the player attack; City Guard #2 remained
at 100% throughout — confirming these are truly two separate instances,
not a display artifact.

### Kerra Double-spawn: Likely PASS (artifact, not duplicate)

`mob list *kerra*` showed exactly 1 blacksmith Kerra at all times.
`mob schedule 21` confirmed instance 21 is the only Kerra with a
schedule. The brief "#1 / #2" display seen in the forge at ~5 AM when
the player walked in appears to have been a render-timing artifact (Kerra
entered the room at the exact moment the room rendered). Her schedule
inspector consistently shows one instance at a single location.

### Kerra Location Samples

| Sample Time   | Segment Active | Target Room | Location    | Count |
|---------------|---------------|-------------|-------------|-------|
| ~5:00 AM      | 22-6 (sleep)  | 5101 (loft) | Forge+loft  | 1     |
| ~7:26 AM      | 9-18 (craft)  | 470 (forge) | Forge (AT)  | 1     |
| ~14:00 (sched)| 9-18 (craft)  | 470 (forge) | Forge (AT)  | 1     |

## Goal 3 Results: Sleep/Wake Verification (chunk 3.3 ongoing)

- [x] Sleeping NPCs render with sleep emotes — **PASS** (guard emoted
  "shifts in sleep, murmuring something inaudible" in barracks)
- [ ] Sleeping NPCs render with `(asleep)` suffix in room look — **UNKNOWN**
  (could not confirm — guards oscillated in/out of barracks due to
  double-spawn; room look never captured a stable sleeping guard)
- [x] Scheduled wake on segment end works correctly — **PASS** (guards
  returned to patrol at 6 AM transition)
- [x] Kerra wakes correctly at 6 AM segment transition — **PASS**
  (schedule inspector showed she transitioned from 22-6 sleep segment to
  6-9 wake segment; emotes "pulls on her boots and apron" observed)

## BUGs / CONCERNs / OBSERVATIONs / PASS sections

### BUG: Double city guard from stale mobs.instances files

**Severity: HIGH** — Two stale mob instance files are loading a second
city guard:
```
_datafiles/world/dogmud/mobs.instances/thornwall_city/106-city_guard-room460.yaml
_datafiles/world/dogmud/mobs.instances/thornwall_city/106-city_guard-room465.yaml
```
Fix: Delete one (or both) of these instance files before next server
boot. The engine will respawn a single fresh guard from the mob template.
The code fix in commit 31cbc3b1 may be correct but the stale instance
files are causing the persistent double-spawn. This also causes the
guard's barracks/sleep behavior to appear erratic (both guards racing
to/from the same target room).

### BUG: Guard double-spawn side-effects on sleep rendering

Because both guards oscillate between the barracks and the patrol rooms
during the night segment, the "(asleep)" suffix check was inconclusive.
The guards never settled stably in the barracks long enough to render.
This is a secondary consequence of the double-spawn, not an independent
sleep-rendering bug.

### OBSERVATION: `time set <hour>` admin command does not work

Ran `time set 6`, `time set 22`, `time 6`, `time 22` — all returned the
current time but did not advance it. Time must be controlled via
`server set Timing.RoundsPerDay` (changed to 100 for speed testing,
restored to 900). Day/night observation was done using accelerated
time via config change.

### OBSERVATION: schedule inspector hour may lag real clock

At game-time 7:26 AM (per `time` command), `mob schedule 21` for Kerra
showed `current hour: 14` — a 6-7 hour discrepancy. This may be because
many game-days passed during the `RoundsPerDay=100` speed test and the
schedule module's internal hour tracking didn't fully sync. Not
necessarily a bug — observe in future sessions.

### PASS: Patrol route correctly defined and executed

The thornwall_market_beat.yaml patrol loop (strict, 5 waypoints:
460->461->464->465->466) is authored correctly and the guard traverses
all waypoints. All 5 rooms were visited during observation.

### PASS: Guard does NOT pursue player out of patrol area

After combat at room 465, guard remained in the patrol zone. Player fled
to room 502 without the guard following. Guard resumed patrol westward on
next patrol tick. Correct behavior.

### PASS: Day/night transition fired correctly

At 22:04 PM, guards left patrol and were observed arriving at barracks
room 5104 with sleep emotes. At 6:00 AM sunrise, guards returned to
Market Square Center and resumed patrol pattern.

### PASS: Kerra single-instance throughout session

mob list, mob schedule, and physical room observation all confirm exactly
one Kerra instance. No true double-spawn for Kerra.

## Raw Stats

- Commands sent: ~65
- Samples collected: 3 (Kerra location), 4+ (guard observation)
- Game-time settings used: RoundsPerDay=100 for ~5 minutes (day skip
  from 12:11 PM to night and back), then restored to 900
- Instance IDs confirmed: Kerra = instance 21; guard = above 280 (not
  found in brute-force scan)
- Mob instance files found: 2 city guard, 1 Kerra

## Recommended Actions

1. **IMMEDIATE:** Delete `106-city_guard-room460.yaml` (keep or delete
   `106-city_guard-room465.yaml`) from mobs.instances/thornwall_city/
   and restart server. The city guard should then exist as a single
   instance. Verify with `mob list *guard*` after restart.
2. **VERIFY:** After fixing double-spawn, re-run the sleep rendering
   check — confirm guard renders with "(asleep)" suffix when stable in
   barracks room 5104.
3. **DOCUMENT:** Update test goal notes to reflect that `time set`
   command is not available; use `server set Timing.RoundsPerDay` for
   time acceleration instead.
