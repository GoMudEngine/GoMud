# Splash Context

## Purpose

`internal/splash` decides **who sees a full-screen splash** — the large ASCII-art
moments such as a mutation reveal. It is an audience-resolution package: given a
`Splash` describing the event and its target scope, it returns the users who
should receive it.

## API

```go
type SplashTarget uint8
type Splash struct { /* target scope, originating user, zone, payload */ }

func (Splash) Type() string          // "Splash" — satisfies the event interface
func Recipients(s Splash) []*users.UserRecord
```

`Recipients` resolves the target scope to a concrete user list; `filterByZone`
is the zone-scoped case.

## Gotchas

- **This package chooses the audience, not the art.** The splash content is
  supplied by the caller; changing what a reveal looks like is not a change
  here.
- **Recipients are resolved at send time.** A player who arrives a round later
  will not see it — splashes are moments, not state.
- **Splash text can interleave with other output.** A reveal line landing in the
  middle of a splash is a known, accepted quirk rather than a bug; do not
  "fix" it by serialising output globally.
- **`Type()` exists so a `Splash` can travel on the event bus.** It is not a
  string you should switch on outside that context.

## Dependencies

`users`, `events`.

## Consumers

`internal/mutations` (the reveal path) and `internal/hooks`.
