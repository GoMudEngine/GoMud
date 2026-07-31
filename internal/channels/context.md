# Channels Context

## Purpose

`internal/channels` is the registry of global chat channels and the rules for
who hears them. It is a lookup table plus two predicates — the actual message
delivery lives in `messaging`.

## API

```go
type Channel struct { /* name, aliases, colour, config key */ }

func All() []Channel
func Get(name string) (Channel, bool)
func Enabled(cfgValue any) bool
func ShouldReceive(isSender, deafened bool, cfgValue any) bool
```

## The `ShouldReceive` contract

Three inputs decide delivery:

- **`isSender`** — the speaker normally sees their own message even when their
  own subscription would exclude them.
- **`deafened`** — the `deafen` moderation state suppresses receipt.
- **`cfgValue`** — the player's per-channel setting, typed `any` because it
  arrives from the generic config bag.

Keeping all three in one function is deliberate: every channel that
hand-rolled this logic got at least one case wrong.

## Gotchas

- **`cfgValue` is `any` and `Enabled` interprets it loosely.** An unset value is
  not the same as `false`; check `Enabled`'s behaviour before assuming a default.
- **The registry is fixed at build time.** Adding a channel means adding a
  `Channel` entry, a config key, and a command — the registry alone does not
  create one.
- **`Get` matches the canonical name.** Alias resolution belongs to the caller.

## Dependencies

`configs`.

## Consumers

`internal/usercommands` (the chat commands), `internal/messaging`, and the web
client's channel list.
