# Follow Module Context

## Purpose

`modules/follow` implements `follow` for both players and mobs: a follower
moves when its target moves. It is a thin relationship map plus a large set of
event listeners whose real job is **breaking the link at the right moments**.

## Types

```go
type followId struct { /* user or mob identity in one value */ }
type FollowModule struct { /* target → followers */ }
```

`followId` unifies users and mobs so one map serves both, with
`getFollowIdInstance()` resolving back to whichever is real.

## Behaviour

```go
startFollow(target, source followId, cutoffRound ...uint64)
stopFollowing(source followId) followId
loseFollowers(target followId) []followId
isFollowing(followId) bool
getFollowers(target followId) []followId
```

`startFollow` takes an optional **cutoff round** — a follow that expires — which
is how temporary escorting is expressed without a separate mechanism.

## Where the work actually is

The listener set is the substance of the module:

- `roomChangeHandler` — the actual following.
- `onNewRound` — expiry and reconciliation.
- `onPartyChange` — party membership changes the relationship.
- `idleMobHandler` — a following mob acts on its own idle tick.
- `playerDespawnHandler`, `onPlayerDeath`, `onMobDeath` — break the link.

Every one of those exists because a stale follow edge produces a mob trailing a
player who no longer exists, or a follower that cannot be shaken off.

## Gotchas

- **Any new way for an actor to leave the world needs a listener here.**
  Despawn, death, and disconnect are covered; a future teleport-out or
  instance-transfer is not automatically.
- **`loseFollowers` returns the dropped followers** so the caller can message
  them. Discarding the return leaves players silently un-followed.
- **Follow is not the same as party.** They interact (`onPartyChange`) but a
  party member is not automatically a follower.
- **Mobs follow via their idle tick**, so a followed player moving faster than
  the mob's tick can outrun it. That is intended.

## Dependencies

`plugins`, `events`, `users`, `mobs`, `rooms`.

## Consumers

Registered as a plugin; provides the user and mob `follow` commands.
