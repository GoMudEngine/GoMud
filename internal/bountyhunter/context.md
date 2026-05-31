# Bounty Hunter Package Context

## Overview

`internal/bountyhunter` is the bounty-hunting dispatch substrate (5.2).
It owns two responsibilities:

1. **Dispatch sweep** (`dispatch.go`): a per-round scan that decides when
   to spawn a new hunter for a wanted player and manages the hunt lifecycle
   (reap resolved/dead hunts, enforce redispatch cooldown).

2. **Hunter spawn + scaling** (`bountyhunter.go`): creates a statpool-scaled,
   affix-geared hunter instance at the issuer faction's station, stamps the
   per-instance pursuit target into `MiscData`, and adds the
   `hunt_bounty_target` goal so the planner fires.

**What this package does NOT cover:** pursuit decisions. Those live in
`internal/planners/hunt_bounty_target.go` (the goal planner) and the
`bounty_hunter` behavior archetype (`try_goal_planner` drives it).

---

## Key Files

| File | Role |
|------|------|
| `bountyhunter.go` | Pure scaling helpers (`scaledStatpool`, `gearGold`) + `spawnHunter` |
| `dispatch.go` | `RunDispatchSweep` + active-hunt registry + `despawnHunter` |
| `bountyhunter_test.go` | Unit tests: statpool scaling, gear-gold formula |
| `dispatch_test.go` | Unit tests: `shouldDispatch` decision table |

---

## Dispatch Trigger

`RunDispatchSweep(nowRound uint64)` is called once per round from
`internal/hooks/NewRound_MobRoundTick.go` (before the per-mob loop).

Spawn condition (all must hold):
- At least one open, faction-issued player bounty with
  `GoldReward >= BountyHunterGoldThreshold`.
- No hunter is already pursuing this target (`activeHunts[uid]` is nil).
- Enough rounds have elapsed since the last hunter was killed
  (`nowRound - lastKilledFor[uid] >= BountyHunterRedispatchCooldown`).

One hunter per player — the highest-gold faction bounty wins when a
player has multiple open faction bounties.

---

## Hunter Spawn (`spawnHunter`)

1. **Seat** — the spawn room is the issuer faction's `release_room` field
   (the same room used as the arrest release point). Factions without a
   `release_room` are skipped (`spawnHunter` returns 0).

2. **Statpool scaling** — `clamp(round(base + gold × perGold), min, max)`.
   At default knobs: `clamp(round(250 + gold×0.25), 300, 500)`.

3. **Mob template** — template id `110` (the `bounty_hunter` archetype).
   `mobs.NewMobByIdFresh(110, seat, statpool)` overrides the template's
   nominal statpool with the computed value.

4. **Faction tag** — the issuer faction is appended to `hunter.Groups`.
   This makes `factions.FactionsForMob` include the issuer, so the
   `PlayerDeath_BountyResolve` hook's `killGuard` path credits the kill
   and calls `justice.ClearFactionRecord`.

5. **Affix gear** — mirrors the instance-loot path in `rooms.go`. For
   each item in `hunter.LootPool`, `items.GenerateAffixedItem(baseId,
   gearGold, lootBudgetScalar)` is called and the result worn immediately.
   `gearGold = statpool / BountyHunterGearGoldDivisor`.

6. **Target stamp** — `hunter.Character.SetMiscData("bh_target_user_id",
   targetUserId)` and `"bh_bounty_id"`. Per-hunter target lives on the
   instance, not in goal params, because the goals store is template-keyed
   (all hunter instances share template 110's ONE goal record). The goal
   is a param-less intent marker; the planner reads `bh_target_user_id`
   from instance MiscData.

7. **Goal** — `goals.Add(mobId, namesimple, &goals.Goal{Type:
   "hunt_bounty_target", Priority: 100})` so `try_goal_planner` selects
   and fires the planner. `ConflictError` is expected and harmless when a
   second hunter is spawned while the first is still active.

8. **World placement** — `rooms.LoadRoom(seat).AddMob(hunter.InstanceId)`.

9. **Telegraph** — online target receives a `CategorySystem` message:
   "Word reaches you that a hunter has taken the contract on your head."

---

## Pursuit (Goal + Planner)

The `hunt_bounty_target` goal type lives in
`internal/goals/catalog/hunt_bounty_target.go`. Its predicate returns
`true` (goal is still active) while the triggering bounty is open.

The planner (`internal/planners/hunt_bounty_target.go`) runs each tick
the hunter is idle. Decision logic (`huntDecision`, pure + unit-testable):

| Condition | Command emitted |
|-----------|-----------------|
| Target is jailed (`jail_until_round` in MiscData) | `""` (hold; never pursue into a cell) |
| Hunter is in the same room as target | `"attack @<uid>"` |
| Otherwise | `"pathto <targetRoom>"` |

Target offline → hold (`StatusRunning`). The dispatch sweep will detect
the bounty closed and despawn the hunter.

---

## Claim and Record Clear (Death pays the debt)

When a wanted player dies and the killing mob belongs to the issuer
faction (via the dynamically-appended `Groups` entry), the existing
`PlayerDeath_BountyResolve` hook's `killGuard` branch:

1. Calls `bounties.TryClaim(bountyId, MobSubject(killerMobId))`.
2. Transfers `GoldReward` to the hunter.
3. Calls `justice.ClearFactionRecord(issuerFaction, userId)` — resolves
   the player's crimes, withdraws the faction's open bounties, and resets
   rep to the floor. Identical clearing to serving a jail sentence.

The dispatch sweep detects the bounty is no longer open at the next round
and removes the hunt from `activeHunts` without a separate signal.

---

## Hunt Lifecycle Registry

Two package-level maps (in-memory, not persisted):

- `activeHunts map[int]*hunt` — keyed by target userId; one entry per
  active pursuit. Each entry holds `hunterInstanceId`, `bountyId`, and
  `issuerFaction`.

- `lastKilledFor map[int]uint64` — records the round a hunter for a given
  target was last killed, enforcing the redispatch cooldown.

**Reap (Phase 1 of sweep):**
- Bounty closed/claimed/expired → `despawnHunter` + forget the hunt.
- Hunter instance gone (hunter killed) → record cooldown round, forget the
  hunt (the player gets a reprieve).

---

## Jailed-Target Safety

Two layers prevent a hunter from entering a jail cell:

1. **Planner hold.** When `u.Character.GetMiscData("jail_until_round") !=
   nil`, `huntDecision` returns an empty command (`StatusRunning`). The
   hunter loiters at its current room.

2. **No-aggro-target net.** Buff 88 (Jailed) carries the `no-aggro-target`
   flag. The `NewRound_DoCombat` round-tick drops any mob's stale aggro on
   a jailed player — so even if the hunter somehow entered the cell, it
   would not re-engage.

If the player pays their fine or serves their sentence,
`justice.ResolveDetention` calls `ClearFactionRecord`, which withdraws the
bounty. The dispatch sweep then calls off the hunt at the next round.

---

## Config Knobs (Balance)

| Knob | Default | Purpose |
|------|---------|---------|
| `BountyHunterGoldThreshold` | 500 | Minimum faction-bounty gold to trigger a hunter |
| `BountyHunterBaseStatpool` | 250 | Base of the scaled statpool formula |
| `BountyHunterStatpoolPerGold` | 0.25 | Statpool added per gold of triggering bounty |
| `BountyHunterMinStatpool` | 300 | Clamp floor for hunter statpool |
| `BountyHunterMaxStatpool` | 500 | Clamp ceiling for hunter statpool |
| `BountyHunterRepathRounds` | 5 | Rounds between pursuit re-paths (planner interval) |
| `BountyHunterRedispatchCooldown` | 500 | Rounds before re-dispatch after a hunter dies |
| `BountyHunterGearGoldDivisor` | 5 | `gearGold = statpool / divisor` fed to `GenerateAffixedItem` |

---

## Architectural Note: Template-Keyed Goals vs. Instance MiscData

The goals store (`internal/goals`) is keyed by mob **template** id, not
instance id. All instances of template 110 share one goal record. Storing
`target_user_id` in goal params would make every concurrent hunter pursue
the same target.

The design sidesteps this via instance MiscData: `bh_target_user_id` is
stamped on each individual hunter instance, and the planner reads it from
the live mob rather than from goal params. The goal itself is param-less
and acts only as a template-level intent signal that activates
`try_goal_planner`.

If more instance-divergent goal-param use cases accumulate, investing in
instance-keyed goal state would be the right path.

---

## Integration Notes

| Caller / site | What it calls |
|---------------|---------------|
| `internal/hooks/NewRound_MobRoundTick.go` | `RunDispatchSweep(nowRound)` — once per round |
| `internal/hooks/PlayerDeath_BountyResolve.go` | Hunter kill → `justice.ClearFactionRecord` (via `killGuard` branch, not a `bountyhunter` call) |
| `internal/planners/hunt_bounty_target.go` | Reads `bh_target_user_id` from instance MiscData |
| `internal/goals/catalog/hunt_bounty_target.go` | Registers the goal type predicate |

`bountyhunter` imports: `bounties`, `configs`, `factions`, `goals`,
`items`, `knowledge`, `messaging`, `mobs`, `rooms`, `users`, `util`.
It does not import `internal/justice` or `internal/actions` (no import
cycle from those directions).
