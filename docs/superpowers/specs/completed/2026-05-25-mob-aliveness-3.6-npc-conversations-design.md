# Mob Aliveness 3.6 — NPC↔NPC Idle Conversations (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.6 (Phase 3 — Routine layer)
**Size:** M
**Branch:** `feature/mob-aliveness-3.6-npc-conversations`
**Depends on:** 1.6 (NPC relationships, shipped 2026-05-09)

## Goal

Two related NPCs in the same room occasionally exchange a short
2-4 line conversation, drawn from a relationship-type-keyed
library with optional per-pair overrides. The player witnesses
real back-and-forth dialogue between townspeople instead of two
independent monologues. The world feels woven, not flat.

This is the qualitative leap that chunk 3.2's per-segment
`say` idlecommands can't deliver — even with authored sayings,
chunk 3.2 NPCs never actually talk *to each other*. 3.6
introduces interactive exchanges gated by the chunk 1.6
relationship graph and chunk 3.2/3.3/3.4 life-state checks.

## In scope

1. New `internal/conversations/` package holding types, loader,
   registry, picker, and the per-conversation state machine.
2. New conversation YAML format in two subdirectories under
   `_datafiles/world/dogmud/conversations/`:
   - `types/<relationship-type>.yaml` — generic pools for each
     of the six relationship types (`family`, `friend`, `rival`,
     `lover`, `employer`, `employee`)
   - `pairs/<smaller-mob-id>_<larger-mob-id>.yaml` — per-pair
     overrides (extend the type pool, don't replace)
3. Per-tick trigger in `NewRound_IdleMobs` (default 1% chance
   per idle NPC per tick) — low base rate so conversations feel
   organic and unaware of the player.
4. Player-arrival boost in `internal/usercommands/go.go`
   post-arrival hook (default 25% chance when arriving in a
   room with 2+ NPCs that have at least one relationship edge
   between them) — biases conversations toward player presence
   without making the world go silent when unobserved.
5. Multi-round exchange state machine — line-per-round pacing,
   stamps state on both NPCs' MiscData, drives subsequent
   speaker rotations until the exchange completes.
6. Per-NPC cooldown after exchange completes (default 50
   rounds) — prevents back-to-back chatter from the same NPC.
7. Subtype sub-pools — pool YAMLs can define
   `subtypes.<key>.exchanges` lists; when the initiator's
   relationship subtype matches a key, the engine draws from
   `default ∪ matching_subtype` (additive, not replacing).
8. Load-time validation: filename↔id, pool id is a known
   relationship type, pair mob ids resolve, pair has an actual
   relationship edge, every exchange has ≥1 line, each line
   has a valid speaker (A/B) and non-empty text.
9. Three config knobs: `ConversationBaseChancePct` (1.0),
   `ConversationPlayerArrivalBoostPct` (25),
   `ConversationCooldownRounds` (50).
10. Pilot content: four Thornwall tavern back-room NPCs (Dal +
    Fen + Gobb + Wrex) gain relationship edges; new `friend`
    type pool (10-15 exchanges); optional small `rival` pool
    for variety; optional per-pair override for Dal↔Wrex.
11. Documentation: schema doc, package context.md, hooks
    context.md note, CLAUDE.md subsection, helpfile append.
12. Smoketester goal file that observes the tavern back room
    for 20-30 rounds and asserts at least one full exchange.

## Out of scope

- **NPC↔NPC opinion store** — `internal/opinions` is NPC→player
  only. v1 "mood" = relationship type + life-state gating + the
  optional subtype sub-pools. Future work can add a NPC↔NPC
  opinion delta if richer "they're annoyed with each other
  today" semantics matter.
- **"Spoken about you" gossip** — explicit per roadmap. Player-
  overheard conversations are flavor-only in v1; "the guard
  tells the baker about your crime" needs chunk 5.1 (Town
  Justice) crime hooks.
- **Cross-room conversations** — same-room only. No shouting
  across rooms, no overheard-from-next-door propagation.
- **Player joining a conversation** — they can only overhear.
  No "interject" or "reply" verb in 3.6.
- **Multi-party (3+) conversations** — strict pairs only.
- **Conversation chains** — one exchange completes, then both
  NPCs go on cooldown; no follow-on exchange immediately
  chained to the previous one.
- **Translation / language barriers** — assume common speech.
- **Conversation persistence across server restart** —
  in-progress conversations dropped on restart (NPCs go idle
  on respawn anyway; transient MiscData on both NPCs is wiped).
- **Player-tunable global mute** — if a player finds chatter
  annoying, the existing room "Deafened" or chat-suppress
  flag should cover it. No new mute knob for conversations
  specifically.
- **Admin inspector for live conversation state** — could
  extend `mob schedule <instId>` to show conversation state,
  but defer. Smoketester observation is the v1 verification.
- **Crafter NPCs talking shop while working** — out of scope.
  Crafter ticks happen inside `activity: craft` segments; a
  conversation trigger would compete with the crafter tick.
  v1 gates conversations on "fully idle" — no patrol, no
  craft, no sleep, no in-progress conversation, no player
  dialogue.

## Architecture

### Data model

**Conversation types** in new file
`internal/conversations/conversation.go`:

```go
type ConversationLine struct {
    Speaker string `yaml:"speaker"` // "A" or "B"; A is the initiator
    Text    string `yaml:"text"`
}

type Exchange struct {
    Lines []ConversationLine `yaml:"lines"`
}

type Pool struct {
    Id          string             `yaml:"id"`          // e.g., "family"
    Description string             `yaml:"description,omitempty"`
    Exchanges   []Exchange         `yaml:"exchanges"`
    Subtypes    map[string]Subpool `yaml:"subtypes,omitempty"`
}

type Subpool struct {
    Exchanges []Exchange `yaml:"exchanges"`
}

type PairOverride struct {
    Id        string     `yaml:"id"`        // unique label, free-form
    MobA      int        `yaml:"mob_a"`     // template mob id
    MobB      int        `yaml:"mob_b"`     // template mob id
    Exchanges []Exchange `yaml:"exchanges"`
}

// Package-level registries, populated by Load() at startup.
var (
    poolsMu sync.RWMutex
    pools   = map[relationships.Type]*Pool{}

    pairsMu sync.RWMutex
    pairs   = map[pairKey]*PairOverride{}
)

type pairKey struct {
    LowId  int
    HighId int
}

func GetPool(t relationships.Type) *Pool
func GetPairOverride(mobA, mobB int) *PairOverride
```

Both lookups normalize the pair to sorted-id order so call-site
order doesn't matter.

### Pool YAML example

`_datafiles/world/dogmud/conversations/types/friend.yaml`:

```yaml
id: friend
description: "Generic friend banter — small talk, gossip, complaints."
exchanges:
  - lines:
      - speaker: A
        text: "How's the brew today?"
      - speaker: B
        text: "Same as ever, thank Aen."
      - speaker: A
        text: "Could be worse."
  - lines:
      - speaker: A
        text: "Bones aching with the weather?"
      - speaker: B
        text: "Aye. Cold mornings tell."
      - speaker: A
        text: "Hot tea fixes more than the apothecary admits."
      - speaker: B
        text: "Truer words."

subtypes:
  fond:
    exchanges:
      - lines:
          - speaker: A
            text: "Always good to share a quiet hour with you."
          - speaker: B
            text: "Same, my friend. Same."
  estranged:
    exchanges:
      - lines:
          - speaker: A
            text: "..."
          - speaker: B
            text: "..."
          - speaker: A
            text: "Hmm."
```

### Pair-override YAML example

`_datafiles/world/dogmud/conversations/pairs/116_117.yaml` —
override for Dal (117) and Wrex (116):

```yaml
id: dal_and_wrex
mob_a: 116
mob_b: 117
exchanges:
  - lines:
      - speaker: A
        text: "The usual, Wrex?"
      - speaker: B
        text: "You know me too well, lass."
  - lines:
      - speaker: A
        text: "Mind the spill on table three — I'll get to it."
      - speaker: B
        text: "Take your time. We've got nowhere to be."
```

Filename convention: `<smaller-mob-id>_<larger-mob-id>.yaml`.
The pair-override loader normalizes mob_a/mob_b to sorted order
in the registry key so the filename order is the source of
truth.

### Loader & validation

**Loader** lives in `internal/conversations/loader.go`. Called
from `mobs.LoadDataFiles()` AFTER mob templates have loaded
(so cross-checks against mob ids work). Mirrors the chunk
3.2/3.4 loader pattern.

**Load-time validation (panic):**

| Check | Rule |
|---|---|
| `types/*.yaml` filename ↔ id | `ConvertForFilename(id)` equals filename |
| Pool id is a known relationship type | Must be one of `family`, `friend`, `rival`, `lover`, `employer`, `employee` (mapped via `relationships.Type` constants) |
| Each exchange has ≥1 line | Empty `lines:` list rejected |
| Each line's `speaker` is `"A"` or `"B"` | Anything else rejected |
| Each line has non-empty `text` | Empty strings rejected |
| `pairs/*.yaml` filename matches sorted-id convention | `<min(mob_a, mob_b)>_<max(mob_a, mob_b)>.yaml` |
| Pair `mob_a != mob_b` | No self-conversations |
| Pair `mob_a` and `mob_b` resolve to loaded mob templates | Cross-check via mobs registry |
| Pair has at least one relationship edge | `relationships.RelationsBetween(mob_a, mob_b)` returns non-empty |

**Load-time warnings (non-fatal, logged once):**

- Subtype key in a pool that's not in a documented convention
  (free-string, but warn for typo detection — known good
  subtypes: `fond`, `estranged`, `professional`, `bitter`).
- Exchange with only 1 line (monologue, not a conversation).
- Exchange with >6 lines (player tab-out risk).
- Any `text` line longer than ~78 characters (per CLAUDE.md
  MUD line-width — warn but accept).
- Pool with zero exchanges (the relationship type will be a
  no-op until exchanges are added).

### Trigger mechanism

**Per-tick trigger** — runs in `NewRound_IdleMobs` after the
chunks 3.2/3.3/3.4 schedule/sleep/patrol branches, before the
existing path-walker. Sits behind a guard that requires the
NPC to be fully idle:

```go
// Chunk 3.6: idle conversation trigger.
// Requires the NPC to be fully idle this tick:
//   - not in combat
//   - not asleep
//   - not currently in a conversation (own state)
//   - not currently being talked to by a player
//   - not patrolling between waypoints (path queue non-empty)
//   - not transitioning schedules this tick (SegmentChanged)
if conversationsEligible(mob, plan, schedulePlan) {
    if util.Rand(10000) < int(configs.GetBalanceConfig().
        ConversationBaseChancePct*100) {
        conversations.TryStart(mob, room)
    }
}
```

`conversationsEligible` is a small helper combining the gates.
`ConversationBaseChancePct` is a `float64`-equivalent config
knob; default `1.0` → 1% per tick → roughly one trigger attempt
per 100 ticks per idle NPC (≈ once per 400 sec at default
tick rate).

**Player-arrival boost** — in `internal/usercommands/go.go`
post-arrival hook, after the existing chunk 3.3 light-source
wake block:

```go
// Chunk 3.6: player-arrival boost. If the arriving player
// lands in a room with 2+ relateable NPCs that aren't already
// busy, roll once at the higher chance for an opening exchange.
pairs := conversations.FindRelateablePairsInRoom(destRoom)
if len(pairs) > 0 {
    if util.Rand(100) < int(configs.GetBalanceConfig().
        ConversationPlayerArrivalBoostPct) {
        // Pick a random eligible pair.
        a, b := pairs[util.Rand(len(pairs))].A, pairs[util.Rand(len(pairs))].B
        conversations.TryStartBetween(a, b, destRoom)
    }
}
```

`FindRelateablePairsInRoom` is a helper that enumerates
unordered (mobA, mobB) pairs in the room where both are fully
idle and `relationships.AreRelated(a, b)` is true.

### `TryStart` + state machine

```go
// TryStart attempts to start a conversation initiated by `mob`.
// Picks a random in-room related partner, picks a pool +
// exchange (uniform draw from type pool ∪ matching subtype ∪
// pair override), stamps state on both NPCs, fires the first
// line immediately. No-op if no eligible partner / both on
// cooldown / no exchanges for the relationship type.
func TryStart(initiator *mobs.Mob, room *rooms.Room)

// TryStartBetween skips partner selection (caller has already
// identified the pair). Used by the player-arrival boost.
func TryStartBetween(a, b *mobs.Mob, room *rooms.Room)
```

Both fan into:

```go
func startConversation(a, b *mobs.Mob, ex Exchange, room *rooms.Room)
```

which:
1. Picks one of the two NPCs as speaker A randomly (a single
   exchange's "A is the initiator" is conceptual — the picker
   doesn't care which was the rolling-NPC for the trigger).
2. Stamps MiscData on both NPCs (see keys below). Both NPCs
   start with `conversation_line_idx = 0`.
3. Fires the FIRST line immediately via
   `mob.Command(fmt.Sprintf("say %s", line.Text))` from
   whichever NPC is the speaker of `Lines[0]`.
4. Stamps `conversation_line_idx = 1` on **both NPCs** (shared
   progress counter — see state-machine note below).

**Per-NPC MiscData keys:**

| Key | Type | Purpose |
|---|---|---|
| `conversation_partner_id` | int | Mob instance id of the other NPC. 0 (or unset) means not in a conversation. |
| `conversation_role` | string | `"A"` or `"B"` (which speaker am I in this exchange). |
| `conversation_pool_id` | string | Pool id (relationship type) the exchange came from. |
| `conversation_pair_override_id` | string | Pair-override id if applicable; empty if exchange came from a type pool. |
| `conversation_exchange_id` | int | Index into the chosen exchange list. Stable for the duration of the exchange. |
| `conversation_line_idx` | int | The index of the NEXT line in the exchange (SHARED progress counter — both NPCs stamp it to the same value when a line fires). |
| `conversation_last_round` | uint64 | Round count when I last spoke. Gates the "one line per round" pacing. |
| `conversation_cooldown_until_round` | uint64 | Rounds before which I won't initiate or accept a new exchange. |

All keys are transient (mob MiscData is not persisted for non-
essential NPCs). On respawn / server restart, the state is
wiped → no in-progress conversation persists. NPCs return to
idle on respawn, and the next trigger may start fresh.

**Per-tick state machine** runs in the same IdleMobs branch as
the trigger:

```go
// Chunk 3.6: drive an in-progress conversation if this NPC has one.
if partnerId := getMiscDataInt(&mob.Character, "conversation_partner_id"); partnerId > 0 {
    conversations.TickConversation(mob, partnerId, room)
}
```

`TickConversation` does:

1. **Resolve partner.** If partner instance is gone or in a
   different room → abort: clear MiscData on both, no
   cooldown (graceful — the partner left through no fault).
2. **Re-validate state.** If partner is in combat / asleep /
   mid-conversation-with-player → abort, no cooldown.
3. **Find the exchange and the next line.** Look up
   `conversation_pool_id` + `conversation_pair_override_id` +
   `conversation_exchange_id` → resolves to a specific
   `Exchange`. Read the shared `conversation_line_idx` to find
   `Lines[conversation_line_idx]`.
4. **Speaker gate.** If `Lines[conversation_line_idx].Speaker
   != conversation_role`, the line belongs to the partner —
   wait silently. (The partner's own TickConversation will
   fire it on its own tick, then advance the shared counter,
   and on the tick after that this NPC will see its own line
   come up.)
5. **Pacing gate.** If `currentRound == conversation_last_round`,
   we've already spoken this round. Skip (defensive — alternation
   normally prevents this).
6. **Fire the line:** `mob.Command(fmt.Sprintf("say %s", line.Text))`.
   Stamp `conversation_last_round = currentRound` on self.
   **Advance `conversation_line_idx` to `conversation_line_idx + 1`
   on BOTH NPCs** (write to partner's MiscData too — both have
   stable MiscData maps; `mobs.GetInstance(partnerId).Character.SetMiscData(...)`
   is safe under the per-tick serial IdleMobs loop).
7. **If this was the LAST line of the exchange:** clear all
   conversation MiscData on both NPCs, stamp
   `conversation_cooldown_until_round = currentRound + ConversationCooldownRounds`
   on both. The conversation is complete.

**Shared-counter rationale:** the line_idx is shared progress,
not per-NPC. Both NPCs stamp it together when a line fires.
The speaker gate at step 4 ensures only the right NPC speaks
each line; the partner waits one tick. Concretely for a 4-line
ABAB exchange starting at round R with speaker A first:
- Round R: A speaks line 0, both stamp line_idx=1.
- Round R+1: B speaks line 1, both stamp line_idx=2.
- Round R+2: A speaks line 2, both stamp line_idx=3.
- Round R+3: B speaks line 3, both stamp line_idx=4. Last line
  detected → MiscData cleared, cooldown stamped on both.

Net pacing: one line per game round, exchange complete in
`len(Lines)` rounds, deterministic.

**Abort triggers fire in step 1 or 2.** When aborting, the
remaining lines never fire — natural for the world (NPCs in
real life don't always finish their sentences either).

### Config knobs

Added to `internal/configs/config.balance.go`:

| Knob | Default | Purpose |
|---|---|---|
| `ConversationBaseChancePct` | `1.0` | Per-tick % chance per fully-idle NPC to attempt starting a conversation. 1% ≈ once per 100 ticks per NPC. |
| `ConversationPlayerArrivalBoostPct` | `25` | On player arrival in a room with relateable NPC pairs, % chance to start one. |
| `ConversationCooldownRounds` | `50` | Rounds before an NPC can start another conversation after one completes. Matches chunk 3.3's `ScheduleWakeGraceRounds` default for consistency. |

## Edge cases & failure modes

| Situation | Behavior |
|---|---|
| 3+ NPCs in room with multiple relationship edges | Per-NPC tick trigger picks ONE of their in-room related partners uniformly. Multiple conversations can run concurrently — like a real tavern. The partner-busy check serializes per-NPC (each NPC is in at most one conversation at a time). |
| Initiator and partner both roll a trigger on the same tick | `TryStart` checks partner is not already in a conversation. If both roll the same tick, the second one finds the first already stamped → skips. |
| Player walks in during an in-progress exchange | Remaining lines fire normally (room-level say broadcasts). Player-arrival boost check skips firing a NEW exchange if any room NPC is already mid-conversation. |
| Player walks OUT mid-exchange | Conversation continues uninterrupted between the NPCs. The player misses the rest. |
| Schedule transition mid-conversation (patrol → sleep) | The leaving NPC's schedule executor (chunks 3.2/3.4) starts pathing or applies the Sleeping buff. The conversation's next tick detects the mid-conversation-with-player / sleep / not-in-room state and aborts gracefully. |
| Same-pair conversation tries to start within cooldown | TryStart's cooldown check rejects; silently no-ops. |
| Conversation pool empty for the chosen relationship type | TryStart returns without firing. Load-time warning surfaces the empty pool. |
| Per-pair override exists but the pair's edge type doesn't match any pool | Per-pair exchanges still fire (they're additive); just no type-pool augmentation. Pair overrides are always eligible regardless of subtype matching. |
| Combat in an adjacent room | No effect; conversations only abort on direct gating triggers (same-room combat aggro on either participant). |
| Two players in the same room when conversation fires | Both witness it. Standard say semantics — visible to all room occupants. |
| NPC's `say` text triggers a player's dialog tree match | `say` emits room text but does NOT trigger NPC dialog handlers. Different verb than `ask`. Safe. |
| Server restart mid-conversation | MiscData wiped. NPCs return to idle on respawn. No partial-conversation persistence. Acceptable. |
| Charmed NPC in the room | Charmed NPCs don't participate in conversations (treated as not-idle by the eligibility check). |
| Mob instance respawns mid-conversation | The respawned instance has empty MiscData (fresh instance). The other NPC's next tick detects the partner is gone (instance id no longer matches) and aborts. |
| Cross-zone "relationship" edge (mob in different zone) | Same-room gate filters it out naturally. The pair will simply never fire unless both happen to be in the same room. Cross-zone NPCs reaching the same room is rare but legal. |

## Validation summary

**Load time (panic):**
- Filename↔id mismatch (pool or pair)
- Pool id is not a known relationship type
- Empty exchange (no lines)
- Speaker field is not "A" or "B"
- Empty `text` field
- Pair `mob_a` or `mob_b` doesn't resolve to a loaded mob
- Pair `mob_a == mob_b`
- Pair has no relationship edge

**Load time (warn-only, dedup'd):**
- Subtype key outside the documented convention
- Single-line exchange (degenerate)
- >6-line exchange (player tab-out risk)
- `text` line longer than 78 characters
- Pool with zero exchanges

**Runtime:** no per-tick warnings (the trigger paths silently
no-op when ineligible — the smoketester is the verification).

## Testing

### Unit tests

| File | Coverage |
|---|---|
| `internal/conversations/conversation_test.go` | Pool resolution (`GetPool` returns the right pool for each relationship type), pair-override lookup with sorted-key normalization, exchange picker (uniform draw across type pool ∪ matching subtype ∪ pair override) |
| `internal/conversations/loader_test.go` | All load-time panics fire correctly (bad pool id, empty lines, bad speaker, missing pair mobs, pair without relationship edge, filename mismatch). Warn-only cases log without panicking. |
| `internal/hooks/conversation_trigger_test.go` | `conversationsEligible` gates (sleep / combat / patrol / existing conversation / player dialogue all block). `TickConversation` advances line-by-line in correct order. Abort triggers fire correctly (partner leaves room, partner sleeps, partner combat). Cooldown stamps on completion. |

### Manual smoke pass

1. Boot. Confirm `conversations.LoadConversations() pools=N pairs=M` log line.
2. Enter the tavern back room (where Dal + Fen + Gobb + Wrex
   should all be present).
3. Stand idle. Within ~50-100 rounds expect to witness at
   least one full 2-4 line exchange between two of the four
   NPCs.
4. Use the player-arrival boost: leave the room, wait 10s,
   re-enter. Expect higher likelihood of immediate exchange.
5. Attack one of the NPCs mid-exchange. Confirm the
   conversation aborts cleanly (no further lines fire).
6. Wait for the cooldown to expire, observe a fresh exchange
   start later.

### Autonomous smoketester

New goal file `tools/testing/goals/3.6-conversation-observation.yaml`:

- Enter the tavern back room.
- Stand idle for 30 game rounds.
- Count overheard `say` broadcasts from any of the four target
  NPCs (Dal, Fen, Gobb, Wrex).
- Pass criteria: at least one full multi-line exchange
  observed; at least 2 different relationship-type pools fire
  (if rival edge added: friend + rival).
- Optional: enter and exit the room 3 times in 60 rounds,
  confirm player-arrival boost increases observed exchanges
  compared to passive observation.

## Documentation

| File | Change |
|---|---|
| `docs/schemas/conversation.md` (NEW) | Full schema reference for pools + pair overrides, validation rules, subtype convention, runtime behavior, composition with relationships |
| `internal/conversations/context.md` (NEW) | Package context: loader, registry, trigger, state machine, MiscData key list, abort triggers |
| `internal/hooks/context.md` | Append note on the conversation trigger + state machine in IdleMobs; mention the player-arrival boost in the go.go hook chain |
| `internal/configs/context.md` | Add three new config knob rows |
| `CLAUDE.md` | New "NPC↔NPC Conversations" subsection: relationship-keyed type pools + per-pair overrides, line-per-round pacing, life-state gates, deferred items (NPC↔NPC opinion store, gossip about player) |
| `_datafiles/world/dogmud/templates/help/ask.template` | Append one sentence: "You can sometimes overhear townspeople chatting with each other — pause in busy rooms to catch the gossip." |

## Pilot content (Thornwall tavern back room)

Three layers of content authoring:

### Relationship edges (mob YAML edits)

Author edges among Dal (mob 117), Fen (114), Gobb (115), Wrex
(116):

- Fen ↔ Gobb, Fen ↔ Wrex, Gobb ↔ Wrex — all `friend`
- Dal ↔ Fen, Dal ↔ Gobb, Dal ↔ Wrex — all `friend`
- (Optional flavor): Fen ↔ Wrex also `rival` — the two old
  men argue about which war they fought in. Chunk 1.6 supports
  multiple edge types between the same pair.

Total: 6 friend edges + 1 optional rival edge. Author one
direction per pair; chunk 1.6 auto-mirrors symmetric types.

### Conversation pools

**`_datafiles/world/dogmud/conversations/types/friend.yaml`** —
10-15 exchanges covering generic friend banter (weather, brews,
bones aching, gossip, life updates, weather complaints). Two
optional subtypes (`fond`, `estranged`) with 2-3 exchanges
each.

**(Optional) `_datafiles/world/dogmud/conversations/types/rival.yaml`** —
5-8 exchanges of pointed jabs and disagreements. Only used if
the rival edge between Fen and Wrex is added.

### Per-pair override

**`_datafiles/world/dogmud/conversations/pairs/116_117.yaml`** —
2-3 unique exchanges specific to Dal (117) and Wrex (116).
"The usual, Wrex?" / "You know me too well, lass." Extends
(does not replace) the friend pool.

## Commit shape

Suggested split:

1. `feat(conversations): package skeleton + types + registry`
2. `feat(conversations): YAML loader + validators`
3. `feat(hooks): conversation trigger + state machine in IdleMobs`
4. `feat(hooks): player-arrival boost in go.go`
5. `feat(configs): three Conversation* config knobs`
6. `feat(content): Thornwall tavern back room — relationship edges + friend pool`
7. `feat(content): (optional) rival pool + Dal↔Wrex pair override`
8. `docs: conversation schema + context.md + CLAUDE.md + helpfile`
9. `chore(roadmap): mark 3.6 Done`

Each commit independently reviewable.

`PATCH_NOTES.md` updated at push time per pre-push SOP.

## Roadmap closeout

`MOB_ALIVENESS_ROADMAP.md`:
- Flip chunk 3.6 status to **Done**.
- Add "Shipped:" summary describing the type-pool/pair-override
  authoring model, line-per-round pacing, trigger model, life-
  state gating, pilot content.

## Open questions

None — design fully scoped during brainstorming.

## References

- Roadmap: `MOB_ALIVENESS_ROADMAP.md` chunk 3.6
- Hard dependency (shipped): chunk 1.6 (NPC relationships),
  spec at
  `docs/superpowers/specs/2026-05-09-mob-aliveness-1.6-npc-relationships-design.md`
- Relationship API: `internal/relationships/relationships.go`
  (`RelationsBetween`, `AreRelated`, `Type` enum)
- Chunks 3.2 / 3.3 / 3.4 composition partners:
  - 3.2: per-segment idlecommands give NPCs one-sided sayings
    (3.6 adds two-sided exchanges on top, gated to fully-idle
    NPCs)
  - 3.3: sleeping NPCs are ineligible for conversations
  - 3.4: patrolling NPCs (mid-walk between waypoints) are
    ineligible; at-waypoint patrollers are eligible
- IdleMobs hook (where trigger lives): `internal/hooks/NewRound_IdleMobs.go`
- Player-arrival hook (where boost lives): `internal/usercommands/go.go`
- Existing `say` command (verb the conversation will invoke):
  `internal/mobcommands/say.go`
