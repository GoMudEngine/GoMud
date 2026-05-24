# Mob Aliveness 1.3 — Crime / Wanted State

**Status:** Design • **Roadmap chunk:** 1.3 (Phase 1, Substrate) •
**Size:** M • **Depends on:** 1.2

## Goal

Per-faction, per-player log of crimes committed against faction
members — murder, assault, theft. Witnessed crimes identify the
perpetrator; unwitnessed crimes record the act with `perp: unknown`
to support future rumor systems (chunk 1.7) and forensic-style
gameplay. Crimes drive faction-rep changes through a unified
combat hookup that supersedes 1.2's flat kill-rep delta.

Without this, "the citizens of Thornwall know you murdered the
beggar in the market" is impossible — 1.2 only tracks per-faction
*reputation*, not the *acts* that drove the reputation. Town
justice (5.1) needs the act log to make sentencing decisions;
bounty hunting (5.2) needs it to declare bounties; rumors (1.7)
need it to seed gossip.

## Non-goals (explicit Out of scope)

- **Guard reactions / town justice.** Per the roadmap brief, guard
  behavior in response to crimes is **chunk 5.1**. 1.3 ships
  substrate + a rep-bump consumer; guards reading the crime log
  to decide aggro/arrest/escalation is later work.
- **Cry-for-help propagation.** Assault on a faction-member should
  arguably alert nearby NPCs in adjacent rooms — that's chunk
  **2.8 (mob scout/track)** territory. The 1.3 witness check is
  room-scoped only.
- **Forensic evidence trails.** A murder in an empty room could
  plausibly leave a corpse with stab wounds for guards to
  investigate. DOGMud doesn't model that today; lone murder =
  unrecorded perpetrator (perp: unknown).
- **Mob-on-mob harm as a crime.** Bandits killing a Thornwall
  citizen is plausibly a crime against the citizenry, but the 1.3
  hooks are scoped to player-caused harm. Mob-vs-mob crime
  recording could be added later as a sibling consumer of the
  same substrate.
- **Tavern gossip / rumor surfaces.** Chunk **1.7 (World-model
  facts)** consumes the crime log to seed rumor content. 1.3 just
  stores the data.
- **Crime-driven dialogue gating.** Same as 1.2's deferred
  dialogue work — the data is queryable, but no dialogue-engine
  wiring lands here.
- **Player-facing crime numbers.** Players never see numeric crime
  counts or rep deltas. CLAUDE.md "no hard numbers" rule applies.

## Decisions

### Crime kinds

Three kinds in scope for chunk 1.3:

- **`assault`** — recorded at first-aggression on a faction-member
  mob (same hook point as 1.2's opinion bump in `attack.go` /
  `target.go`). Victim is alive → self-witness. Other faction
  members in the room are also witnesses.
- **`murder`** — recorded at `MobDeath` of a faction member with
  a player damager. Victim is dead → not a self-witness. Witness
  check is "other faction-members in the room at death time."
  Lone murder records the row with `perp: unknown`.
- **`theft`** — recorded at FAILED `steal` / `pickpocket` only.
  Successful theft is silent in DOGMud's existing skill mechanic
  (no broadcast, no victim notification) and produces NO crime
  record. Failed theft already triggers `room.SendTextVisual`
  ("X catches you in the act") and a counterattack — that's the
  natural hook point.

Adding new kinds later is a matter of declaring the const and
wiring a hook — no schema break.

### Faction attribution

Crimes are **keyed by victim's factions**. Multi-faction
membership on victims (e.g., a guard who is also a citizen)
naturally produces multiple crime records — one per faction,
with the same metadata.

A witness is a mob INSTANCE in the same room as the act, whose
`mobs.Mob.Groups` include at least one of the victim's faction
IDs. The victim themselves counts as a self-witness EXCEPT when
they're dead (murder case — pass `excludeInstanceId = victim's
instance` to the witness helper).

**No allied-faction propagation.** Allies/enemies on faction
definitions stay declarative data. If `thornwall_citizens` and
`thornwall_guards` are allies and a guard witnesses a citizen
being murdered, the crime is NOT auto-recorded against the
guards' log — but the guards become witnesses to the citizens'
crime via the citizens-faction membership they share through
multi-membership. The data model handles allied-attribution
through tagging, not through propagation logic.

### Perpetrator identification

```go
type Perpetrator struct {
    Type PerpType // player | mob | unknown
    Id   int     // userId or mobId; absent for unknown
}
```

If at least one valid witness is present, the perpetrator is
identified (`Type: player, Id: userId`). If no witness, the
perpetrator is `unknown` and the crime is recorded but with no
rep impact (the world doesn't know whom to blame).

Future expansion (out of 1.3): mob-on-mob crimes would set
`Type: mob, Id: mobId`. The substrate doesn't enforce this in
1.3 — only player-caused harm fires the hooks.

### Storage layout

One file per faction at:

```
_datafiles/world/dogmud/factions.crimes/{slug}.yaml
```

(Gitignored; runtime state. Mirrors `factions.rep/`.)

```yaml
faction_id: thornwall_citizens
crimes:
  - id: 1
    kind: murder
    zone: "Thornwall City"
    room_id: 467
    round: 1843201
    victim_mob_id: 100
    victim_instance_id: 250
    perpetrator: { type: player, id: 17 }
    resolved_round: 0
    resolved_by: ""
  - id: 2
    kind: theft
    zone: "Thornwall City"
    room_id: 462
    round: 1845102
    victim_mob_id: 102
    victim_instance_id: 312
    perpetrator: { type: unknown }
    resolved_round: 0
    resolved_by: ""
```

Crime IDs are monotonic per-faction. On disk reload, the package
computes `nextId = max(crimes[].id) + 1`.

`.gitignore` adds `_datafiles/**/factions.crimes` alongside the
existing runtime-state entries.

### Persistence model

Mirrors `internal/factions/persistence.go`:

- In-memory cache `crimeCache: map[string]*FactionCrimes`,
  protected by `sync.RWMutex`.
- Lazy load on first access for a faction; on miss, returns an
  empty `FactionCrimes`.
- Synchronous save on every `Record` / `Resolve` mutation.
- `saveMu sync.Mutex` serializes file I/O to avoid Windows
  ERROR_SHARING_VIOLATION on concurrent saves to the same file.
- `SaveAllCrimes()` defined for parity; not currently wired to
  shutdown.

### Crime expiry / atonement

**Persist until resolved (preferred), with 365-day game-time
safety net.**

- Each crime has `resolved_round: uint64` (default 0 = unresolved)
  and `resolved_by: string` (free-form reason).
- `Resolve(factionId, crimeId, reason)` snaps the row to resolved
  state. Idempotent. Doesn't touch faction rep — restoration of
  rep is the consumer's call (5.1 town justice can decide whether
  paying a fine restores rep too).
- 365-day game-time safety net: a `PruneStale(factionId)` function
  iterates the log and resolves any unresolved crimes older than
  `Balance.CrimeStaleAfterRounds` (default ~365 days converted to
  rounds) with `resolved_by: "stale"`. Bounded-storage discipline.
- Resolved rows stay in the file as historical record (consumers
  may want to read sentencing history). The 365-day safety net is
  about ensuring no row stays unresolved indefinitely; rows
  themselves are never deleted in 1.3.

### Crime → faction-rep coupling (consolidated with 1.2)

Chunk 1.2 ships a flat `-10` rep drop per faction-member kill via
`MobDeath_FactionRep`. Chunk 1.3 **rewrites that hook** to be
crime-aware:

- Mob death → record murder crime → if perp identified, drop rep
  by `Balance.CrimeRepDeltaMurder` (-25).
- Assault → record assault crime → if perp identified, drop rep
  by `Balance.CrimeRepDeltaAssault` (-10).
- Failed theft → record theft crime → if perp identified, drop
  rep by `Balance.CrimeRepDeltaTheft` (-5).

For non-citizen factions (warren, generic enemies), the witness
check still works because faction members ARE the same-faction
witnesses (e.g., killing a warren scout when other warren scouts
are in the room → other scouts witness → perp identified → rep
drops by `CrimeRepDeltaMurder` = -25). The deliberate change from
1.2: the rep delta is now -25 instead of -10. Killing a faction
member is a more severe act than 1.2 modeled.

A player who stalks and kills lone faction members one at a time
no longer takes rep damage — the substrate respects "evidence
required." Intentional fiction lever.

**Assault upgrades to murder in place.** When `MobDeath` fires
for a player who already has a recent unresolved assault record
against the victim's faction, the existing assault row is
upgraded (kind changed to murder, perpetrator re-confirmed, room
+ round may be re-stamped to the death event). Rep delta for the
upgrade is `CrimeRepDeltaMurder - CrimeRepDeltaAssault` (-15
incremental). One row per fight, not two.

### Integration scope for chunk 1.3

Substrate + the consolidated `MobDeath_FactionRep` rewrite +
`steal` failure hookup + `attack`/`target` first-aggression
assault hookup + `thornwall_citizens` faction authoring + admin
command + helpfile.

Not wired in 1.3:
- Dialogue mood / quest reward → crime mutation (defer).
- Mob-on-mob crime recording (defer).
- Cry-for-help on assault (chunk 2.8).
- Adjacent-room witness propagation (chunk 2.8).
- Town-justice consumer (chunk 5.1).
- Rumor-system consumer (chunk 1.7).

### `thornwall_citizens` faction authoring

The 1.2 spec deferred this faction to 1.3. Definition file:

```yaml
# _datafiles/world/dogmud/factions/thornwall_citizens.yaml
faction_id: thornwall_citizens
display_name: "Thornwall Citizenry"
description: |
  The townsfolk of Thornwall — merchants, craftspeople, beggars,
  city officials, the elders who gather at the gate. They keep
  the city alive and watch each other's backs in small ways.
default_rep: 0
allies: [thornwall_guards]
enemies: []
```

(`thornwall_guards.yaml` gets `allies: [thornwall_citizens]`
added in this chunk to mirror the alliance.)

**Members tagged in 1.3** (existing mobs gain `thornwall_citizens`
in their `groups:` list):

- 95 temple priest Olen
- 96 tavern keeper Marek
- 97 blacksmith Kerra
- 98 apothecary Voss
- 99 records clerk Pell
- 100 city beggar
- 101 street performer
- 102 market merchant
- 103 food vendor
- 104 fence dealer Siv
- 108 jeweler Tess
- 109 enchanter Vael
- 113 weaver Maren
- 114 Old Fen
- 115 Old Gobb
- 116 Old Wrex
- 117 barmaid Dal
- 120 bank clerk
- 248 tavern cook Brynn
- 315 Sable

Plus **multi-faction membership for the three guards**: 92 city
gate guard, 94 guard captain Velk, 106 city guard each gain
`thornwall_citizens` ALONGSIDE their existing `thornwall_guards`.

Total: 23 mobs in `thornwall_citizens` (20 named civilians + 3
guards via multi-membership).

**Explicitly excluded:**
- 90 highwayman, 91 crop pest, 105 thornwall_thug,
  249 Torvan Cresk (smuggler leader), 357/358/359
  (caravan-org, separate faction)
- 244-247 smugglers under tavern (player-flagged exclusion)
- 270-273 phantom subzone (player-flagged exclusion)
- 107 Elara (not from Thornwall — player-flagged exclusion)
- 374 wagon, 375 Hob, 376 Bran (vehicles/animals)
- 89 farmer Dorn (outskirts, player-flagged)

**Pattern note for chunk 6.5a (broader content pass):** every
faction with both rank-and-file civilians AND a specialized
enforcement role should layer the memberships (e.g., Stillwater
militia members would also be Stillwater citizens). The 1.3
thornwall_citizens + thornwall_guards multi-membership is the
template.

### Admin command `crime`

Subcommands:

```
crime list <factionId>                          # unresolved crimes
crime list <factionId> --all                    # include resolved
crime show <player>                             # cross-faction unresolved for one player
crime show <player> --all                       # include resolved
crime resolve <factionId> <crimeId> <reason>    # mark resolved
crime prune-stale <factionId>                   # apply the 365-day stale safety net
```

Output style mirrors `opinion show` / `faction show` (ANSI
table, 80-col wrap). `--all` flag is admin-only nicety.

Helpfile at
`_datafiles/world/dogmud/templates/admincommands/help/command.crime.template`,
mirroring the existing peer command helpfiles. Registry entry:
`` `crime`: {Crime, true, true, true}, // Admin only ``,
inserted alphabetically between `craft` and `deafen`.

## Architecture

```
                ┌────────────────────────────────────────────────┐
                │ _datafiles/world/dogmud/factions.crimes/       │
                │   {slug}.yaml          (runtime, gitignored)   │
                └──────────────────┬─────────────────────────────┘
                                   │ lazy load + sync save
                                   ▼
   ┌──────────────────────────────────────────────────────────┐
   │              internal/crimes package                     │
   │                                                          │
   │  crimeCache: factionId → *FactionCrimes  (sync.RWMutex)  │
   │                                                          │
   │  Public API: Record, Resolve, FindRecentAssault,         │
   │              AllForFaction, AllForPlayer, SaveAllCrimes  │
   │              + helpers WitnessesInRoom, IdentifiedPerp   │
   └─────┬────────────────────────────────────┬───────────────┘
         │ Record / Resolve                   │ AllForFaction / Player
         ▼                                    ▼
  ┌────────────────────────────┐   ┌─────────────────────────────┐
  │ Combat hookup (1.3):        │   │ Future consumers (later     │
  │  MobDeath_FactionRep        │   │ chunks):                    │
  │  rewrite (replaces 1.2's    │   │  - 1.7 rumors / world facts │
  │  flat kill-rep)             │   │  - 5.1 town justice         │
  │                             │   │  - 5.2 bounty hunting       │
  │ steal-failure hookup in     │   └─────────────────────────────┘
  │  skill.skullduggery.steal   │
  │                             │
  │ first-aggression assault    │
  │  hookup in attack.go +      │
  │  target.go                  │
  └────────────────────────────┘

  ┌────────────────────────────┐
  │ admin.crime.go (1.3)        │
  │ list / show / resolve /     │
  │ prune-stale                 │
  └────────────────────────────┘
```

## Public API

```go
package crimes

import (
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/rooms"
)

type Kind string

const (
    KindAssault Kind = "assault"
    KindMurder  Kind = "murder"
    KindTheft   Kind = "theft"
)

type PerpType string

const (
    PerpPlayer  PerpType = "player"
    PerpMob     PerpType = "mob"      // future use
    PerpUnknown PerpType = "unknown"
)

type Perpetrator struct {
    Type PerpType `yaml:"type"`
    Id   int      `yaml:"id,omitempty"`
}

type Crime struct {
    Id               int         `yaml:"id"`
    Kind             Kind        `yaml:"kind"`
    Zone             string      `yaml:"zone"`
    RoomId           int         `yaml:"room_id"`
    Round            uint64      `yaml:"round"`
    VictimMobId      int         `yaml:"victim_mob_id"`
    VictimInstanceId int         `yaml:"victim_instance_id"`
    Perpetrator      Perpetrator `yaml:"perpetrator"`
    ResolvedRound    uint64      `yaml:"resolved_round"`
    ResolvedBy       string      `yaml:"resolved_by"`
}

// Record creates a new crime row on each affected faction's log.
// Returns the new crime IDs (parallel to factionIds).
func Record(
    factionIds []string,
    kind Kind,
    perp Perpetrator,
    victim *mobs.Mob,
    instanceId int,
    roomId int,
    zone string,
) []int

// Resolve marks a specific crime as resolved. Idempotent.
func Resolve(factionId string, crimeId int, resolvedBy string)

// FindRecentAssault returns the most recent unresolved assault
// crime committed by `userId` against any mob in `factionId`
// within `lookbackRounds`. Used to upgrade-in-place when a fight
// ends in death.
func FindRecentAssault(factionId string, userId int, lookbackRounds uint64) *Crime

// AllForFaction returns crimes against the given faction.
func AllForFaction(factionId string, includeResolved bool) []*Crime

// AllForPlayer returns crimes naming this userId as the
// identified perpetrator, across all factions.
func AllForPlayer(userId int, includeResolved bool) []*Crime

// PruneStale resolves all unresolved crimes older than
// Balance.CrimeStaleAfterRounds with reason "stale".
func PruneStale(factionId string) int

// SaveAllCrimes flushes dirty caches.
func SaveAllCrimes()

// Helpers

// WitnessesInRoom returns the list of faction-aligned mob
// instances in the given room that witness a crime against the
// listed factions. Pass excludeInstanceId = victim's instance
// for murders (victim is dead, not a self-witness); pass 0 for
// assault and theft (victim is alive and a self-witness).
func WitnessesInRoom(factionIds []string, room *rooms.Room, excludeInstanceId int) []int

// IdentifiedPerp returns PerpPlayer if witnesses is non-empty,
// otherwise PerpUnknown.
func IdentifiedPerp(userId int, witnesses []int) Perpetrator
```

## Data flow — combat death

1. Player kills a faction-member mob (warren scout, citizen, etc.).
2. Combat round detects health <= 0 → calls `mob.Command(suicide)`.
3. `mobcommands/suicide.go` queues `events.MobDeath{...}`.
4. Hook `MobDeath_FactionRep` fires (after instance is destroyed).
5. Hook resolves the mob TEMPLATE via `mobs.GetMobSpec(MobId)` —
   not `GetInstance` (already verified bug-fixed in chunk 1.2's
   tail end of work).
6. Hook calls `factions.FactionsForMob(spec)` to get the list of
   defined factions the victim belonged to.
7. Hook calls `crimes.WitnessesInRoom(factionIds, room,
   evt.InstanceId)` to find faction-member witnesses (excluding
   the victim).
8. For each player in the build set (damagers + same-room party
   members from chunk 1.2's logic), for each affected faction:
   a. Look up `crimes.FindRecentAssault(fid, userId, 100)` — if
      present, upgrade kind to murder + re-stamp metadata.
   b. Otherwise, `crimes.Record([fid], KindMurder, perp, ...)`.
   c. If `perp.Type == PerpPlayer`, `factions.BumpRep(fid,
      userId, CrimeRepDeltaMurder)` — the upgrade case bumps by
      the incremental `Murder - Assault` delta only (since the
      assault delta was already applied at first-aggression).

## Data flow — first-aggression assault

1. Player invokes `attack <target>` against a faction-member mob.
2. `attack.go` detects `isFreshAggro` (existing 1.2 logic).
3. `factions.BumpRep(fid, userId, OpinionAttackBump)` — the chunk
   1.2 opinion bump (per-NPC opinion store).
4. **NEW in 1.3:** for each defined faction the mob is in:
   a. `crimes.WitnessesInRoom(factionIds, room, 0)` — victim
      counts as self-witness here.
   b. `crimes.Record([fid], KindAssault, perp, ...)`.
   c. If perp identified, `factions.BumpRep(fid, userId,
      CrimeRepDeltaAssault)`.
5. Same hook for `target.go`'s target-switch helper
   `bumpOpinionOnTargetSwitch` (extended to also record the
   crime).

## Data flow — failed theft

1. Player invokes `steal <target>`.
2. `skill.skullduggery.steal.go` rolls the opposed check.
3. On failure path (existing code: room broadcast + counterattack):
   a. `factions.FactionsForMob(m)` — does target have a faction?
   b. If yes, `crimes.WitnessesInRoom(factionIds, room, 0)` —
      victim is alive, self-witnesses.
   c. `crimes.Record([fid], KindTheft, perp, ...)`.
   d. If perp identified, `factions.BumpRep(fid, userId,
      CrimeRepDeltaTheft)`.
4. Successful theft path: NO crime record (silent mechanic).

## Concurrency

Same model as `internal/factions/`:
- `crimeCacheMu sync.RWMutex` guards the per-factionId map.
- `saveMu sync.Mutex` serializes file I/O on Windows.
- `nextId` per `FactionCrimes` is mutated under cache write lock.

## Test plan

### Unit (`internal/crimes/crimes_test.go`)
- `Record` writes a row with new ID and persists.
- `Resolve` marks the row; idempotent.
- `FindRecentAssault` returns most recent unresolved assault for
  (player × faction) within window; nil if none / all resolved.
- `AllForFaction(includeResolved=false)` filters resolved rows.
- `AllForPlayer` walks every loaded faction.
- `PruneStale` snaps unresolved crimes older than threshold to
  `resolved_by: "stale"`.

### Witness helper tests
- `WitnessesInRoom` returns instance IDs whose `Groups` overlap
  with `factionIds`, excluding `excludeInstanceId`.
- Empty room → empty.
- `IdentifiedPerp` empty witnesses → unknown; non-empty → player.

### Persistence (`internal/crimes/persistence_test.go`)
- Save/load round-trip preserves rows including resolved status.
- Missing file → empty FactionCrimes.
- Corrupt YAML → log warning, treat as empty.
- `nextId` resumes from `max(crimes[].id) + 1` on reload.
- Concurrency: parallel `Record` on same faction converges with
  unique IDs (saveMu).

### Combat hookup integration (`internal/hooks/MobDeath_FactionRep_test.go` extended)
- Player kills faction member with same-faction witness → murder
  recorded with player perp, `CrimeRepDeltaMurder` applied.
- Lone murder → murder recorded with `unknown` perp, NO rep bump.
- Non-faction kill → no crime, no rep change.
- Assault-then-kill: ONE row, kind=murder (upgrade-in-place), NOT
  two.
- Party-room propagation preserved from chunk 1.2.

### Steal-failure (`internal/usercommands/skill.skullduggery.steal_test.go` new)
- Failed steal from faction-member → theft crime recorded with
  player perp, `CrimeRepDeltaTheft` applied.
- Successful steal → no crime.
- Failed steal from non-faction mob → no crime.

### Assault hookup (`internal/usercommands/attack_test.go` extended)
- Fresh aggression on faction-member records assault crime in
  addition to chunk 1.2's opinion bump.
- Same-target re-attack does not double-record.
- Both `attack` and `attack #other` (target switch) record
  assault.

### Admin command (`internal/usercommands/admin.crime_test.go`)
- `crime list <factionId>` outputs unresolved rows; `--all`
  includes resolved.
- `crime show <player>` walks all factions.
- `crime resolve <factionId> <crimeId> <reason>` marks the row.
- Unknown faction / unknown crime ID → friendly error, no panic.

### Live smoke test (manual smoketester walkthrough)
1. Edit user 17 yaml: place in a Thornwall city room, ensure no
   pre-existing warren rep / no crime records.
2. Boot server. Watch for clean factions + crimes load.
3. Smoketester walks:
   - Attack a beggar in front of other citizens → admin
     `crime list thornwall_citizens` shows assault row.
   - Kill the beggar → assault upgraded to murder in place;
     `faction show smoketester` shows the murder rep delta.
   - Find a lone citizen, murder them → admin
     `crime list thornwall_citizens` shows the murder row with
     `perp: unknown`, no additional rep change.
   - Attempt steal-and-fail → theft crime recorded.
   - Admin `crime show smoketester` lists all identified crimes.
   - Admin `crime resolve thornwall_citizens <id> "fine paid"`
     clears one row.
4. Cleanup: kill server per SOP, reset smoketester.

## Non-functional requirements

1. **Single source of truth.** All crime mutations route through
   `crimes.Record` / `crimes.Resolve`. No direct cache writes.
2. **Synchronous save on every mutation.** `saveMu` serializes
   file I/O.
3. **Lazy load.** Server startup must not eagerly read every
   faction.crimes file.
4. **Witness check is room-scoped.** No adjacent-room propagation.
5. **No player-facing crime numbers.** All player-visible signals
   come through fiction.
6. **Storage size discipline.** Per-faction file growth bounded
   by 365-day stale safety net + manual `crime prune-stale`.
   Resolved rows stay (historical record); unresolved rows can't
   accumulate beyond the threshold.

## Files touched (estimate)

**New:**
- `internal/crimes/types.go` — Kind, PerpType, Perpetrator, Crime, FactionCrimes
- `internal/crimes/crimes.go` — public API (Record, Resolve, FindRecentAssault, AllForFaction, AllForPlayer, PruneStale, SaveAllCrimes, WitnessesInRoom, IdentifiedPerp)
- `internal/crimes/persistence.go` — file load/save, cache, saveMu
- `internal/crimes/crimes_test.go`
- `internal/crimes/persistence_test.go`
- `internal/crimes/test_main_test.go` — mudlog init
- `internal/usercommands/admin.crime.go`
- `internal/usercommands/admin.crime_test.go`
- `_datafiles/world/dogmud/factions/thornwall_citizens.yaml`
- `_datafiles/world/dogmud/templates/admincommands/help/command.crime.template`

**Modified:**
- `internal/configs/config.balance.go` — add `CrimeRepDeltaMurder` (-25), `CrimeRepDeltaAssault` (-10), `CrimeRepDeltaTheft` (-5), `CrimeStaleAfterRounds` (~365 days as rounds)
- `internal/configs/config.balance.misc.go` — defaults
- `internal/hooks/MobDeath_FactionRep.go` — rewrite to be
  crime-aware (replaces flat -10 with kind-specific deltas via
  the crimes package)
- `internal/hooks/MobDeath_FactionRep_test.go` — extended tests
- `internal/usercommands/skill.skullduggery.steal.go` — failure
  hookup
- `internal/usercommands/skill.skullduggery.steal_test.go` — new
  failure tests
- `internal/usercommands/attack.go` — first-aggression assault
  recording (alongside 1.2's opinion bump)
- `internal/usercommands/target.go` — same hookup in
  `bumpOpinionOnTargetSwitch`
- `internal/usercommands/attack_test.go` — extended
- `internal/usercommands/usercommands.go` — register `crime`
- `_datafiles/world/dogmud/factions/thornwall_guards.yaml` — add
  `allies: [thornwall_citizens]`
- 20 thornwall_city/outskirts mob YAMLs — add
  `thornwall_citizens` to existing groups
- 3 thornwall guard mob YAMLs (92, 94, 106) — add
  `thornwall_citizens` alongside existing `thornwall_guards`
- `.gitignore` — add `_datafiles/**/factions.crimes`
- `MOB_ALIVENESS_ROADMAP.md` — flip 1.3 to `In progress`, then
  `Done` after smoke test

## Open questions for implementation

These don't block design approval but will be resolved during
plan-writing:

- **Round-count for 365-day game-time.** DOGMud's day length is
  configurable; the conversion `365 days × hours/day × rounds/hour`
  needs to read from configs. Concrete value lives in
  `Balance.CrimeStaleAfterRounds` and gets computed during
  default-setting if zero.
- **Steal-failure hook placement.** The existing failure broadcast
  is inside an else-branch; the simplest insert point is right
  after that block. Verify during planning that the early-return
  paths (skill rank gate, target validation rebuff) DON'T trigger
  the crime hook.
- **Assault-upgrade semantics.** When upgrade-in-place fires,
  should the row's `round`, `room_id`, `victim_instance_id` get
  re-stamped to the death event? Probably yes for last-known-state
  accuracy, but the `id` stays the same. Decide during planning.
- **Steal-failure-with-no-room-witness edge case.** What if the
  steal fails but there are NO faction members in the room
  (player stealing from a lone NPC who isn't in a faction)? Then
  `factionIds` is empty, no crime recorded. That's correct — no
  faction was wronged. But for ungrouped victims, this means
  theft is silent regardless of success/failure. Document.
