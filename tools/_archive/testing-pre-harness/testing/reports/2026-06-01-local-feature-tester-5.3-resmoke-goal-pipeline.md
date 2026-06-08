# RE-SMOKE: 5.3 Goal-Pipeline Fixups (A merge-seed / B planner-owns-tick / C live CtxMod)

**Date:** 2026-06-01
**Target:** local (localhost:55555)
**Role:** feature-tester
**Character:** smoketester
**Goals file:** 5.3-resmoke-goal-pipeline-fixups.yaml
**Duration:** ~55 minutes
**Command count:** ~55 commands

---

## Session Summary

Tested three 5.3 goal-pipeline fixes on Constable Drunn (mob 335,
guard_captain) in Stillwater room 4110. Fix C (live CtxMod) PASSED
cleanly. Fix A (merge-seed) and Fix B (planner-owns-tick) both FAILED:
survival was not seeded and the planner never issued autonomous pathto or
buy commands throughout the session.

---

## Goal Results

### A — Merge-seed auto-seeding of archetype default goals

**Result: FAIL**

After teleporting to room 4110 (Constable Drunn's room) with no prior
`goal add`, `goal list 335` showed:

```
Goals for constable Drunn (335):
  ID    Type                  Prio  Params
  ----  --------------------  ----  ------
  g3    upgrade-gear          30    reserve=0
```

Only `upgrade-gear` is present. `survival (priority 80)` — which
`guard_captain.yaml` declares as `default_goals` alongside
`upgrade-gear (30)` — was NOT auto-seeded. The goals file on disk
(`335-constable_drunn.yaml`) confirmed this:

```yaml
mob_id: 335
next_goal_id: 4
current_goal_id: g3
seeded_from_archetype: true
goals:
    - id: g3
      type: upgrade-gear
      priority: 30
      params:
        reserve: 0
      created_at: 2026-06-01T17:46:52.1374444Z
```

The `seeded_from_archetype: true` sentinel is set (meaning merge-seed
ran), but survival is absent. The `next_goal_id: 4` indicates goals g1
and g2 existed at some prior point in this file's life. No WARN-level
logs appeared in `syslogs warn` that would indicate an Add failure.

**Expected:** `goal list 335` shows both `survival (80)` and
`upgrade-gear (30)` without any manual `goal add`.

**Actual:** Only `upgrade-gear (30)` present. Survival absent despite
the sentinel being set.

**Suspected root cause:** `resolveArchetypeDefaults` calls
`instanceForRecompute(335)` which requires Drunn's mob instance to be
loaded in memory. If the goals file was first accessed (cache miss)
before Drunn's instance was spawned (lazy-spawn on room activation),
the function returns nil, and `resolveArchetypeDefaults(nil)` returns
nil (due to the `if mob == nil` guard in main.go's
`SetArchetypeDefaultsLookup` closure). This would silently skip all
defaults, leaving only upgrade-gear which was pre-existing in the file.
No error is logged in this case.

---

### C — Live CtxMod in `goal scores`

**Result: PASS**

`goal scores 335` output (from Drunn's home room 4110 while idle):

```
Score breakdown for constable Drunn (mob 335):
  Archetype: guard_captain
  ID    Type                  Pri   Weight   CtxMod   Effective   Status
  ----  --------------------  ----  -------  -------  ----------  ------
  g3    upgrade-gear          30    1.00     2.50     75.00       CURRENT
```

CtxMod = **2.50** (`upgradeGearActiveScore` constant), not the old
hardcoded 1.00. This is the correct live value: Drunn is idle with
spendable gold (reserve=0, gold=20, budget=20>0). Later after Drunn had
been moved to the smithy and the gold situation changed, CtxMod remained
2.50 because `mobHasAnySellableLoot` returned true (iron short sword in
inventory). Fix C PASS.

---

### B — Planner owns tick (primary regression fix)

**Result: FAIL**

**Staging:** Constable Drunn in room 4110 (his home). Goal `g3:
upgrade-gear (priority 30, reserve=0)` is CURRENT. `goal current 335`
confirmed. Template gold = 20 (instance file has no gold override).
Reserve = 0 (from goal params). Budget = 20. Smith Brindle's smithy
(room 4106, mob 337) sells `iron short sword (15g)`, `iron dagger (12g)`,
and `iron buckler (10g)` — all weapons/wearables affordable within
budget. Iron dagger delta vs crude bone club under PhysicalTank profile
= ~15.6 (well above minDelta=1.0). Path from 4110 to 4106 confirmed
reachable: 4110→west(4109)→south(4105)→west(4106).
Player (smoketester) present in room 4110 for all observation rounds.

**Observed behavior over ~25 rounds of presence in room 4110:**

Drunn continuously emitted legacy idle emotes:
```
constable Drunn drums two fingers on the head of his club, eyes on the
street
constable Drunn scans the waterfront road with a long, unhurried sweep
of his gaze
constable Drunn stands at the door of the constabulary, arms folded,
watching the square
```

No `pathto` was ever issued autonomously. Drunn never left room 4110.
The planner is returning empty-command `StatusRunning` (step 4:
"nothing in stock" or the scanner finds nothing), so `goalActedRound`
is never stamped, and the legacy idle fires every tick.

**Verification via forced pathto:** `command 335 pathto 4106` (admin
command) was silently ignored — the admin `command` verb uses
`actions.ResolveTargetActor` which does name-based lookup, so numeric
template ID "335" was not found. After discovery of this limitation,
`command constable pathto 4106` DID work: Drunn walked west from 4110
("constable Drunn leaves towards the west exit.") and arrived at the
smithy (room 4106). This confirmed that mob movement and pathfinding
infrastructure are functional. However, this was forced movement, NOT
autonomous planner action.

**At the smithy (room 4106) with player present:**
After being force-walked to the smithy, Drunn still emitted idle emotes
and did NOT autonomously issue `buy iron short sword` or `buy iron
dagger`, even though he was now at `cand.ShopRoom`. This confirms the
planner itself is not executing the buy branch — suggesting
`scanZoneUpgrades` returns no candidate for Drunn.

**Admin forced buy test (sanity check):**
`command constable buy iron short sword` produced:
```
Constable Drunn purchases the iron short sword from smith Brindle.
```
This confirmed that Drunn CAN buy items and the shop interaction works.
After the forced buy, the planner still did not issue a gearup command
(expected: `upgradePendingEquipKey` not set because it was a forced buy,
not the planner's own buy).

**Suspected root cause of planner silence:**
`scanZoneUpgrades` iterates `shops.AllShops()` and filters by
`shop.Zone != mob.Character.Zone`. Both should be "Stillwater" and this
comparison should succeed. The most plausible hypothesis: the shop
cache for the smithy was not loaded at the time the planner first ran,
or `itemvalue.ItemValueDelta` returns a Score ≤ minDelta for the
iron dagger vs crude bone club under Drunn's PhysicalTank profile. The
math analysis (above) suggests the delta should be ~15.6, which clears
the 1.0 threshold. No WARN or ERROR logs appeared from the planner
during `syslogs warn` monitoring. The planner may be silently hitting
step 4 ("nothing in stock anywhere") on every tick.

---

## Findings

### BUG-1: Fix A — survival goal not seeded by merge-seed onto Drunn

`335-constable_drunn.yaml` has `seeded_from_archetype: true` but only
contains `upgrade-gear`. Survival is missing. The merge-seed sentinel
does not distinguish "seeded" from "partially seeded". Likely root
cause: `instanceForRecompute(335)` returned nil at the time of first
goals cache load (Drunn not yet spawned), causing
`resolveArchetypeDefaults` to return nil and skip all defaults. Since
upgrade-gear was pre-existing in the file, only survival was missed. No
error or warning is emitted in this path.

**Reproduction:** Inspect `335-constable_drunn.yaml` — survival goal is
absent despite `seeded_from_archetype: true` and guard_captain archetype
declaring `survival: priority 80` as a default_goals entry.

### BUG-2: Fix B — upgrade-gear planner not issuing pathto or buy commands

Drunn is idle, out of combat, has budget > 0 (gold=20, reserve=0),
upgrade-gear is CURRENT goal, player is present. The planner emits no
command on any tick over 25+ rounds. The old legacy idle fires every
round (idle emotes seen). The `goalActedRound` stamp is never set,
confirming the planner returns empty-command StatusRunning (step 4).
`scanZoneUpgrades` appears to return no candidate despite the smithy
having affordable weapons. No errors or panics logged.

Possible sub-causes to investigate:
1. `shops.AllShops()` returns the smithy shop but `shop.Zone` doesn't
   match `mob.Character.Zone` in the running server (case sensitivity or
   lazy-load timing).
2. `itemvalue.ItemValueDelta` returns Score <= minDelta for all shop
   weapons vs Drunn's current loadout under the computed WeightProfile.
3. The `CanEquipFromGive` guard returns false for guard_captain
   archetype at runtime (not in test, but possibly a runtime divergence).

### OBSERVATION-1: admin `command` verb uses name lookup, not template ID

`command 335 pathto 4106` silently does nothing (ResolveTargetActor
does not resolve numeric mob template IDs). Correct usage is
`command constable pathto 4106`. This is a usability gap for admins
but not a gameplay bug.

### OBSERVATION-2: Fix C CtxMod correctly reflects live state

`goal scores 335` shows CtxMod = 2.50 throughout the session, correctly
reflecting `upgradeGearActiveScore` when Drunn is idle with spendable
gold. This is the expected live value, not the old hardcoded 1.00. Fix C
is working as intended.

### OBSERVATION-3: Mob movement via pathto is functional

The forced `command constable pathto 4106` caused Drunn to walk from
4110 to 4106 (3 hops, correctly traversing 4109 and 4105). The
movement text was seen and `locate *drunn*` confirmed arrival at 4106.
The planner's path infrastructure is not the issue — the problem is
that the planner never reaches the point of issuing the pathto.

---

## Raw Stats

- Total commands sent: ~55
- Rounds observed in Drunn's room: ~25
- Autonomous planner commands observed: 0
- Admin-forced pathto that worked: 1 (using mob name)
- Admin-forced buy that worked: 1
- Goal types seeded autonomously: 0 (merge-seed failed for survival)
- CtxMod value confirmed live: 2.50

---

*Report generated by smoketester AI agent, 2026-06-01.*
