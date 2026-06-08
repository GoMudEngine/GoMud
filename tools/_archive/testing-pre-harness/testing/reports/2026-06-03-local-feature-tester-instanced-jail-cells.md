# Feature Test: Per-Prisoner Instanced Jail Cells

**Date:** 2026-06-03  
**Target:** local (localhost:55555)  
**Role:** feature-tester  
**Character:** smoketester (admin)  
**Goals file:** instanced-jail-cells.yaml  
**Duration:** ~90 minutes  
**Commands issued:** ~200  

---

## Session Summary

Tested the per-prisoner instanced jail cell system (zone: "Instance Jail Cell",
template room 5107). Successfully triggered multiple arrests across three server
sessions. The critical finding: **instance cell creation silently fails on every
arrest attempt**, and the arrest system silently falls back to the static Thornwall
holding cell (room 5105) every time. The instanced cell path never activated. All
other mechanics — arrest declaration, fine decay, payfine release, sentence-expiry
release, and logout/login sentence clock persistence — behaved correctly.

---

## Goal Results

### Goal 1 — Reach a guarded town, confirm arrival: PASS

Arrived at Thornwall Gate Ward (room 460) via admin `teleport 460`. Guard Captain
Velk present in barracks (room 473). City guard patrols active via
`thornwall_market_beat` patrol.

```
teleport 460
Moved to room 460.
.: [*] Gate Ward [Thornwall City]
...
Also here: City Guard (100%)
Exits: east, north, west
```

### Goal 2 — Trigger an arrest: PASS (with caveats)

Successfully triggered three separate arrests. Method: spawned city beggar victims
(`mob spawn city beggar`) in Thornwall rooms while a guard was present, attacked
them with `attack city`, which recorded assault crimes against `thornwall_citizens`.
Guard ran `RunGuardEnforcement` and issued `arrestOutcomeDeclare` then
`arrestOutcomeHaul`.

**Key blocker encountered:** City guards with `tank_taunter` behavior archetype
entered combat immediately when I had hostile rep, preventing the enforcement loop
(`RunGuardEnforcement` gates on `!mob.Character.IsInCombat()`). Workaround:
Guard Captain Velk (behavior_archetype: `guard_captain`) did NOT auto-enter combat,
allowing the arrest path to complete cleanly.

Arrest message observed three times:
```
guard captain Velk says, "Move along is past — you're under arrest. Come quietly."
...
A guard seizes you and hauls you to the holding cell. You have been placed under
arrest by the Thornwall Guards.
```

### Goal 3 — Verify the cell is a PRIVATE INSTANCE: FAIL (Critical Bug)

All three arrests placed me in **static room 5105** (Thornwall City zone), NOT an
instanced ephemeral room. This is a confirmed silent failure of instance creation.

**Room after each arrest:**
```
room info
RoomId: 5105 (Zone root is 460)
Filepath: thornwall_city\5105.yaml
Zone: Thornwall City
Title: Holding Cell
Description: A cramped cell carved into the stone foundations beneath the guard
  barracks, reached by a short flight of worn steps. Iron bars set deep into the
  rock form the front wall, their lock heavy and well-oiled...
Exits: [up => 473]
```

**Expected (instanced):** Room ID > 1,000,000, Zone: "Instance Jail Cell",
description should contain "The seal of the Thornwall Guards is stamped into the
iron."

**Character file inspection** confirms `jail_instance_id` key is ABSENT from
`miscdata` — set to 0 by `ExecuteArrest` (indicating `aCreateCellFn` returned
`(0, false)`). The `keyJailInstanceId` with value 0 is omitted from YAML
serialization.

**Exits test:** The static cell has `up => 473` listed but blocked by Jailed buff.
All cardinal directions return "You can't do that!" as expected, but there IS one
exit (up to barracks). This is the static cell behavior; an instanced cell should
have NO exits at all (zone-config has SuppressReturnPortal + no hard exits in
5107.yaml).

**Recall test:** `recall` returned "Recall not recognized" (command not available),
so recall blocking can't be confirmed in isolation, but the zone-config has
`allow_recall: false`.

**Faction description:** The description shown is the generic static cell
description — the instanced cell's faction-stamped description was never generated.

### Goal 4 — Check fine + release via payfine: PASS

`fine` command correctly shows outstanding fine with decay message.

```
fine
Your fine to walk free now is 25 gold. It drops the longer you sit.
Pay it with payfine.
```

`payfine` successfully released me to the correct release room (473, Guard
Barracks):

```
payfine
You count out 15 gold and settle your fine with the guards.
The cell door swings open. You are free to go.
```

Room after release:
```
.: [*] Guard Barracks [Thornwall City]
...
Exits: down (locked), south, up
```

Movement confirmed working after release (walked south to Gate Ward).

### Goal 5 — Logout/login resume: PARTIAL

Tested twice via server restart and manual `quit`.

**Test A — Server restart during sentence (unintended):** Server went down while
I was in the static cell (controller-initiated restart). On reconnect,
`RestoreJailOnLogin` ran. `jail_until_round: 1384192`, server restart round:
~1384223. Since sentence had elapsed, `ResolveDetention` was called and I was
released to the barracks (room 473) correctly. The "sentence elapsed while offline
→ release on login" path works.

**Test B — Manual quit mid-sentence:** Quit while jailed (third arrest, static
cell). After logout + re-login: `RestoreJailOnLogin` fired. With the sentence
barely elapsed, I was placed at the release room (473) with the Jailed buff still
active (buff timer not yet zeroed). Fine command immediately after login said
"You aren't being held anywhere" (jail record cleared), but the Jailed buff took
~72 rounds to expire naturally. Movement blocked until buff expired.

**Notable:** The "fresh private cell on login when sentence still running" path was
NEVER tested because instance creation always fails. The bug that prevents instanced
cells also prevents proper logout/login resume for the instanced-cell path.

**BUG (see below):** After `ResolveDetention`, the Jailed buff (88) persists until
its natural timer expires rather than being immediately cleared. This creates a
window where `fine` says "not held" but `conditions` still shows "Jailed" and
movement is blocked.

### Goal 6 — Stability (no panics, no TTL eviction): PASS

No panics observed. No "portal collapsing" or instance eviction messages appeared
in any session. The server handled three arrests, multiple server restarts, and
various admin commands cleanly.

Static holding cell (5105) never auto-evicted. No orphaned instances (since none
were created).

---

## Findings

### BUG-1 (Critical): Instance jail cell creation silently fails on every arrest

**Severity:** Critical — the core feature under test never functions.

**Evidence:**
- Three separate arrests all landed in static room 5105.
- Character file shows `jail_instance_id` absent (value 0, omitted by YAML).
- No WARN or ERROR logged during arrest (even with `syslogs info` enabled).
- The `aCreateCellFn` in `internal/justice/arrest.go:91-104` returns `(0, false)`
  and falls back to `staticCell` silently — no logging on failure path.

**Root cause analysis:**

The `aCreateCellFn` calls `rooms.CreateZoneInstanceWithOpts("Instance Jail Cell",
...)` which calls `rooms.CreateEphemeralZone("Instance Jail Cell")` which does:

```go
func CreateEphemeralZone(zoneName string) (map[int]int, error) {
    roomIds := make([]int, len(roomManager.zones[zoneName].RoomIds))
    for roomId := range roomManager.zones[zoneName].RoomIds {
        roomIds[idx] = roomId
        idx++
    }
    return CreateEphemeralRoomIds(roomIds...)
}
```

`roomManager.zones["Instance Jail Cell"].RoomIds` is populated ONLY when
`addRoomToMemory(room)` is called for room 5107. However, `loadAllRoomZones()`
at server startup only populates `roomIdToFileCache` (the filepath lookup), NOT
`RoomIds`. Room 5107 only enters `RoomIds` when a player visits it (which triggers
`LoadRoom → addRoomToMemory`).

During all three arrest attempts, no player had visited room 5107 beforehand, so
`RoomIds` was empty, causing `CreateEphemeralRoomIds` to receive an empty slice
and return `errNoRoomIdsProvided`.

**Confirmation:** When I manually teleported to room 5107 (loading it into memory)
and then got arrested again, it STILL failed. This suggests either:
1. The room was unloaded from memory by the idle-unload cycle before the arrest
   tick ran (room `lastVisited < unloadRoundThreshold`), or
2. There is another failure mode I could not isolate without server-side logging
   (LogToFile: false in this session).

**Fix direction:** Either (a) pre-load room 5107 at startup by visiting it from a
startup fixture or initialization path, or (b) add a fallback in `CreateEphemeralZone`
that calls `LoadRoomTemplate` for rooms not yet in `RoomIds` but present in
`roomIdToFileCache`, or (c) add error logging in `aCreateCellFn` so the failure is
visible in server logs.

### BUG-2 (Medium): Jailed buff persists after `ResolveDetention` clears the record

**Severity:** Medium — creates player-visible inconsistency.

**Evidence:**
```
fine
You aren't being held anywhere.   <- jail record cleared

conditions
Jailed  (fading)                  <- buff still active, movement blocked
```

After `ResolveDetention` is called (sentence-expiry path), the buff's natural timer
continues to run rather than being immediately removed. `player.RemoveBuff(88)` is
called but the in-game effect persists for some rounds. Root cause unclear — may be
a delayed buff expiry propagation or a stale buff state from `AddBuffScaled` when
the scaled count doesn't align with the current round.

### BUG-3 (Low): No error logging when instance creation fails in ExecuteArrest

**Severity:** Low — operational blind spot.

The `aCreateCellFn` wrapper in `arrest.go` discards the error from
`CreateZoneInstanceWithOpts`. Even with `syslogs info` enabled, no message appears
when the fallback to static cell is used. Operators cannot diagnose why instanced
cells aren't working.

**Fix direction:** Add `mudlog.Warn(...)` in the `else if staticCell != 0` branch
of `ExecuteArrest` and the equivalent branch in `RestoreJailOnLogin`.

### CONCERN-1: Instance cell logout behavior places player in release room, not cell

On logout-while-jailed: `HandleJailedDespawn` should set `player.RoomId = fallback
(5105)`. But the saved character file shows `roomid: 473` (release room), not 5105.
The jail record keys are correct but the room ID doesn't match. This may be a save-
ordering issue where the player's current room is saved before `HandleJailedDespawn`
updates it, or an admin-teleport interaction.

### CONCERN-2: Guard combat prevents arrest enforcement from completing

City guards with `tank_taunter` behavior archetype enter combat via the legacy
hostile-check (go.go line 590) when a player has TierHostile faction rep. Once in
combat, `RunGuardEnforcement` gates on `!mob.Character.IsInCombat()` and stops
running — preventing the `arrestOutcomeHaul` from ever firing. Guard Captain Velk
(guard_captain archetype, no auto-attack player_enter) correctly arrested via the
enforcement path. This may be the intended design for TierHostile players (fight
rather than arrest), but it means a player who has fought guards cannot be arrested
at all by city guards — they can only be arrested by the captain.

### OBSERVATION-1: Static cell correctly restricts movement

Room 5105 correctly blocked movement on all exits while the Jailed buff (88) was
active. The "up" exit exists in the room template but is blocked by the buff's
`no-go` flag. `payfine` released to room 473 correctly.

### OBSERVATION-2: Fine decay works as expected

Fine decremented at 5 gold/round as configured (`JusticeFineDecayPerRound`). A 363
gold fine took ~72 rounds to reach 0, as expected.

### OBSERVATION-3: Sentence-elapsed release on login works correctly

When the sentence elapsed during a server restart (Test A in Goal 5), login
correctly placed me in the release room (473) with no Jailed buff. The
`until <= now` path in `RestoreJailOnLogin` works.

---

## Cell Description (Static Cell — Instance Never Created)

All three arrests used room 5105. The description shown:

```
A cramped cell carved into the stone foundations beneath the guard barracks,
reached by a short flight of worn steps. Iron bars set deep into the rock form
the front wall, their lock heavy and well-oiled. A thin pallet of straw lies in
one corner beside a battered tin bucket, and a high, narrow slit near the ceiling
lets in a grudging bar of daylight. The air is cool and close, smelling of damp
stone and old straw. Footsteps and muffled voices drift down from the duty room
above.
```

The expected instanced cell description (from `factionCellDescription("thornwall_guards")`
in `internal/justice/arrest.go:286-295`) was never observed. Expected description:

```
Four close walls of cold, mortared stone press in around a single iron-strapped
door with no handle on this side. The seal of the Thornwall Guards is stamped
into the iron. There is no way out but the law's mercy and the slow passage of time.
```

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Arrests triggered | 3 |
| Instance cells created | 0 |
| Static cell fallbacks | 3 |
| payfine releases | 1 |
| Sentence-expiry releases | 2 (1 via server restart, 1 via timer) |
| Server restarts during session | 2 |
| Panics observed | 0 |
| Portal evictions observed | 0 |
| Server log entries for instance creation | 0 (LogToFile: false) |
