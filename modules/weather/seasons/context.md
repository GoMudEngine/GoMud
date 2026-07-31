# Weather Seasons Context

## Purpose

`modules/weather/seasons` turns the calendar into a climate modifier. A
**track** is a named yearly cycle of seasons assigned to months; resolving a
track for a given day yields the current season, the previous one, and a
**blend** factor, so climate shifts gradually across a season boundary instead
of snapping.

The output feeds `sim.Climate`, which the weather simulation uses to decide
what fronts form where.

## Files

- **track.go** — `Season`, `Track`, `Tracks`, and YAML loading.
- **resolve.go** — day-of-year → (current, previous, blend).
- **apply.go** — climate blending and per-zone season resolution.
- **doc.go** — package doc.

## Types

```go
type Season struct  { /* name, months, climate multipliers */ }
type Track  struct  { /* ordered seasons covering the year */ }
type Tracks map[string]Track

type CalendarPos struct { /* where in the year we are */ }
type ZoneSeason  struct { /* the resolved season for one zone */ }
```

## API

```go
func Load(fsys fs.FS, dir string, monthsPerYear, daysPerYear int) (Tracks, []error)
func (t Track) Resolve(dayOfYear int) (current, previous string, blend float64)
func EffectiveClimate(base sim.Climate, tracks Tracks, pos CalendarPos) sim.Climate
func ZoneSeasons(g *sim.Graph, base sim.Climate, tracks Tracks, pos CalendarPos) map[sim.ZoneId]ZoneSeason
```

`Load` returns `(Tracks, []error)` — it loads what it can and reports every
problem, rather than failing on the first bad file.

## Blending

`Resolve` returns a blend in `[0, 1]` measuring how far into the current season
the day is; `lerp(previous, current, blend)` produces the climate. Early in a
season the previous one still dominates, which is what makes the transition
feel like weeks of changing weather rather than a switch.

A season's months must be **contiguous** (`contiguousStart` handles wrapping
across the year boundary). A track with a gap is a load error.

## Gotchas

- **`Load` takes `monthsPerYear` and `daysPerYear` as arguments** rather than
  reading config, because the calendar is configurable. Passing the wrong
  values silently shifts every season.
- **Multipliers are per weather type** (`mult`), and a missing key means "no
  change", not zero. Do not treat an absent multiplier as suppression.
- **This package is vendored from a standalone weather repo** and its purity is
  enforced by architecture tests. It must not import engine packages —
  `internal/` never imports `modules/`, and the weather crawler keeps its own
  zone graph for the same reason. See the header of
  `internal/mapper/mapper.crosszone.go` for why unification was rejected.

## Dependencies

`modules/weather/sim` and the standard library only.

## Consumers

`modules/weather/engine`.
