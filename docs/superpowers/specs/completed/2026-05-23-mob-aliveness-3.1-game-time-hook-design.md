# Mob Aliveness 3.1 — Game-time Hook (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.1 (Phase 3 — Routine layer, foundation)
**Size:** Truly small (smaller than the roadmap's "S" estimate — almost all the infrastructure already exists)
**Branch:** `feature/mob-aliveness-3.1-game-time-hook`

## Goal

Extend the existing `time_of_day` btree condition with hour-range
support so chunk 3.2 schedules can author hour-precise gates like
"smith works 9-17". Preserves backward compatibility with the existing
binary `period: day` / `period: night` form.

## What already exists (discovered during brainstorming)

Almost every roadmap requirement for chunk 3.1 is already in the
codebase:

| Roadmap requirement | Status |
|---|---|
| Time tick | ✅ `util.GetRoundCount()` advances every round |
| Day/night flag | ✅ `gametime.IsNight()` + `GameDate.Night` |
| Configurable day length | ✅ `configs.GetTimingConfig().RoundsPerDay` |
| Btree condition (binary day/night) | ✅ `time_of_day` condition exists at `internal/behaviortree/conditions_state.go:64` |
| Visible time-of-day UI (out of scope per roadmap) | ✅ Already shipped via `modules/time/time.go` (`time` command) + `look.go` passes `IsNight` to room templates |

The ONLY gap: the existing `time_of_day` condition only handles
`period: day` / `period: night`. Chunk 3.2 schedules need
hour-precision (e.g., "smith works 9-17"), which neither the existing
condition nor any other btree primitive provides.

## In scope

Add a `range:` parameter to the existing `time_of_day` btree
condition.

**Parameter:** `range: "<start_hour>-<end_hour>"` (string). Hours are
0-23 (24h format, matching `GameDate.Hour24`). Wraps midnight when
`start_hour > end_hour`.

**Semantics:**

| YAML | Meaning |
|---|---|
| `range: "9-17"` | Success if `9 <= Hour24 < 17` (smith working hours) |
| `range: "22-6"` | Success if `22 <= Hour24` OR `Hour24 < 6` (wraps midnight, night shift) |
| `range: "0-24"` | Always Success (full day — likely author misconfig, warn) |
| `range: "5-5"` | Never Success (empty range — likely author misconfig, warn) |

**YAML examples:**

```yaml
# Existing binary form (unchanged)
- condition:
    type: time_of_day
    period: day

# New hour-range form
- condition:
    type: time_of_day
    range: "9-17"          # smith working hours

- condition:
    type: time_of_day
    range: "22-6"          # nightwatch shift, wraps midnight
```

Both parameters supported simultaneously. If both `period:` and
`range:` are set on the same node, `range:` takes precedence (single
source of truth — author error to set both, but range wins quietly
rather than failing). Document this precedence in `context.md`.

## Out of scope

- New btree condition with a different name (`time_of_day_is` per the
  roadmap sketch) — just extend the existing `time_of_day` condition.
  Keeps registry small, preserves backward compat.
- Numeric alternative `start_hour: int / end_hour: int` — `range:
  "9-17"` string form is sufficient. YAGNI.
- Day-of-week or month gating — chunk 3.2 explicitly defers
  weekday/weekend/holiday variation, so 3.1 doesn't need it.
- Sub-hour precision (e.g., "9:30-17:30") — minute-level scheduling is
  not a chunk 3.2 requirement. Hour granularity is enough for the
  "smith works 9-17, sleeps 22-6" use case.
- Player-facing time UI — already shipped.

## Validation

Parse `range:` at evaluation time (not load time, matching the
existing btree-action validation pattern from chunk 2.10
`try_mutation_active`):

- Non-numeric, wrong format, or out-of-bounds → `mudlog.Error` once
  per misconfig + return `Failure`.
- Empty range (`"5-5"`) → `mudlog.Warn` once + always `Failure`.
- Full-day range (`"0-24"`) → `mudlog.Warn` once + always `Success`.

A small `sync.Map` (or similar) tracks already-logged misconfigs so
warnings don't spam every tick.

## Architecture

**Single file modified:** `internal/behaviortree/conditions_state.go`.

The `condTimeOfDay` function gains a `range:` parameter check before
the existing `period:` switch. Parsing helper `parseHourRange(s
string) (start, end int, valid bool)` extracts and validates the
range string. `inHourRange(hour, start, end int) bool` handles the
wrap-around math.

No package-level state changes needed. No new dependencies.

## Testing

Unit tests in `internal/behaviortree/conditions_state_test.go` (or
wherever the existing condition tests live — verify during
implementation). Cover:

**Existing binary form (regression):**
- `period: day` returns Success at noon, Failure at midnight
- `period: night` returns Success at midnight, Failure at noon

**Hour range — basic:**
- `range: "9-17"` Success at 10am, 4:59pm (just before end)
- `range: "9-17"` Failure at 8am, 5pm (exclusive end), midnight

**Hour range — wrap-around:**
- `range: "22-6"` Success at 23, 0, 5 (wraps midnight)
- `range: "22-6"` Failure at 6 (exclusive end), 7, 21

**Hour range — edge cases:**
- `range: "0-24"` always Success + logged warning
- `range: "5-5"` always Failure + logged warning

**Hour range — malformed:**
- `range: ""` → falls back to `period:` if set, otherwise Failure
- `range: "abc"` → Failure + logged error
- `range: "9-25"` (out of bounds) → Failure + logged error
- `range: "9"` (missing end) → Failure + logged error
- `range: "-9-17"` (negative) → Failure + logged error

**Precedence:**
- Both `period: day` AND `range: "22-6"` set → range wins, midnight
  returns Success (range path), 10am returns Failure (range path)

Hour-of-day is sourced from `gametime.GetDate()`. Tests use
`util.SetRoundCount(N)` to a known round to make `Hour24`
deterministic. Verify test setup helpers exist or add minimal ones.

## Documentation

Update `internal/behaviortree/context.md`'s conditions table row for
`time_of_day` to document the new `range:` parameter alongside
`period:`. Single-line addition (or expand the existing row to two
lines if the table format allows).

## Commit shape

One commit: `feat(btree): time_of_day condition supports hour ranges`.
Includes the parser helper, the extended condition, the tests, and
the context.md update.

## Open questions

None — all clarifications captured during brainstorming, scope locked.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 3.1
- Existing condition: `internal/behaviortree/conditions_state.go:64`
  (`condTimeOfDay`)
- Existing gametime API: `internal/gametime/gametime.go` (`GetDate`,
  `IsNight`, `GameDate.Hour24`)
- Existing player time UI: `modules/time/time.go`, `internal/usercommands/look.go:692`
- Btree validation pattern precedent: chunk 2.10 `try_mutation_active`
  in `internal/behaviortree/actions_mutation.go` (log + Failure at
  first-call time)
- Downstream consumer: chunk 3.2 NPC schedules — needs `range:` to
  author hour-precise routines.
