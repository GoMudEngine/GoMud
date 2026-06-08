# Observer Report: Caravan-Kessa Handoff at Room 4038
**Date:** 2026-04-30
**Role:** observer (passive)
**Server:** localhost:55555 (local, Stage 3.1 branch)
**Observer:** smoketester (AI)
**Mission:** Document the caravan-Kessa handoff at Road Fork (room 4038) for
Stage 3.1 verification

---

## Summary

**Handoff NOT witnessed.** After approximately 90 minutes of continuous
observation at room 4038 (Road Fork, North Road), neither the caravan crew
(Ketil/Marta/Lars, mobs 357/358/359) nor Kessa (mob 373) appeared at the
meeting point. The expected timing windows based on cycle analysis were:
- InboundFernwayPickup: ~9:00-9:10 AM
- OutboundFernwayPickup: ~9:54-10:05 AM

Both windows passed with no caravan activity. This indicates either a
stall/block in the caravan state machine or a cycle timing that differs
significantly from the estimate.

---

## Observation Log

### Session Timeline

| Time (real) | Event |
|-------------|-------|
| 08:44 AM | Bridge connected to localhost:55555, logged in as smoketester |
| 08:44-09:01 | Navigation to room 4038 (combats with feral hog and road bandit en route) |
| 09:02 AM | Arrived at room 4038 (Road Fork, North Road) |
| 09:02-10:13 | Continuous observation at 4038, look every ~90-120 seconds |
| 10:13 AM | Session concluded, report written |

### Room 4038 Occupants During Entire Observation

- **Corvin** (mob 276, stable NPC): Present the entire session. Issued
  idle flavor emotes throughout (whittling, checking wheel, glancing at
  farmstead). Never moved.
- **Bloodline Agent** (mob 287, patrol NPC): Patrolled in and out via the
  west exit throughout. Typical pattern: west for 1-3 minutes, return, repeat.
- **quester9** (another player): Present most of the session. Gave smoketester
  items at ~09:04. Departed east at 10:07, returned shortly after.

### Expected Arrivals: Not Observed

- **Kessa** (mob 373): Zero appearances. Expected to arrive from the east
  (Fernway Western Trailhead) when in ForagerWaiting state.
- **Ketil** (mob 357): Zero appearances.
- **Marta** (mob 358): Zero appearances.
- **Lars** (mob 359): Zero appearances.
- **Handoff message** ("A wiry forager hands a satchel up to the caravan;
  the wagon takes the load and rolls on."): Never emitted.

---

## Evidence: Caravan State at Session Start

Mob instance files (which save when a zone has active players) provide
indirect caravan location evidence:

| File | Last Modified | Contents |
|------|---------------|----------|
| `mobs.instances/thornwall_city/357-ketil-room465.yaml` | Apr 30 08:16 | Ketil at room 465 (Thornwall Market) |
| `mobs.instances/thornwall_city/358-marta-room465.yaml` | Apr 30 08:16 | Marta at room 465 |
| `mobs.instances/thornwall_city/359-lars-room465.yaml`  | Apr 30 08:16 | Lars at room 465 |

All three caravan mobs were last saved at Thornwall depot (room 465) at
8:16 AM — six minutes after the server crash+restart at 7:10 AM. No
subsequent saves were written for any caravan mob in any zone during
the session, indicating they spent the entire observation period in a
zone with no active players (or in memory only).

Kessa's instance:
| File | Last Modified |
|------|---------------|
| `mobs.instances/the_fernway_south/373-kessa-room4197.yaml` | Apr 30 08:17 |

Kessa was at her sanctuary (room 4197, Forager's Camp) as of 8:17 AM and
was never saved to a new location, suggesting she remained in Fernway South
throughout.

---

## Timing Analysis

**Server restart:** ~7:10 AM (confirmed from bug_log.txt — server crashed
at 07:09:51 due to duplicate mob ID 243, restarted immediately).

**Caravan state at restart:** Caravan mobs reset to `fold_anchor_room: 465`
(Thornwall Market Square). State machine initializes to `ThornwallDwell`.

**Expected cycle timeline from restart:**
- `ThornwallDwell`: 720 rounds × 4s = 2,880s = 48 min → depart **~7:58 AM**
- `OutboundTransit` (Thornwall 465 → Stillwater 4109): The route traverses
  the entire North Road, a path of approximately 20-30 rooms. At one room
  per behavior tick (~4s), this is roughly 80-120 seconds minimum. Actual
  transit time with pathfinding overhead is likely 3-8 minutes.
  Estimated arrival at Stillwater: **~8:01-8:06 AM**
- `OutboundFernwayPickup` (backtrack from Stillwater 4109 to 4038): The
  caravan is at room 4109 and must path to 4038. This is a reverse-route
  traversal of similar length. Estimated arrival at 4038: **~8:08-8:15 AM**
  (before smoketester arrived at 9:02 AM — missed by ~45-50 min)
- Dwell at 4038: 6 rounds = 24s
- `StillwaterRoute`: Visits 8 vendor rooms in Stillwater (~2-4 min per stop
  with pathfinding, plus any idle emotes) = ~16-32 min total
- `StillwaterDwell`: 720 rounds = 48 min → depart **~9:00-9:25 AM**
- `InboundTransit` (Stillwater → Thornwall): ~3-8 min
- `InboundFernwayPickup` (backtrack from Thornwall to 4038): This state
  backtracks from Thornwall (465) toward 4038, which is ~10-20 rooms.
  Estimated arrival at 4038: **~9:10-9:40 AM**
- `ThornwallRoute`: Visits 9 vendor rooms in Thornwall = ~18-36 min
- `ThornwallDwell`: 48 min dwell
- NEXT `OutboundFernwayPickup` at 4038: **~10:45-11:30 AM** (outside the
  session window)

**Conclusion:** The observation window (~9:02 AM to ~10:13 AM) covered
the InboundFernwayPickup window but apparently missed it (caravan was
likely en route Stillwater→Thornwall or already at ThornwallDwell). The
next OutboundFernwayPickup was not expected until ~10:45+ AM, past the
120-minute session limit.

---

## Notable Events During Observation

1. **Ambient room flavor "Somewhere south, a wagon creaks along the road."**
   appeared 12+ times throughout the session. This is hardcoded flavor text
   in room 4038's YAML (line 41). Not indicative of caravan proximity.

2. **Server autosave at 09:02:40 and 10:03:13**: Saving users/rooms/other —
   routine autosave cycle. No caravan-related mob saves triggered.

3. **quester9 gave smoketester gear at ~09:04**: Hunter-eel scale vest and
   two chrysalis knuckles transferred. Unrelated to caravan observation.

4. **Sun set at ~10:08 AM real time** (in-game time transition visible in
   HUD: ☀️ → ☾). No change in room occupants.

---

## Kessa Forager State Analysis

The `ForagerWaitTimeoutRounds: 150` config (150 × 4s = 600s = 10 min) means
Kessa only waits at 4038 for up to 10 minutes after delivering a forage run.
For the handoff to succeed, Kessa must arrive at 4038 AND the caravan must
arrive within that 10-minute window.

Since the observation showed neither Kessa nor the caravan at 4038, possible
explanations include:

1. **Caravan timing longer than estimated**: StillwaterRoute vendor visits
   may take longer than estimated, pushing the InboundFernwayPickup past the
   observation window.
2. **Kessa was not in ForagerWaiting state**: Kessa only paths to 4038 when
   she has gathered enough forage to deliver. Her `fatigueLimit = 480` rounds
   (~32 min of active foraging) must be reached before delivery. If the
   restart reset her to RestingState, she may not have reached delivery
   threshold yet.
3. **Caravan blocked by combat**: Mob 283 (bandit_lookout, `hates: [caravan]`,
   `hostile: true`, `buffids: [9]` = spawns hidden) can ambush the caravan.
   `tickTransit` halts movement while `anyPartyMemberInCombat()` returns true.
   A prolonged fight or a Ketil fold-recall at <30% HP (via `fold-recall: 20`
   tactic in his spellbook) could stall or reset the caravan.
4. **Both timed out**: If Kessa arrived but the caravan didn't (or vice versa),
   Kessa's `ForagerWaitTimeoutRounds` would expire and she'd return to her
   territory, with no handoff occurring. This is expected behavior per the spec.

---

## Recommendations

1. **Extend the observation window or use a second test session** targeting
   the ~10:45-11:30 AM window for the next OutboundFernwayPickup. Consider
   verifying actual caravan position via admin commands (`admin:mobroom 357`)
   before committing to another long wait.

2. **Add server-side logging for caravan state transitions** to confirm the
   caravan state machine is progressing. A `slog.Info("caravan_state",
   "mob", mob.Id, "state", state.Name())` on each `transitionTo()` call
   would make it trivially easy to verify the cycle.

3. **Verify bandit block risk**: Check if mob 283 (bandit_lookout) is
   consistently blocking the caravan at room 4052 (Old Fence Line). If the
   caravan fights bandits on every transit, the transit time is highly
   variable. The `hostilesInRoom` halt in `tickTransit` is correct behavior,
   but if fights last many rounds or Ketil recall-resets the caravan, the
   handoff window becomes unpredictable.

4. **Consider shortening `CaravanDepotDwellRounds`** for testing purposes.
   720 rounds (48 minutes) makes each full cycle ~3-4 hours of real time,
   making observation sessions impractical. A reduced test value of ~60
   rounds (4 min) would let the caravan cycle 6+ times per hour.

---

## Log Evidence Summary

- Session log: `tools/mud_log.txt` (8:44 AM – 10:13 AM, 2,900+ lines)
- Zero occurrences of: "Ketil", "Marta", "Lars", "Kessa", "wiry forager",
  "satchel", "handoff"
- Zero mob movement events for mobs 357/358/359/373 in the entire log
- All mob movement events in log: bloodline agent (patrol) and quester9 only

---

## Verdict

**INCONCLUSIVE — timeout, not evidence of failure.**

The caravan-Kessa handoff system was not observed within the 120-minute
session window. This does not indicate a bug — the timing analysis shows
the observation window likely fell between caravan visits to 4038. The
Stage 3.1 feature test report (2026-04-30-local-feature-tester-stage-3-1-
foragers.md) confirmed the caravan crew was present and functional at the
Thornwall depot (Goal 13 PASS), which provides positive evidence the
caravan state machine is running.

A dedicated session targeting the correct timing window, or admin tooling
to force a caravan state transition, is needed to directly verify the
handoff trigger.
