# DOGMud NPC↔NPC Conversation System Context

## Overview

The conversations package provides relationship-keyed idle dialogue between
NPCs. Two mobs with an active relationship edge can randomly exchange 2-4
line conversations drawn from a library organized by relationship type
(friend, rival, professional, etc.). Pair overrides allow per-pair-specific
scripting.

The system decouples from `internal/mobs/` via a `MobConversant` interface
to keep the import graph acyclic; `internal/conversationadapter` is the
bridge.

## Public API Surface

### Main Entry Points

```go
// Load all type pools and pair overrides from YAML
func Load()

// Set up DI hooks for world validation (mob existence, relationship edge lookup)
func SetConversationWorldValidator(mobExists func(int) bool,
                                   relationshipBetween func(int, int) *relationships.Edge)

// Attempt to start a conversation between initiator and one randomly-chosen
// partner from the room's fully-idle, relateable mobs
func TryStart(initiator MobConversant, roomMobIds []int) bool

// Explicit pair entry point; returns false if gating fails
func TryStartBetween(a, b MobConversant) bool

// Per-round driver; increments line index and fires mob commands
func TickConversation(self MobConversant, partnerId int) bool

// Finalize a conversation (clear state, apply cooldown)
func FinishConversation(mobA, mobB MobConversant)

// Abort (no cooldown applied)
func AbortConversation(mobA, mobB MobConversant)
```

### Read Accessors

```go
// Fetch type pool by relationship type (e.g., "friend")
func GetPool(t relationships.Type) *Pool

// Fetch pair-specific override if present; nil if not found
func GetPairOverride(mobA, mobB int) *PairOverride

// Query conversation state (prefer MobConversant methods)
func IsInConversationChar(c *characters.Character) bool
func IsOnCooldownChar(c *characters.Character) bool
```

### MobConversant Interface

Any type with this shape can participate in conversations. Decouples the
conversations package from mobs:

```go
type MobConversant interface {
    GetId() int                              // Instance ID
    GetRoom() int                            // Current room ID
    GetCharacter() *characters.Character     // Reference to character
    Command(cmd string)                      // Execute a command
}
```

## Internal Flow

### Startup: `Load()`

1. Walk `_datafiles/world/dogmud/conversations/types/` and load all
   type-pool YAMLs into a registry map (keyed by relationship type).
2. Walk `_datafiles/world/dogmud/conversations/pairs/` and load all
   pair-override YAMLs; store them in a pair map (keyed by sorted
   `(mobA, mobB)` tuple).
3. Call validators (panics on filename mismatch, missing edges, etc.).
4. Log summary: `conversations.Load() loadedCount=<types>/<pairs>`.

### Trigger: `TryStart(initiator, roomMobIds)`

1. Filter `roomMobIds` to mobs with an active relationship edge to initiator.
2. Filter to fully-idle mobs (no conversation, no combat, no sleep, off cooldown).
3. If none found, return false.
4. Randomize selection or pick first; call `TryStartBetween(initiator, partner)`.

### Picker: `TryStartBetween(a, b)`

1. Guard: both mobs same room, both idle, relationship edge exists, neither on cooldown.
2. Pick exchange uniformly from the union of:
   - Type pool for the relationship type (required)
   - Pair override exchanges (if the override exists for this pair)
3. If the NPCs' relationship has a subtype, prefer the subtype's exchange
   pool first; fall back to base `exchanges` if subtype missing.
4. Cache the selected exchange (copy all lines) into `ActiveExchange` keyed
   by synthesized exchange ID.
5. Stamp both mobs' MiscData: `conversation_partner_id`, `conversation_exchange_id`,
   `conversation_line_idx=0`.
6. Return true.

### Tick: `TickConversation(self, partnerId)`

1. Guard: both mobs still exist, still in same room, still idle.
2. Read `conversation_line_idx` from MiscData.
3. Compute the current line: `lines[idx]` from the cached exchange.
4. Compute the speaker: if `idx % 2 == 0`, speaker is "A"; else "B".
5. Determine who plays "A": the NPC with the lower ID.
6. Route the command to the correct NPC: `Command("say <text>")`.
7. Increment `conversation_line_idx` in both MiscData.
8. If incremented idx equals `len(lines)`, call `FinishConversation(a, b)`.
9. Return true if conversation is still active, false if finalized.

### Finalize: `FinishConversation(a, b)`

1. Clear both NPCs' MiscData: conversation keys.
2. Apply cooldown: stamp `conversation_cooldown_round` on both to
   `util.GetRoundCount() + ConversationCooldownRounds`.
3. Unset `IsInConversation` state.

### Abort: `AbortConversation(a, b)`

1. Clear both NPCs' MiscData: conversation keys.
2. Do **not** apply cooldown.
3. Unset `IsInConversation` state.

## MiscData Keys

Eight MiscData keys are used for state tracking. All keys are string
constants exported from the package:

| Key | Type | Semantics |
|-----|------|-----------|
| `ConversationPartnerId` | int | NPC instance ID of the conversation partner |
| `ConversationExchangeId` | string | Synthesized ID for the cached exchange (constructed at conversation start) |
| `ConversationLineIdx` | int | Current line index (0-based); shared with partner; increments each round |
| `ConversationCooldownRound` | uint64 | Round number at which cooldown expires; `> currentRound` blocks new conversations |
| `ConversationState` | string | enum-like ("active", "finalizing", "aborted") for edge-case tracing |
| `ConversationInitiator` | int | Which NPC instance started the conversation (bookkeeping; not load-bearing) |
| `ConversationStartRound` | uint64 | Round number at conversation start (bookkeeping; log only) |
| `ConversationLastTickRound` | uint64 | Last round we ticked this conversation (guard against double-tick) |

## Active-Exchange Cache

At conversation start, a shallow copy of the selected `Exchange` is stored
in an in-memory cache, keyed by a synthesized `exchange_id` (e.g.,
`"friend-7"` for the 7th exchange in the friend pool). This cache lives
for the duration of the exchange; when the conversation finishes, the entry
is deleted.

Cache is accessed by `TickConversation` to fetch the current line without
re-searching the registry. Cheap operation; small universe (2-4 lines per
conversation).

## Abort Triggers

Conversation silently aborts (no cooldown) when:

1. **Partner moved room** — partner's `GetRoom()` != initiator's
2. **Partner sleeps** — partner has Sleeping buff
3. **Partner enters combat** — partner's `Character.Aggro != nil`
4. **Partner starts player dialogue** — (future: hooks from dialogue engine)
5. **Line out of range** — `conversation_line_idx >= len(lines)` (shouldn't happen; guard)
6. **Partner disappeared** — partner lookup returns nil (mob despawned)

Detection happens in `TickConversation` at round start. On abort, call
`AbortConversation`, log a one-line trace, and let the initiator resume
normal idle ticking.

## Config Knobs

All live under `Balance` in `_datafiles/config.yaml`:

| Knob | Default | Purpose |
|------|---------|---------|
| `ConversationBaseChancePct` | 1.0 | Per-tick % chance a fully-idle NPC attempts to start a conversation. |
| `ConversationPlayerArrivalBoostPct` | 25 | When a player enters a room with relateable, idle NPCs, % chance to start one. |
| `ConversationCooldownRounds` | 50 | Cooldown (rounds) applied to both NPCs after a conversation completes. |

## Hook Integration Points

**Trigger wiring:** `internal/hooks/NewRound_IdleMobs.go` calls
`conversations.TryStart(mob, roomMobIds)` after the patrol branch and before
the path-walker. If TryStart returns true, skip normal idle command dispatch
for this round (NPC is busy talking).

**Tick wiring:** Same hook calls `conversations.TickConversation(mob, partnerId)`
each round while the mob is in a conversation, returning false when the
exchange finishes.

**Player-arrival boost:** `internal/usercommands/go.go` calls
`conversations.TryStart(initiator, room.GetMobIds())` with elevated %
chance (the `ConversationPlayerArrivalBoostPct` knob is read and applied
as a per-attempt override).

**Relationship edge validation:** At load time, `SetConversationWorldValidator`
is called from `main.go` with DI callbacks so the package can validate that
all mobs and edges exist without importing `mobs` or `relationships`.

## Dependencies

- `internal/relationships` — relationship type constants, Edge lookup
- `internal/characters` — Character interface (buff checks)
- `internal/util` — `GetRoundCount()`, logging
- `internal/fileloader` — YAML parsing (via loader.go)
- `sync` — read-write lock for registry

## Testing Hooks

Two test-registration functions:

```go
// Register a test pool (bypasses file I/O)
func RegisterPoolForTest(t relationships.Type, pool *Pool)

// Unregister
func UnregisterPoolForTest(t relationships.Type)
```

Use in unit tests to inject mock pools without a full load.

## Future Scope

**Opinion store:** NPC↔NPC relationships tracked via a persistent store
(what NPC A thinks of NPC B, including spoken-about-you gossip). Deferred
beyond chunk 3.6.

**Conversation chains:** Multi-exchange sequences triggered by ambient
conditions (e.g., a thief overhears merchant complaints). Deferred.

**Dynamic topics:** Conversations referencing world events or quest state.
Deferred.
