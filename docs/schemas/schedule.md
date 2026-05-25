# Schedule YAML schema

Schedules are daily routines attached to NPCs via the
`schedule_id:` field on mob specs. Each schedule references a file
at `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml` (filename
= `ConvertForFilename(id)`).

## Required fields

```yaml
id: <string>                      # must match the filename
description: <string>             # short prose, used in admin debug output
segments: [<segment>, ...]        # must cover all 24 hours, no overlaps
```

## Segment fields

```yaml
- start: <int 0-23>               # inclusive
  end: <int 1-24>                 # exclusive; end may equal 24 for the
                                  # day-boundary
  target_room: <int>              # must exist; mob is steered here
  activity: <"" | "craft">        # gates engine-side activity verbs
  idlecommands:                   # mob's idle pool while in this segment
    - emote <text>
    - say <text>
```

Wrap-around: when `start > end`, the segment covers `[start, 24)`
and `[0, end)`. Example: `start: 22 end: 6` covers 22-5.

Two segments may share a `target_room` (e.g. the priest visits the
temple twice with different `idlecommands`).

## Validation (load-time, panics)

- Filename must equal `ConvertForFilename(id)`.
- Each segment must satisfy `0 <= start < 24` and `0 < end <= 24`.
- `start != end` (no empty segments).
- Every hour 0-23 must be claimed by exactly one segment (no
  overlaps, no gaps).
- `target_room` must exist.
- `mapper.GetPath` must succeed for every consecutive segment pair
  (chronological order, including the wrap-around transition).
- The mob's `schedule_id` (in its mob YAML) must resolve.

## Validation (load-time, warn-only)

- `activity:` value not in `{"", "craft"}`.
- `activity: craft` on a mob that lacks `crafter: true`.
- `target_room` is outside the mob's `zone`.
- Segment has zero `idlecommands` (mob will be silent).

## Future: per-day variation

Single-day routines are the current shape. When per-day variation
lands, the loader will recognise an optional top-level `days:` map
and prefer it over flat `segments:`:

```yaml
# Not implemented today — shown for forward-compatibility.
id: thornwall_smith
description: "..."
days:
  default: [ ...segments... ]
  weekend: [ ...segments... ]
  holiday: [ ...segments... ]
```

Existing flat-segment schedules continue working unchanged. The
schedule `id` is the stable reference; mob YAMLs do not move.

## Authoring workflow

1. Pick an `id` (snake_case, zone prefix recommended).
2. Identify target rooms in the zone — confirm reachability from
   each other.
3. Author segments covering all 24 hours; each segment gets idle
   flavor written in NPC voice (`say` and `emote` lines).
4. Save to
   `_datafiles/world/dogmud/schedules/<zone>/<filename>.yaml`.
5. Add `schedule_id: <id>` to the mob spec.
6. Restart the server. The validator will surface any coverage,
   pathing, or reference issues at boot.
7. In-game, use `mob schedule <instId>` to confirm the executor
   resolves the right segment for the current hour.

## See also

- Spec: `docs/superpowers/specs/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`
- Mob spec field: `internal/mobs/mobs.go` (`Mob.ScheduleId`)
- Loader: `internal/mobs/schedule_loader.go`
- Executor: `internal/hooks/NewRound_IdleMobs_schedule.go`
- Admin inspector: `mob schedule <instId>`
