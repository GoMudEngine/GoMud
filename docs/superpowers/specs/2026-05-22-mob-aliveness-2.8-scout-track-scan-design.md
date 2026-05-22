# Mob Aliveness 2.8 — Mob Scout / Track / Scan

> **Phase 2 tactical (eighth chunk).** Lift the three remaining
> information-gathering verbs (`scan`, `track`, `search`) into the
> `internal/actions/` package via the actor pattern. Player commands
> become thin wrappers. Mob wrappers added for all three verbs. Four
> new btree action primitives (three `try_*` + one
> `move_toward_tracked` consumer) and two new state-query conditions.
> One new mob archetype (`scout`) wired on `217-goblin_scout`, plus
> one targeted graft each onto `lookout`, `thief`, and `leader`.
>
> `consider` already shipped in chunk 2.4 and is not revisited here;
> the roadmap chunk brief is mildly stale on that point.

## Goal

Patrol-flavored mobs become patrollers. A goblin scout standing
watch on the steppe should sweep adjacent rooms each idle tick,
spot an approaching player one room out, alert its pack, and move
to engage — not stand frozen until the player walks into its tile.

Three unlocked behaviors from one consolidated set of primitives:

- **Forward awareness (scout pattern):** an idle mob scans
  adjacent rooms, sets soft-target on the first hostile sighting,
  calls for help, and closes the distance.
- **Stealth detection (scout / thief / lookout):** a periodic
  in-room search reveals hidden hostiles to the searcher only,
  letting the mob respond instead of being ambushed.
- **Chase persistence (leader / scout):** active tracking via
  buff 26 lets a mob pursue a fleeing aggro target across room
  boundaries instead of giving up at the threshold.

The verbs ship as a single consolidated set because their player-
side cousins already share infrastructure (`skills.Search` for
both `track` and `search`; visitor-trail data for both `scan`
and `track`). Lifting them together avoids duplicate plumbing.

## Architectural musts

Brainstorming refined the framing:

1. **Three verbs lift to `actions/` via the actor pattern**,
   mirroring chunks 2.1 (`actions.Buy`), 2.4 (`actions.Consider`),
   and 2.7 (`actions.Steal` et al). Each exposes a single entry
   point `<Verb>(actor Actor, opts <Verb>Options) <Verb>Result`.
   Both player wrappers and mob wrappers thin to ~25 lines.

2. **Mob-actor side effects are silent.** `MobActor.SendText`
   is a no-op (existing convention). The action returns a
   structured result; the btree consumes it. Player-only flair
   (template rendering, colored tier descriptions) lives in the
   player wrapper's `formatXResultForPlayer` helper, not in the
   action.

3. **Search-reveal semantics: scout-only awareness.** When a mob
   `try_search` succeeds on a hidden mob or hidden player, the
   target is exposed to the searcher (via `SoftTarget` and result
   payload) but NOT to other room occupants. The hidden buff is
   not stripped. Matches the player path exactly: the searcher
   sees the `(hiding)` tag, others see nothing. No public-reveal
   broadcast.

4. **Tier-1 and Tier-3 search results are player-only flavor.**
   Hidden exits, hidden containers, and hidden nouns are player
   discovery affordances with no consumer on the mob side. The
   action's `SearchResult` still populates these fields for the
   player wrapper, but `try_search` ignores them — only hidden
   mobs and hidden players (Tier 2) feed into soft-target
   selection.

5. **Track persistence: buff 26 symmetric with players.** When
   `try_track` succeeds on a named target, the action applies
   buff 26 to the actor (mob or player) and stores
   `tracking-user` / `tracking-mob` misc data on the actor's
   `Character`. The btree consumes the buff state via the
   `mob_is_tracking` condition and acts via the new
   `move_toward_tracked` action. See the **Buff 26 mob
   compatibility** section for the contingency plan if buff 26's
   tick handler is player-coupled.

6. **Btree primitives are verb-shaped, plus one consumer.**
   `try_scan`, `try_track`, `try_search` mirror the verb lift.
   `move_toward_tracked` is the buff-26 consumer that dispatches
   a `go <direction>` to chase a tracked target. Conditions
   `room_has_hidden_entity` and `mob_is_tracking` are cheap
   pre-checks so archetypes can branch before paying for the
   action's cooldown.

7. **Target resolution follows the standard chain.** For
   `try_track`: prefer `ctx.Event.UserId` → `Aggro.UserId` →
   `Aggro.MobInstanceId`. Configurable via `target_from:`
   parameter so archetypes can pin the source explicitly
   (`aggro` for chase, `event` for "this player just left",
   etc.). `try_scan` picks the first hostile sighting found
   while iterating exits in deterministic order.

8. **One new archetype (`scout`), one wired test mob, three
   grafts.** Following the chunk 2.4 / 2.7 precedent: `scout.yaml`
   lives alongside the others in
   `_datafiles/world/dogmud/behaviors/archetypes/`. The Ironwind
   goblin scout (mob 217) is the single test target for the new
   archetype. The three grafts target one representative mob per
   archetype and add a single new branch each:
   - `lookout` gains scan-before-ambush
   - `thief` gains search-before-steal
   - `leader` gains track-on-aggro-lost

9. **"Hostile" determination on the mob side.** `try_scan` uses
   `mob.HatesMob(other)` for mob-on-mob (chunk 2.4 precedent) and
   `actor.Aggro` + faction-rep tier (chunks 1.1/1.2 substrate)
   for mob-on-player. If faction-rep substrate isn't trivially
   callable from the action, fall back to "any player in an
   adjacent room is a sighting" — surfaced as a hand-off during
   implementation.

## Architecture & module layout

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/actions/scan.go` | NEW | `Scan(actor, opts) ScanResult` |
| `internal/actions/track.go` | NEW | `Track(actor, opts) TrackResult` — covers both trail-read and active-track modes |
| `internal/actions/search.go` | NEW | `Search(actor, opts) SearchResult` |
| `internal/actions/scan_test.go` | NEW | Unit tests, 2.7 shape |
| `internal/actions/track_test.go` | NEW | Unit tests |
| `internal/actions/search_test.go` | NEW | Unit tests |
| `internal/usercommands/scan.go` | REWRITE | Thin wrapper (~25 LoC) |
| `internal/usercommands/skill.track.go` | REWRITE | Thin wrapper |
| `internal/usercommands/skill.search.go` | REWRITE | Thin wrapper |
| `internal/mobcommands/scan.go` | NEW | Thin wrapper |
| `internal/mobcommands/track.go` | NEW | Thin wrapper |
| `internal/mobcommands/search.go` | NEW | Thin wrapper |
| `internal/mobcommands/mobcommands.go` | MODIFY | Register `scan`, `track`, `search` in `mobCommands` map |
| `internal/behaviortree/actions_scout.go` | NEW | Four btree actions: `try_scan`, `try_track`, `try_search`, `move_toward_tracked` |
| `internal/behaviortree/conditions_scout.go` | NEW | Two btree conditions: `room_has_hidden_entity`, `mob_is_tracking` |
| `internal/behaviortree/actions.go` | MODIFY | Register the four new actions in `init()` |
| `internal/behaviortree/conditions.go` | MODIFY | Register the two new conditions in `init()` |
| `internal/behaviortree/context.md` | MODIFY | Document new actions + conditions |
| `_datafiles/world/dogmud/behaviors/archetypes/scout.yaml` | NEW | New archetype tree |
| `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` | MODIFY | Scan-before-ambush graft |
| `_datafiles/world/dogmud/behaviors/archetypes/thief.yaml` | MODIFY | Search-before-steal graft |
| `_datafiles/world/dogmud/behaviors/archetypes/leader.yaml` | MODIFY | Track-on-aggro-lost graft |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml` | MODIFY | `behavior_archetype: scout` |
| `internal/actions/context.md` | MODIFY | Document the new actions |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.8 Done, roll-up 16/41 |

## Public API

### `actions.Scan`

```go
package actions

// ScanOptions parameterizes a one-step adjacent-room sweep.
type ScanOptions struct {
    // HostileOnly: when true, SoftTarget population (mob path)
    // requires a hostile sighting (HatesMob or faction-hostile).
    // Default true.
    HostileOnly bool
}

// ScanSighting describes one occupant in one adjacent room.
type ScanSighting struct {
    ExitName    string
    RoomId      int
    RoomTitle   string
    Mobs        []ScanEntity
    Players     []ScanEntity
}

type ScanEntity struct {
    Id      int    // instance id (mob) or user id (player)
    Name    string
    IsMob   bool
}

// ScanResult is the structured outcome. Sightings is in
// deterministic exit-iteration order.
type ScanResult struct {
    Sightings []ScanSighting
}

// Scan walks each visible (non-secret) exit, loads the adjacent
// room, and lists non-hidden mobs and players in each. No skill
// check, no cooldown (matches the existing player behavior).
// MobActor + UserActor both supported. UserActor receives
// SendText for the rendered list; MobActor is silent. The result
// is returned in both cases.
func Scan(actor Actor, opts ScanOptions) ScanResult
```

### `actions.Track`

```go
// TrackOptions parameterizes the trail-read.
type TrackOptions struct {
    // TargetNoun empty: trail-scan current room mode (no-arg
    // player path). Non-empty: active-track mode — resolves
    // target by name in current/adjacent room visitor logs.
    TargetNoun string

    // For mob callers: resolve target from a context source
    // instead of TargetNoun. Ignored if TargetNoun set.
    // Values: "aggro", "event", "soft_target", "none".
    TargetFrom string
}

// TrackingInfo matches the existing usercommands struct (lifted
// from skill.track.go). Strength is a tiered label
// ("Dead"/"Weak"/"Good"/"Warm"/"Hot"); NumericStrength is the
// raw 0..1 visitor decay.
type TrackingInfo struct {
    Name            string
    Type            string // "mob" or "user"
    Strength        string
    NumericStrength float64
    ExitName        string
}

// TrackResult is the structured outcome.
type TrackResult struct {
    // Trail-scan mode populates Visitors.
    Visitors []TrackingInfo

    // Active-track mode populates these.
    ActiveTargetUserId    int    // 0 if not user target
    ActiveTargetMobInstId int    // 0 if not mob target
    DirectionExit         string // best exit toward target
    BuffApplied           bool   // true when buff 26 applied

    // Common.
    RollValue   int  // for tier-band introspection
    OnCooldown  bool // 1-round cooldown collision
    Reason      string
}

// Track runs a Perception+Search trail-read. With TargetNoun or
// TargetFrom set, attempts active-tracking: locates the target's
// trail across adjacent rooms, applies buff 26 to the actor, and
// stores tracking-user/tracking-mob misc data. Without either,
// reports the visitor log of the current room.
func Track(actor Actor, opts TrackOptions) TrackResult
```

### `actions.Search`

```go
// SearchOptions is intentionally empty v1; in-room search is the
// only mode. Reserved for future "search container" path.
type SearchOptions struct{}

// SearchResult is the structured outcome.
type SearchResult struct {
    // Tier 1 (target 125) — player-flavor only.
    HiddenExitsFound      []string
    HiddenContainersFound []string

    // Tier 2 (target 135) — feeds mob soft-target selection.
    StashedItemsFound  []SearchStashedItem
    HiddenPlayersFound []int // user ids
    HiddenMobsFound    []int // mob instance ids

    // Tier 3 (target 175) — player-flavor only.
    HiddenNounsFound []string

    OnCooldown bool   // 2-round cooldown collision
    Reason     string
}

type SearchStashedItem struct {
    ItemId      int
    DisplayName string
}

// Search rolls Perception+Search per discovery candidate in the
// room. Result aggregates all hits. UserActor receives the existing
// template-rendered output; MobActor is silent. The cooldown is
// shared with the player path (2 rounds on the `search` key).
func Search(actor Actor, opts SearchOptions) SearchResult
```

### Player wrapper shape (example — `track`)

```go
func Track(rest string, user *users.UserRecord, room *rooms.Room,
           flags events.EventFlag) (bool, error) {
    opts := actions.TrackOptions{TargetNoun: rest}
    if rest == "stop" || rest == "clear" {
        return cancelTracking(user)
    }
    actor := &actions.UserActor{User: user, Room: room}
    result := actions.Track(actor, opts)
    formatTrackResultForPlayer(user, room, result)
    return true, nil
}
```

`cancelTracking`, `formatTrackResultForPlayer` are package-local
helpers in `usercommands/`. The action stays I/O-clean except for
`actor.SendText` calls required for room-broadcast messaging
(e.g., search's "is snooping around" emote).

### Mob wrapper shape (example — `scan`)

```go
func Scan(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
    actor := &actions.MobActor{Mob: mob, Room: room}
    _ = actions.Scan(actor, actions.ScanOptions{HostileOnly: false})
    return true, nil
}
```

Mob wrappers exist primarily so `scan` / `track` / `search` are
typeable mob commands (debuggability + symmetry with player
verbs). The btree calls into the action directly via its
primitive — it doesn't route through the mobcommand wrapper.

## Btree primitive shapes

### Actions (`actions_scout.go`)

```yaml
# Scan adjacent rooms. On first hostile sighting (HostileOnly:
# true is default), populates ctx.SoftTarget with that target.
# Returns Success when ANY entity sighted; Failure when adjacent
# rooms are empty or no hostiles found (HostileOnly mode).
- type: action
  do: try_scan
  hostile_only: true

# Trail-track. target_from picks the resolution source:
#   - "aggro": Aggro.UserId then Aggro.MobInstanceId (default for
#              chase use case)
#   - "event": ctx.Event.UserId
#   - "soft_target": ctx.SoftTarget
# On success, applies buff 26 to the mob, stores
# tracking-user/tracking-mob misc data, sets ctx.SoftTarget to the
# tracked target. Returns Success on trail hit; Failure on miss,
# cooldown, or unresolved target.
- type: action
  do: try_track
  target_from: aggro

# Read buff 26 + tracking misc data, derive direction toward the
# tracked target, dispatch `go <direction>`. Returns Failure if
# no buff 26, no resolvable direction, or movement fails (locked
# exit, etc.). Composes with try_track in a sequence: try_track →
# move_toward_tracked each tick keeps the chase alive.
- type: action
  do: move_toward_tracked

# Search current room for hidden entities. On Tier-2 hits (hidden
# mobs / hidden players), populates ctx.SoftTarget with the first
# hostile detection. Tier-1 + Tier-3 hits are populated in the
# SearchResult but ignored by the btree primitive (player-flavor
# only). Returns Success on any hidden-hostile detection;
# Failure otherwise.
- type: action
  do: try_search
```

### Conditions (`conditions_scout.go`)

```yaml
# True when at least one hidden mob or hidden player is in the
# room. Cheap pre-check before paying for try_search's cooldown
# + per-discovery rolls.
- type: condition
  check: room_has_hidden_entity

# True when self carries buff 26 (active tracking).
- type: condition
  check: mob_is_tracking
```

## `scout` archetype YAML sketch

```yaml
# _datafiles/world/dogmud/behaviors/archetypes/scout.yaml
name: scout
description: |
  Patrol-flavored mob. Scans adjacent rooms each idle tick;
  on hostile sighting, alerts pack and moves toward target.
  Searches current room periodically for hidden threats.
  Pursues fleeing aggro targets across room boundaries.

tree:
  - type: selector
    children:

      # 1. Shared panic-flee branch
      - { type: include, archetype: "_panic_flee" }

      # 2. Self-defense
      - type: sequence
        on: mob_attacked
        children:
          - { type: include, archetype: "generic_fighter" }

      # 3. Engage on aggro (target in same room)
      - type: sequence
        on: mob_idle
        children:
          - { type: condition, check: has_aggro }
          - { type: include, archetype: "generic_fighter" }

      # 4. Active-track loop — aggro target fled the room
      - type: sequence
        on: mob_idle
        children:
          - { type: condition, check: mob_is_tracking }
          - { type: action, do: move_toward_tracked }

      # 5. Search current room for hidden threats
      - type: sequence
        on: mob_idle
        children:
          - { type: condition, check: room_has_hidden_entity }
          - { type: action, do: try_search }
          - { type: action, do: call_for_help }

      # 6. Scan adjacent rooms; on hostile, alert + chase
      - type: sequence
        on: mob_idle
        children:
          - { type: action, do: try_scan, hostile_only: true }
          - { type: action, do: try_track, target_from: soft_target }
          - { type: action, do: call_for_help }
          - { type: action, do: move_toward_tracked }

      # 7. Fallback wander
      - { type: action, do: idle_wander }
```

(Exact include/inversion syntax to match existing archetype
conventions discovered during implementation; this sketch is
intent-level.)

## Grafts on existing archetypes

One new branch added to each archetype's tree, preserving all
existing behavior.

### `lookout.yaml` — scan-before-ambush

Prepend a `try_scan`-gated branch to the existing `player_enter`
ambush so the mob "spots" approaching intruders one room early.
On `try_scan` success, the lookout sets soft-target and either
begins moving toward the room (if its tree supports advance) or
prepares an ambush primed for that direction. Baseline ambush
math preserved — the lookout still ambushes when the player
actually enters; the new branch just lights up earlier.

### `thief.yaml` — search-before-steal

Insert `room_has_hidden_entity` → `try_search` → `call_for_help`
as a new branch ahead of the existing steal-and-flee loop. A
thief now catches sneaking rivals or hidden players before
attempting to lift a coin purse from an "empty" room.

### `leader.yaml` — track-on-aggro-lost

Append `mob_is_tracking` → `move_toward_tracked` as a new branch
on the leader's idle path. Also wire `try_track` into the
combat-aggro-lost flow (exact event name verified during impl —
`mob_target_lost` if available, otherwise a synthetic
`mob_idle` + an `aggro_target_not_in_room` condition that does
not currently exist and would be added inline if needed).

Each graft is ≤ 10 lines of YAML. No existing branches modified.

## Test mob — Ironwind Goblin Scout

`_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml`
changes:

- Set `behavior_archetype: scout` (currently `lookout`).
- Confirm `search` skill rank present (bump to ≥ 1 if absent).
- All other fields preserved.

Single mob flip for v1. Additional `scout` flips (e.g., warren
scout, ranger archetypes in other zones) follow in a content pass.

## Buff 26 mob compatibility

Risk surface: buff 26 was authored for the player active-tracking
display loop. Its tick handler may be player-coupled in ways that
break or silently no-op for mob owners.

Plan:

1. **First implementation step:** read the buff 26 YAML/script
   (`_datafiles/world/dogmud/buffs/26-*.yaml`) and audit the tick
   handler for `users.GetByUserId` patterns or `UserRecord`
   parameters that wouldn't apply to mobs.

2. **If tick handler is portable** (operates on `Character`, not
   `UserRecord`): apply buff 26 to mobs and let it tick normally.
   `move_toward_tracked` reads misc data identically to the player
   path. Symmetric design intact.

3. **If tick handler is player-coupled** (the more likely case):
   apply buff 26 to mobs for introspection only (`mob_is_tracking`
   condition + admin debug visibility), and have
   `move_toward_tracked` re-derive direction from misc data on
   each call by re-running the adjacent-room visitor scan. This
   is no worse than re-evaluating each btree tick — which we'd
   want anyway for tactical responsiveness. The buff becomes a
   state-token, not a control mechanism.

4. **Hand off** before deciding between (2) and (3). Don't refactor
   buff 26's tick handler in this chunk — that's a separate audit.

## Hostile determination

`try_scan` and `try_search` need a "hostile?" predicate to drive
soft-target selection.

- **Mob-on-mob:** use `mob.HatesMob(other)` — established in
  chunk 2.4.
- **Mob-on-player:** preferred path is faction-rep tier (chunk
  1.2 substrate). If `factions.GetRep(slug, playerId)` is
  trivially callable from the action layer, use it: hostile = rep
  tier ≤ "Disliked" or matching enemy-faction membership. If the
  call site requires plumbing that pushes scope, fall back to
  "any non-charmed player in an adjacent room counts as a
  sighting" — flagged as a hand-off during implementation.

## Testing & smoke validation

### Unit tests

`internal/actions/{scan,track,search}_test.go` — for each action:

- Success path (sighting / trail hit / hidden entity found).
- Failure path (empty adjacent rooms / no trail / no hidden
  entities).
- Result-shape contract: `ScanResult.Sightings` length matches
  sighted entities; `TrackResult.DirectionExit` set on active-track
  success; `SearchResult.HiddenMobsFound` populated correctly.
- MobActor silent-side-effects assertion (no SendText calls).
- Cooldown path for search (second call within 2 rounds returns
  `OnCooldown: true`, no rolls fired).
- Skill progression: `OnSkillUse("search")` called on every roll-
  fired path.

### No new btree primitive unit tests

Consistent with chunks 2.4 / 2.6 / 2.7 — primitives validated via
in-game smoke. The unit-test surface for btree primitives is
mostly registration + param parsing, not per-condition math.

### Smoke test plan

Against `217-goblin_scout` for the new archetype, and one
representative mob each for the three grafted archetypes.

1. **Scout idle-scan loop.** Stand two rooms away from the goblin
   scout's spawn. Walk one step closer. Expect: scout's next idle
   tick sights you via `try_scan`, sets aggro, calls for help,
   begins moving toward your room. Continue retreating; expect
   the scout to chase via `move_toward_tracked` across at least
   2 room boundaries.

2. **Scout-room search.** Sneak (buff 9) into the goblin scout's
   room. Expect: scout's idle `try_search` branch detects you via
   `room_has_hidden_entity`, the scout aggros and calls for help.
   Other room occupants (if any) do NOT see you revealed.

3. **Lookout scan-before-ambush.** Approach a lookout-archetype
   mob (e.g., 283-bandit_lookout) from an adjacent room. Expect:
   lookout reacts a tick earlier than baseline because `try_scan`
   spotted you before `player_enter` fired. Baseline ambush math
   preserved.

4. **Thief search-before-steal.** Sneak into a thief mob's room
   (90-thornwall_highwayman). Expect: thief's idle `try_search`
   catches you, aggros (or calls for help, depending on power-
   overmatch math) instead of attempting to steal from an
   "empty" room.

5. **Leader track-on-aggro-lost.** Engage a leader-archetype mob,
   then flee one room over. Expect: leader's `mob_is_tracking`
   branch fires `move_toward_tracked` and pursues. Verify chase
   persists across at least 2 room boundaries.

6. **Active-tracking buff observability.** During (5), inspect
   the mob's buff list via admin command. Confirm buff 26
   actually applied to the mob.

7. **Build/data smoke:** `go run .` clean boot past
   `mobs.LoadDataFiles()` and `behaviors.LoadDataFiles()`. All
   package tests pass.

Kill test servers after smoke per the standing SOP.

## Risks / known limitations

- **Buff 26 tick handler may be player-coupled.** Mitigated via
  the buff-26 fallback plan above. Surfaces as a hand-off
  decision early in implementation.
- **`move_toward_tracked` direction staleness.** If the tracked
  target moves between the actor's `try_track` and the actor's
  `move_toward_tracked` call (same tick or next tick), the
  direction may already be wrong. Acceptable v1: the next tick
  re-runs `try_track` to refresh. Tuning lever if smoke reveals
  jank.
- **Faction-rep substrate plumbing for "hostile" predicate.** May
  not be trivially callable from `actions/`. Mitigated via the
  fallback plan in **Hostile determination**.
- **Scout grafted onto a single mob in v1.** Other scout-named
  mobs (warren_scout 72) stay on `lookout` for now; flip in a
  content pass after the archetype proves out.
- **`try_search` cooldown shared with player path.** Two mobs
  searching the same room within 2 rounds will collide via the
  global `search` cooldown key on each character. Acceptable —
  cadence-control feature, not a bug.
- **No mob-side "is snooping around" room broadcast for
  `try_search`.** Player path emits the emote via room-broadcast;
  mob path is currently silent. Could be added as flavor in a
  follow-up.

## Open questions

- **Buff 26's exact tick handler shape** — resolved during
  implementation by reading the buff YAML and any associated
  script.
- **`mob_target_lost` btree event existence** — verified during
  implementation. If absent, leader graft uses a synthetic
  condition or a `mob_idle` poll.
- **Faction-rep call ergonomics from `actions/`** — verified
  during implementation. Fallback plan covers the worst case.

## Out of scope

- **`consider` re-lift.** Already shipped in chunk 2.4.
- **`shadow` revisit.** Already shipped in chunk 2.7.
- **Public-reveal broadcast on `try_search`.** Architectural
  must #3 pins this to scout-only awareness.
- **Refactoring buff 26's tick handler for cross-actor parity.**
  Followup if the player-coupled fallback ships and we later
  want true tick symmetry.
- **Additional `scout`-archetype mob flips beyond goblin_scout
  (217).** Content-pass item.
- **`scan` cooldown.** Player path has none; preserved.
- **Adjacent-room iteration beyond one step.** `try_scan` peeks
  one exit deep, like the player command.
- **Stealth-state dual-path consolidation** (logged in chunk 2.7
  as followup; not touched here).

## Roadmap impact

- Chunk 2.8 marked Done.
- Roll-up: 16 / 41 done • 0 in progress • 25 not started.
- Unblocks: chunks 5.2 (bounty hunting — consumes `try_track`,
  `try_scan`, and the `scout` archetype precedent), 3.4 (waypoint
  patrols — `try_scan` natural fit for patrol-while-watching),
  6.1 (Stillwater town-flavor pass — guards can gain `try_scan`
  for early intrusion detection).
