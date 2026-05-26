# Conversation YAML schema (chunk 3.6)

Conversations are relationship-keyed exchanges between NPCs drawn from
a type-pool or pair-override library at
`_datafiles/world/dogmud/conversations/`. Type pools organize generic
exchanges by relationship type (friend, rival, etc.). Optional pair
overrides add per-pair-specific variations.

## Locations

- **Type pools:** `_datafiles/world/dogmud/conversations/types/<type>.yaml`
- **Pair overrides:** `_datafiles/world/dogmud/conversations/pairs/<lower>_<higher>.yaml`

where `<type>` matches a relationship type string and `<lower>` / `<higher>`
are mob IDs with the smaller ID listed first.

## Type Pool YAML shape

```yaml
id: <string>                         # must match filename ConvertForFilename(id)
description: <string>                # optional; used in admin debug output
exchanges:                           # at least one exchange
  - lines:
      - speaker: "A"                 # "A" (initiator) or "B" (partner)
        text: <string>               # NPC voice; max ~80 chars
      - speaker: "B"
        text: <string>
  - lines:
      - speaker: "A"
        text: <string>
      - speaker: "B"
        text: <string>
subtypes:                            # optional sub-pool variation
  fond:                              # subtype key (warn if unknown)
    - lines:
        - speaker: "A"
          text: <string>
        - speaker: "B"
          text: <string>
  bitter:
    - lines:
        - speaker: "A"
          text: <string>
        - speaker: "B"
          text: <string>
```

## Pair Override YAML shape

```yaml
id: <string>                         # arbitrary identifier
mob_a: <int>                         # lower mob ID
mob_b: <int>                         # higher mob ID (mob_a < mob_b required)
exchanges:                           # same shape as type pool
  - lines:
      - speaker: "A"
        text: <string>
      - speaker: "B"
        text: <string>
```

Pair overrides extend the type pool — if no pair override exists for two
NPCs, they use the type pool for their relationship. If a pair override
exists, they draw from both the override's `exchanges` and the type pool.

## Filename conventions

**Type pools:** filename must equal `ConvertForFilename(id)`. Example:
`id: fond` → `fond.yaml`.

**Pair overrides:** filename is `<lower>_<higher>.yaml` where lower and
higher are mob IDs. Example: mobA=50, mobB=97 → `50_97.yaml`. The file
must be placed in `_datafiles/world/dogmud/conversations/pairs/`.

## Speaker semantics

- **"A"** is the **initiator-role** — the NPC who suggests the conversation.
- **"B"** is the **partner-role** — the NPC who responds.

The engine randomizes which physical NPC plays "A" per conversation start.
Scripts that bake mob names or gender into specific speaker roles lose
flavor 50% of the time. Author **role-agnostic scripts**: use generic
pronouns ("I", "you"), neutral topic language, and lines that flow
sensibly regardless of which NPC plays which role.

**Example of role-agnostic vs. role-locked:**

```yaml
# BAD: assumes a specific NPC or gender
- speaker: "A"
  text: "Say, Elena, have you finished that order?"
  
# GOOD: role-agnostic
- speaker: "A"
  text: "Have you finished that order we discussed?"
```

## Subtype convention

Subtypes allow flavor variation per relationship subtype string. Known keys
(**warn if encountered but unknown**):
- `fond` — warm, affectionate exchange
- `estranged` — cool, formal exchange
- `professional` — business-focused exchange
- `bitter` — hostile, sarcastic exchange

Runtime behavior: when a conversation starts, the system checks if both NPCs
have a shared subtype in the relationship definition (in their mob specs or
the relationship registry). If yes, pool exchanges from that subtype first;
if the subtype is missing or unknown, pool exchanges from the base `exchanges`
list. Unknown subtypes trigger a loader warning.

## Validation (load-time, panics)

- Filename must equal `ConvertForFilename(id)` (type pools only).
- At least one exchange in the base `exchanges` list.
- Each exchange has at least one `lines` entry.
- Each line has a valid `speaker` ("A" or "B") and non-empty `text`.
- **Pair override only:** `mob_a < mob_b`.
- **Pair override only:** no self-pair (mob_a != mob_b).
- Relationship edge exists between both mobs (relationship registry must
  contain an edge entry for the pair).

## Validation (load-time, warn-only)

- Single-line exchange (degenerate; at least two lines recommended).
- Line text exceeds ~80 characters (display wrapping concern).
- Pair override with missing or unresolved relationship edge.
- Subtype value not in `{fond, estranged, professional, bitter}`.
- Type pool with zero subtypes (subtype keys present but all empty).

## Runtime behavior

### Trigger

Per-tick chance `ConversationBaseChancePct` (default 1%) fires on each
fully-idle NPC. **Player-arrival boost:** `ConversationPlayerArrivalBoostPct`
(default 25%) when a player enters a room containing 2+ relateable,
fully-idle NPCs. (Relateable = both NPCs have an active relationship
edge in the registry.)

### Pacing

One line per round. The engine stamps a shared `conversation_line_idx`
MiscData counter on both NPCs; each round increments it deterministically
so line alternation (A, B, A, B, …) follows the stored counter, not per-NPC
state.

### Cooldown

After a complete exchange, both NPCs enter a `ConversationCooldownRounds`
cooldown (default 50 rounds) — they cannot start a new conversation during
this window. Cooldown is tracked via a MiscData key per NPC.

### Abort

If the partner moves room / sleeps / enters combat / starts a player dialogue,
the conversation aborts gracefully with no cooldown applied (the initiator
resumes attempting new conversations immediately on the next idle tick).

## Runtime gating

Conversations only fire when **both NPCs are fully idle**:
- Not in combat (`Character.Aggro == nil`)
- Not sleeping (no Sleeping buff)
- Not in an existing conversation
- Not on cooldown (outside the cooldown window)
- Both in the same room
- Relationship edge exists (type-keyed in registry)

## Pilot: Thornwall tavern back-room

Four NPCs with a defined friend-relationship network:
- Dal (mob 87) ↔ Fen (mob 88): friend
- Dal ↔ Gobb (mob 89): friend
- Fen ↔ Gobb: friend
- Wrex (mob 90) ↔ Gobb: friend
- Optional: Fen ↔ Wrex: rival (subtype: bitter)
- Optional: Dal ↔ Wrex pair override (per-pair exchanges)

Exchanges drawn from:
- Type pool: `friend.yaml` (generic friend interactions)
- Type pool: `rival.yaml` (if Fen↔Wrex subtype=bitter and available)
- Pair override: `87_90.yaml` (if Dal↔Wrex pair-specific lines defined)

## See also

- Spec: `docs/superpowers/specs/2026-05-25-mob-aliveness-3.6-npc-conversations-design.md`
- Loader: `internal/conversations/loader.go`
- Trigger: `internal/hooks/NewRound_IdleMobs.go` (TryStart call)
- Executor: `internal/hooks/NewRound_IdleMobs_conversations.go`
- Player-arrival boost: `internal/usercommands/go.go`
- Relationship registry: `internal/relationships/`
