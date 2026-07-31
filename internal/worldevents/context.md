# World Events Context

## Purpose

`internal/worldevents` is a ring buffer of notable things that have happened in
the world — deaths, discoveries, economic swings, faction shifts — kept so that
UI surfaces (the web dashboard, "what's been going on" queries) can show recent
history without every producer maintaining its own log.

This is **not** the engine event bus. `internal/events` is the synchronous
pub/sub the game runs on; this package is a passive, queryable record.

## API

```go
type WorldEventType int
type Significance int

type WorldEvent struct { /* type, significance, actors, zone, text, round */ }
type WorldEventFilter struct { /* type / significance / zone constraints */ }

func InitWorldEvents()
func EmitWorldEvent(evt WorldEvent)
func GetRecentWorldEvents(n int, filter *WorldEventFilter) []WorldEvent
```

`Significance` is what keeps the feed readable: a routine mob death and a
faction war are both world events, and the consumer filters by importance
rather than by type.

## Gotchas

- **`InitWorldEvents` must run before the first emit.** Emitting into an
  uninitialised buffer is not useful and is not diagnosed for you.
- **The buffer is bounded and in memory.** Events are lost on restart and old
  ones are overwritten. Nothing here is an audit trail — if a subsystem needs
  durable history, it must persist its own.
- **`GetRecentWorldEvents` filters *after* selecting recent events**, so a
  narrow filter over a busy feed can legitimately return fewer than `n` results
  even though matching events exist further back.
- **A nil filter means no filtering.**

## Dependencies

`util` (round count) and `mudlog`.

## Consumers

`internal/web` (dashboard) and the world-event query commands.
