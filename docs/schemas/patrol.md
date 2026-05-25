# Patrol YAML schema (chunk 3.4)

Patrols are looped multi-room routes attached to NPCs via the
`patrol_id` field on mob specs, OR via a schedule segment with
`activity: patrol` + `patrol_id`. Each patrol references a file
at `_datafiles/world/dogmud/patrols/<zone>/<id>.yaml` (filename
= `ConvertForFilename(id)`).

## Required fields

```yaml
id: <string>                      # must match the filename
description: <string>             # short prose; used in admin debug output
waypoints: [<waypoint>, ...]      # at least one
loop_shape: <"strict" | "yo-yo">  # default "strict" if omitted
```

## Waypoint fields

```yaml
- room: <int>                     # target room id; must exist
  dwell_rounds: <int>             # rounds to stay at this waypoint before
                                  # moving on; default 0 (move immediately)
```

## Loop semantics

**Strict (default):** A → B → C → D → A → B → … After the last
waypoint, the next target is the first waypoint.

**Yo-yo:** A → B → C → D → C → B → A → B → … Direction flips at
endpoints. State carried in per-mob MiscData (`patrol_direction`).

## Composition with schedules (chunk 3.2)

A schedule segment can opt in to running a patrol for its
duration:

```yaml
- start: 6
  end: 22
  # target_room is optional for activity: patrol segments
  activity: patrol
  patrol_id: thornwall_market_beat
  idlecommands:
    - say All clear here.
```

When the schedule executor enters a patrol segment, it stamps
`active_patrol_id` MiscData. The patrol executor (which runs
after the schedule executor on each idle tick) consumes the
stamp and drives the patrol.

If a patrol segment also sets `target_room` (legal but
redundant), the `target_room` wins for spawn-override
placement. The patrol's first waypoint serves as the fallback
when no `target_room` is set.

## Validation (load-time, panics)

- Filename must equal `ConvertForFilename(id)`.
- At least one waypoint.
- Each waypoint's `room` exists in the rooms registry.
- Each waypoint's `dwell_rounds >= 0`.
- `loop_shape` is empty, `"strict"`, or `"yo-yo"`.
- Inter-waypoint pathfinding resolves for every consecutive
  pair (and the wrap pair for `strict`).
- Mob `patrol_id` references resolve.
- Schedule segment `patrol_id` references resolve.

## Validation (load-time, warn-only)

- Single-waypoint patrol — degenerate.
- Schedule segment with `patrol_id` set but `activity` is not
  `patrol` — field has no effect.
- Cross-zone waypoint — out of scope for chunk 3.4 (will be
  handled by chunk 3.7).

## Runtime

The patrol executor runs in `NewRound_IdleMobs`. Per tick, for
mobs with an active patrol context, it consults the mob's
current waypoint index + direction (in MiscData) and decides
whether to dwell, advance, path toward the current target, or
fall back to `pathto home` after `ScheduleMaxPathRetries`
(default 20) consecutive path failures.

Combat interrupts patrols via the existing IdleMobs combat
guard. On the next idle tick after combat ends, the patrol
executor sees the same `patrol_waypoint_idx` and resumes
pathing.

## Future: cross-zone + caravan unification (chunk 3.7)

Cross-zone waypoint references and caravan-movement
unification (caravans become a yo-yo patrol with cargo + vendor
semantics layered on top) are deferred to chunk 3.7. The 3.4
loop_shape choices were made with future caravan migration in
mind.

## See also

- Spec: `docs/superpowers/specs/2026-05-25-mob-aliveness-3.4-waypoint-patrols-design.md`
- Mob spec field: `internal/mobs/mobs.go` (`Mob.PatrolId`)
- Loader: `internal/mobs/patrol_loader.go`
- Executor: `internal/hooks/NewRound_IdleMobs_patrol.go`
- Admin inspector: `mob schedule <instId>` (shows patrol state)
