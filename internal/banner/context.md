# Banner Context

## Purpose

`internal/banner` formats the framed, centred announcement blocks the game
shows for milestone moments — a mutation emerging, a tier change, an
achievement. One place owns the box drawing and centring so every milestone
looks the same.

## API

```go
type Kind int
type TierChange struct { /* from → to */ }

func Format(kind Kind, name string, tier *TierChange) string
```

`tier` is optional — pass nil for a milestone with no before/after.

## Gotchas

- **Centring assumes an 80-column client.** `center` pads to the project's
  fixed MUD line width; a banner built from a longer name will not wrap
  gracefully, it will overflow. Keep names short.
- **The returned string contains ANSI tags, not raw escapes.** Do not measure
  its length for layout — the tags are not printable width.
- **`Format` returns text; it does not send.** Delivery is the caller's job,
  which is what lets the same banner go to a player, a room, or a log.

## Dependencies

`colorpatterns` and the ansitags conventions.

## Consumers

`internal/mutations` (emergence and apex reveals), `modules/achievements`, and
progression milestone paths.
