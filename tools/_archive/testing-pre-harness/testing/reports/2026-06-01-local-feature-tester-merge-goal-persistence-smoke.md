# Smoke Report: 5.3 Goal-Pipeline Fixups + Mob Instance Goal Persistence Merge

- **Date:** 2026-06-01
- **Target:** local (localhost:55555)
- **Role:** feature-tester
- **Character:** smoketester (admin, AI)
- **Duration:** ~22 minutes
- **Commands sent:** ~55

---

## Session Summary

Merged `master` was tested against the two linked features: 5.3 goal-pipeline
fixups (named NPCs with per-mob behavior trees now reach the goal planner) and
mob instance goal-progress persistence (gold, equipment, planner state survive
performance despawns). The server started cleanly, rooms/mobs rendered without
errors, and the goal pipeline showed full stack operation: seeding, archetype
weight lookup, context modifiers, and selection. Several subsystems
(Town Justice, bounty hunting, schedules, caravan NPCs) all showed continued
correct operation. The persistence end-to-end verification (mob saves up ->
despawns -> respawns geared) was not observable as expected per the gate note.

---

## Goal Results

### A: Server Health / No Merge Regression

**PASS**

- Connected cleanly. Server started on freshly-merged `master`.
- `look` rendered rooms correctly with NPC presence, ASCII map tiles, exits.
- Moved through Stillwater, North Road, Thornwall Outskirts, Thornwall City,
  Thornwall barracks holding cell, and back without crashes.
- No error messages, no crash dumps, no log errors. Server persisted for the
  full session.
- Caravan members (Ketil, Lars, Hob, Bran, Marta) observed moving on road.
- Scheduled NPCs (Constable Drunn) observed walking room-to-room multiple times.
- NPC idle emotes firing (Smith Brindle, multiple others).
- World saving triggered mid-session ("Saving users/rooms/other...Done.")
  without error.

### B: Goal Pipeline Live

**PASS — Strong confirmation of full pipeline operation.**

All three admin `goal` subcommands (`list`, `current`, `scores`) verified
functional. Key results:

**Smith Brindle (mob 337, noncombat_shopkeeper)**
```
goal scores 337:
  Archetype: noncombat_shopkeeper
  ID    Type          Pri   Weight   CtxMod   Effective   Status
  g1    upgrade-gear  30    1.00     2.50     75.00       CURRENT
```
Non-trivial CtxMod (2.50) confirms context evaluation is running, not defaulting
to 1.0. Effective = 75.00 = 30 * 1.00 * 2.50. Goal CURRENT marker resolves.

**Constable Drunn (mob 335, guard_captain)**
Queried remotely (mob not loaded): archetype showed `(none)`, survival=0.00.
This is expected behavior -- the score command falls back to safe defaults when
no live mob instance exists. Re-queried from Drunn's room (4110):
```
goal scores 335 (in-room):
  Archetype: guard_captain
  g4    survival      80    1.00     0.00     0.00        candidate
  g3    upgrade-gear  30    1.00     2.50     75.00       CURRENT
```
Archetype correctly resolves when mob is loaded. Survival scores 0.00 (full HP
-- correct). Upgrade-gear CURRENT at 75.00.

**Bounty Hunter (mob 110, bounty_hunter)**
```
goal scores 110:
  Archetype: bounty_hunter
  ID    Type                Pri   Weight   CtxMod     Effective    Status
  g1    hunt_bounty_target  100   5.00     100.00     50000.00     CURRENT
```
Weight=5.00 matches `goal_weights: hunt_bounty_target: 5.0` in
`bounty_hunter.yaml`. CtxMod=100.00 indicates active target exists (smoketester
had an active bounty). Effective=50,000 correctly dominates all other goal
candidates. Complete weight lookup + context modifier path verified.

**Guard Captain Velk (mob 94, guard_captain)**
```
goal list 94:
  g1    revenge-mob   75    target_id=17 target_kind=player
```
Revenge goal against smoketester (userId=17) from previous session. Confirms
reactive goal seeding (chunk 4.5) persisted across server restarts. The YAML
backing file (`goals/94-guard_captain_velk.yaml`) had `created_at:
2026-05-30T02:48:20Z` -- seeded days ago, still live.

**Numeric ID and namesimple both work.** `goal list 337` and
`goal list smith_brindle` return identical results.

### C: Mobs Actively Behaving

**PASS**

- Smith Brindle: idle emotes running ("wipes soot from his forearms...",
  "thrusts a blade into the slack tub...", "taps a piece of hot iron on
  the anvil..."). Running continuously throughout session.
- Constable Drunn: observed walking east-to-west and west-to-east on schedule
  twice, including entering Brindle's smithy (cross-room schedule path).
- Caravan NPCs (Ketil, Lars, Hob, Bran, Marta): observed moving north together
  as a group on the North Road during session.
- Storekeeper Wulf: idle greeting emote ("nods in greeting from behind the
  counter").
- Town Justice: Guards in Market Square recognized wanted status, issued arrest
  message, transported to holding cell. No combat required (surrender policy
  active).
- Bounty hunter: spawned, tracked across zone boundaries to Stillwater, engaged
  in grapple combat, then broke off cleanly when grapple resolved.
- World feels alive; no evidence of idle handler deadlock.

### D: Shop Sanity

**PASS**

- Brindle's Smithy `list` returned 11 items including pine pitch (dynamic price:
  16g at session start, 21g later -- stock reduced, price rose via living
  economy). Correct.
- Tinder & Tackle: rendered shop interface.
- Food Vendor in Thornwall: `list` rendered.
- All shop interfaces rendered without errors.

### E: Best-Effort Persistence Observation

**BLOCKED (expected per pre-task note)**

Goal YAML files on disk were observed showing persisted state across server
restarts (e.g., Guard Captain Velk's revenge-mob goal created 2026-05-30 still
active 2026-06-01, Brindle's upgrade-gear goal file timestamp 2026-06-01 with
`current_since_round: 1377189`). However, the full end-to-end loop (mob saves
up -> performance despawn -> respawn geared) was not observed, per the noted
dependency on the unmerged ghost-guards fix. Treated as blocked/deferred per
spec.

---

## Findings

### PASS

- **PASS-1:** `goal list/current/scores` all work by both numeric ID and
  namesimple. No crashes, correct data.
- **PASS-2:** Goal selection pipeline confirms non-trivial CtxMod scores
  (2.50 for upgrade-gear, 100.00 for hunt_bounty_target). Not all-1.0 defaults.
- **PASS-3:** Archetype weight lookup confirmed working: bounty_hunter 5.0x
  weight for hunt_bounty_target correctly applied.
- **PASS-4:** Reactive goal persistence: Velk's revenge-mob goal from days
  earlier survived server restart (goal YAML on disk).
- **PASS-5:** Town Justice system (5.1) operating: guards arrested on sight,
  fine system responsive (`fine` command returned 405g), holding cell used.
- **PASS-6:** Bounty hunter system (5.2) operating: hunt_bounty_target goal with
  Effective=50,000, hunter tracked across zone boundary, grappled, then
  disengaged correctly.
- **PASS-7:** Schedule system: Constable Drunn observed moving room-to-room per
  schedule at least 4 times. Cross-zone pathfinding to Brindle's smithy.
- **PASS-8:** Caravan still running: Lars/Ketil/Hob/Bran/Marta observed
  moving as a group on North Road.
- **PASS-9:** No server crash or disconnect during the full session.
- **PASS-10:** Dynamic pricing in living economy: pine pitch rose from 16g to
  21g during session as stock decreased.

### BUG

- **BUG-1 (pre-existing, known):** `sleep` fails with "Something prevented you
  from sleeping: failed to add buff. target: 'smoketester' buffId: 15."
  Matches known bug [[project_sleep_buffid_leak]] in MEMORY.md. Not new.

### CONCERN

- **CONCERN-1:** Goal YAML files for many mobs have `seeded_from_archetype:
  true` but `goals: []` -- meaning the file was created before the archetype
  had `default_goals`, or the archetype had none, and the system won't re-seed
  because the flag is already set. Mobs affected include: thornwall_highwayman
  (90, behavior_archetype: thief), bandit_caster (285), soren (286), barmaid_dal
  (117, no behavior_archetype). Thieves/bandits are missing survival/wealth-gold
  goals they'd get on a clean first spawn. A one-time goal re-seeding pass
  (clearing empty stale files) may be worth doing before the next prod push to
  ensure all archetype NPCs have their full goal set.

- **CONCERN-2 (minor, not a regression):** `goal scores` when mob is NOT loaded
  in memory shows `Archetype: (none)` and all CtxMods as 0.00. This is not
  wrong (it's a deliberate fall-back per the code at admin.goal.go:329-331) but
  it could mislead an admin who queries a mob in an unloaded room. A note in the
  help output or a `(mob not loaded — scores may be incomplete)` warning would
  help. Low priority.

### OBSERVATION

- **OBS-1:** smoketester character had a pre-existing bounty/wanted status from
  a prior session, which caused immediate guard hostility in Thornwall. This
  triggered active combat, bounty hunter spawn, and eventual arrest -- while
  unintended from a test setup standpoint, it gave excellent coverage of the
  5.1/5.2 systems running alongside the 5.3 goal work.
- **OBS-2:** Guard Captain Velk was killed in the opening Thornwall encounter
  (fell to smoketester's attacks while fleeing). A new Velk will respawn per
  normal cycle; the revenge goal persists on mob 94 template and will be active
  on the next Velk instance.
- **OBS-3:** Goal persistence on disk is clearly working: the goal YAML files
  show state from prior sessions (Velk revenge from 2026-05-30, Drunn
  upgrade-gear from earlier today). The `current_since_round` and
  `last_switch_round` fields are being updated.

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Commands sent | ~55 |
| Rooms visited | ~25 |
| Mobs observed | 20+ |
| Named mobs queried via `goal` | 8 |
| Goals queried | 15+ |
| Goal files on disk | 80+ |
| Server crashes | 0 |
| Disconnects | 0 |
| Errors in mud_log.txt | 0 |
