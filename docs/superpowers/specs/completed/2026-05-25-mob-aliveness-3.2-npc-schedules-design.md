# Mob Aliveness 3.2 — NPC Schedules (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.2 (Phase 3 — Routine layer)
**Size:** L
**Branch:** `feature/mob-aliveness-3.2-npc-schedules`
**Depends on:** 3.1 (game-time hook, shipped 2026-05-23) for the
hour-precision time source the executor consults.

## Goal

Authored daily routines that move NPCs between rooms and swap their
idle flavor by the hour. A town that empties at night and fills in
the morning feels a thousand percent more alive than a static town.

The chunk delivers:

- A `schedule_id:` field on mob specs.
- A new `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml` file
  type, loaded at startup.
- A Go-side executor wired into `NewRound_IdleMobs` that steers
  scheduled mobs toward the current segment's target room and swaps
  their idle-command pool.
- A spawn-override so scheduled mobs appear at the right room for
  the current hour on cold start and on death-respawn.
- A new `mob_at_target_room` btree condition for archetype-side
  activity gating.
- An `activity:` field on segments that gates `TickMobCraft` so
  Blacksmith Kerra only forges at the forge.
- A `mob schedule <instanceId>` admin inspector for debugging.
- Three Thornwall City pilots — Blacksmith Kerra, Tavern Keeper
  Marek, Temple Priest Olen — plus three new "above-shop" home
  rooms.

## In scope

- Single-day routines (one segment list per schedule).
- Full 24-hour coverage required; load-time validator panics on
  gaps or overlaps.
- Per-segment idlecommand pool swap (reuses existing
  `Mob.IdleCommands` plumbing).
- `pathto`-driven movement (reuses existing path-walker in
  `NewRound_IdleMobs`).
- Spawn override at current segment's target room on server start
  and on death-respawn.
- One activity tag (`craft`) recognised by the engine; the slot is
  authored on every segment for forward compatibility but unknown
  values are warn-only, not fatal.
- Schema designed to absorb the chunk-3.X future per-day variation
  without breaking existing files.

## Out of scope

- **Per-day variation** (weekday/weekend/holiday). The schema is
  shaped to absorb a `days:` layer later — see "Future per-day
  extensibility" below — but today every schedule is a single flat
  segment list.
- **Sleeping state** — chunk 3.3 owns the real sleeping condition,
  room descriptions, wake triggers, and combat-on-sleeper
  consequences. Today, "asleep" is approximated by a segment with
  `target_room: <home>` and sleepy idlecommands.
- **Multi-room segments** — each segment has exactly one
  `target_room`. Multi-room "smith picks up tools, walks to anvil,
  hammers" is chunk 3.5's territory.
- **Real maintenance activity** — `activity: craft` only gates the
  existing crafter tick; it doesn't make the smith actually swing a
  hammer at an anvil entity. Real maintenance verbs are chunk 3.5.
- **Waypoint patrols** — chunk 3.4 owns looping routes with dwell
  times. Schedules are point-to-point per segment, not patrols.
- **NPC↔NPC conversations** — chunk 3.6.
- **`/new-schedule` content generator command** — schedules are
  hand-authored for the pilot. A generator may come later.
- **Schedule reload at runtime** — load-once at startup, like
  rooms/mobs.

## Architecture

### Data model

**Mob spec gains one field** (`internal/mobs/mobs.go`):

```go
ScheduleId string `yaml:"schedule_id,omitempty"`
```

Empty / missing = no schedule (existing wander behaviour preserved).

**Schedule type** (new file `internal/mobs/schedule.go`):

```go
type Schedule struct {
    Id          string
    Description string
    Segments    []ScheduleSegment
}

type ScheduleSegment struct {
    Start        int      // 0-23 inclusive
    End          int      // 1-24 inclusive (24 = midnight upper bound)
    TargetRoom   int
    Activity     string   // "" | "craft" | future maintenance verbs
    IdleCommands []string
}

// Package-level registry, populated by LoadDataFiles().
var schedules = map[string]*Schedule{}

// Get returns the schedule for an id, nil if missing.
func GetSchedule(id string) *Schedule { ... }

// CurrentSegment returns the segment active at hour24 (0-23).
// Returns nil if no segment covers this hour (defensive — coverage
// is validated at load time).
func (s *Schedule) CurrentSegment(hour24 int) *ScheduleSegment { ... }
```

`CurrentSegment` handles wrap-around segments (start > end) the
same way chunk 3.1's `inHourRange` helper does — borrow that helper
verbatim or factor it out into `internal/gametime`.

### Schedule YAML files

**Location:** `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`.
Zone subfolder convention matches rooms/mobs/behaviors. Filename
must equal `ConvertForFilename(id)` (engine convention; mismatch
panics at startup).

**Schema (chunk 3.2 form):**

```yaml
id: thornwall_smith
description: "Kerra's smith routine: forge by day, home at night"
segments:
  - start: 6
    end: 9
    target_room: 1234        # home loft above the forge
    activity: ""             # no engine-recognised activity
    idlecommands:
      - emote rubs sleep from her eyes.
      - emote pulls on her boots and apron.
  - start: 9
    end: 18
    target_room: 5678        # the forge
    activity: craft          # gates TickMobCraft
    idlecommands:
      - emote raises the hammer once more.
      - emote tongs a glowing bar from the coals.
      - say I'll have it done by sunset.
  - start: 18
    end: 22
    target_room: 9012        # tavern
    activity: ""
    idlecommands:
      - emote sips from a tankard.
      - say Long day at the forge.
  - start: 22
    end: 6                   # wraps midnight
    target_room: 1234        # back to the loft
    activity: ""
    idlecommands:
      - emote breathes evenly, asleep on the cot.
      - emote turns over with a soft snore.
```

### Future per-day extensibility (planned, not built)

When per-day variation lands, the loader will check for an optional
top-level `days:` map first; if absent, fall back to the flat
`segments:` form. Day-one schedules keep working unchanged:

```yaml
# Future shape, not implemented in 3.2
id: thornwall_smith
description: "..."
days:
  default: [ ...segments... ]
  weekend: [ ...segments... ]
  holiday: [ ...segments... ]
```

The schedule `id` stays stable across this change; mob YAMLs don't
move. Calling out this shape in `docs/schemas/schedule.md` keeps
authors from inventing the wrong structure.

### Loader & validation

**Loader** lives in `internal/mobs/schedule_loader.go`. Called from
`mobs.LoadDataFiles()` once at startup (after rooms are loaded, so
target_room existence checks resolve). Mirrors the existing
buff/quest/mob loaders. Filesystem walks
`_datafiles/world/dogmud/schedules/**/*.yaml`.

**Load-time validation (panic on any failure):**

| Check | Rule |
|---|---|
| Filename matches id | `ConvertForFilename(id) == filename` |
| Segment hour bounds | `0 <= start < 24`, `0 < end <= 24`, `start != end` |
| Target room exists | `rooms.LoadRoom(target_room) != nil` |
| 24h full coverage | segments (including wrap-around) claim every hour 0-23 exactly once |
| No overlapping segments | implied by exactly-once-coverage |
| Inter-segment path resolves | for every consecutive segment pair (including wrap-around), `mapper.GetPath(seg[i].target_room, seg[i+1].target_room)` succeeds |

**Load-time warnings (non-fatal, logged once):**

| Check | Rule |
|---|---|
| Unknown `activity:` value | Anything other than `""` or `craft` (or future-known activities). Warn so authors notice typos but don't break boot. |
| `activity: craft` on non-crafter mob | Mob YAML has `crafter: false` but a segment requests `craft`. Could be a planned future crafter — warn, don't fail. |
| Cross-zone target_room | Segment's target_room is outside the mob's zone. Weird but not forbidden — warn. |
| Segment has zero idlecommands | Mob will be silent at that location — warn. |

**Mob-side cross-check** runs after both mobs and schedules are
loaded: every mob with `schedule_id` must reference a schedule
that exists. Mismatch → panic with the mob file path and the
missing id.

### Executor — `NewRound_IdleMobs` hook

The schedule branch sits in `internal/hooks/NewRound_IdleMobs.go`,
*after* the combat / conversation / despawn guards and *before*
the existing path-walker branch.

Pseudocode:

```go
if mob.ScheduleId != "" {
    sched := mobs.GetSchedule(mob.ScheduleId)
    if sched != nil {
        hour := gametime.GetDate().Hour24
        seg := sched.CurrentSegment(hour)

        // Detect segment transition since last tick.
        // schedule_last_seg_start is stored in MiscData (transient,
        // not persisted — first tick after spawn always counts as
        // a transition, which is the right behavior).
        lastSegStart := mob.GetMiscDataInt("schedule_last_seg_start")
        if seg != nil && seg.Start != lastSegStart {
            mob.SetMiscData("schedule_last_seg_start", seg.Start)
            mob.Path.Clear()            // cancel stale path from old seg
            mob.IdleCommands = seg.IdleCommands
        }

        if seg != nil && mob.Character.RoomId != seg.TargetRoom {
            // Not at target → queue pathto if no path is in flight.
            if mob.Path.Len() == 0 && mob.Path.Current() == nil {
                mob.Command(fmt.Sprintf("pathto %d", seg.TargetRoom))
            }
            // The existing path-walker below picks up the queued path.
        }

        // While a schedule is active, suppress wander entirely —
        // simplest is to zero MaxWander in-memory each tick. The
        // existing wander code already no-ops on MaxWander == 0.
        mob.MaxWander = 0
    }
}
```

Key invariants:

- Combat / conversation / charm guards (already present higher in
  the function) suspend the schedule branch entirely. The schedule
  resumes cleanly on the next idle tick.
- Player-displacement recovery is free: the existing path-walker
  already detects `currentStep.RoomId() != mob.Character.RoomId`
  and reissues `pathto home`. Our hook then observes "still not at
  target," queues a fresh `pathto <target>` on the next tick.
- Idlecommand pool swap is a single assignment; the existing
  idlecommand-firing code at the bottom of the round needs no
  change. The swap is **destructive** — the YAML-loaded base
  `IdleCommands` is overwritten while a schedule is active. This
  is intentional: a scheduled mob's "default" pool is the schedule,
  not the YAML, so no save/restore plumbing is needed. The base
  pool is restored implicitly on respawn (YAML reloads).
- Segment transitions mid-travel cancel the stale path and reissue
  `pathto` to the new target.

**Pathto failure handling.** `mob.Command("pathto N")` swallows
unreachable destinations silently for non-`home` targets. The
executor stores a counter in `MiscData` (key:
`schedule_path_fail_count`). The counter **increments on every
tick where `pathto` to the segment target fails**, and **resets to
0 the first tick the mob is at `seg.TargetRoom`**. After
`ScheduleMaxPathRetries` (new config, default 20 ≈ 80 seconds at
default tick rate), the mob runs `pathto home` to avoid stranding,
then the counter resets and retries resume on the next segment
transition. A dedup'd warning logs once per `(mob_instance,
target_room)` so the misconfig surfaces without spam.

### Spawn override

In the mob spawn path (`mobs.NewMobById`, `mobs.NewMobByIdFresh`,
or whichever entry point sets `Character.RoomId` to `HomeRoomId`
for new instances), after the standard fields are populated but
before the mob is added to its initial room:

```go
if mob.ScheduleId != "" {
    if sched := GetSchedule(mob.ScheduleId); sched != nil {
        if seg := sched.CurrentSegment(gametime.GetDate().Hour24); seg != nil {
            mob.Character.RoomId = seg.TargetRoom
        }
    }
}
```

`HomeRoomId` is unchanged — it stays the "true home" for the
existing `pathto home` semantic and for `GetVisibility` / other
hooks that key off home rooms.

### `TickMobCraft` activity gate

In `internal/mobs/crafter.go`, at the top of `TickMobCraft`:

```go
if mob.ScheduleId != "" {
    sched := GetSchedule(mob.ScheduleId)
    if sched != nil {
        seg := sched.CurrentSegment(gametime.GetDate().Hour24)
        if seg == nil || seg.Activity != "craft" {
            return nil  // off-duty — no crafting tick
        }
    }
}
// existing tick logic unchanged
```

Mobs without `schedule_id` skip the gate entirely → no regression
on existing crafters who don't have schedules yet.

### `mob_at_target_room` btree condition

Added to `internal/behaviortree/conditions_state.go` (alongside
the chunk 3.1 `time_of_day` condition; registered through the
condition registry in the same file's package init):

```go
// Returns Success when the mob is at its current schedule
// segment's target room; Failure when the mob has no schedule,
// no current segment, or is in transit.
func condMobAtTargetRoom(params map[string]any, ctx *EvalContext) Result
```

YAML:
```yaml
- condition:
    type: mob_at_target_room
```

This is the btree-side parity of the Go-side activity gate. Lets
archetype authors gate behaviour-tree branches on "am I where my
schedule says I should be?" without needing to know the
specifics. Used by future archetypes for emotes / verb triggers
tied to location-bound activities.

### Admin debug inspector

New subcommand on the existing `mob` admin command:

```
mob schedule <instanceId>
```

Output:
```
Schedule for Blacksmith Kerra (mob 97, instance 142):
  schedule_id:     thornwall_smith
  current hour:    14
  current segment: forge (9-18)
    target_room:   5678
    activity:      craft
  next segment:    tavern (18-22) in 4 hours
  mob location:    AT TARGET (5678)
  path queue:      empty
```

Lives in the dogmud-specific override
`_datafiles/world/dogmud/templates/admincommands/help/command.mob.template`
(beside the existing `mob heal` extension).

## Pilot content

Three Thornwall NPCs span the routine variety we want for a real
smoke test.

### NPCs

| Mob | NPC | Routine sketch |
|---|---|---|
| 97 | **Blacksmith Kerra** | 6-9 loft (waking) → 9-18 forge (`activity: craft`) → 18-22 tavern → 22-6 loft (sleep). Canonical smith arc; load-bearing crafter-gate test. |
| 96 | **Tavern Keeper Marek** | 6-10 quarters (prep) → 10-22 tavern (long shift) → 22-6 quarters (sleep). Demonstrates long-segment + simple two-target case. |
| 95 | **Temple Priest Olen** | 4-6 chamber (rise) → 6-10 temple (dawn prayers) → 10-12 chamber (rest) → 12-18 temple (afternoon prayers) → 18-22 tavern (yes, the priest drinks) → 22-4 chamber (sleep). Same room appearing in two non-adjacent segments — load-bearing variety check for the segment resolver. |

### New rooms (3)

Each NPC's "home" is added as an `up` exit from their workplace:

| Workplace | New home room | Exits added |
|---|---|---|
| Kerra's forge | "Kerra's loft above the forge" | `up` from forge → loft; `down` from loft → forge |
| Marek's tavern (back room) | "Marek's quarters above the tavern" | `up` from tavern back room → quarters; `down` → tavern back room |
| Olen's temple (cloister) | "Olen's chamber above the temple" | `up` from cloister → chamber; `down` → cloister |

Rooms are small, single-purpose bedrooms with short descriptions,
no useful affordances, no loot. Players can `up` into them too —
not locked, just nondescript. Specific room IDs assigned during
plan-writing (run `id_inventory.py --zone thornwall_city --alloc
rooms 5` for a clean block).

### File deltas

- 3 new room YAMLs in `_datafiles/world/dogmud/rooms/thornwall_city/`
- 3 existing room YAMLs edited (forge, tavern back room, cloister)
  to add the `up` exit
- 3 new schedule YAMLs in
  `_datafiles/world/dogmud/schedules/thornwall_city/`
  (`thornwall_smith.yaml`, `thornwall_tavern_keeper.yaml`,
  `thornwall_temple_priest.yaml`)
- 3 mob YAMLs edited to add `schedule_id:` field
  (97-blacksmith_kerra, 96-tavern_keeper_marek, 95-temple_priest_olen)
- Any `rooms.instances/` stale saves for the edited workplace rooms
  must be deleted (CLAUDE.md instance-save SOP)

## Edge cases & failure modes

| Situation | Behaviour |
|---|---|
| Mob in combat at segment transition | Combat guard at top of `IdleMobs` suspends schedule. After combat ends, next idle tick re-syncs to current segment. |
| Player drags scheduled mob to wrong room | Existing path-walker detects `currentStep.RoomId() != mob.Character.RoomId`, reissues `pathto`. Schedule layer idempotent — next tick queues a fresh `pathto` toward correct segment target. |
| Segment transition fires mid-travel | Tick detects `seg.Start != lastSegStart`, clears `mob.Path`, reissues `pathto` to new target on next iteration. |
| Target room temporarily unreachable | `pathto` fails silently. Per-mob retry counter increments. Dedup'd warning logged once per `(mob_instance, target_room)`. After `ScheduleMaxPathRetries` (default 20) consecutive failures, mob runs `pathto home` to avoid getting stranded. Counter resets on first successful arrival at target. |
| Target room *permanently* unreachable (zone refactor) | Caught at startup by inter-segment pathfinding sanity check. Panic — pre-push SOP boot-test surfaces this before prod. |
| `schedule_id` references missing schedule | Startup panic (mob-side cross-check). |
| Scheduled mob is charmed | Existing `isCharmed` guard at top of `IdleMobs` short-circuits schedule. When charm expires, schedule resumes from current hour. |
| Mob killed during a segment | Standard despawn timer; respawn applies the spawn-override → reappears at the current segment's target room. |
| Server restart mid-segment | Spawn-override applies at boot → scheduled mobs appear at correct rooms, idlecommands already swapped. World "already in motion." |
| Player in conversation with NPC across segment transition | Conversation guard at top of `IdleMobs` suspends idle tick during conversation. Schedule transition picks up cleanly on next idle tick after conversation ends. |
| Locked door / quest-gated path | Load-time pathfinding doesn't model locks. Runtime treats it as unreachable: dedup'd warning + retry counter + fallback to home. Treated as content-authoring bug. |

## Validation summary

**Load time (panic):**
- Filename ↔ id mismatch
- Segment hour bounds outside [0, 24]
- Coverage gaps or overlaps across 24 hours
- Target room doesn't exist
- Inter-segment pathfinding fails (mapper.GetPath returns
  ErrPathNotFound)
- Mob's `schedule_id` doesn't resolve

**Load time (warn-only, dedup'd):**
- Unknown `activity:` value
- `activity: craft` on non-crafter mob
- Cross-zone target_room
- Segment with zero idlecommands

**Runtime (warn-only, dedup'd):**
- `pathto` failure for a scheduled mob (logged once per
  mob_instance + target_room, then suppressed)

## Configuration knobs

One new config knob in `internal/configs/config.balance.go` (or
wherever similar mob knobs live):

| Knob | Default | Purpose |
|---|---|---|
| `ScheduleMaxPathRetries` | 20 | After N consecutive `pathto` failures, scheduled mob falls back to `pathto home` to avoid stranding |

No other new knobs — the system is content-driven.

## Testing

### Unit tests

| Test file | Coverage |
|---|---|
| `internal/mobs/schedule_test.go` | `Schedule.CurrentSegment(hour)`: basic hour-in-range, exclusive-end boundary, wrap-around (start > end), same room appearing in two non-adjacent segments (Olen pattern), nil/no-coverage defensive return |
| `internal/mobs/schedule_loader_test.go` | All load-time panics fire correctly: coverage gap, overlap, bad hour bounds, missing target_room, filename mismatch, unresolvable schedule_id on mob, unreachable inter-segment path. All warn-only cases log without panicking |
| `internal/hooks/NewRound_IdleMobs_schedule_test.go` | Scheduled mob at target fires segment idlecommands; mob in wrong room queues `pathto`; segment transition mid-travel clears stale path; charm short-circuits schedule; combat short-circuits schedule; MaxWander zeroed while schedule active; pathto retry counter increments + falls back to home at threshold |
| `internal/mobs/crafter_test.go` (additions) | `TickMobCraft` returns nil when scheduled mob's current segment activity != `craft`; unchanged behavior for mobs with no `schedule_id` |
| `internal/mobs/spawn_test.go` (additions) | Spawn override places scheduled mob at current segment's target room, not HomeRoomId; mob without schedule_id spawns at HomeRoomId as before |
| `internal/behaviortree/conditions_state_test.go` (additions) | `mob_at_target_room` returns Success at target, Failure in transit, Failure when no schedule |

### Manual smoke pass

After build:
1. Boot fresh, confirm no startup panic.
2. `time set 09:00` → confirm Kerra at forge, Marek at tavern,
   Olen at temple. Wait several rounds; observe segment
   idlecommands firing.
3. `time set 13:00` → confirm Olen at chamber (rest segment).
4. `time set 22:00` → confirm all three at their respective home
   rooms (loft / quarters / chamber).
5. `mob schedule <kerra_instance>` confirms expected output at
   each time.
6. Attack Kerra mid-shift → confirm combat suspends schedule,
   schedule resumes after combat ends.

### Autonomous smoke (dispatched alongside 3.3 brainstorming)

A smoketester session observes one full game-day's schedule cycle.

- **Target NPC:** Blacksmith Kerra (most variety: 4 segments,
  crafter-gate test, same-tavern overlap with Olen).
- **Tool:** `/test-mud local feel-tester` with a new one-off goal
  file `tools/testing/goals/3.2-schedule-observation.yaml`.
- **Tester instructions (encoded in goal file):**
  1. Locate Kerra at session start; log her room and the game time.
  2. Every N rounds (compute from `RoundsPerDay` so ~12 samples
     across one game-day), re-check her location and log any
     change with the game-time.
  3. When Kerra is at the forge, confirm crafting output is
     visible. When she's elsewhere, confirm crafting output is NOT
     visible.
  4. Watch for "lost" status, error spam, or her getting stuck.
- **Output:**
  `tools/testing/reports/3.2-schedule-observation-<timestamp>.md`
  with timestamped location log + crafting-gate observations +
  anomalies.
- **Pass criteria:**
  - All 4 segments visited in the correct hour-ranges.
  - No startup panics.
  - No warn-spam in server logs.
  - Crafting fires only during the `activity: craft` segment.

Session length is enough real-time to span one game-day —
documented in the goal file based on `RoundsPerDay × tick rate`.

## Documentation

### Internal context.md updates

| File | Change |
|---|---|
| `internal/mobs/context.md` | New "Schedules" section: `schedule_id` field, schedule YAML schema link, spawn-override, `TickMobCraft` activity gate, `mob_at_target_room` btree condition pointer |
| `internal/hooks/context.md` | Update `NewRound_IdleMobs` row: schedule branch inserted before path-walker. Update `MobIdle_HandleIdleMobs` row: crafter tick respects schedule `activity:` |
| `internal/behaviortree/context.md` | New conditions table row: `mob_at_target_room` |
| `internal/configs/context.md` | New config knob row: `ScheduleMaxPathRetries` |

### Schema reference

`docs/schemas/schedule.md` — full YAML reference. Documents the
chunk 3.2 flat-segments form as current, calls out the planned
`days:` layer so authors don't invent the wrong shape. Linked from
`docs/CONTENT_GENERATION_GUIDE.md`.

### Player-facing helpfiles

| File | Change |
|---|---|
| `_datafiles/world/default/templates/help/ask.template` | Append a short paragraph: "NPCs follow daily routines. The smith might not be at the forge after sunset — try the tavern or come back in the morning." |
| `modules/time/files/datafiles/templates/help/time.template` | Append one sentence: "Many townspeople follow daily routines — they work, eat, drink, and sleep at different times." |

A standalone `help schedules` is held in reserve. The line in
`help time` and the paragraph in `help ask` together cover the
"why can't I find the smith at midnight" question. Decide during
implementation whether the standalone helpfile adds value.

### Admin helpfile

Append a `mob schedule` block to
`_datafiles/world/dogmud/templates/admincommands/help/command.mob.template`
(the dogmud override, not the upstream default).

### CLAUDE.md

Short subsection under "Project Context":

> ### NPC Schedules
> Townspeople NPCs can carry a `schedule_id:` field that references a
> daily routine in `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`.
> Schedules cover all 24 hours, swap the mob's idle command pool per
> segment, steer the mob between rooms via the existing `pathto`
> plumbing, and gate `TickMobCraft` via segment `activity:`. Schedule
> validators panic at startup on coverage gaps, unreachable target
> rooms, or unresolved `schedule_id` references — pre-push SOP
> boot-test catches these.

### Content generation guide

`docs/CONTENT_GENERATION_GUIDE.md` gets a short "Schedules"
subsection: "No `/new-schedule` command yet; author by hand using
`docs/schemas/schedule.md`. Restart required after authoring."

## Commit shape

Suggested split (each commit independently reviewable):

1. `feat(mobs): schedule YAML loader + validator`
2. `feat(mobs): schedule executor in NewRound_IdleMobs + spawn override + crafter activity gate + mob_at_target_room`
3. `feat(content): Thornwall pilot — Kerra, Marek, Olen schedules + above-shop homes`
4. `feat(admin): mob schedule inspector command`
5. `docs: schedule schema + helpfiles + context.md + CLAUDE.md + roadmap closeout`

`PATCH_NOTES.md` gets a single entry at push time covering the
whole chunk, per the pre-push SOP. Pre-push checklist also
requires:
- Set `Logging.LogToFile: false` in `_datafiles/config.yaml`.
- Boot server locally and confirm clean startup (the new
  pathfinding-sanity validator will surface authoring drift here
  if any exists).
- Clean any `rooms.instances/` stale saves for the three edited
  workplace rooms.

## Roadmap closeout

`MOB_ALIVENESS_ROADMAP.md`:
- Flip chunk 3.2 status to **Done** with a one-line "Shipped:"
  describing the pilot (Kerra, Marek, Olen + crafter activity gate +
  spawn override).
- Add a note on chunk 3.5 (Maintenance routines) explaining its new
  load-bearing dependency on 3.2's `activity:` field.

## Open questions

None — design fully scoped during brainstorming.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 3.2
- Dependency: chunk 3.1 spec at
  `docs/superpowers/specs/completed/2026-05-23-mob-aliveness-3.1-game-time-hook-design.md`
- Existing path infrastructure:
  `internal/mobs/mobs_path.go` (`PathQueue`),
  `internal/mobcommands/pathto.go` (sets the queue),
  `internal/hooks/NewRound_IdleMobs.go` (walks the queue)
- Existing crafter tick:
  `internal/mobs/crafter.go:194` (`TickMobCraft`),
  called from `internal/hooks/MobIdle_HandleIdleMobs.go:74`
- Pathfinder: `internal/mapper/mapper.path.go` (`GetPath`)
- Game-time API: `internal/gametime/gametime.go` (`GetDate`,
  `GameDate.Hour24`)
- Btree validation pattern precedent: chunk 2.10
  `try_mutation_active`
- Loader pattern precedent: existing buffs / mobs / quests loaders
- Helpfile structure: `_datafiles/world/default/templates/help/`
  (default) +
  `_datafiles/world/dogmud/templates/admincommands/help/`
  (dogmud overrides)
