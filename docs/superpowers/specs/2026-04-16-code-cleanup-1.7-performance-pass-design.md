# Code Cleanup 1.7: Performance Pass — Design Spec

## Goal

Reduce wasted CPU on mobs in unpopulated zones, eliminate data races
on global registries, and pick up a cheap allocation win on room
cleanup. No behavior change in active zones — every mob in a room
that currently contains a player behaves identically to today.

Four targeted changes from the 1.x overview, collapsed into one
substage because they share the same rollout and testing surface.

## Scope

**In scope:**

1. Zone-activity tracking index (`zone → player count`) with hooks
   in `Room.AddPlayer` / `Room.RemovePlayer`.
2. Active/idle lane split in `handleMobCombat` AND
   `NewRound_MobRoundTick` (the overview's #1, extended at the
   user's request).
3. `sync.RWMutex` on 6 package-global maps across `internal/mobs`
   and `internal/users`.
4. `PruneVisitors` empty-map fast path.

**Out of scope:**

- `Room.GetMobs` per-tick cache. Decision: the zone split already
  cuts the vast majority of these calls; adding a cache layer for
  the remainder is not worth the invalidation complexity. Revisit
  only if post-deploy profiling shows GetMobs is still hot in
  active zones.
- Per-mob / per-user state mutexes. See "Future work" at the bottom.
- God-function refactor (Stage 1.2). MobRoundTick helper extraction
  falls out of this work as a side effect; the other 7 wait.
- Any behavior change in active zones.

## Decisions Locked During Brainstorming

- **"Active" = zone-level.** A mob is active iff its current room's
  zone contains ≥1 player. Rationale: players expect mobs in the
  zone they're exploring to keep progressing (packs fight each
  other, ambient life continues). Room-level is too tight;
  world-level is a no-op.
- **Idle lane scope = minimal, every round.** Cooldowns, combat
  memory expiry, charm duration, buff ticks (regen + DoT),
  condition ticks. Everything else — combat, progression,
  mutation acquisition, charm-state management, crafting, stat
  revalidation — only runs for active-zone mobs. Rationale: no
  player in zone means no one to benefit from or witness
  progression, and timers still need to tick so a returning player
  sees coherent state.
- **Zone index hooks go in AddPlayer/RemovePlayer**, not MoveToRoom.
  Both methods already deduplicate, so the hook fires exactly once
  per genuine membership change and transparently covers login,
  logout, movement, teleport, death, fold-recall, and instance
  portals without per-caller wiring.

## Architecture

### Zone Activity Tracking

New state in `internal/rooms/`:

```go
var (
    zonePlayerCount   = map[string]int{}
    zonePlayerCountMu sync.RWMutex
)

// ZoneHasPlayers returns true if the zone has ≥1 player.
func ZoneHasPlayers(zone string) bool { ... }

// SnapshotActiveZones returns a set of zones with ≥1 player.
// Intended for loop hot paths; single lookup avoids re-locking.
func SnapshotActiveZones() map[string]bool { ... }

// ResetZonePlayerCount is a test helper.
func ResetZonePlayerCount() { ... }

// VerifyZonePlayerCount rebuilds the map from ground truth and
// returns any drift. Used by tests and the admin diagnostics page.
func VerifyZonePlayerCount() (drift map[string]int) { ... }
```

Hooks inside `Room.AddPlayer` and `Room.RemovePlayer`:

```go
func (r *Room) AddPlayer(userId int) int {
    for _, v := range r.players {
        if v == userId { return len(r.players) }
    }
    r.players = append(r.players, userId)
    incrementZonePlayerCount(r.Zone)
    return len(r.players)
}

func (r *Room) RemovePlayer(userId int) (int, bool) {
    for i, v := range r.players {
        if v == userId {
            r.players = append(r.players[:i], r.players[i+1:]...)
            decrementZonePlayerCount(r.Zone)
            return len(r.players), true
        }
    }
    return len(r.players), false
}
```

Startup: `rebuildZonePlayerCount()` iterates all rooms with players
once, called from `world.Start` after the room manager finishes
loading.

**Coverage verification.** Production call sites for
`AddPlayer`/`RemovePlayer`:

| Caller | Path | Covered |
|--------|------|---------|
| Normal movement (`go`) | MoveToRoom → both | ✅ |
| Fold-recall | MoveToRoom → both | ✅ |
| Admin teleport | MoveToRoom → both | ✅ |
| Suicide/death | MoveToRoom → both | ✅ |
| Instance portals | MoveToRoom → both | ✅ |
| Login | enterWorld → MoveToRoom(isSpawn) → AddPlayer | ✅ |
| Logout | HandleLeave → RemovePlayer | ✅ |
| Character creation | NewCharacter → AddPlayer | ✅ |

No code path mutates `user.Character.RoomId` without also calling
`AddPlayer` or `RemovePlayer` on the relevant rooms. Verified by
grep across all production files.

**Empty Zone string.** `zonePlayerCount[""]` is a valid key if a
room happens to have an empty `Zone` field. This is harmless: the
count simply tracks players in un-zoned rooms as a pseudo-zone.
Existing data has no un-zoned rooms in practice.

### Active/Idle Lane Split

**Split #1 — `handleMobCombat`** (`internal/hooks/NewRound_DoCombat.go:138`)

Add a single guard at the top of the per-mob loop:

```go
activeZones := rooms.SnapshotActiveZones()
for _, mobId := range mobs.GetAllMobInstanceIds() {
    mob := mobs.GetInstance(mobId)
    if mob == nil || mob.Character.Health <= 0 {
        continue
    }
    room := rooms.LoadRoom(mob.Character.RoomId)
    if room == nil || !activeZones[room.Zone] {
        continue
    }
    // ... existing combat logic unchanged
}
```

Combat simply does not run in idle zones. A mob "in combat" when
its last player left the zone sits frozen in combat state until
combat-memory expiry (idle-lane) clears the flag. On re-entry, the
next `handleMobCombat` call picks up where it left off subject to
target still existing.

**Split #2 — `NewRound_MobRoundTick`** (`internal/hooks/NewRound_MobRoundTick.go:88`)

Extract the current ~10 inline operations into named helpers, then
branch on zone activity:

```go
activeZones := rooms.SnapshotActiveZones()
for _, mobId := range mobs.GetAllMobInstanceIds() {
    mob := mobs.GetInstance(mobId)
    if mob == nil { continue }

    room := rooms.LoadRoom(mob.Character.RoomId)
    active := room != nil && activeZones[room.Zone]

    // Idle lane — runs for every live mob every round
    tickMobCooldowns(mob)
    expireMobCombatMemory(mob)
    tickMobCharmDuration(mob)
    tickMobBuffs(mob)
    tickMobConditions(mob)

    if !active { continue }

    // Active-only — skipped in idle zones
    tickMobProneRecovery(mob)
    tickMobMutationAcquisition(mob)
    tickMobCharmState(mob)
    tickMobCrafting(mob)
    revalidateMobStats(mob)
}
```

Each helper takes the body of its current inline block verbatim.
Package-private, same file.

**Why this particular 5/5 split.** Idle-lane operations are
timers — returning players expect them to have been counting.
Active-only operations are either progression (no one in zone to
benefit), state machines coupled to combat, or player-facing work
(crafting).

### Registry Mutexes

Six maps to protect across two packages.

**`internal/mobs/mobs.go`:**

```go
var (
    mobsMu              sync.RWMutex // mob templates
    mobInstancesMu      sync.RWMutex // live instances
    mobsHatePlayersMu   sync.RWMutex
    mobNameCacheMu      sync.RWMutex
    recentlyDiedMu      sync.RWMutex
    // allMobNames guarded by mobsMu (derived from same data)
)
```

**`internal/users/users.go`:**

```go
type ActiveUsers struct {
    mu    sync.RWMutex
    Users map[int]*UserRecord
    // ... existing fields
}
```

**Locking rules:**

1. RLock for reads, Lock for writes. No upgrading inside a
   critical section.
2. Release before re-entering the package. `sync.RWMutex` is
   not re-entrant. Load pointer, release, operate.
3. Return copies of slices for iteration (pattern used already
   by `GetAllMobInstanceIds`). Callers iterate outside the lock.
4. Do not hold a lock across event dispatch, I/O, or any call
   into another package.

**Functions that need wrapping:**

`internal/mobs/mobs.go` reads (RLock): `GetInstance`,
`GetAllMobInstanceIds`, `GetMobSpec`, `GetAllMobInfo`,
`GetAllMobNames`, lookups in `mobNameCache`, `recentlyDied`
reads, `mobsHatePlayers` reads.

`internal/mobs/mobs.go` writes (Lock): template load/reload,
`NewInstance` / spawn paths, `DestroyInstance`, `instanceCounter`
increments, `mobsHatePlayers` writes, `mobNameCache` writes,
`recentlyDied` writes.

`internal/users/users.go` reads (RLock): `GetByUserId`,
`GetActiveUsers`, connection lookups.

`internal/users/users.go` writes (Lock): `LoginUser`, logout
paths, connection association.

**Not in scope here:** protecting `Mob.Character.*` or
`UserRecord.Character.*` state from concurrent mutation. That is
a much larger effort (see "Future work" below). These mutexes
protect only map membership — "does this id still resolve to an
instance?" — not the fields of the instance it points to.

### PruneVisitors Fast Path

`internal/rooms/rooms.go:1931`:

```go
func (r *Room) PruneVisitors() {
    if len(r.visitors) == 0 {
        return
    }
    // ... existing loop unchanged
}
```

That's the complete change.

## Constraints

- **Zero behavior change in active zones.** Every mob in a zone
  with ≥1 player behaves identically — same events, same
  progression, same combat, same buffs.
- **Idle mobs still tick timers.** Cooldowns, buff durations, DoTs,
  charm duration, combat memory expiry all run every round
  regardless of zone activity.
- **No mob is lost.** Idle mobs remain in `mobInstances`, remain in
  their rooms, respond to re-entry correctly on the next tick.
- **Tests pass under `-race`.** All new mutex work verified.
- **No deadlocks.** Fine-grained critical sections only. No
  cross-package lock holds.

## Testing Strategy

### Per-step tests

**PruneVisitors fast path:**
- Unit: `PruneVisitors` on a `visitors` map with `len==0` does not
  allocate and does not mutate the map.

**Mutexes:**
- All existing tests pass under `go test -race ./...`.
- New test: N concurrent `GetInstance` reads racing with
  `NewInstance`/`DestroyInstance` writes, under `-race`. Zero
  violations.
- Manual: `go run . -race`, click through admin dashboard pages
  during combat, verify no race warnings in server log.

**Zone index:**
- Unit: `AddPlayer` increments, `RemovePlayer` decrements.
- Unit: `AddPlayer` on existing member does not double-count.
- Unit: `RemovePlayer` on non-member does not decrement.
- Unit: Cross-zone `MoveToRoom` moves count from old to new.
- Unit: Same-zone `MoveToRoom` leaves count unchanged.
- Unit: Startup `rebuildZonePlayerCount` matches incrementally-
  maintained state exactly.
- Unit: `VerifyZonePlayerCount` returns empty drift map after any
  sequence of adds/removes/moves.
- Edge cases tested: login, logout, death→recovery teleport,
  fold-recall, admin teleport, instance portal entry/exit,
  ephemeral room despawn while player inside.

**Lane split:**
- Unit: each new `tickMobX` helper tested in isolation with
  constructed mob state.
- Integration: 10 mobs in active zone + 10 mobs in idle zone run
  for 100 rounds each. Active-zone mobs show progression diffs.
  Idle-zone mobs show only timer/regen diffs.
- Regression: before/after snapshot of a full round on a
  representative world state. Same events dispatched, same mob
  state mutations for active-zone mobs.
- Manual smoke:
  1. Combat in populated zone — unchanged behavior.
  2. Leave zone during combat. Mob sits frozen.
  3. Return before combat-memory expiry. Mob resumes combat.
  4. Return after combat-memory expiry. Mob is neutral.
  5. Arena death → ejection → arena zone goes cold. Mobs idle.
  6. Admin dashboard clicks during combat. No `-race` warnings.
  7. Fold-recall mid-combat. Zone change updates correctly.

### Benchmarks

New file `internal/hooks/bench_mobroundtick_test.go`:

```go
func BenchmarkMobRoundTick_AllActive(b *testing.B)
func BenchmarkMobRoundTick_MostlyIdle(b *testing.B)  // 10% active
func BenchmarkMobRoundTick_AllIdle(b *testing.B)
```

Target metric: "MostlyIdle" case ≥5× faster than baseline after
lane split.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Zone counter drifts (missed call path) | Low | Mobs freeze in a zone that should be active | `VerifyZonePlayerCount()` exposed via admin diagnostics page + covered by unit tests on every AddPlayer/RemovePlayer path. No automatic scheduled sweep in v1 — add one only if drift is observed in the wild |
| Idle-lane helper skips something needed | Low | Invisible bug in idle zones | Scope is strictly explicit: 5 helpers run always. No player can witness a diff in an idle zone by definition |
| Mutex deadlock | Low | Server hang | Fine-grained locks; no cross-package holds; `go vet` + `-race` in CI |
| Combat-memory expires before player returns | Medium (by design) | Returning player's attacker is neutral | Intended. Previous behavior left aggro forever, which was the actual bug |
| Instance zone with zero players + live mobs | Low | Arena boss stops ticking down instance age | Instance expiration is a separate per-instance hook that doesn't iterate per-mob — unaffected |

## Rollout Order

Lowest-risk first, each step commits independently and is
individually revertable.

1. **PruneVisitors fast path.** 15 min. Commit and verify.
2. **Registry mutexes.** ~2–3h. Add RWMutex to 5 mob-package maps
   and the users map. Add `-race` CI test. Manual dashboard test.
3. **Zone activity tracking foundation.** ~2h. Add
   `zonePlayerCount`, accessors, AddPlayer/RemovePlayer hooks,
   startup rebuild, `VerifyZonePlayerCount` debug helper. No
   consumers yet — index runs in the background, validated by
   unit tests.
4. **Apply the split.** ~4h. Extract 10 MobRoundTick inline blocks
   into helpers, add zone check to `handleMobCombat`, add
   active/idle branch to MobRoundTick. Benchmarks. Full manual
   smoke test.

**Estimated total: ~10h**, matching the overview.

## Future Work (Not In 1.7)

### Per-mob / per-user state mutexes

**The problem.** Our 1.7 mutexes protect map membership
(`does id → pointer resolve?`) but not the fields of the instance
that pointer names. A concurrent `Mob.Character.Health -= x` from
two goroutines is still a data race.

**Why we didn't fix it here.** Scope. Every `Mob` and
`UserRecord` is touched by dozens of call sites that currently
assume single-threaded access. Serializing access would mean
either (a) a mutex per instance on every field access, or (b)
funneling all mutations through an actor/queue pattern. Both are
architectural changes far beyond a cleanup pass.

**Pros of doing it in a later stage:**
- Would let us move more work to dedicated worker goroutines
  (e.g., offload AI decisions, behavior-tree evaluation, or
  crafting ticks from the main loop).
- Makes the admin web dashboard a first-class citizen — today it
  reads instance state concurrently and can observe torn writes
  (Health halfway updated, etc.). Fixing registry maps closes the
  "instance exists" race but not the "instance state is coherent"
  race.
- Enables safe parallelism within a single round (batched
  regen/buff ticks across mobs).

**Cons:**
- Pervasive churn. Every field access in `Character`, `Mob`,
  `UserRecord` needs either a mutex or a message.
- Lock ordering rules proliferate. Mob A locks first, mob B
  locks second — classic deadlock surface.
- Performance: taking an RWMutex on every `Mob.Character.Health`
  read is not free. A more fruitful pattern is often a single
  "this mob is being processed this round" ownership token.
- Alternative approach: instead of per-instance locks, commit to
  a single-threaded model and add a "GetSnapshot" API for readers
  (admin, Discord integration). Goroutines get copies, not live
  pointers. Much simpler rule to enforce.

**Recommended path if pursued:** start with the snapshot API
(copy-for-readers) before considering per-instance locks.
Snapshots cost one alloc per read but eliminate the race entirely
with no lock ordering to reason about. Only introduce per-instance
locks if a genuine write-write race surfaces that can't be handled
by routing through events.

Leave this for a post-1.x cleanup phase once we have data on
whether the current race surface is causing real user-visible
problems.
