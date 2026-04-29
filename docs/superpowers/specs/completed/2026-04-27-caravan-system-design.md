# Caravan System — Design (Stage 2 of Caravan Effort)

**Date:** 2026-04-27
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Build a continuously-running caravan that walks between Thornwall and
Stillwater, visiting each vendor in both towns and triggering restock on
arrival. Replace the existing per-mob auto-restock for caravan-served
zones — the caravan IS the restock mechanism. Use the Stage 1 NPC party
primitives for the caravan crew's coordinated movement and combat.

## Multi-stage context

This spec is **Stage 2** of a four-stage effort. Stage 1 (NPC party
system) shipped 2026-04-27 — see
`docs/superpowers/specs/2026-04-27-npc-party-system-design.md`.

1. ✅ NPC groups (Stage 1 — shipped)
2. **Basic caravan** (THIS SPEC) — Thornwall↔Stillwater route + restock-on-visit
3. Forager NPCs + region-split mat lists — wilderness mobs that gather and sell locally; mats split between regions
4. Real item transfer — caravan literally hauls items between town inventories (replaces v1 top-off-to-max)

Each stage gets its own spec → plan → implementation cycle.

## Worldbuilding

The caravan crew are wholesalers seeking arbitrage between regions. Some
goods are more valuable in one town than another, and they make the run
profitable by moving deliveries where they're needed. They are valued
visitors at each town's market — local merchants tolerate them as
suppliers but would push back hard if they tried to sell direct to
players, so they don't (Stage 2 keeps them as pure delivery agents; no
shop interface). Players see the caravan as a recurring "delivery day"
event roughly once per game day.

The dangerous road between the towns is the bandit-haunted North Road
(Stage 1's smoke-test consumers). Caravans hire on rough escorts to
fight through. Stage 2 establishes the recurring spectacle: every cycle,
the caravan crosses the bandit camp, the bandits engage, the bandits
lose, the caravan rolls on.

## Architecture overview

**What exists today:**

- `internal/parties/` — actor-aware party system from Stage 1 with
  `party_ensure_npc_party`, `party_call_help`, `party_respond_to_help`,
  `party_assist_target`, etc.
- `internal/hooks/MobIdle_HandleIdleMobs.go` line 55 — calls
  `mobs.TickMobShopRestock(mob)` every idle tick; restocks fire every
  ~200 rounds per `CrafterMaterialRestockRate` config.
- `internal/shops/shopinventory.go` — `Restock()` method tops off
  `RestockQty` items up to `MaxStock` per stock entry.
- Mob YAML `non_combatant: true` flag — wired into 7+ player commands
  (`attack`, `bash`, `kick`, `grapple`, `taunt`, `trip`, `shoot`,
  `throw`) and `steal` for shopkeeper-style attack rejection.
- `internal/mobcommands/lookfortrouble.go` — scans rooms for hostile
  player targets via `mob.Hostile` + `HatesSpecies`.

**What Stage 2 adds:**

```
internal/caravan/                              (NEW package)
  routes.go        # hardcoded route definitions
  state.go         # caravan state enum + transition logic
  visit.go         # vendor restock invocation
  routes_test.go
  state_test.go
  visit_test.go

internal/behaviortree/actions_caravan.go       (NEW)
  caravan_step  — single workhorse btree action;
                  reads MobState[caravan_state], dispatches to
                  internal/caravan state machine, advances state

internal/hooks/MobIdle_HandleIdleMobs.go
  Skip TickMobShopRestock for mobs in CaravanServedZones (~3 lines)

internal/mobcommands/lookfortrouble.go
  Extend hostile-target scan to also check HatesGroup against other
  mobs' groups field — bandits aggro on caravan via group hatred
  (~15 lines)

internal/usercommands/{attack,bash,grapple,kick,shoot,taunt,throw,trip}.go
internal/usercommands/skill.skullduggery.steal.go
  Add mob.PlayerAttackImmune check next to existing IsNonCombatant()
  check, with the same rebuff message (~3 lines per file)

internal/mobs/mobs.go
  Add PlayerAttackImmune bool field to Mob struct (~1 line)

internal/configs/config.balance.go
  Add CaravanServedZones []string,
      CaravanTransitDwellRounds int,
      CaravanDepotDwellRounds int

_datafiles/world/dogmud/mobs/thornwall_city/
  {N}-ketil.yaml      — caravan master, party leader
  {N+1}-marta.yaml    — guard, hammer & gambeson
  {N+2}-lars.yaml     — guard, ranged/off-hand support

_datafiles/world/dogmud/behaviors/thornwall_city/
  {N}-ketil.yaml      — leader btree (caravan_step + party combat)
  {N+1}-marta.yaml    — follower btree (party_assist_target + follow)
  {N+2}-lars.yaml     — follower btree

_datafiles/world/dogmud/dialogue/thornwall_city/
  {N}-ketil.yaml      — caravan-master flavor dialogue
  {N+1}-marta.yaml    — guard flavor
  {N+2}-lars.yaml     — guard flavor

_datafiles/world/dogmud/rooms/thornwall_city/{depot_room}.yaml
  Add caravan crew to spawninfo

_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml
_datafiles/world/dogmud/mobs/north_road/{284,285,286}-*.yaml
  Add `caravan` to HatesGroup; statpool detune ~25-30%
```

## Data model: state machine

The caravan operates as a continuous state machine. State is stored in
the leader's (Ketil's) `MobState["caravan_state"]` and progresses as
follows:

```
   ┌──────────────────────┐
   │  thornwall_dwell     │  ← initial spawn here at server start
   │  (CaravanDepotDwell  │
   │   ~360 rounds)       │
   └──────────┬───────────┘
              ↓ dwell timer expires
   ┌──────────────────────┐
   │  outbound_transit    │  pathto Stillwater depot via Stage 1
   │  (~60 rounds)        │  party_follow_leader
   └──────────┬───────────┘
              ↓ leader arrives at Stillwater depot room
   ┌──────────────────────┐
   │  stillwater_route    │  visit each Stillwater vendor room in
   │  (~30 rounds)        │  hardcoded order; trigger Restock()
   └──────────┬───────────┘
              ↓ all route stops complete
   ┌──────────────────────┐
   │  stillwater_dwell    │
   │  (~360 rounds)       │
   └──────────┬───────────┘
              ↓ dwell timer expires
   ┌──────────────────────┐
   │  inbound_transit     │  pathto Thornwall depot
   │  (~60 rounds)        │
   └──────────┬───────────┘
              ↓ leader arrives at Thornwall depot
   ┌──────────────────────┐
   │  thornwall_route     │  visit each Thornwall vendor room
   └──────────┬───────────┘
              ↓ all route stops complete
        (loops to top — thornwall_dwell)
```

**Total cycle:** ~900 rounds = ~1 hour real time = 1 in-game day. The
narrative beat is "the caravan arrives roughly once a day." Both dwell
durations are config-tunable via `CaravanDepotDwellRounds`; a
`CaravanTransitDwellRounds` knob is reserved for future use (e.g.,
forced dwell at intermediate inn rooms).

**Initial spawn**: Thornwall depot, state defaults to `thornwall_dwell`
on first btree tick. The choice of Thornwall as the home depot is
arbitrary — Stillwater would work equally well; Thornwall is just the
convention.

**Followers** (Marta, Lars) don't run their own state machine — they
follow the leader via Stage 1's `party_follow_leader` (extended for
cross-zone if needed; see implementation notes below).

## Route data

```go
// internal/caravan/routes.go

type Route struct {
    DepartFromRoomId int      // e.g., Thornwall depot room
    ArriveAtRoomId   int      // e.g., Stillwater depot room
    VendorStopIds    []int    // vendor rooms visited at the destination
}

var (
    OutboundRoute = Route{
        DepartFromRoomId: ThornwallDepotRoomId,
        ArriveAtRoomId:   StillwaterDepotRoomId,
        VendorStopIds:    StillwaterVendorRooms,  // ordered list
    }
    InboundRoute = Route{
        DepartFromRoomId: StillwaterDepotRoomId,
        ArriveAtRoomId:   ThornwallDepotRoomId,
        VendorStopIds:    ThornwallVendorRooms,   // ordered list
    }
)
```

Hardcoded for v1. Stage 4+ migrates to YAML config when there are
multiple caravans / multiple routes.

**Concrete depot + vendor rooms TBD during planning** — not blocking
spec approval. Likely candidates:

- Thornwall depot: a room near the east-end square (existing).
- Stillwater depot: a room in or adjacent to the Stillwater merchant
  cluster (likely Lakefront Square area).
- Vendor rooms: every Stillwater + Thornwall mob YAML with a non-empty
  `shop:` field — surveyed during plan task 0.

## Restock semantics

**Suppression of per-mob auto-restock**

```go
// _datafiles/config.yaml
Balance:
  CaravanServedZones:
    - "Stillwater"
    - "Thornwall City"
  CaravanDepotDwellRounds: 360  # ~24 min real, half a game day each
```

```go
// internal/hooks/MobIdle_HandleIdleMobs.go
// (around line 55, before TickMobShopRestock)

if !configs.GetBalanceConfig().IsCaravanServedZone(mob.Zone) {
    if mobs.TickMobShopRestock(mob) {
        // existing restock flavor message logic
    }
}
```

**Restock invocation on caravan visit**

```go
// internal/caravan/visit.go

// VisitVendorsInRoom calls Restock() on every shop-bearing mob in the
// given room, returning the list of mob names that received a delivery
// (for flavor message generation).
func VisitVendorsInRoom(roomId int) []string {
    room := rooms.LoadRoom(roomId)
    if room == nil {
        return nil
    }
    var visited []string
    for _, instId := range room.GetMobs(rooms.FindAll) {
        mob := mobs.GetInstance(instId)
        if mob == nil || !mob.HasShop() {
            continue
        }
        if mob.ShopInventory.Restock() {
            visited = append(visited, mob.Character.Name)
        }
    }
    return visited
}
```

`caravan_step`, when in a `*_route` state and arrived at the next vendor
room, calls `VisitVendorsInRoom` and emits a visible flavor message
("Ketil's crew unloads a crate of supplies; Smith Brindle nods his
thanks.") to the room.

If the caravan is destroyed mid-trip and the next visit is missed,
served-zone vendors run dry until a fresh caravan respawns and
completes its first cycle. This is the intended pressure of restock
semantics A — the caravan has real, visible weight. Stage 3 foragers
will provide a local floor for affected zones.

## Combat at room 4052: bandits aggro on caravan

**Group-hate primitive in `lookfortrouble`**

Bandit lookout currently scans for hostile players via
`mob.Hostile`. Extend it to also fire on mobs whose `groups:` list
intersects the caller's `HatesGroup:` list.

```go
// internal/mobcommands/lookfortrouble.go (sketch)

// (existing player-target scan stays as-is)

// NEW: hostile-mob scan
if len(mob.HatesGroup) > 0 {
    for _, otherInstId := range room.GetMobs(rooms.FindAll) {
        if otherInstId == mob.InstanceId { continue }
        other := mobs.GetInstance(otherInstId)
        if other == nil { continue }
        if mob.HatesAnyGroup(other.Groups) {
            possibleMobTargets = append(possibleMobTargets, otherInstId)
        }
    }
}
```

**Caravan crew YAML wiring**

```yaml
# _datafiles/world/dogmud/mobs/thornwall_city/{N}-ketil.yaml
groups:
  - caravan
  - merchant_train
```

```yaml
# _datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml
hates_groups:
  - caravan
```

When the caravan enters room 4052, the lookout's idle tick runs
`lookfortrouble`, finds Ketil's group `caravan` matches the lookout's
`hates_groups`, sets Aggro. Stage 1 `party_call_help` fires; camp mobs
respond. Combat plays out via existing party combat logic.

Caravan party uses Stage 1 `party_assist_target` to coordinate
targeting back at the bandits. Statted to win the 3-vs-4 with margin.

## Player attack rebuff

Caravan mobs cannot be `non_combatant: true` (they fight bandits). New
flag on Mob struct: `PlayerAttackImmune bool`. Same gates that already
check `IsNonCombatant()` in player attack commands get a sibling check.

```go
// internal/usercommands/attack.go (~line 161)
if m.IsNonCombatant() || m.PlayerAttackImmune {
    user.SendText(fmt.Sprintf("You can't attack <ansi fg=\"mobname\">%s</ansi>.", m.Character.Name))
    mobs.FireAttackRejected(...)
    return true, nil
}
```

Same pattern in: `bash.go`, `grapple.go`, `kick.go`, `taunt.go`,
`trip.go`, `shoot.go`, `throw.go`, `skill.skullduggery.steal.go`.

```yaml
# _datafiles/world/dogmud/mobs/thornwall_city/{N}-ketil.yaml
non_combatant: false
player_attack_immune: true
```

The flag affects only player-originated attacks. Mob-originated attacks
(the bandits) pass through unchanged.

## Bandit detune (in-scope; smoke-test consumer)

The Stage 1 bandit pack is too tough as a group, per playtest feedback.
Drop `statpool` by ~25-30% across the four bandit mobs:

| Mob | Current statpool | Detuned target |
|---|---|---|
| 283 bandit_lookout | 140 | 100 |
| 284 bandit_fighter | (current) | -25% |
| 285 bandit_caster  | (current) | -25% |
| 286 Soren          | (current) | -25-30% |

Goal: a baseline player (stats around 100, the human-baseline center)
should have a fighting chance against the lookout 1v1. The full pack
should still be a meaningful threat to a 3-person baseline party but
shouldn't be a guaranteed wipe.

Exact per-mob numbers settled in the implementation plan after a
quick tuning pass.

## Stage 1 primitives reused

| Primitive | Usage |
|---|---|
| `party_ensure_npc_party` | Caravan party forms at first btree tick (Ketil = leader, Marta + Lars = members; HomeRoomId = Thornwall depot) |
| `party_assist_target` | Combat at 4052: Marta/Lars copy Ketil's target |
| `party_call_help` | Available if caravan is jumped between camps; not expected to fire on the main route |
| `party_follow_leader` | Marta and Lars follow Ketil through transit states |
| `party_at_home_stand` | Caravan members at Thornwall depot don't wander off |
| MobDeath caller-clear | Inherited automatically — if Ketil dies, party dissolves; respawn + ensure rebuilds |

**Cross-zone movement note**: Stage 1's `party_follow_leader` and the
engine's `pathto` command are both already cross-zone capable — they
operate on roomids, not zone-relative paths. The existing
`internal/mapper/` A* finds routes across zone boundaries as long as
the exit graph is connected. Stage 2 should validate this end-to-end
during the smoke test (caravan transit Thornwall → Stillwater spans 4
zones: thornwall_city → thornwall_outskirts → marches_spur_road →
north_road → stillwater).

## Edge cases

| Scenario | Behavior |
|---|---|
| Caravan wiped at 4052 | Stage 1's leader-death path dissolves the party. All three mobs respawn at Thornwall depot per spawninfo timer. New btree ticks form a fresh party via `party_ensure_npc_party`. State defaults back to `thornwall_dwell`. Cycle resumes. ~5-15 min real-time outage of restock service. |
| Player attacks Ketil with `attack` | Rebuff message: "You can't attack Ketil." No aggro generated. |
| Player tries to steal from Ketil | Same rebuff path via `skill.skullduggery.steal.go`. |
| Player tags along with caravan in transit | Caravan ignores them. Player can observe the bandit fight. Mob-vs-player aggro logic unchanged — bandits still aggro on the player too. |
| Caravan route blocked (room destroyed, exit removed) | `pathto` fails. `caravan_step` falls through to legacy idle for that tick. Next tick retries. If permanently blocked, the caravan stalls — surfaces as a content bug for fix. No engine-level handling required. |
| Server restart | Caravan respawns at Thornwall depot per spawninfo. State defaults to `thornwall_dwell`. Cycle starts fresh. |
| Two players in the depot when caravan leaves | Caravan walks past them, players see "Ketil heads west toward Stillwater." Standard mob-leaves-room flavor. |
| Player kills a bandit while caravan is fighting them | Player gets the loot; combat resolves normally. Caravan continues on its route. |

## Testing strategy

**Unit tests** (`internal/caravan/*_test.go`):
- `state_test.go`: every state transition (thornwall_dwell → outbound_transit → stillwater_route → ...) returns the correct next state given the right preconditions
- `routes_test.go`: route data integrity (no zero room IDs, no duplicate stops)
- `visit_test.go`: `VisitVendorsInRoom` calls `Restock()` on each shop-bearing mob, returns list of names

**Btree integration tests** (`internal/behaviortree/actions_caravan_test.go`):
- `caravan_step` reads/writes `MobState["caravan_state"]` correctly
- Each state's tick action does the right thing (issues pathto, fires restock, advances dwell counter)
- Partial-party safety: caravan can advance even if one follower is briefly behind

**Hook test** (`internal/hooks/`):
- `TickMobShopRestock` is correctly skipped when `mob.Zone` is in `CaravanServedZones`
- Non-served zones still tick as before

**lookfortrouble test** (`internal/mobcommands/`):
- Hostile-mob group-hate scan adds the right mob to `possibleMobTargets`
- Doesn't fire when no group overlap

**In-game smoke test** (manual):
1. Boot server. `party admin list-npc` shows the caravan party with Ketil as leader, Marta + Lars as members.
2. Wait for `outbound_transit` state. Watch Ketil/Marta/Lars walking the road through 4 zones to Stillwater.
3. At room 4052, witness the brawl: bandit lookout aggros Ketil, calls help, camp mobs respond, both parties brawl, caravan wins. Bandits respawn after caravan moves on.
4. Caravan reaches Stillwater depot, transitions to `stillwater_route`. Visits each vendor room in order.
5. At each vendor stop, observe flavor message ("Ketil's crew unloads a crate of supplies; <vendor> nods.") AND verify the vendor's shop stock topped off.
6. Caravan returns to Thornwall depot, visits each Thornwall vendor.
7. Wait for next cycle. Verify the cycle repeats indefinitely.
8. `attack ketil` → confirm rebuff message.
9. Kill a Stillwater vendor's stock down to 0 (`buy iron ingot` repeatedly). Verify it does NOT auto-restock. Wait for next caravan visit. Verify it tops off.
10. Wipe the caravan (admin `kill ketil` or similar). Verify mobs respawn at Thornwall depot, state resets, cycle restarts.

## Out of scope (explicitly)

- **Foragers** — Stage 3 spec
- **Real item transfer** — Stage 4 spec (caravan moves actual items between town inventories instead of triggering top-off)
- **Multiple caravan routes** (e.g., Thornwall ↔ Watchers Crossing, Sanctum Basin ↔ anywhere) — deferred until route data moves to YAML config
- **YAML-driven route configs** — hardcoded in v1
- **Player-rideable caravan / passenger system**
- **Caravan-related quests** (escort, ambush, etc.) — possible Stage 3+ content
- **Town crier / notice board announcements** of caravan arrivals — possible polish
- **Multiple caravans per route** (staggered cadence) — single caravan in v1
- **Caravan storage / cargo theft mechanics** — Stage 4

## Files affected

| Action | File | Purpose |
|---|---|---|
| CREATE | `internal/caravan/routes.go` | Hardcoded route definitions |
| CREATE | `internal/caravan/state.go` | State enum + transition logic |
| CREATE | `internal/caravan/visit.go` | Vendor restock invocation |
| CREATE | `internal/caravan/routes_test.go` | Route data integrity |
| CREATE | `internal/caravan/state_test.go` | State machine unit tests |
| CREATE | `internal/caravan/visit_test.go` | Visit logic unit tests |
| CREATE | `internal/behaviortree/actions_caravan.go` | `caravan_step` btree action |
| CREATE | `internal/behaviortree/actions_caravan_test.go` | Btree integration tests |
| MODIFY | `internal/hooks/MobIdle_HandleIdleMobs.go` | Skip auto-restock in caravan-served zones |
| MODIFY | `internal/mobcommands/lookfortrouble.go` | Add hostile-mob group-hate scan |
| MODIFY | `internal/mobcommands/lookfortrouble_test.go` | Test group-hate scan |
| MODIFY | `internal/mobs/mobs.go` | Add `PlayerAttackImmune bool` field |
| MODIFY | `internal/usercommands/attack.go` | Add `PlayerAttackImmune` rebuff check |
| MODIFY | `internal/usercommands/bash.go` | Same |
| MODIFY | `internal/usercommands/grapple.go` | Same |
| MODIFY | `internal/usercommands/kick.go` | Same |
| MODIFY | `internal/usercommands/shoot.go` | Same |
| MODIFY | `internal/usercommands/taunt.go` | Same |
| MODIFY | `internal/usercommands/throw.go` | Same |
| MODIFY | `internal/usercommands/trip.go` | Same |
| MODIFY | `internal/usercommands/skill.skullduggery.steal.go` | Same |
| MODIFY | `internal/configs/config.balance.go` | New caravan config knobs |
| MODIFY | `_datafiles/config.yaml` | Default values for new knobs |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/{N}-ketil.yaml` | Caravan master |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/{N+1}-marta.yaml` | Guard |
| CREATE | `_datafiles/world/dogmud/mobs/thornwall_city/{N+2}-lars.yaml` | Guard |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/{N}-ketil.yaml` | Leader btree |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/{N+1}-marta.yaml` | Follower btree |
| CREATE | `_datafiles/world/dogmud/behaviors/thornwall_city/{N+2}-lars.yaml` | Follower btree |
| CREATE | `_datafiles/world/dogmud/dialogue/thornwall_city/{N}-ketil.yaml` | Caravan-master flavor |
| CREATE | `_datafiles/world/dogmud/dialogue/thornwall_city/{N+1}-marta.yaml` | Guard flavor |
| CREATE | `_datafiles/world/dogmud/dialogue/thornwall_city/{N+2}-lars.yaml` | Guard flavor |
| MODIFY | `_datafiles/world/dogmud/rooms/thornwall_city/{depot_room}.yaml` | Add caravan crew to spawninfo |
| MODIFY | `_datafiles/world/dogmud/mobs/north_road/283-bandit_lookout.yaml` | Add HatesGroup; statpool 140 → 100 |
| MODIFY | `_datafiles/world/dogmud/mobs/north_road/284-bandit_fighter.yaml` | HatesGroup + ~25% statpool detune |
| MODIFY | `_datafiles/world/dogmud/mobs/north_road/285-bandit_caster.yaml` | HatesGroup + ~25% statpool detune |
| MODIFY | `_datafiles/world/dogmud/mobs/north_road/286-soren.yaml` | HatesGroup + ~25-30% statpool detune |
| MODIFY | `docs/schemas/mob.md` | Document `player_attack_immune` and `hates_groups` fields |
| MODIFY | `docs/schemas/behavior.md` | Document `caravan_step` action |
| MODIFY | `PATCH_NOTES.md` | Stage 2 summary entry |

## Verification plan

**Phase 1 — unit + integration tests pass:**
- `go test ./internal/caravan/...`
- `go test ./internal/behaviortree/...`
- `go test ./internal/mobcommands/...`
- `go test ./internal/hooks/...`
- `go test ./...` overall (no regressions)

**Phase 2 — server boot clean:**
- Per the Pre-Push SOP in `CLAUDE.md`, boot the server locally and
  confirm no panics during data-file loading. Verify caravan crew mobs
  load (`mobs.LoadDataFiles() loadedCount=...`).

**Phase 3 — caravan smoke test (manual, in-game):**
- The 10-step smoke sequence from "Testing strategy" above.

**Phase 4 — backward compat smoke test:**
- Existing player-party features unchanged.
- Existing shopkeeper attack-rebuff (`non_combatant: true` mobs) unchanged.
- Stage 1 bandit pack behavior unchanged structurally — just detuned numbers.
- Caravan-served zones: vendors no longer auto-restock outside caravan visits (verify by reading shop tick log + waiting past 200 rounds).
- Non-served zones (Watchers Crossing, Sanctum Basin, etc.): vendors still auto-restock as before.

## Open implementation questions (for the plan stage)

These are detail-level decisions to make during planning, not
brainstorming-level decisions:

- Exact depot room IDs (Thornwall + Stillwater) — survey existing
  rooms and pick the best fit during plan task 0.
- Exact vendor room IDs and visit order in each town — surveyed in plan
  task 0.
- Caravan crew mob IDs — pick the next available range in
  `thornwall_city/`.
- Btree event vs. tick model for `caravan_step` — likely event-driven on
  `mob_idle` matching the Stage 1 pattern, but confirm during plan.
- Flavor message wording for caravan-arrives / vendor-restocked /
  caravan-leaves — drafted in plan task that creates the dialogue files.
- Whether to add a `RoutedCaravan` config option to disable the caravan
  entirely (for testing / edge cases) — probably yes, default true.
