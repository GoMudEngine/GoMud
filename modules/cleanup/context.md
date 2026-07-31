# Cleanup Module Context

## Purpose

`modules/cleanup` provides `trash` and `bury` — the two ways to remove clutter
from a room. Each is implemented twice, once for players and once for mobs, so
NPCs can tidy up after themselves without a separate mechanism.

## API

```go
func (c *CleanupModule) userTrashCommand(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)
func (c *CleanupModule) mobTrashCommand(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error)
func (c *CleanupModule) userBuryCommand(...) (bool, error)
func (c *CleanupModule) mobBuryCommand(...) (bool, error)
```

The paired user/mob signatures are the plugin API's shape: `AddUserCommand` and
`AddMobCommand` take different handler types, so a command available to both
is registered twice.

## Gotchas

- **These destroy items.** `trash` is irreversible; there is no bin to recover
  from.
- **Player and mob variants must be kept in step.** A rule added to one and not
  the other produces NPCs that can do something players cannot, or vice versa —
  the project tracks this as command parity and warns about unpaired commands
  at boot.
- **`bury` is corpse-oriented, `trash` is item-oriented.** They read similarly
  and are not interchangeable.

## Dependencies

`plugins`, `events`, `users`, `mobs`, `rooms`, `items`.

## Consumers

Registered as a plugin.
