# Bounties System Context

## Overview

The `internal/bounties` package maintains a single registry of active bounties — contracts issued by factions, quests, or NPCs offering gold or reputation rewards for specific conditions (currently kill-only). Each bounty identifies the issuer, the target (player or mob template), the reward (auto-computed from target statpool at declaration time, or overridden by the issuer), the condition, and the expiry round. Bounties are recorded synchronously and persisted to disk. The auto-claim hook on `MobDeath` transfers rewards to the killer when a bounty's mob target dies, and is consumed by future systems (bounty hunters, town justice escalation).

## Key Components

### Core Files
- **types.go**: Enums and data structures (`IssuerType`, `Issuer`, `Condition`, `Status`, `Bounty`, `Registry`)
- **bounties.go**: Public API (`FactionIssuer`, `QuestIssuer`, `NPCIssuer`, `Declare`, `Get`, `AllOpen`, `OpenForTarget`, `OpenForIssuer`, `OpenAgainstPlayer`, `AllForTarget`, `AllRows`, `TryClaim`, `Withdraw`, `MarkExpired`, `PruneExpired`); helper functions (`computeDefaultGold`, `computeDefaultRep`, `statpoolFor`); lazy cache initialization; test seams
- **persistence.go**: Disk I/O (`saveRegistry`, `loadRegistryFromDisk`, `loadOrLazyInit`); cache management; file path generation
- **bounties_test.go**: Unit tests for Declare (defaults and overrides), reward computation, query methods, claim/withdraw/expiry logic, and persistence
- **persistence_test.go**: Tests for disk I/O, cache lifecycle, and concurrent saves
- **test_main_test.go**: TestMain setup with environment-variable override for temp-dir isolation

## Key Functions

### Issuer Helpers
- **FactionIssuer(slug string) Issuer**: Constructs an `Issuer` with `Type: IssuerFaction` and `Id: slug`. Returns a faction issuer by slug.
- **QuestIssuer(questId int) Issuer**: Constructs an `Issuer` with `Type: IssuerQuest` and `Id: strconv.Itoa(questId)`. Returns a quest issuer by stringified quest ID.
- **NPCIssuer(mobId int) Issuer**: Constructs an `Issuer` with `Type: IssuerNPC` and `Id: strconv.Itoa(mobId)`. Returns an NPC issuer by stringified mob template ID.

### Write API
- **Declare(issuer Issuer, target knowledge.Subject, condition Condition, expiryRound uint64, opts DeclareOpts) (int, error)**: Creates a new bounty and persists it. Returns the assigned bounty ID. Gold and rep rewards are auto-computed from target statpool unless overridden via `DeclareOpts.GoldOverride` and `DeclareOpts.RepOverride`. Stamps `DeclaredRound` to current round. Sets status to `StatusOpen`. Optionally records `DeclaredReason` from opts.

- **TryClaim(bountyId int, claimer knowledge.Subject) (*Bounty, bool)**: Records a claim on the given bounty. Returns (bounty, true) on success — status was open and transitions to claimed, claimer is recorded, claimed_round is stamped. Returns (nil, false) if the bounty was already non-open (already claimed, expired, or withdrawn).

- **Withdraw(bountyId int)**: Transitions an open bounty to withdrawn status. Idempotent — re-withdrawing a withdrawn bounty is a no-op.

- **MarkExpired(bountyId int)**: Transitions an open bounty to expired status. Idempotent — re-marking an expired bounty is a no-op.

- **PruneExpired() int**: Walks open bounties and flips any past their expiry round to status=expired. Returns the count flipped. Expiry round of 0 means never expires. Persists synchronously.

### Read API
- **Get(bountyId int) *Bounty**: Returns the bounty with the given ID, or nil if not found. Does not trigger cache load if the bounty doesn't exist; uses in-memory cache.

- **AllOpen() []*Bounty**: Returns all bounties with status=open. Snapshot copy.

- **OpenForTarget(target knowledge.Subject) []*Bounty**: Returns all bounties with status=open targeting the given subject (player or mob template). Snapshot copy.

- **OpenForIssuer(issuer Issuer) []*Bounty**: Returns all bounties with status=open issued by the given issuer. Snapshot copy.

- **OpenAgainstPlayer(userId int) []*Bounty**: Convenience wrapper for `OpenForTarget(PlayerSubject(userId))`. Returns all open bounties targeting the given player.

- **AllForTarget(target knowledge.Subject, includeNonOpen bool) []*Bounty**: Returns all bounties targeting the given subject. If `includeNonOpen=false`, only status=open rows are included. If true, all statuses are included. Snapshot copy.

- **AllRows() []*Bounty**: Returns a snapshot of every row in the registry, regardless of status. Used by admin `--all` listing.

## Global State

### Cache and Mutex
- **registry**: `*Registry` holding the loaded bounty registry in memory. Nil until first access.
- **registryMu**: `sync.RWMutex` protecting the registry and all in-cache mutations. Write operations (Declare, TryClaim, Withdraw, MarkExpired, PruneExpired) acquire the write lock; read operations acquire the read lock.
- **saveMu**: `sync.Mutex` serializing disk writes (Windows file-lock safety). Mirrors the `internal/crimes`, `internal/opinions`, and `internal/knowledge` patterns.

### Test Seams
- **roundForTest**: Injection hook for `currentRound()` — lets tests freeze or advance the round count without reading `util.GetRoundCount()`. Production never sets this; tests assign a closure and must clear it after use.
- **goldMultiplierForTest**: Injection hook for `goldMultiplier()` — lets tests override `configs.GetBalanceConfig().BountyGoldDefaultMultiplier` without standing up the full configs stack.
- **goldFloorForTest**: Injection hook for `goldFloor()` — lets tests override `configs.GetBalanceConfig().BountyGoldFloor` without standing up the full configs stack.
- **statpoolForTest**: Injection hook for `statpoolFor()` — lets tests provide a fake implementation of target statpool lookup (mobs.GetMobSpec and users.GetByUserId). Production maps to the real lookups.

## Data Structure Design

### Issuer Polymorphism
```go
type IssuerType string
const (
    IssuerFaction IssuerType = "faction"
    IssuerQuest   IssuerType = "quest"
    IssuerNPC     IssuerType = "npc"
)

type Issuer struct {
    Type IssuerType `yaml:"type"`
    Id   string     `yaml:"id"`  // faction: slug; quest/npc: stringified int
}
```
The `Issuer` is a tagged (type, id) pair. For factions, id is the faction slug. For quests and NPCs, id is the stringified integer (questId or mobId). This allows the same bounty registry to service issuers from three different substrates.

### Target Reuse: knowledge.Subject
```go
// From internal/knowledge
type SubjectType string
const (
    SubjectPlayer SubjectType = "player"
    SubjectMob    SubjectType = "mob"
)

type Subject struct {
    Type SubjectType `yaml:"type"`
    Id   int         `yaml:"id"`
}
```
Bounties reuse `knowledge.Subject` for the target. For players, `Type: SubjectPlayer` and `Id` is the userId. For mobs, `Type: SubjectMob` and `Id` is the mob TEMPLATE id (not instance). This polymorphism enables the same bounty substrate to serve both wanted-player bounties (town justice) and notorious-mob bounties (bounty hunters).

### Status and Condition Enums
```go
type Status string
const (
    StatusOpen      Status = "open"
    StatusClaimed   Status = "claimed"
    StatusExpired   Status = "expired"
    StatusWithdrawn Status = "withdrawn"
)

type Condition string
const (
    ConditionKill Condition = "kill"  // v1 only; future: capture, deliver_item
)
```

### Bounty Row
```go
type Bounty struct {
    Id             int               `yaml:"id"`             // monotonic per registry
    Issuer         Issuer            `yaml:"issuer"`
    Target         knowledge.Subject `yaml:"target"`
    GoldReward     int               `yaml:"gold_reward"`    // locked at declare
    RepReward      int               `yaml:"rep_reward"`     // locked at declare
    Condition      Condition         `yaml:"condition"`
    DeclaredRound  uint64            `yaml:"declared_round"`
    ExpiryRound    uint64            `yaml:"expiry_round"`   // 0 = never
    Status         Status            `yaml:"status"`
    ClaimedBy      knowledge.Subject `yaml:"claimed_by,omitempty"`
    ClaimedRound   uint64            `yaml:"claimed_round,omitempty"`
    DeclaredReason string            `yaml:"declared_reason,omitempty"`
}
```
Each bounty carries full contract details. The id is monotonic per registry (scoped to all bounties, not per-issuer). Gold and rep are fixed at declaration time to give issuers predictable costs. Status tracks the contract lifecycle: open (available), claimed (successful completion), expired (past expiry round), withdrawn (issuer cancelled).

### Registry (On Disk)
```yaml
# _datafiles/world/dogmud/bounties.yaml (gitignored)
next_id: 7
bounties:
  - id: 1
    issuer:
      type: faction
      id: thornwall_guards
    target:
      type: player
      id: 17
    gold_reward: 300
    rep_reward: 6
    condition: kill
    declared_round: 2065600
    expiry_round: 2073400
    status: open
    claimed_by:
    claimed_round: 0
    declared_reason: "Murder of Tavern Keeper Marek"
  - id: 2
    issuer:
      type: npc
      id: 3102
    target:
      type: mob
      id: 3103
    gold_reward: 200
    rep_reward: 2
    condition: kill
    declared_round: 2065700
    expiry_round: 0
    status: claimed
    claimed_by:
      type: player
      id: 18
    claimed_round: 2065750
    declared_reason: ""
```
The `Registry` struct holds `NextId` (auto-increment counter) and `Bounties` (slice of `*Bounty`). One file per server, stored at `_datafiles/world/dogmud/bounties.yaml`, gitignored. The registry is loaded lazily on first access and cached in memory.

## Reward Auto-Compute

Gold and rep rewards are computed at declaration time from the target's statpool. Compute-at-declare (not compute-at-claim) means the issuer "knows" what they are paying when they post the contract, and the reward doesn't drift if the target is buffed or debuffed between declaration and claim.

### Gold Reward Calculation
```
gold = floor(target_statpool × BountyGoldDefaultMultiplier), with minimum BountyGoldFloor
```
- **Config**: `BountyGoldDefaultMultiplier` (default 0.5), `BountyGoldFloor` (default 50)
- **Rationale**: Scales with target power (statpool). The multiplier is a balance knob controlling issuer cost. The floor ensures even trivial mobs have a meaningful reward.
- **Override**: Via `DeclareOpts.GoldOverride`. If set to non-zero, the auto-computed value is ignored.

### Rep Reward Calculation
```
rep = max(1, floor(target_statpool / 100))
```
- **Rationale**: Dividing by 100 normalizes statpool (which centers at 100 per stat, 6 stats = 600 baseline) to roughly 1 rep per 100 statpool. The floor of 1 ensures every target yields at least 1 rep.
- **Override**: Via `DeclareOpts.RepOverride`. If set to non-zero, the auto-computed value is ignored.

### Statpool Lookup
The `statpoolFor(target knowledge.Subject)` helper resolves target statpool as follows:
- **Mob targets**: Uses `mobs.GetMobSpec(target.Id).StatPool` — the pre-spawn point budget from the mob template.
- **Player targets**: Sums `ValueAdj` across all 6 stats (`Strength.ValueAdj + Dexterity.ValueAdj + ...`) from the online user. Offline player lookup is not supported in v1; returns 0, which causes gold floor and rep floor of 1 to apply.

## Integration Notes

### Auto-Claim on MobDeath
When a mob target dies and a player is the highest damager, the `MobDeath_BountyClaim` hook (in `internal/hooks/`) auto-claims all open bounties on that target:
1. Find all open bounties with `Target == MobSubject(deadMobTemplateId)`.
2. Call `TryClaim(bountyId, PlayerSubject(killerId))` for each.
3. Transfer gold via `user.Character.Gold += bounty.GoldReward`.
4. If issuer is a faction, bump rep via `factions.BumpRep(issuer.Id, killerId, bounty.RepReward)`.

Companion damage already rolls up to the owning player via `combat.go`'s `GetCharmedUserId` path, so companion-heavy builds get the bounty credit they earned with no extra bounty-layer logic.

### Quest Engine Integration
The quest engine's `declare_bounty` action allows quest scripts to issue bounties:
```go
declare_bounty: {issuer_type: "quest", target_type: "player", target_id: 17, 
                 condition: "kill", expiry_round: 3000, 
                 gold_override: 500}
```
The action resolves issuer type and target type from the quest definition, calls `Declare` with the supplied options, and persists synchronously.

### Player Command
The `bounty` command with role-gated subcommands:
- **bounty list**: Lists all open bounties. Show-only (no perms required).
- **bounty show <id>**: Shows details of a single bounty (open or closed). Show-only.
- **bounty declare**: Admin-only subcommand to manually declare a bounty. Used for debugging and town-justice integration.
- **bounty withdraw <id>**: Admin-only subcommand to cancel an open bounty.
- **bounty prune-expired**: Admin-only subcommand to mark all past-expiry rows as expired.

### Physical Bounty Boards
Two flavor nouns (lookable objects):
- **Thornwall Guard Barracks, room 473**: `bounty board` — lists all open faction-issued bounties.
- **Stillwater Constabulary, room 4110**: `wanted board` — lists all open player-target bounties (wanted posters).

Flavor text reads from `AllOpen()` and displays per-issuer sections.

### Dependencies
- **configs**: Reads `BountyGoldDefaultMultiplier` and `BountyGoldFloor` for reward auto-compute.
- **knowledge**: Imports `Subject` type for polymorphic target. `SubjectType`, `SubjectPlayer`, `SubjectMob` enums and constructors.
- **mobs**: Calls `GetMobSpec()` to fetch statpool for mob targets in `statpoolFor()`.
- **users**: Calls `GetByUserId()` to fetch statpool for player targets in `statpoolFor()`.
- **factions**: Called by `MobDeath_BountyClaim` to bump rep when faction-issued bounties are claimed.
- **util**: Uses `GetRoundCount()` for timestamps and `ConvertForFilename()` for path generation.
- **mudlog**: Logs warnings on disk I/O errors.

### Imported By
- `internal/hooks/MobDeath_BountyClaim` (auto-claim on mob death)
- `internal/questengine/actions` (declare_bounty action)
- `internal/usercommands/admin.bounty` (admin API + player list/show commands)

## Memory Reporting

`memory.go` registers a `util.AddMemoryReporter` under the section name
`Bounties`, surfacing the in-memory store size in the `server stats` admin
command (mob-aliveness chunk 6.4). One row is reported: `registry`
(total number of bounty rows loaded, nil-safe — 0 when the registry has
not yet been loaded from disk).

## Testing Notes

### Test Files
- **bounties_test.go**: ~250 lines covering computeDefaultGold (multiplier + floor), computeDefaultRep (statpool / 100, floor of 1), Declare (defaults and overrides, status/round stamping), Get (cache lookup), AllOpen/OpenForTarget/OpenForIssuer (filtering), AllForTarget (with includeNonOpen toggle), AllRows (snapshot), TryClaim (transition to claimed, idempotence), Withdraw (transition to withdrawn, idempotence), MarkExpired (transition to expired, idempotence), and PruneExpired (batch expiry).
- **persistence_test.go**: ~80 lines covering disk I/O (save/load roundtrip), cache lifecycle (lazy init, double-check-lock), and concurrent save serialization via saveMu.
- **test_main_test.go**: TestMain that sets `DOGMUD_BOUNTIES_DIR_OVERRIDE` to a temp dir so tests don't touch the real bounties.yaml.

### Test Seam Pattern
Tests inject fake implementations via global function pointers (`roundForTest`, `goldMultiplierForTest`, `goldFloorForTest`, `statpoolForTest`) without building the full configs, mobs, or users stacks. After each test, seams are cleared so production code sees nil and uses the real implementations. The pattern is identical to the 1.3 crimes and 1.4 knowledge packages.

### Key Test Scenarios
- `computeDefaultGold` applies multiplier with a floor minimum.
- `computeDefaultRep` divides by 100 with a floor of 1.
- `Declare` auto-computes rewards from statpool or accepts overrides.
- `Declare` assigns monotonic IDs and stamps DeclaredRound.
- `TryClaim` transitions open → claimed, records claimer and round.
- `TryClaim` returns (nil, false) on non-open bounties.
- `Withdraw` / `MarkExpired` idempotently flip status.
- `PruneExpired` respects expiry round threshold, skips expiry_round=0.
- `AllOpen` / `OpenForTarget` / `OpenForIssuer` filter correctly.
- `AllForTarget` respects includeNonOpen toggle.
- Persistence: files are created on first write, loaded lazily on first access, saved synchronously on every write operation.
