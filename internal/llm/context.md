# LLM Context

## Purpose

`internal/llm` is an optional, asynchronous bridge to a local Ollama instance
for generating NPC dialogue that has no authored answer. It is a **flavour**
feature: the game must work identically when it is disabled or unreachable.

## Files

- **types.go** — `LLMProfile`, `ConversationContext`, and the Ollama wire types.
- **client.go** — `AskAsync`.
- **cache.go** — the response cache and the per-mob pending flag.

## API

```go
type LLMProfile struct         { /* model, persona, generation options */ }
type ConversationContext struct { /* who is talking, about what, with what history */ }

func AskAsync(...)
```

The API is **async-only**. There is no blocking `Ask`, deliberately — a game
round must never wait on a model.

## Caching and pending state

```go
checkCache(mobInstanceId, topic) (string, bool)
storeCache(mobInstanceId, topic, response, ttl)
isPending(mobInstanceId) bool
setPending(mobInstanceId, val)
```

The cache is keyed by `(mob instance, topic)` with a TTL, so repeatedly asking
one NPC the same thing costs one generation. The **pending flag is per mob
instance**, not per topic: while a mob is waiting on a generation it will not
start another, which bounds concurrent requests to one per NPC.

## Gotchas

- **Both callbacks run on the async goroutine, under `util.LockMud()`.**
  `AskAsync` takes the mud lock around `onResponse` *and* `onUnavailable` before
  invoking them, because every real implementation touches shared game state
  (`mobs.GetInstance`, `dialogue.Load` / `ShiftMood`, `mob.Command`) and the
  `internal/dialogue` caches are unguarded maps that `MainWorker` writes. So:
  call mob/room/dialogue functions directly from a callback, and **never** take
  the mud lock inside one — `mudLock` is not reentrant and double-locking hangs
  the whole server.
- **Every failure path must degrade to authored dialogue.** A timeout, a
  refused connection, or a disabled config must produce the normal fallback
  line, never an error shown to a player.
- **Responses arrive after the player's turn.** The reply is delivered whenever
  it lands, so an NPC may answer a beat late, or after the player has left. The
  caller must handle a recipient who is no longer there.
- **The cache is in memory and unbounded by count** — only by TTL. A long
  uptime with many NPCs accumulates entries.
- **Generated text is not validated.** It bypasses the authoring conventions
  the rest of the game is held to (voice, line width, no raw numbers), so keep
  it to low-stakes flavour and never to quest-critical information.

## Dependencies

`configs`, `mudlog`, plus `net/http` and JSON. No dependency on `mobs` or
`users` — context is passed in.

## Consumers

`internal/dialogue` fallback paths, gated on config.
