# Mob Aliveness 3.4 — Waypoint Patrols (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.4 (Phase 3 — Routine layer)
**Size:** M
**Branch:** `feature/mob-aliveness-3.4-waypoint-patrols`
**Depends on:** None hard. Composes with chunk 3.2 (NPC schedules)
when a schedule segment opts into `activity: patrol`.

## Goal

Authored multi-room patrol routes. NPCs walk a sequence of
waypoints with optional per-waypoint dwell. Combat interrupts;
on resume, the mob heads to the same target waypoint it was
walking toward when combat started.

Patrols are the movement foundation chunk 5.1 (Town Justice)
will consume — guards walking a beat are required infrastructure
for crime-reactive justice. 3.4 builds the movement primitive
only; crime detection and reaction stay 5.1's concern.

## In scope

1. New `patrol_id` field on mob spec — symmetric to chunk 3.2's
   `schedule_id`. Empty/missing → no patrol.
2. New patrol YAML file type loaded from
   `_datafiles/world/dogmud/patrols/<zone>/<id>.yaml`. Filename
   = `ConvertForFilename(id)`.
3. Two loop shapes:
   - `strict` (default) — A→B→C→D→A→B→…
   - `yo-yo` — A→B→C→D→C→B→A→B→…
4. Per-waypoint `dwell_rounds: N` (integer, default 0).
5. Standalone executor — mobs with `patrol_id` (no schedule)
   patrol always.
6. Schedule integration via `activity: patrol` + `patrol_id`
   on a schedule segment. The schedule executor stamps the
   active patrol id; the patrol executor consumes it the same
   tick.
7. Combat interrupt — patrol pauses via the existing IdleMobs
   combat guard. Resumes pathing to the same target waypoint on
   the next idle tick after combat ends.
8. Path-failure retry + home fallback — reuses the existing
   `ScheduleMaxPathRetries` config knob from chunk 3.2.
9. Loader validation: target rooms exist, inter-waypoint
   pathfinding resolves (mirrors 3.2's load-time sanity check).
10. Mob-side cross-check: every `patrol_id` reference must
    resolve to a loaded patrol; panic at boot on miss.
11. Spawn override for scheduled-patrol mobs: at server start /
    respawn, if the current segment is `activity: patrol` and
    has no `target_room`, place the mob at the patrol's first
    waypoint.
12. Loader change to the chunk 3.2 schedule schema: `target_room`
    becomes optional for segments with `activity: patrol`.
13. Admin debug extension: `mob schedule <instId>` (chunk 3.2's
    inspector) also shows patrol state when the mob has one —
    current waypoint index, target, direction, dwell remaining,
    path retry count.
14. Pilot content: one composed Thornwall City guard patrol
    (mob 106 city_guard) with a 6-22 patrol segment + 22-6
    sleep segment at a new guard barracks room.

## Out of scope

- **Dynamic re-routing when paths blocked** — explicit per
  roadmap. Hard-fail with home-fallback after retries.
- **Inter-zone patrols** — single-zone only. Cross-zone is
  chunk 3.7 (deferred — see roadmap).
- **Caravan unification** — caravans stay on their own movement
  system. 3.4's yo-yo shape opens the door for future refactor;
  chunk 3.7 owns that work.
- **Randomized / probabilistic waypoint order** — strict ordered
  only. "Wander a territory" is the forager pattern (chunk 2.9),
  not patrol.
- **Patrol-aware crime detection** — chunk 5.1 (Town Justice).
- **Per-waypoint emote/idlecommand pools** — schedule's
  per-segment idlecommands cover the "say something at this
  stop" case for composed patrols. Standalone patrols use the
  mob's base IdleCommands. A future refinement could add
  per-waypoint emotes if needed.
- **Player-facing patrol commands** — patrols are NPC behavior
  only. No player verb in 3.4.

## Architecture

### Data model

**Mob spec gains one field** (`internal/mobs/mobs.go`):

```go
PatrolId string `yaml:"patrol_id,omitempty"` // chunk 3.4: patrol route reference
```

Placed near `ScheduleId`.

**Patrol type** in new file `internal/mobs/patrol.go`:

```go
type Patrol struct {
    Id          string           `yaml:"id"`
    Description string           `yaml:"description,omitempty"`
    LoopShape   string           `yaml:"loop_shape,omitempty"` // "strict" | "yo-yo"; default "strict"
    Waypoints   []PatrolWaypoint `yaml:"waypoints"`
}

type PatrolWaypoint struct {
    Room        int `yaml:"room"`
    DwellRounds int `yaml:"dwell_rounds,omitempty"` // 0 = move on immediately
}

// Package-level registry, populated by LoadPatrols() at startup.
var (
    patrolsMu sync.RWMutex
    patrols   = map[string]*Patrol{}
)

func GetPatrol(id string) *Patrol { ... }
```

### Patrol YAML schema

```yaml
id: thornwall_market_perimeter
description: "Market square perimeter beat — corner shrine, bank, gate, tavern porch."
loop_shape: strict
waypoints:
  - room: 4200       # market square center
    dwell_rounds: 5
  - room: 4201       # NE corner shrine
    dwell_rounds: 10
  - room: 4202       # bank corner
    dwell_rounds: 8
  - room: 4203       # gate plaza
    dwell_rounds: 5
  - room: 4204       # tavern porch
    dwell_rounds: 3
```

`loop_shape` omitted → defaults to `"strict"`. Empty
`description` is allowed.

### Loop semantics

**Strict (`loop_shape: strict` or omitted):**
After reaching `waypoints[last]` and finishing its dwell, the
next target is `waypoints[0]`. Mob walks back to the start, then
forward through the loop again. Direction state is irrelevant.

**Yo-yo (`loop_shape: yo-yo`):**
Direction state in `patrol_direction` MiscData (+1 = forward,
-1 = reverse). When advancing:
- Forward at `waypoints[last]` → flip to reverse, next target
  is `waypoints[last-1]`.
- Reverse at `waypoints[0]` → flip to forward, next target is
  `waypoints[1]`.

### Per-mob runtime state (MiscData keys)

| Key | Value | Purpose |
|---|---|---|
| `patrol_waypoint_idx` | int | Current target waypoint index in the Waypoints slice |
| `patrol_direction` | int (+1 / -1) | Yo-yo direction; ignored for strict |
| `patrol_dwell_remaining` | int | Rounds left at current waypoint before advancing |
| `patrol_path_fail_count` | int | Consecutive failed `pathto` attempts; triggers home fallback at `ScheduleMaxPathRetries` |
| `active_patrol_id` | string | Set by schedule executor for the patrol executor to consume on the same tick; cleared after read |

All keys persist tick-to-tick during a mob instance's lifetime
but are transient across server restart / mob respawn (mob
MiscData isn't persisted to disk for non-essential NPCs).
Cold-start defaults are 0 / empty, equivalent to "first tick
from scratch, head to waypoints[0]."

### Loader & validation

**Loader** lives in `internal/mobs/patrol_loader.go`. Called from
`mobs.LoadDataFiles()` alongside the existing `LoadSchedules()`
call. Mirrors the 3.2 loader structure (filepath.Walk, yaml
unmarshal, validators).

**Load-time validation (panic on failure):**

| Check | Rule |
|---|---|
| Filename ↔ id | `ConvertForFilename(id)` equals the filename base |
| Waypoints non-empty | `len(Waypoints) > 0` |
| Each waypoint's `room` | non-zero; resolved via injected `roomExists` (DI pattern, see below) |
| `dwell_rounds >= 0` | negative is a typo |
| `loop_shape` | empty, `"strict"`, or `"yo-yo"` |
| Inter-waypoint pathfinding | for every consecutive `(waypoint[i] → waypoint[i+1])` pair, `mapper.GetPath` resolves. For `strict`, also check the wrap (`waypoint[last] → waypoint[0]`). For `yo-yo`, the forward path serves both directions — no extra check. |

**Mob-side cross-check** (after both mobs and patrols load):
every mob with `patrol_id` set must reference a loaded patrol.
Mismatch → boot panic.

**Warn-only checks** (log once via `mudlog.Warn`):
- `len(Waypoints) == 1` — degenerate patrol (mob will stand and
  dwell forever); author probably meant a schedule segment or
  a static spawn
- Cross-zone `room:` waypoint — explicitly out of scope for 3.4
  (chunk 3.7 lifts the restriction)
- `loop_shape` set to an unrecognized non-empty value — treated
  as `strict`, warn

**DI for world checks:** Extend the existing
`SetScheduleWorldValidator` pattern from chunk 3.2. Either reuse
the same injection (the two validators consume `roomExists` and
`hasPath` identically) or add a peer `SetPatrolWorldValidator`
in `patrol_loader.go`. Implementation-detail decision — pick
whichever produces cleaner code. The wiring in `main.go` adds
the patrol injection alongside the existing schedule one,
between `rooms.LoadDataFiles()` and `mobs.LoadDataFiles()`.

### Schedule integration

**Schema change to schedule YAML** (chunk 3.2 extension):

```yaml
- start: 6
  end: 22
  # target_room: OPTIONAL when activity is "patrol"
  activity: patrol
  patrol_id: thornwall_market_beat
  idlecommands:
    - say All clear here.
    - emote scans the square.
```

`target_room` becomes optional for segments with
`activity: patrol`. The schedule loader's existing
`target_room != 0` check is relaxed for patrol segments only.

A new field `PatrolId string` is added to `ScheduleSegment`.

If a patrol segment also sets `target_room` (legal but redundant),
it's ignored — the patrol's first waypoint serves as the
spawn-override anchor.

**Schedule loader validation extensions:**
- If `activity: patrol`, `patrol_id` must be non-empty (panic on
  miss) and must resolve to a loaded patrol (cross-check after
  both loaders run).
- A non-patrol segment with `patrol_id` set is a warning (the
  field has no effect; author probably meant `activity: patrol`).

### Spawn override

The chunk 3.2 spawn override
(`applyScheduleSpawnOverride(scheduleId, homeRoomId, hour)`)
returns the current segment's `target_room`. For patrol segments
with no `target_room`, fall back to:

1. The patrol's `waypoints[0].Room` (if the patrol resolves)
2. `homeRoomId` (if the patrol doesn't resolve — defensive)

A guard at server-boot during their dayshift segment thus
appears at the start of their beat, not at the barracks.

### Patrol executor

**Pure decision helper** in new file
`internal/hooks/NewRound_IdleMobs_patrol.go`:

```go
type patrolPlan struct {
    HasPatrol         bool
    WantsDwellWait    bool   // mob at current target with dwell > 0
    WantsPath         bool   // mob not at current target
    TargetRoom        int
    WantsAdvance      bool   // dwell expired (or 0); advance this tick
    NextWaypointIdx   int
    NextDirection     int    // +1 / -1; only meaningful for yo-yo
    WantsHomeFallback bool   // after MaxPathRetries
    FailureMessage    string
}

func patrolTickPlan(mob *mobs.Mob, patrolId string) patrolPlan
```

Pure over its inputs (`mob.Character.RoomId`, MiscData,
`patrolId`). Easy to unit-test without driving the full
IdleMobs loop.

**Decision flow per tick:**

1. Resolve patrol. If missing or empty, return empty plan
   (HasPatrol=false).
2. Read MiscData: `patrol_waypoint_idx` (default 0),
   `patrol_direction` (default +1), `patrol_dwell_remaining`
   (default 0), `patrol_path_fail_count` (default 0).
3. Resolve current target waypoint = `patrol.Waypoints[idx]`.
4. Three cases:
   - **At target + dwell > 0:** `WantsDwellWait = true`. Applier
     decrements `patrol_dwell_remaining`.
   - **At target + dwell == 0:** `WantsAdvance = true`. Compute
     next waypoint per loop_shape + direction.
   - **Not at target:** check retry counter. If
     `patrol_path_fail_count >= ScheduleMaxPathRetries`,
     `WantsHomeFallback = true`. Else `WantsPath = true` with
     `TargetRoom = current target waypoint.Room`.

**Side-effect applier:**

```go
func applyPatrolPlan(mob *mobs.Mob, plan patrolPlan)
```

- `WantsDwellWait` → `SetMiscData("patrol_dwell_remaining", current-1)`
- `WantsAdvance` → `SetMiscData("patrol_waypoint_idx", NextWaypointIdx)`,
  `SetMiscData("patrol_direction", NextDirection)`,
  `SetMiscData("patrol_dwell_remaining", newWaypoint.DwellRounds)`,
  `SetMiscData("patrol_path_fail_count", 0)` (fresh waypoint = fresh retry slate)
- `WantsPath` → `mob.Command(fmt.Sprintf("pathto %d", TargetRoom))`,
  increment `patrol_path_fail_count`
- `WantsHomeFallback` → log dedup'd warning,
  `mob.Command("pathto home")`,
  `SetMiscData("patrol_path_fail_count", 0)`

**At-target arrival** (the tick the mob finally reaches the
waypoint) → reset `patrol_path_fail_count` to 0 even on a
"WantsDwellWait" or "WantsAdvance" path. Cleanest: applier
always zeros it when the mob is at target.

### Wiring in `NewRound_IdleMobs`

Patrol branch placed AFTER the chunk 3.2 schedule branch (so the
schedule's stamp of `active_patrol_id` is visible). Pseudocode:

```go
// Chunk 3.4: patrol executor.
var activePatrolId string

// Schedule branch (chunk 3.2) may have stamped this MiscData
// during this tick for activity:patrol segments.
if id := getMiscDataString(&mob.Character, "active_patrol_id"); id != "" {
    activePatrolId = id
    mob.Character.SetMiscData("active_patrol_id", "")
}
// Standalone patrol: mob has PatrolId directly, no schedule.
if activePatrolId == "" && mob.PatrolId != "" {
    activePatrolId = mob.PatrolId
}

if activePatrolId != "" {
    plan := patrolTickPlan(mob, activePatrolId)
    applyPatrolPlan(mob, plan)
}
```

The schedule executor (chunk 3.2) gains a small addition: when
the active segment has `activity: patrol` AND `patrol_id != ""`,
the applier stamps `active_patrol_id` MiscData. The patrol branch
above consumes and clears it.

### Admin debug extension

Extend `mob schedule <instId>` (chunk 3.2 admin command) to also
render patrol state when the mob has `PatrolId != ""` or has an
active patrol context from a schedule segment. Sample additional
lines:

```
patrol_id:               thornwall_market_beat
patrol loop shape:       strict
patrol current waypoint: 2 (room 4202, "bank corner")
patrol direction:        +1
patrol dwell remaining:  4 rounds
patrol path retries:     0
```

## Pilot content

**One composed pilot:** Thornwall city guard (mob 106) with a
6-22 patrol segment + 22-6 sleep segment.

| File | Change |
|---|---|
| `_datafiles/world/dogmud/patrols/thornwall_city/thornwall_market_beat.yaml` | NEW — ~5 waypoint perimeter route in market/civic district. Exact rooms picked during plan-writing by reading the zone map. |
| `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_city_guard_dayshift.yaml` | NEW — schedule with 6-22 segment (`activity: patrol`, `patrol_id: thornwall_market_beat`) + 22-6 segment (`activity: sleeping`, `target_room: <new barracks room>`) |
| `_datafiles/world/dogmud/rooms/thornwall_city/<barracks_id>.yaml` | NEW — guard barracks. Same "above-shop" pattern as chunk 3.3 pilots; likely above an existing constabulary or guard-hut room. |
| `_datafiles/world/dogmud/rooms/thornwall_city/<existing>.yaml` | EDIT — add `up` exit to the new barracks |
| `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml` | EDIT — add `schedule_id: thornwall_city_guard_dayshift` |

Plan-writing resolves the specific room IDs via
`tools/id_inventory.py` and the Thornwall zone map.

## Edge cases & failure modes

| Situation | Behavior |
|---|---|
| Mob in combat during patrol | Existing IdleMobs combat guard suspends patrol. Next idle tick after combat ends, patrol executor sees same `patrol_waypoint_idx` and resumes pathing toward that waypoint. |
| Patrol mob dragged to a non-patrol room | Existing path-walker re-issues `pathto current_target_waypoint`. Same idempotent recovery as chunk 3.2 schedules. |
| Path between waypoints permanently broken (zone refactor) | Caught at boot by inter-waypoint pathfinding validator → panic. Pre-push SOP boot-test catches it before prod. |
| Path temporarily broken at runtime (locked door, temp exit removed) | Per-mob retry counter; after `ScheduleMaxPathRetries`, fall back to `pathto home`. Counter resets on next successful arrival at target. |
| Yo-yo direction lost across server restart | `patrol_direction` MiscData is transient. On respawn, defaults to +1 → starts at `waypoints[0]` going forward. Mob falls into rhythm within one loop. |
| Player follows patroller around | No special handling; the player just experiences the patrol. |
| Patrol with a waypoint at the mob's HomeRoomId | Works fine — HomeRoomId is just a room id like any other. |
| Patroller wakes from sleep (chunk 3.3) mid-patrol-segment | `OnSleeperWoken` stamps `schedule_wake_round`. Schedule executor's grace cooldown prevents immediate re-sleep. Patrol executor sees active patrol context and resumes from current waypoint. |
| `activity: patrol` schedule segment with no `patrol_id` | Loader-time panic. |
| Schedule transitions OUT of a patrol segment | The patrol executor reads-and-clears `active_patrol_id` every tick, so it goes empty the moment the schedule stops stamping it. The non-patrol new segment runs normally (target_room movement + idlecommands). Patrol MiscData (`patrol_waypoint_idx` etc.) is preserved — if the same patrol is later re-entered, the mob resumes mid-loop. If a *different* patrol is entered, the first-tick patrolTickPlan handles index validity (idx out-of-bounds → reset to 0). |
| Mob with both `patrol_id` (standalone) AND `schedule_id` whose segment has `activity: patrol` referencing a different patrol | Schedule-segment patrol wins (it's the active context for this tick). Standalone PatrolId is only used as a fallback when the schedule doesn't stamp `active_patrol_id`. Document this precedence. |
| Patrol mob killed mid-patrol | Standard despawn + respawn. Respawn applies the spawn-override (segment target_room if scheduled, or waypoints[0] for patrol segment); MiscData is transient so patrol state starts fresh. Acceptable cold-start. |
| Author edits the patrol YAML at runtime via /reload | Schedule loader pattern (chunk 3.2 fix): full registry replacement on reload, not additive. Patrol loader matches. Live patrol mobs' MiscData indices may now point at stale waypoint positions; first failed lookup falls through to "no current segment" or "no target" and the mob re-syncs by next tick. |

## Validation summary

**Load time (panic):**
- Filename ↔ id mismatch
- Empty waypoints list
- Waypoint room is 0 / doesn't exist
- `dwell_rounds < 0`
- Inter-waypoint pathfinding fails
- Mob's `patrol_id` doesn't resolve
- Schedule segment with `activity: patrol` and no `patrol_id`
- Schedule segment's `patrol_id` doesn't resolve

**Load time (warn-only, dedup'd):**
- Single-waypoint patrol (degenerate)
- Cross-zone waypoint (out of 3.4 scope)
- `loop_shape` unrecognized value (treated as `strict`)
- Non-patrol schedule segment with `patrol_id` set (no effect)

**Runtime:** Per-mob path-failure warning (dedup'd by
`(mob_instance, waypoint_room)` after the home-fallback trips).

## Configuration knobs

**No new knobs.** Reuses `ScheduleMaxPathRetries` (chunk 3.2,
default 20) for the patrol-path home-fallback threshold —
identical semantics, no reason to add a parallel knob.

## Testing

### Unit tests

| File | Coverage |
|---|---|
| `internal/mobs/patrol_test.go` | Patrol type + helpers — strict loop next-waypoint computation; yo-yo direction flip at endpoints; advance through 5-waypoint patrol covering forward + reverse direction transitions |
| `internal/mobs/patrol_loader_test.go` | All load-time panics (empty waypoints, bad shape, missing room, negative dwell, filename mismatch, mob-side cross-check) plus warn-only cases |
| `internal/hooks/NewRound_IdleMobs_patrol_test.go` | `patrolTickPlan` behaviors: at target with dwell → WantsDwellWait; at target with no dwell → WantsAdvance; not at target → WantsPath with retry counter; retries hit ScheduleMaxPathRetries → WantsHomeFallback; counter reset on arrival; yo-yo direction flip path |
| `internal/hooks/NewRound_IdleMobs_schedule_test.go` (additions) | Schedule segment with `activity: patrol` stamps `active_patrol_id`; clears on segment transition; patrol_id-on-non-patrol-segment is no-op (with warning) |
| `internal/mobs/schedule_loader_test.go` (additions) | Schedule with `activity: patrol` and no `target_room` validates cleanly; without `patrol_id` panics |

### Manual smoke pass

1. Boot fresh, confirm `mobs.LoadPatrols() loadedCount=1`.
2. Find the city guard. Use `mob schedule <inst>` to confirm
   patrol state: current waypoint, target, dwell, retries.
3. Watch the guard for two loops. Verify visits to all waypoints
   in order; dwell timing visible (lingers at corner shrine).
4. `time set 23` — guard returns to barracks, sleeps (per chunk
   3.3). `mob schedule` shows segment transition; patrol
   MiscData persists but `active_patrol_id` is unset.
5. `time set 7` — guard resumes patrol from waypoint 0 (or
   whichever index she had cached); resumes correctly.
6. Attack the guard mid-patrol. Combat. Win or `flee`.
   Confirm: next idle tick, guard resumes pathing to the same
   waypoint she was walking toward when combat started.

### Autonomous smoketester

New goal file `tools/testing/goals/3.4-patrol-observation.yaml`.
Observe the city guard for ~two full game-day loops, verifying:
- All waypoints visited in order
- Dwell timing approximately correct at each waypoint
- Day/night schedule transition (dayshift patrol → barracks
  sleep at 22, back to patrol at 6)
- Combat interrupt + resume

Pass criteria: 2 complete loops observed, no "lost" adjective,
no error/panic in chat, schedule + patrol integration both fire
correctly.

## Documentation

| File | Change |
|---|---|
| `docs/schemas/patrol.md` | NEW — full patrol schema, loop_shape semantics, composition with schedules, validation rules |
| `docs/schemas/schedule.md` | Extend activity-vocabulary section — `activity: patrol` with `patrol_id`, optional `target_room` |
| `internal/mobs/context.md` | New "Patrols" subsection peer to existing "Schedules" |
| `internal/hooks/context.md` | Note on patrol executor in IdleMobs section; schedule executor's `active_patrol_id` stamp |
| `internal/configs/context.md` | Note `ScheduleMaxPathRetries` also governs patrol path retries |
| `CLAUDE.md` | New "Patrols" subsection near the existing "NPC Schedules" / "Sleep Mechanics" subsections |
| `_datafiles/world/dogmud/templates/admincommands/help/command.mob.template` | Update `mob schedule` block to mention the patrol-state output |

No new player-facing helpfile — patrols are NPC behavior, the
existing `help time` / `help ask` notes cover the player-side
hint generically.

## Commit shape

Suggested split:

1. `feat(mobs): patrol type + loader + validators`
2. `feat(mobs): patrol_id field on Mob + load-time cross-check`
3. `feat(hooks): patrolTickPlan + applyPatrolPlan + IdleMobs hook`
4. `feat(mobs): schedule loader recognizes activity: patrol + optional target_room`
5. `feat(hooks): schedule executor stamps active_patrol_id`
6. `feat(mobs): spawn override falls back to patrol first waypoint`
7. `feat(admin): mob schedule inspector shows patrol state`
8. `feat(content): Thornwall city guard pilot (patrol + dayshift schedule + barracks room)`
9. `docs: patrol schema + context.md + CLAUDE.md`
10. `chore(roadmap): mark 3.4 Done`

`PATCH_NOTES.md` gets one entry at push time per pre-push SOP.

## Roadmap closeout

`MOB_ALIVENESS_ROADMAP.md`:
- Flip chunk 3.4 status to **Done**.
- Add "Shipped:" summary describing the patrol primitive, loop
  shapes, composition with schedules, pilot.
- Cross-note chunk 3.7 dependency on 3.4 patrols is now
  satisfied for the single-zone case.

## Open questions

None — design fully scoped during brainstorming.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 3.4
- Dependency for 3.7 (inter-zone patrols + caravan unification):
  this chunk
- Chunk 3.2 (schedules) — composition partner; spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`
- Chunk 3.3 (sleeping) — composition partner for day/night
  shift patrols; spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.3-sleeping-wake-states-design.md`
- Path infrastructure: `internal/mobs/mobs_path.go`,
  `internal/mobcommands/pathto.go`,
  `internal/hooks/NewRound_IdleMobs.go`
- Mapper: `internal/mapper/mapper.path.go`
- Existing schedule loader (precedent): `internal/mobs/schedule_loader.go`
- Existing schedule executor (precedent): `internal/hooks/NewRound_IdleMobs_schedule.go`
- Future chunk 5.1 (Town Justice) consumer: `MOB_ALIVENESS_ROADMAP.md`
- Future chunk 3.7 (inter-zone + caravan unification):
  `MOB_ALIVENESS_ROADMAP.md`
