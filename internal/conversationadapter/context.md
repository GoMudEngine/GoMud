# Conversation Adapter Context

## Purpose

`internal/conversationadapter` exists for one reason: to keep the import graph
acyclic.

`internal/conversations` implements NPC↔NPC dialogue and needs to ask questions
about mobs — where are you, are you fighting, are you asleep, who is your
partner. It cannot import `internal/mobs` without creating a cycle. So
`conversations` declares a narrow `MobConversant` interface, and **this package
is the only place that knows about both**.

## API

```go
func AdaptMob(m *mobs.Mob) conversations.MobConversant
```

That is the whole public surface. `mobAdapter` implements every
`MobConversant` method by delegating to the real mob:

```go
ConvInstanceId() int          ConvMobId() int
ConvRoomId() int              ConvGetMiscData(key string) any
ConvSetMiscData(key string, val any)
ConvIsInCombat() bool         ConvHasBuffFlag(f buffs.Flag) bool
ConvAggro() bool              ConvPathLen() int
ConvPathCurrentNonNil() bool  ConvCommand(text string)
ConvGetPartner(instanceId int) conversations.MobConversant
```

The `Conv` prefix is deliberate — it marks these as interface-satisfying
shims rather than a second mob API, and it keeps them from colliding with the
real method names.

## Gotchas

- **Add nothing here but delegation.** Any logic placed in the adapter is
  invisible to both packages that think they own the behaviour. Conversation
  rules belong in `conversations`; mob behaviour belongs in `mobs`.
- **Widening `MobConversant` is a design decision, not a convenience.** Each new
  method is another piece of mob surface the conversation layer depends on.
  Prefer passing the value in.
- **`ConvGetPartner` returns an adapted mob, not a raw one** — the interface is
  closed over itself so the conversation layer never sees a `*mobs.Mob`.
- **Idle gating is checked through several of these at once** (combat, aggro,
  path state, buff flags). A conversation only fires when a mob is fully idle,
  so a new "busy" state needs a corresponding predicate here or NPCs will chat
  through it.

## Dependencies

`mobs`, `buffs`, `conversations`. By construction it is the only package
importing both sides.

## Consumers

`internal/hooks`, which drives the per-tick conversation attempts.
