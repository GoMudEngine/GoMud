# NPC Party System — Design (Stage 1 of Caravan Effort)

**Date:** 2026-04-27
**Status:** Approved (brainstorming complete, ready for implementation plan)

## Goal

Extend the existing `internal/parties` package to support NPC actors as
first-class party members and leaders. Refactor the existing bandit pack
tactics onto the new system as the smoke-test consumer. Introduce a small
set of behavior tree primitives (actions, conditions, events) that any
group of mobs can use for coordinated movement, combat, and retreat.

## Multi-stage context

This spec is **Stage 1** of a four-stage effort:

1. **NPC groups** (THIS SPEC) — extend party system; refactor bandits as smoke test
2. **Basic caravan** — Stillwater↔Thornwall caravan + delivery mob; uses Stage 1 party primitives for movement/combat/retreat coordination
3. **Forager NPCs + region-split mat lists** — wilderness foraging mobs that gather and sell locally; mat distribution split between regions
4. **Real item transfer** — caravan literally buys low / sells high, moving items between town inventories (replaces the v1 top-off-to-max restock model)

Each stage gets its own spec → plan → implementation cycle. This Stage 1
spec is fully independent of stages 2-4 (they consume Stage 1 primitives).

## Architecture overview

**What exists today:**

- `internal/parties/parties.go` — Party struct keyed on player `UserId int`
- Bandit pack tactics — ad-hoc per-mob behavior tree code + `MobDeath_PackFlee.go` hook
- Combat system already actor-aware via `actions.Actor` (see `internal/actions/actor.go`)
- Existing `actions.Actor` interface provides `GetUserId()` (player or 0) and `GetMobInstanceId()` (mob or 0) for disambiguation

**What Stage 1 changes:**

```
internal/parties/parties.go
  Party struct refactored to use actions.Actor for all member references
  partyMap registry keyed by leader Actor (lookup helpers handle the
    disambiguation between user-keyed and mob-keyed parties)
  All public APIs take/return actions.Actor where appropriate
  NPC parties exist in-memory only — die with leader mob instance
  Player parties: zero behavioral change

internal/usercommands/party.go
  Updated to use Actor abstraction internally (player party UX unchanged)
  New admin subcommands: party admin list-npc, party admin show <id>

internal/behaviortree/actions_party.go        (NEW)
  party_call_help, party_respond_to_help,
  party_follow_leader, party_assist_target,
  party_flee_to_room, party_at_home_stand

internal/behaviortree/conditions_party.go     (NEW)
  party_member_below_pct, party_in_combat, party_leader_in_combat,
  party_in_room, party_at_home

internal/events/eventtypes.go
  Add party_help_requested and party_dissolved event types

internal/hooks/MobDeath_PackFlee.go
  Migrate to fire party_dissolved if dead mob is a party leader
  (or delete entirely if the new behavior tree handles it cleanly)

mobs/<bandit zones>/...
  Bandit mob YAMLs updated to auto-join Soren's party on spawn
  (via behavior tree on-spawn or lazy on first call_help)
```

**Stage 2 (caravan) consumption preview** (not in this spec, but informs the design):

- Caravan crew = NPC party with Ketil as leader, Marta + Lars as members
- `party_follow_leader` powers caravan transit movement (extended to cross-zone in Stage 2)
- Existing combat primitives + `party_assist_target` handle coordinated road combat
- `party_flee_to_room` is N/A for caravan (caravan supposed to win) but available

## Data model changes

```go
// internal/parties/parties.go

type Party struct {
    Leader        actions.Actor          // was LeaderUserId int
    Members       []actions.Actor        // was UserIds []int
    Invitees      []actions.Actor        // was InviteUserIds []int
    AutoAttackers []actions.Actor        // was AutoAttackers []int
    Position      map[ActorKey]string    // actor → "front"/"middle"/"back"
    HomeRoomId    int                    // 0 if none designated
    HelpRoomId    int                    // 0 if no active call; rally room when set
}

// ActorKey is an internal lookup type for map keys. Implementation can be:
//   - typed string ("user:42" / "mob:1234"), OR
//   - struct{ UserId, MobInstanceId int } if pointer-equality on Actor isn't reliable
// Pick whichever fits existing patterns in the parties package; doesn't
// affect external API.
```

**Lookup model:**

```go
// Single global registry, replaces the existing partyMap[UserId]*Party
var partyRegistry []*Party  // or map keyed by leader ActorKey

// Public API examples (signatures change from int → Actor):
func New(leader actions.Actor) *Party
func GetByLeader(leader actions.Actor) *Party
func GetByMember(actor actions.Actor) *Party  // search any party containing this actor
func (p *Party) Add(actor actions.Actor) error
func (p *Party) Remove(actor actions.Actor) error
func (p *Party) Dissolve()  // fires party_dissolved for all members; removes from registry
```

**Persistence:**

- Player parties: in-memory, identical to today (already non-persistent)
- NPC parties: in-memory only; die with the leader mob instance
- Server restart: all NPC parties gone; reform lazily on next `party_call_help` signal

## New behavior tree primitives

### Actions (`internal/behaviortree/actions_party.go`)

| Action | Description |
|--------|-------------|
| `party_call_help` | Fired by ANY member. Sets the party's HelpRoomId to current room. Fires `party_help_requested` event for all members not already in this room. Used by lookouts on intruder spot AND by any member needing reinforcements. |
| `party_respond_to_help` | Member-side: navigate toward HelpRoomId if not already there. Higher priority than `party_follow_leader`. |
| `party_follow_leader` | Default movement: move toward leader's room if not already there. Used during transit/idle. |
| `party_assist_target` | In combat: target the same enemy the leader is targeting (or, configurable, follow the lookout's target if leader is offline). |
| `party_flee_to_room` | All members navigate toward the action's `room_id` parameter (typically the home/camp room). Triggered by leader on group-pressure threshold. |
| `party_at_home_stand` | If party's `HomeRoomId` matches current room, suppress flee behavior; mob fights to the death. |

### Conditions (`internal/behaviortree/conditions_party.go`)

| Condition | Description |
|-----------|-------------|
| `party_member_below_pct` | Any party member's HP/SP/CP under N% (params: `pool` "hp"/"sp"/"cp", `percent` int) |
| `party_in_combat` | Any party member currently in combat |
| `party_leader_in_combat` | Specifically the leader is in combat |
| `party_in_room` | All party members in same room (used to gate "all assembled" actions) |
| `party_at_home` | All party members in their `HomeRoomId` (only relevant if HomeRoomId != 0) |

### Events (`internal/events/eventtypes.go`)

| Event | Trigger | Payload |
|-------|---------|---------|
| `party_help_requested` | Fired when a member calls help | `RallyRoomId int`, `CallerActor Actor` |
| `party_dissolved` | Fired when leader dies OR party is explicitly disbanded | `Reason string` ("leader_died" / "disbanded" / "all_dead") |

Member behavior trees can listen for these events the same way they listen
for `mob_hurt` / `mob_die` today.

## Bandit migration plan (smoke test)

**Step 1: Identify the bandit camp room.**

Per quest 17 description: "off the road near the drainage ditch." Need to
scan north_road or marches_spur_road room files to find the canonical
camp room ID. Recorded in the implementation plan as task 0.

**Step 2: Establish the bandit party.**

When the lookout (283) spawns, OR on first `party_call_help` invocation
if Soren isn't yet spawned: create a party with Soren (286) as leader,
add lookout/fighter/caster (283/284/285) as members, set
`HomeRoomId` = camp room ID.

If Soren is dead and a non-leader is forced to call help, that mob
becomes the de-facto leader (lazy promotion at party-creation time only;
no mid-life leader changes per the dissolution model).

**Step 3: Wire bandit btrees.**

- **Lookout (283)** btree:
  - On `player_enter` (hostile player): emote "whistles sharply" →
    `party_call_help`.
  - On `party_help_requested` event from another lookout in same party:
    `party_respond_to_help`.
  - In combat: `party_assist_target`.
  - On `party_flee_to_room` signal: navigate to camp room.
  - At camp room: `party_at_home_stand` (overrides flee).

- **Fighter (284) / Caster (285)** btrees:
  - On `party_help_requested` event: `party_respond_to_help`.
  - In combat: `party_assist_target`.
  - On `party_flee_to_room` signal: navigate to camp.
  - At camp room: `party_at_home_stand`.

- **Soren leader (286)** btree:
  - In combat: `party_assist_target` for own targeting.
  - Periodic check (every few rounds): `party_member_below_pct` (any
    member at 30% HP, OR member count below threshold) → fire
    `party_flee_to_room` with target = camp.
  - At camp room: `party_at_home_stand`.

**Step 4: Migrate or delete `MobDeath_PackFlee.go`.**

The existing hook fires when a pack member dies and triggers other
pack members to flee. With the new system:

- If the dead mob is the party leader: fire `party_dissolved` for all
  members; let their individual btrees handle the chaos (some flee, some
  fight on, etc. — Model B dissolution behavior we agreed on).
- If the dead mob is a non-leader member: remove them from the party;
  no automatic flee. The leader's threshold check will eventually trigger
  flee if the group is degraded enough.

Existing pack-flee behavior is functionally absorbed into the new
primitives; the hook can likely be deleted, OR repurposed as the place
where `party_dissolved` is fired on leader death.

## Edge cases

| Scenario | Behavior |
|----------|----------|
| Leader dies | `party_dissolved` event fires for all members; party removed from registry; member btrees revert to solo behavior |
| Leader despawns (room unload) | Party persists; member btrees should handle "leader not currently in any active room" by holding position (configurable per btree, but default = hold) |
| All members die | Party garbage-collected automatically next tick |
| Server restart | All NPC parties gone; reform lazily on next help-call signal |
| Player attacks lookout while reinforcements en route | Reinforcements still arrive on schedule; party state unaffected by player damage |
| Lookout dies before reinforcements arrive | Reinforcements still navigate to the rally room (HelpRoomId is set on party, not on the caller); they'll find an empty or player-occupied room and engage |
| Two parties with overlapping members | Disallowed at API level — `Party.Add(actor)` returns error if actor is already in another party |

## Testing strategy

**Unit tests** (`internal/parties/parties_test.go`):
- Backward compat: existing player-party tests still pass after Actor refactor
- New: NPC-party creation, member add/remove, dissolution, lookup by leader, lookup by member
- Mixed parties (player leader + NPC members): create, dissolve, query

**Behavior-tree integration tests** (`internal/behaviortree/actions_party_test.go`):
- `party_call_help` sets HelpRoomId and fires event
- `party_respond_to_help` navigates the listening mob toward the rally room
- `party_flee_to_room` triggers all members to attempt move
- `party_at_home_stand` suppresses flee at home room only

**In-game smoke test** (the bandit pack):
1. Walk to bandit lookout room
2. Confirm lookout emotes "whistles sharply" + `party_call_help` fires
3. Confirm fighter + caster (and Soren if not already in lookout room) navigate toward lookout's room
4. Engage in combat; confirm Soren targets coordinate via `party_assist_target`
5. Bring group health below threshold (kill one member or reduce others to ~30% HP)
6. Confirm Soren fires `party_flee_to_room` with target = camp
7. Confirm bandits navigate toward camp (path may go through several rooms)
8. Walk to camp room; confirm bandits at camp now fight to the death (no further flee)
9. Kill Soren; confirm `party_dissolved` event fires; surviving bandits revert to solo behavior (some flee individually, some hold ground depending on per-mob btree)

## Out of scope (explicitly)

- **Caravan-specific anything** — Stage 2 spec
- **Cross-zone party movement** — Stage 2 (Stage 1 only handles same-zone)
- **Forager NPC behaviors** — Stage 3 spec
- **Real item transfer** — Stage 4 spec
- **Player-invites-NPC** into a player party — deferred indefinitely; companions are a different system
- **Leader promotion / succession** — explicitly Model B (dissolution); future opt-in via `party_succession_chain` field if a future encounter needs it

## Files affected

| Action | File | Purpose |
|--------|------|---------|
| MODIFY | `internal/parties/parties.go` | Actor refactor of Party struct + registry |
| MODIFY | `internal/usercommands/party.go` | Use Actor internally; add admin subcommands |
| MODIFY | `internal/events/eventtypes.go` | Add `party_help_requested`, `party_dissolved` events |
| MODIFY | `internal/hooks/MobDeath_PackFlee.go` | Migrate to fire `party_dissolved` on leader death; or delete |
| MODIFY | `internal/hooks/NewRound_DoCombat_unified.go` | Verify party-aware combat still works post-refactor (mostly already actor-aware) |
| MODIFY | `internal/actions/target_resolution.go` | Verify party-aware target resolution still works |
| CREATE | `internal/behaviortree/actions_party.go` | New party btree actions (6 actions) |
| CREATE | `internal/behaviortree/conditions_party.go` | New party btree conditions (5 conditions) |
| CREATE | `internal/parties/parties_test.go` | Unit tests for refactored Party + NPC support |
| CREATE | `internal/behaviortree/actions_party_test.go` | Btree primitive tests |
| MODIFY | `docs/schemas/behavior.md` | Document new party actions/conditions/events |
| MODIFY | `_datafiles/world/dogmud/mobs/<bandit zones>/{283,284,285,286}-*.yaml` | Wire btrees to use new party primitives |
| MODIFY (or CREATE) | Bandit behavior tree files (per-mob btrees if any exist as separate files) | Replace ad-hoc pack code with new primitives |
| DELETE (possibly) | `internal/hooks/MobDeath_PackFlee.go` | If new btree handles all flee behavior |

## Verification plan

**Phase 1 — unit tests pass:**
- `go test ./internal/parties/...`
- `go test ./internal/behaviortree/...`
- `go test ./...` overall (no regressions)

**Phase 2 — server boot clean:**
- Per the recently-added Pre-Push SOP, boot the server locally and
  confirm `mobs.LoadDataFiles() loadedCount=...` and the party
  package initializes without panic.

**Phase 3 — bandit smoke test (manual, in-game):**
- Walk through the 9-step bandit smoke sequence above
- Specifically verify: lookout calls help, reinforcements arrive,
  retreat triggers at threshold, flee path navigates correctly to camp,
  at-camp behavior switches to last-stand, leader death dissolves party
  cleanly

**Phase 4 — backward compat smoke test (manual):**
- Existing player-party features (form, invite, accept, position,
  auto-attack, dissolve) all work unchanged.
- Test with at least one player party of 2+ to confirm tactical
  positioning still affects targeting per the existing combat code path.

## Open implementation questions (for the plan stage, not blocking spec approval)

These are detail-level decisions to make during implementation, not
brainstorming-level decisions:

- ActorKey type: typed string vs struct vs Actor pointer-equality. Pick
  during implementation based on what fits existing patterns.
- Lazy party formation timing: in lookout's btree on first `player_enter`
  vs explicit party-config file vs on-spawn behavior tree. Implementation
  picks the cleanest path.
- Which exact rooms in north_road / marches_spur_road host the bandit
  camp. Identified during plan-task-0 by scanning room files.
- Whether to keep `MobDeath_PackFlee.go` as the `party_dissolved` firing
  point or move that into the parties package itself.
