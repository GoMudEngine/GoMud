# Time Module Context

## Purpose

`modules/time` is the player-facing `time` command: it renders the in-world
calendar — date, hour, day or night, and the moons — from `internal/gametime`.

The module is one file and one function. All the calendar logic lives in
`internal/gametime`; this is presentation only.

## API

```go
func TimeCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)
```

## Gotchas

- **Do not put calendar arithmetic here.** Anything computing a period, a
  phase, or a transition belongs in `internal/gametime` where it can be reused
  and tested; this module should only format what that package returns.
- **Moon phases are floats in `[0, 1]`, not names.** Rendering them means
  bucketing (`CurrentMoonFlavorBucket`) — do not print the number.
- **Use `GetAllPhases()`** rather than three single-moon calls; each one
  recomputes all three.

## Dependencies

`plugins`, `internal/gametime`, `users`, `rooms`, `events`.

## Consumers

Registered as a plugin.
