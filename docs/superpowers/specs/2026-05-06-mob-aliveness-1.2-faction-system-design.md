# Mob Aliveness 1.2 — Faction System

**Status:** Design • **Roadmap chunk:** 1.2 (Phase 1, Substrate) •
**Size:** L • **Depends on:** 1.1 (shares store-backend pattern)

## Goal

First-class factions: definitions, NPC membership, per-player
reputation per faction, faction-vs-faction relationship declarations,
admin tooling. Replaces the `peacefulquest` placeholder. Substrate
for chunk 1.3 (crime/wanted state), chunk 1.5 (bounty state), and
chunk 5.1 (town justice).

Without this, "the warren remembers you killed one of theirs even
though Lars personally never saw it" is impossible — every
relationship lives only on individual NPCs (chunk 1.1) with no
roll-up to the group level.

## Non-goals (explicit Out of scope)

- **Player-facing visibility.** Players never see numeric rep.
  Relationships surface through dialogue, NPC behavior, and fiction.
  Same discipline as 1.1.
- **Dialogue gating on faction rep.** No `questFlagRequired`-style
  gate for faction tier in 1.2. The data is queryable
  (`factions.TierFor` is public), but no dialogue-engine wiring
  lands here. Defer.
- **Flavor text on rep change.** The player doesn't get "you feel
  less welcome among the warren" messages on bumps. Substrate is
  silent.
- **Bargaining / rhetoric mechanics that consume rep.** No verb
  that spends rep to unlock a check. Out.
- **Faction-specific shop pricing.** Dynamic pricing isn't faction-
  aware in 1.2. Out.
- **Auto-propagation of rep changes through the ally/enemy graph.**
  Allies and enemies are *declarative data* readable by consumers,
  not a pipe that rep deltas flow through. If killing the warren
  should also tilt rep with their allies, the consumer making the
  bump call writes both bumps explicitly.
- **Broad faction content pass.** Only `warren` and
  `thornwall_guards` are authored in 1.2. Bandits, ironwind shaman,
  Sanctum Basin guards, Stillwater militia, Dustwalk caravans, etc.
  are deferred to a new roadmap chunk **6.5a Faction definitions
  content pass**.
- **Thornwall citizens faction + alliance-aware guard aggro.**
  Killing a beggar should make Thornwall guards hostile too — but
  that's the same machinery as crime/wanted state and naturally
  belongs with chunk **1.3**. 1.2 ships only the narrow
  `thornwall_guards` faction; 1.3 adds `thornwall_citizens` and the
  alliance-aware consumer pattern.
- **`peacefulquest` field surviving as a fallback.** No legacy
  compat shim. The field is deleted from `mobs.Mob` in this chunk
  after a one-shot live-player migration seeds rep for existing
  players holding the legacy quest token.

## Decisions

### Identity: factions are Groups with definition files

Mobs already have `Groups []string` (e.g., `[humanoid, warren]`).
A *faction* is a Group with a definition YAML file. No new field
on Mob; the existing `Groups` carries membership.

Consequences:
- A group with no definition file (e.g., `humanoid`) stays a
  taxonomy tag for pack tactics / hate propagation, no faction
  semantics.
- A group with a definition file becomes faction-aware: per-player
  rep, allies/enemies, default starting rep.
- Multiple-faction membership is supported by adding multiple
  groups to a mob — e.g., `[humanoid, warren]` makes the mob a
  warren member.

### Storage split — definition vs runtime

Two directories, two lifecycles:

```
_datafiles/world/dogmud/factions/{slug}.yaml          # committed
_datafiles/world/dogmud/factions.rep/{slug}.yaml      # runtime, gitignored
```

Definitions are authored content (load eagerly at startup). Rep
state is engine state (lazy load + synchronous save on mutation).

`.gitignore` adds `_datafiles/world/dogmud/factions.rep/` alongside
the existing `shops/`, `mobs.instances/`, `rooms.instances/`.

### Score: scalar, no decay, opinion-style tiers

- Range `[-100, +100]`, signed scalar `int`, clamped on every
  write.
- **No decay.** Faction rep is institutional memory; it does not
  drift back to neutral on its own. (Individual NPC opinions in
  chunk 1.1 do decay; the two layers compose.)
- **Per-faction default rep** declared in YAML (`default_rep`).
  New players inherit the default until something shifts the rep.
- **Tier banding identical to opinions** — the `factions` package
  imports `internal/opinions` and calls `opinions.TierOf(rep)`
  directly. No duplicate enum, no risk of drift.

### Faction-vs-faction relations: declarative only

Definitions list `allies: [...]` and `enemies: [...]`. Consumer
code is free to read this and act on it (e.g., the guard's aggro
decision factors in allies' rep). Rep changes do **not**
auto-propagate through the alliance graph. If killing X should
tilt rep with Y, the consumer making the bump call writes both.

Reasons (from Q4 of brainstorming):
- Implicit cross-faction effects are debugging hell. "Why is my
  militia rep falling?" → "Because someone bumped Stillwater
  citizens, which the militia is allied with" is a footgun.
- The alliance graph stays useful as data. We just don't pipe rep
  deltas through it.

### Integration scope: API + ONE demo consumer

Same discipline as chunk 1.1:

1. Substrate: registry, rep store, public API.
2. ONE demo consumer: combat-death hookup (kill a faction member
   → bump killer's rep with that faction).
3. `peacefulquest` migration: quest 2 ("The Warren Compact") gains
   a `bump_rep` action; the legacy field is deleted from `Mob`;
   live-player migration entry seeds rep for existing players who
   completed quest 2.
4. Admin command: `faction show / set / bump / reset / list`.
5. Two authored factions: `warren` and `thornwall_guards`.

Everything else (dialogue, quest, shops, broader content) is
deferred to its own chunk.

### Quest engine: new `bump_rep` action

The existing quest action system (`internal/questengine/actions.go`)
gains one new verb. Quest YAML:

```yaml
actions:
  - bump_rep: {faction: warren, delta: 30}
```

Implementation: extend `ActionDef` with `BumpRep *BumpRepDef`;
extend `ActionContext` with `BumpRep(factionId string, delta int)`;
extend `ExecuteAction` with the dispatch case. The real game
bridge calls `factions.BumpRep` directly. ~25 LoC of plumbing.

### `peacefulquest` migration: clean cut

Steps:

1. **Quest engine** gains `bump_rep` action (above).
2. **Quest 2 (The Warren Compact)** end step gains
   `bump_rep: {faction: warren, delta: 30}` alongside the existing
   `grant: 2-end` token (token kept; it may be referenced by other
   dialogue/quest gates).
3. **Warren scout (mobId 72) and warrior (mobId 73)** mob YAMLs
   add `warren` to their groups list and delete the
   `peacefulquest:` line.
4. **Tunnel shaman (mobId 74)** adds `warren` to groups (so killing
   the shaman drops warren rep — the intuitive fiction).
5. **`internal/mobcommands/lookfortrouble.go`** and
   **`internal/behaviortree/actions_party.go`** replace the
   `mob.PeacefulQuest != "" && user.Character.HasQuest(...)` check
   with `factions.IsPeacefulToward(mob, user.UserId)`. The new
   helper is pure faction-rep logic, no legacy fallback.
6. **Live-player migration** (entry name `Migration 0.13.0`,
   matching the existing `0.10.0`/`0.11.0`/`0.12.0` pattern). On
   first server boot post-deploy: for every user file holding
   token `2-end` AND no warren rep entry, call
   `factions.BumpRep("warren", userId, 30)`. Idempotent — runs
   once per user, tracked via the existing migration completion
   flag.
7. **Delete `Mob.PeacefulQuest`** field from `internal/mobs/mobs.go`
   after steps 1-5 land. No callers remain; the YAML field on
   warren mobs was deleted in step 3.

### Combat hookup: kill-faction-member

A new hook file `internal/hooks/MobDeath_FactionRep.go`
subscribed to `events.MobDeath`. On mob death:

- Resolve the killer (player UserId; companion attribution via the
  existing `IsCharmed(userId)` chain).
- If no player killer (env damage, mob-vs-mob), skip.
- If the dead mob has zero defined factions, skip.
- For each defined faction in the dead mob's `Groups`:
  - Bump the killer's rep by `Balance.FactionMemberKillRep`
    (default `-10`, config knob).
  - **Party propagation**: look up the killer's party via
    `parties.GetParty(killerUserId)` (or equivalent). For every
    party member whose `Character.RoomId == mob.Character.RoomId`
    at death-time, bump their rep too.

Magnitude: ~10 kills saturates a player to `-100` (Hostile) on
that faction. Intended fiction — clearcutting a faction's members
makes you their permanent enemy.

Edge cases:
- Mob has multiple defined factions (e.g.,
  `[warren, deep_creatures]`) → both rep entries get the full
  `-10` bump. Authors who want a "primary" faction can leave
  secondary tags as non-faction (no definition file → no rep
  bump).
- Killer is in a party but the party member died/was offline →
  per-member room check filters them out cleanly.
- The killer themselves is filtered separately (the `bump for the
  killer` is its own line, party iteration starts from the
  killer's perspective).

### Admin command `faction`

Subcommands:

```
faction list                                   # list all loaded definitions
faction show <playerName>                      # all (faction × player) rows for that player
faction show <factionId> <playerName>          # one row
faction set <factionId> <playerName> <rep>     # absolute override (clamped)
faction bump <factionId> <playerName> <delta>  # additive
faction reset <factionId> <playerName>         # snap back to default_rep
```

Output style mirrors `opinion show` (ANSI table, 80-col wrap).
`faction list` is unique to this command — definition set is small
and bounded, worth listing whole.

Helpfile at
`_datafiles/world/dogmud/templates/admincommands/help/command.faction.template`,
mirroring `command.opinion.template` style. Registry entry:
`` `faction`: {Faction, true, true, true}, // Admin only ``.

## Architecture

```
                 ┌──────────────────────────────────────────┐
                 │  _datafiles/world/dogmud/factions/       │
                 │     {slug}.yaml          (committed)     │
                 │                                          │
                 │  _datafiles/world/dogmud/factions.rep/   │
                 │     {slug}.yaml          (runtime, gitignored)
                 └──────────┬─────────────────┬─────────────┘
                            │                 │
              eager startup │                 │ lazy load + sync save
                            ▼                 ▼
   ┌──────────────────────────────────────────────────────────┐
   │              internal/factions package                   │
   │                                                          │
   │  definitions: factionId → *Definition  (immutable)        │
   │  repCache:    factionId → *FactionRep  (sync.RWMutex)     │
   │                                                          │
   │  Public API: GetRep, SetRep, BumpRep, TierFor,            │
   │              GetDefinition, AllDefinitions,               │
   │              FactionsForMob, IsPeacefulToward, SaveAllRep │
   └─────┬───────────────────────────────────┬────────────────┘
         │                                   │
         │ BumpRep / SetRep                  │ GetRep / TierFor
         ▼                                   ▼
  ┌──────────────────────────┐    ┌─────────────────────────────┐
  │ Combat hookup (1.2):     │    │ Future consumers (later     │
  │ MobDeath_FactionRep.go   │    │ chunks): dialogue, quest,   │
  │ kills bump rep + party   │    │ town justice, btree...      │
  │                          │    │                             │
  │ Quest engine bump_rep    │    │                             │
  │ action (1.2 plumbing)    │    │                             │
  │                          │    │                             │
  │ lookfortrouble.go +      │    │                             │
  │ actions_party.go peace   │    │                             │
  │ check (post-migration)   │    │                             │
  └──────────────────────────┘    └─────────────────────────────┘

  ┌──────────────────────────┐
  │ admin.faction.go (1.2)   │
  │ list/show/set/bump/reset │
  └──────────────────────────┘
```

## Data model

### Definition file (committed)

```yaml
# _datafiles/world/dogmud/factions/warren.yaml
faction_id: warren
display_name: "The Warren"
description: |
  A coalition of warren scouts, warriors, and the tunnel shaman that
  have carved out territory in the Labyrinth of Low Tunnels.
  Surface-dwellers are mistrusted on sight.
default_rep: -25
allies: []
enemies: [thornwall_guards]
```

```yaml
# _datafiles/world/dogmud/factions/thornwall_guards.yaml
faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: |
  The city watch of Thornwall. They keep the streets safe and the
  gates manned. Indifferent to strangers; hostile to enemies of
  the city.
default_rep: 0
allies: []
enemies: [warren]
```

### Runtime rep file (not committed, gitignored)

```yaml
# _datafiles/world/dogmud/factions.rep/warren.yaml
faction_id: warren
players:
  17:
    rep: 30
    last_updated_round: 1843201
  92:
    rep: -75
    last_updated_round: 1846020
```

### In-memory types

```go
package factions

import "github.com/GoMudEngine/GoMud/internal/opinions"

type Definition struct {
    FactionId   string   `yaml:"faction_id"`
    DisplayName string   `yaml:"display_name"`
    Description string   `yaml:"description"`
    DefaultRep  int      `yaml:"default_rep"`
    Allies      []string `yaml:"allies"`
    Enemies     []string `yaml:"enemies"`
}

type RepEntry struct {
    Rep              int    `yaml:"rep"`
    LastUpdatedRound uint64 `yaml:"last_updated_round"`
}

type FactionRep struct {
    FactionId string             `yaml:"faction_id"`
    Players   map[int]*RepEntry  `yaml:"players"`
}

const (
    RepMin = -100
    RepMax = +100
)
```

## Public API

```go
package factions

// GetRep returns the player's reputation with the given faction,
// or the faction's default_rep if no row exists. Pure read — does
// not write to disk. Lazy cache priming may add an empty FactionRep
// to the cache on first access for a factionId, mirroring the
// opinions.Get pattern. Returns 0 if the faction is unknown.
func GetRep(factionId string, userId int) int

// SetRep assigns an absolute rep, clamped to [-100, +100], stamps
// last_updated_round, and persists synchronously.
func SetRep(factionId string, userId int, rep int)

// BumpRep adds delta to current rep, clamps, re-stamps, persists.
// No auto-propagation through allies/enemies.
func BumpRep(factionId string, userId int, delta int)

// TierFor returns opinions.TierOf(GetRep(factionId, userId)).
func TierFor(factionId string, userId int) opinions.Tier

// GetDefinition returns the immutable faction definition, or nil
// if the faction is unknown.
func GetDefinition(factionId string) *Definition

// AllDefinitions returns a snapshot of every loaded definition.
func AllDefinitions() []*Definition

// FactionsForMob returns the subset of mob.Groups that have
// definition files. Used by combat hookup and consumer code that
// needs to know "which factions does this mob belong to?"
func FactionsForMob(mob *mobs.Mob) []string

// IsPeacefulToward returns true if the player has Warm tier or
// higher with at least one of the mob's defined factions. Used by
// lookfortrouble.go and actions_party.go (replaces the old
// peacefulquest check).
func IsPeacefulToward(mob *mobs.Mob, userId int) bool

// SaveAllRep flushes dirty rep cache to disk. Defined for parity
// with opinions.SaveAllOpinions; not currently wired to shutdown.
func SaveAllRep()
```

Notes:
- **No `Delete`.** Reset to default via `SetRep(faction, user,
  GetDefinition(faction).DefaultRep)` (admin reset subcommand
  wraps this).
- **No event subscriptions.** Callers wire bumps explicitly. Same
  discipline as 1.1.
- **No cross-faction propagation.** If a quest should bump
  multiple factions, the quest YAML lists multiple `bump_rep`
  actions (one per faction).

## Persistence model

Mirrors `internal/opinions/persistence.go`:

- **Eager definition load:** `LoadAllDefinitions()` runs at server
  startup (called from the same path that loads mobs/items/etc.).
  Reads every YAML in `_datafiles/world/dogmud/factions/`. Validates
  every `allies:` / `enemies:` reference resolves to a known
  faction; **panics on unknown reference** so authoring errors
  surface at boot, not at runtime.
- **Lazy rep cache:** First `GetRep`/`SetRep`/`BumpRep` for a
  factionId triggers `loadRepFromDisk(factionId)`. On miss,
  synthesize an empty `FactionRep` and seed the cache (parallel
  to opinions.loadOrLazyInit). No row is added to the inner
  `Players` map until a mutation happens.
- **Synchronous save** on every Set/Bump. Same `saveMu sync.Mutex`
  pattern as opinions to serialize file I/O on Windows.
- **`SaveAllRep`** defined for future use; not called from
  shutdown (sync save covers it).

## Concurrency

Same model as `internal/opinions/`:
- `definitionsMu sync.RWMutex` — definitions are immutable after
  load, but the cache is protected for safety.
- `repCacheMu sync.RWMutex` — guards the per-factionId map.
- `saveMu sync.Mutex` — serializes the marshal+write critical
  section to avoid Windows ERROR_SHARING_VIOLATION on concurrent
  saves to the same file.

## Initial faction definitions (1.2 only)

Two factions ship in 1.2:

### `warren.yaml`

Members tagged in 1.2 (existing mobs gain `warren` in their groups
list):
- 72 — warren scout (`peacefulquest:` line dropped)
- 73 — warren warrior (`peacefulquest:` line dropped)
- 74 — tunnel shaman (warren added; was a neutral broker, but the
  fiction supports rep loss on killing them)

### `thornwall_guards.yaml` (narrow)

Members tagged in 1.2:
- 92 — city gate guard (thornwall_outskirts)
- 94 — guard captain Velk (thornwall_city)
- 106 — city guard (thornwall_city)

Civilians (beggar, merchants, performers, jeweler, enchanter)
explicitly **not** tagged in 1.2 — they belong to a future
`thornwall_citizens` faction added in 1.3.

The thug (105) is intentionally excluded from `thornwall_guards`
membership — they're a criminal, not a guard.

## Roadmap addition

A new chunk slots into Phase 6 (audit & polish) for the broad
faction content pass. Insert into the progress tracker:

```
| 6.5a | Polish | Faction definitions content pass | M | 1.2, 1.3 | Not started |
```

And in the Phase 6 chunk-detail section:

```markdown
### 6.5a Faction definitions content pass
**Status:** Not started • **Size:** M

- **Goal:** Author the rest of the world's factions on top of the
  1.2/1.3 substrate.
- **In:** YAML faction definitions for bandits, warden, ironwind
  shaman, Sanctum Basin guards, Dustwalk caravans, Stillwater
  militia & citizens, etc. Tag remaining faction-relevant mobs
  with their `groups: [<faction_id>]`. Define ally/enemy graphs
  across the full set. Surface any schema gaps the substrate
  didn't anticipate.
- **Out:** Per-faction quests (own content chunk).
- **Depends on:** 1.2 (substrate), 1.3 (citizens + alliance-aware
  consumer pattern, which becomes the template).
- **Why:** 1.2 ships substrate + warren + thornwall_guards. 1.3
  adds thornwall_citizens + alliance-aware guard logic. Bulk
  authoring the rest now would risk schema churn — better to
  validate the substrate against two reference factions, then
  bulk-author once the pattern is settled.
```

This entry commits with the spec.

## Test plan

### Unit (`internal/factions/factions_test.go`)
- `GetRep` returns `default_rep` when no row exists; doesn't write
  a row to disk.
- `SetRep` clamps to [-100, +100], stamps round, persists.
- `BumpRep` reads-then-adds-then-clamps-then-stamps.
- `BumpRep` on unknown faction is a no-op (warned via mudlog, not
  panicked).
- `TierFor` delegates to `opinions.TierOf` correctly across
  boundaries.
- `FactionsForMob` returns only the subset of `mob.Groups` that
  have definitions.
- `IsPeacefulToward` returns true when any of the mob's faction
  reps is at TierWarm or higher; false otherwise.

### Persistence (`internal/factions/persistence_test.go`)
- Save/load round-trip preserves the rep map.
- Missing rep file → return empty cache entry (seeded from
  definition default).
- Corrupt YAML → log warning, treat as missing.
- `SaveAllRep` writes only dirty entries.
- Concurrency: parallel `BumpRep` on the same `(factionId,
  userId)` converges (re-uses the saveMu pattern from opinions).

### Definition loader (`internal/factions/registry_test.go`)
- Loads every YAML in the factions dir at startup.
- Validates `allies:` / `enemies:` references; **panics at boot**
  on unknown reference.
- Re-loading is idempotent.

### Quest engine integration (`internal/questengine/actions_test.go`)
- New `bump_rep` action calls the bridge's `BumpRep` with the
  right faction + delta.
- Mock bridge captures the call for assertion.

### Combat hookup integration (`internal/hooks/MobDeath_FactionRep_test.go`)
- Player kills a warren scout → warren rep bumps by
  `FactionMemberKillRep`.
- Mob has no defined faction → no rep change.
- Mob killed by environment (no killer) → no rep change.
- Mob has multiple defined factions → all of them bump.
- Player in a party, mob killed in same room as party member →
  both party members get the bump.
- Party member in different room → does NOT get the bump.

### `peacefulquest` migration (`internal/users/migration_0_13_0_test.go`)
- User holds quest token "2-end" + has no warren rep → after
  migration, has +30 warren rep.
- User does NOT hold "2-end" → no rep added.
- Migration is idempotent (re-running it doesn't double the rep).

### Admin command (`internal/usercommands/admin.faction_test.go`)
- `faction list` outputs every loaded definition with brief
  metadata.
- `faction show <player>` lists only non-default rows.
- `faction set/bump/reset` mutate via the API and confirm.
- Unknown faction id / unknown player → friendly error, no panic.

### Live smoke test
- Restart preserves rep state across cache drop + file load.
- The `peacefulquest` migration completes cleanly on boot
  (verified visually in startup logs).
- Booting the server with the new factions loads definitions
  without panicking on ally/enemy references.

## Non-functional requirements

1. **Definitions load eagerly at startup.** Validate ally/enemy
   refs at boot; panic on unknown reference (catches authoring
   errors before they reach runtime).
2. **Rep state loads lazily.** Server boot does not eagerly read
   `factions.rep/`. Memory baseline at boot stays roughly
   unchanged.
3. **Synchronous save on every rep mutation.** No write
   amplification (bumps are infrequent). saveMu serializes file
   writes.
4. **Single source of truth for tier banding.** The `factions`
   package never defines its own tier enum; it imports
   `opinions.Tier` and calls `opinions.TierOf`.
5. **No player-facing numbers.** All player-visible relationship
   signals come through fiction. The faction admin command is
   admin-only.
6. **Discipline.** Future faction-rep mutations must route through
   `factions.BumpRep` / `factions.SetRep`. No direct cache writes
   outside the package.

## Files touched (estimate)

**New:**
- `internal/factions/types.go` — Definition, RepEntry, FactionRep,
  RepMin/Max
- `internal/factions/factions.go` — public API (Get/Set/Bump/Tier,
  FactionsForMob, IsPeacefulToward)
- `internal/factions/persistence.go` — file load/save, cache,
  saveMu, ClearCache, SaveAllRep
- `internal/factions/registry.go` — eager definition loader,
  ally/enemy validation
- `internal/factions/factions_test.go`
- `internal/factions/persistence_test.go`
- `internal/factions/registry_test.go`
- `internal/factions/test_main_test.go` — mudlog init
- `internal/hooks/MobDeath_FactionRep.go` — combat hookup
- `internal/hooks/MobDeath_FactionRep_test.go`
- `internal/usercommands/admin.faction.go`
- `internal/usercommands/admin.faction_test.go`
- `_datafiles/world/dogmud/factions/warren.yaml`
- `_datafiles/world/dogmud/factions/thornwall_guards.yaml`
- `_datafiles/world/dogmud/templates/admincommands/help/command.faction.template`

**Modified:**
- `internal/configs/config.balance.go` — add
  `FactionMemberKillRep` (default `-10`)
- `internal/configs/config.balance.misc.go` — default-setter for
  the new knob
- `internal/questengine/types.go` (or actions.go) — add
  `BumpRepDef`, extend `ActionDef` with `BumpRep *BumpRepDef`,
  extend `ActionContext` with `BumpRep(factionId string, delta
  int)`
- `internal/questengine/actions.go` — `ExecuteAction` dispatch
  case for the new verb
- `internal/questengine/bridge.go` — real-bridge `BumpRep`
  implementation calling `factions.BumpRep`
- `internal/usercommands/usercommands.go` — register `faction`
  admin command
- `internal/mobcommands/lookfortrouble.go` — replace
  peacefulquest gate with `factions.IsPeacefulToward`
- `internal/behaviortree/actions_party.go` — same gate
  replacement
- `internal/mobs/mobs.go` — delete `Mob.PeacefulQuest` field
- `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml` —
  add `bump_rep` action to end-step
- `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml`
  — drop peacefulquest, add warren to groups
- `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml`
  — drop peacefulquest, add warren to groups
- `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/74-tunnel_shaman.yaml`
  — add warren to groups
- `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml`
  — add thornwall_guards to groups
- `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml`
  — add thornwall_guards to groups
- `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml`
  — add thornwall_guards to groups
- Server startup path — call `factions.LoadAllDefinitions()`
- User-migration registry — add Migration 0.13.0
- `.gitignore` — add `_datafiles/world/dogmud/factions.rep/`
- `MOB_ALIVENESS_ROADMAP.md` — add chunk 6.5a

## Open questions for implementation

These don't block design approval but will be resolved during
plan-writing:

- **Migration registry path.** Locate where 0.10.0/0.11.0/0.12.0
  are defined and follow the convention for 0.13.0.
- **Companion-pet kill attribution.** Verify the
  `events.MobDeath.KillerUserId` field already populates with the
  pet owner, or whether we need an `IsCharmed(userId)` chain.
- **Party lookup API.** Confirm `parties.GetParty(userId)` exists
  with the right signature, or use whatever the actual accessor
  is.
- **Legacy quest token "2-end" usage audit.** Quick grep before
  writing the quest YAML change: does anything besides
  `peacefulquest` reference `2-end`? If yes, leaving the `grant:
  2-end` action stays the right call. If no, we can drop it too
  (preserves cleanup but is not required).
- **Eager-load ordering.** The factions registry loader must run
  AFTER mobs.LoadDataFiles (because we may want to validate that
  members exist), but it may run BEFORE if we don't add that
  check. Decide during planning.
