# Mob Aliveness 1.5 — Bounty State

> **Phase 1 substrate.** Per-bounty registry: who issued, who's
> wanted, what's the reward, when does it expire. v1 supports all
> three issuer types (faction, quest, NPC) and both target types
> (player, mob template). Auto-claim on MobDeath when target dies.
> Player-facing `bounty list` command + two physical bounty boards
> (Thornwall Guard Barracks, Stillwater Constabulary).

## Goal

Per-bounty record of declared contracts: payer, target, reward,
condition, expiry. Queryable by mobs (5.2 bounty hunters in the
future) and players (`bounty list` and physical boards). Lets town
justice (5.1) escalate from crime → bounty, lets quest content
declare price-on-the-head contracts, lets NPCs post their own
grievances ("the merchant guild offers 500g for the bandit chief").

The chunk's job is **substrate only** — the storage, declaration
API, claim/resolve API, expiry handling, admin command, player
command, and a thin auto-claim hook on MobDeath. Bounty hunter
behavior (5.2) and town-justice escalation (5.1) consume this
substrate later.

## Architectural musts

The brief in MOB_ALIVENESS_ROADMAP.md lists "payer, target, reward,
conditions, expiry" as the schema sketch. During brainstorming we
locked in the following architectural choices:

1. **Polymorphic target.** Reuse chunk 1.4's `knowledge.Subject`
   for the target field. Player-target and mob-template-target
   bounties share the same schema. This is what makes the same
   substrate serve both 5.1 (wanted players) and 5.2 (notorious
   NPC bandits).

2. **Three issuer types.** `Issuer` is a tagged `(type, id)` pair
   with type ∈ {faction, quest, npc} and a string id (faction
   slug, quest id stringified, mob template id stringified). The
   id is string-typed because faction IDs are slugs. Future v2
   can add `IssuerPlayer` for player-funded bounties.

3. **Single registry file.** `_datafiles/world/dogmud/bounties.yaml`
   (gitignored) holds every bounty. There's no natural sharding
   key — quest bounties don't fit per-faction, NPC bounties don't
   fit per-quest. v1 volume is in the dozens; sharding can come
   if it ever balloons.

4. **Auto-compute reward at declaration.** Both gold and rep are
   computed from the target's statpool when the bounty is declared,
   then stored on the row. Issuers can override either via
   `DeclareOpts`. Default formulas:
   - `gold = floor(target_statpool × BountyGoldDefaultMultiplier)`,
     min 50. Multiplier defaults to **0.5**.
   - `rep = max(1, floor(target_statpool / 100))`.
   Compute-at-declare (not compute-at-claim) means the issuer
   "knows" what they're paying when they post the contract, and
   the reward doesn't drift if the target is buffed/debuffed
   between declare and claim.

5. **Auto-claim via MobDeath listener.** When a mob target dies
   and a player is the highest damager, all open bounties on that
   target are claimed by the killer. Gold is transferred via
   `user.Character.Gold`; faction-issued rep is bumped via
   `factions.BumpRep`. Companion damage already rolls up to the
   owning player via `combat.go`'s `GetCharmedUserId` path, so
   companion-heavy builds get the bounty credit they earned with
   no extra logic in the bounty hook. Player-target bounties
   (NPC kills wanted player) are deferred to 5.2 for the auto-fire
   wiring; the `TryClaim` API supports them.

We also commit to **withdraw** (issuer cancels), **expiry** (admin
sweep `PruneExpired` flips past-expiry rows), and **transparent
visibility** (wanted players see their own bounties — narratively
interesting; future v2 `Visible bool` if private contracts matter).

## Architecture & storage

`internal/bounties/` package, parallel to `internal/factions/`,
`internal/crimes/`, `internal/knowledge/`. Reuses the same
patterns: lazy-load on first access, in-memory cache, mutex-guarded
sync-save on write. Marshals YAML under the cache RLock to dodge
the race chunk 1.4's review surfaced.

**On disk:** `_datafiles/world/dogmud/bounties.yaml`, gitignored.
Single registry, not sharded.

**Subject for target:** imports `internal/knowledge.Subject`. v1
chunk dependency on chunk 1.4 is just a struct import; if a future
substrate needs `Subject` independently of knowledge, refactor to
`internal/entities/` then.

## Schema

**File shape (`bounties.yaml`):**
```yaml
next_id: 7
bounties:
  - id: 1
    issuer:
      type: faction          # "faction" | "quest" | "npc"
      id: thornwall_guards   # slug for faction; stringified int for quest/npc
    target:
      type: player           # "player" | "mob"   (knowledge.SubjectType)
      id: 17
    gold_reward: 300         # locked in at declare time
    rep_reward: 6            # locked in at declare time; only matters for faction issuer
    condition: kill          # v1 only kind; future: capture, deliver_item
    declared_round: 2065600
    expiry_round: 2073400    # 0 = never expires
    status: open             # "open" | "claimed" | "expired" | "withdrawn"
    claimed_by:              # only set when status=claimed
      type: player
      id: 0
    claimed_round: 0
    declared_reason: "Murder of Tavern Keeper Marek"  # optional
```

**Go types (sketch):**
```go
type IssuerType string
const (
    IssuerFaction IssuerType = "faction"
    IssuerQuest   IssuerType = "quest"
    IssuerNPC     IssuerType = "npc"
)

type Issuer struct {
    Type IssuerType `yaml:"type"`
    Id   string     `yaml:"id"`  // slug for faction; stringified int for quest/npc
}

func FactionIssuer(slug string) Issuer
func QuestIssuer(questId int) Issuer
func NPCIssuer(mobId int) Issuer

type Condition string
const (
    ConditionKill Condition = "kill"
)

type Status string
const (
    StatusOpen      Status = "open"
    StatusClaimed   Status = "claimed"
    StatusExpired   Status = "expired"
    StatusWithdrawn Status = "withdrawn"
)

type Bounty struct {
    Id             int               `yaml:"id"`
    Issuer         Issuer            `yaml:"issuer"`
    Target         knowledge.Subject `yaml:"target"`
    GoldReward     int               `yaml:"gold_reward"`
    RepReward      int               `yaml:"rep_reward"`
    Condition      Condition         `yaml:"condition"`
    DeclaredRound  uint64            `yaml:"declared_round"`
    ExpiryRound    uint64            `yaml:"expiry_round"`
    Status         Status            `yaml:"status"`
    ClaimedBy      knowledge.Subject `yaml:"claimed_by,omitempty"`
    ClaimedRound   uint64            `yaml:"claimed_round,omitempty"`
    DeclaredReason string            `yaml:"declared_reason,omitempty"`
}

type Registry struct {
    NextId   int       `yaml:"next_id"`
    Bounties []*Bounty `yaml:"bounties"`
}
```

## Configuration knob

`Balance.BountyGoldDefaultMultiplier` (default **0.5**) — gold
auto-compute multiplier. Sized so a baseline 600-statpool mob
yields a 300-gold bounty (modest but noticeable).

`Balance.BountyGoldFloor` (default **50**) — minimum gold reward,
so trivial mobs still pay a meaningful amount.

No knob for rep — formula is fixed at `max(1, statpool/100)`.
Tweakable in code if it ever matters.

## Public API

**Issuer/target construction helpers:**
```go
func FactionIssuer(slug string) Issuer
func QuestIssuer(questId int) Issuer
func NPCIssuer(mobId int) Issuer
// Target uses knowledge.PlayerSubject and knowledge.MobSubject
```

**Writes (all sync-persist):**
```go
type DeclareOpts struct {
    GoldOverride   int     // 0 = use default
    RepOverride    int     // 0 = use default
    DeclaredReason string  // optional flavor text
}

func Declare(issuer Issuer, target knowledge.Subject,
    condition Condition, expiryRound uint64, opts DeclareOpts) (int, error)

func Withdraw(bountyId int)              // open → withdrawn; idempotent
func TryClaim(bountyId int, claimer knowledge.Subject) (*Bounty, bool)
                                         // open → claimed; returns (bounty, true) on success
                                         // returns (nil, false) when already non-open
func MarkExpired(bountyId int)           // single row; admin
func PruneExpired() int                  // sweep; returns count
```

**Reads:**
```go
func Get(bountyId int) *Bounty
func AllOpen() []*Bounty
func OpenForTarget(target knowledge.Subject) []*Bounty
func OpenForIssuer(issuer Issuer) []*Bounty
func OpenAgainstPlayer(userId int) []*Bounty                 // convenience
func AllForTarget(target knowledge.Subject, includeNonOpen bool) []*Bounty
```

## Auto-claim hook

`internal/hooks/MobDeath_BountyClaim.go`, registered alongside
`MobDeath_FactionRep` and `MobRoomChange_KnowledgeObservers`.

**Flow:**
1. Cast event to `MobDeath`. Skip if `len(evt.PlayerDamage) == 0`.
2. Lookup `spec := mobs.GetMobSpec(MobId(evt.MobId))`. Skip if nil.
3. `target := knowledge.MobSubject(int(spec.MobId))`.
4. `open := bounties.OpenForTarget(target)`. Skip if empty.
5. Determine the killer: highest entry in `evt.PlayerDamage`. Same
   convention combat code uses for "who got the kill" elsewhere.
   Companion damage is already rolled up to owners by
   `combat.go:221`'s `GetCharmedUserId` path.
6. For each open bounty: `b, ok := bounties.TryClaim(b.Id, knowledge.PlayerSubject(killerUserId))`.
   On `ok=true`:
   - `user.Character.Gold += b.GoldReward`
   - When `b.Issuer.Type == IssuerFaction`: `factions.BumpRep(b.Issuer.Id, killerUserId, b.RepReward)`
   - Send a system message to the killer:
     `"You collect a bounty: <gold>g."` (raw gold OK per
     existing combat-spoils convention; rep gain implied via the
     existing faction-rep flavor messaging).

**Why split TryClaim from payout:** keeps the bounties package
narrow (no `users` or `factions` import). Future consumers (5.2 NPC
bounty hunters) call TryClaim and hand the gold to the mob's
inventory instead. Tests can assert claim-state without mocking
gold transfer.

**Edge cases:**
- Multi-bounty target: all open bounties on the dead target are
  claimed by the same killer; each pays out independently.
  Realistic — a wanted criminal can have parallel contracts from
  multiple factions.
- Race: the same instance can only be killed once; whoever
  resolved the death wins.
- Friendly fire: kill attribution is purely PlayerDamage-based.
  Whether it WAS a crime is the 1.3 layer's concern, separate.
- Withdraw/expired hits TryClaim: returns `(nil, false)` — hook
  skips payout. Idempotent.

## Quest engine action

New `declare_bounty` action, parallel to chunk 1.2's `bump_rep`.

```yaml
actions:
  - declare_bounty:
      issuer: { type: faction, id: thornwall_guards }
      # OR: issuer: { type: quest, id: <self> }   ← auto-fills with current questid
      target_player: true        # auto-fills with quest holder's userId
      # OR: target: { type: mob, id: 101 }
      condition: kill
      expiry_rounds: 50000
      gold_override: 0           # optional; 0 = use default
      rep_override: 0            # optional
      reason: "Murder of Marek"
```

Engine resolves `target_player: true` by reading the questing
player's userId. Resolves `issuer: { type: quest, id: <self> }`
to the current quest id.

## Admin command

`internal/usercommands/admin.bounty.go`, registered with admin role.

```
bounty list [--all]                     — open by default; --all includes claimed/expired/withdrawn
bounty show <bountyId>                  — full row detail
bounty declare <issuer> <target> [opts] — manual declaration
bounty withdraw <bountyId>
bounty prune-expired
```

**Issuer/target spec format** (dodges the multi-word parser issue
logged in MEMORY's `unified_parser_helper`):
- `<type>:<id>` form, e.g. `faction:thornwall_guards`,
  `mob:101`, `player:17`, `quest:14`, `npc:357`.

**Example:** `bounty declare faction:thornwall_guards player:17 --gold 800 --reason "Murder of Marek"`

**Help template:** `_datafiles/world/dogmud/templates/admincommands/help/command.bounty.template`. Mirror the chunk 1.4 admin command's template format.

**Output formatting:**
- `bounty list` columns: ID, Issuer, Target, Gold, Rep, Reason,
  Expiry. Sort by gold reward descending.
- `bounty show <id>` adds: declared_round, status, claimed_by,
  claimed_round.
- ANSI status colors: open=yellow, claimed=green, expired=gray,
  withdrawn=red.

## Player command

`internal/usercommands/bounty.go`, no role restriction.

```
bounty list [filter]   — all open by default; filter = "mob" | "player" | <issuer-slug>
bounty show <bountyId> — same shape as admin show
```

No declare/withdraw/prune from the player side. Wanted players
CAN see bounties on themselves (transparent v1).

**Help template:** mirror existing user-command helpfile path
during implementation.

## Physical bounty boards

Two reference implementations to validate the content-author path
and provide an in-fiction affordance for smoke testing:

1. **Thornwall: room 473 (Guard Barracks)**
2. **Stillwater: room 4110 (Constabulary)**

Both rooms get a `bounty board` noun and a `room_interact` trigger
that, when the player runs `look bounty board` or `read bounty
board`, displays a flavored intro plus the same `bounty list`
output the user command produces. Implementation reuses the same
`bounties.AllOpen` API call.

Pseudo-shape for the room YAML addition:
```yaml
nouns:
  bounty board:
    look: |
      A weather-worn corkboard stands beside the door, papered
      with wanted notices and contract slips. Recent postings:
    interact:
      action: "show_bounties"   # new room-interact handler that
                                # proxies to bounties.AllOpen
```

**Implementation note:** the room_interact handler is a new tiny
thing in `internal/hooks/` (or wherever room-interact handlers
live) — `~10 lines`. Bounty boards in other zones can be added
later as a content pass.

## Decay / expiry

| Field            | v1 policy                                                         |
|------------------|-------------------------------------------------------------------|
| Status: open     | Default at declare time.                                          |
| Status: claimed  | Set by `TryClaim` on auto-fire or explicit consumer.              |
| Status: expired  | Set by `MarkExpired` (manual) or `PruneExpired` (admin sweep).    |
| Status: withdrawn| Set by `Withdraw` (issuer cancellation).                          |
| `expiry_round=0` | Means never expires.                                              |
| Pruning          | Admin-fired via `bounty prune-expired`. No automatic background sweep v1. |

Non-open bounties remain in the registry for audit (visible via
`bounty list --all`). No automatic deletion. If volume becomes a
concern, add a TTL-on-non-open later.

## Substrate intersections

| Intersection                          | v1 policy                                              |
|---------------------------------------|--------------------------------------------------------|
| Bounties ↔ 1.2 factions               | Auto-claim hook calls `factions.BumpRep` for faction-issued bounties. |
| Bounties ↔ 1.3 crimes                 | Independent v1. 5.1 town justice will issue bounties from crime data; that wiring lives at 5.1. |
| Bounties ↔ 1.4 knowledge              | Bounties imports `knowledge.Subject` for target type. No write-side coupling v1; 5.2 hunters will read knowledge to find targets. |
| Bounties ↔ companions                 | `evt.PlayerDamage` already includes companion damage rolled up to owner via `combat.go:221`. Hook uses it as-is. |

The general rule continues from 1.4: **no retroactive feedback
loops between substrates.** Each event hook applies its effects at
the moment of the event.

## Testing

**Unit tests** in `internal/bounties/`:
- Persistence round-trip (write → evict cache → reload → fields match).
- Lazy-load + double-check-lock concurrent test (50 goroutines, same shape as chunk 1.4 T4).
- Marshal-under-RLock pattern (the lesson from chunk 1.4 review).
- `Declare` with default gold/rep — verify the multiplier math
  (target statpool × 0.5 → gold; statpool/100 → rep, floor 1).
- `Declare` with override gold and override rep.
- `Declare` with all three issuer types and both target types.
- `TryClaim` happy path; `TryClaim` idempotence (already-claimed = no-op returns false).
- `Withdraw` (open → withdrawn).
- `PruneExpired`.
- Read API: `AllOpen`, `OpenForTarget`, `OpenForIssuer`, `OpenAgainstPlayer`.

**Hook integration tests** in `internal/hooks/`:
- `MobDeath_BountyClaim` — single bounty, single damager, claim recorded + gold transferred + faction rep bumped + system message sent.
- Multi-bounty same target → all claim, killer's aggregate gold = sum of rewards.
- No open bounties → no-op.
- Player-target bounties exist but mob died → no-op (only target=mob bounties resolve via this hook v1).
- Companion-only damage path → owner credited (validates the existing rollup in `combat.go:221`).

**Quest engine action test:**
- `declare_bounty` action fires when invoked, creates a row with the correct issuer.
- `target_player: true` resolves to the quest holder's userId.
- Override fields plumb through.

**Smoke test goal file** (`tools/testing/goals/bounty-thornwall-smoke.yaml`):
1. Admin declares a faction bounty against a mob template (e.g.,
   street_performer 101) with default gold and a clear reason.
2. Player runs `bounty list`. The new bounty appears.
3. Player travels to Thornwall Guard Barracks (room 473) and runs
   `look bounty board`. Same bounty shows in the board's output.
4. Player kills the target mob (a different instance of mob 101
   somewhere in the city).
5. Player checks gold (should have increased by the gold reward)
   and rep (should have bumped with thornwall_guards).
6. Player runs `bounty list` again — bounty gone (status=claimed).
7. Admin runs `bounty list --all` and `bounty show <id>` — confirms
   status=claimed, claimed_by=player:smoketester, claimed_round set.
8. Repeat the board check at Stillwater Constabulary (room 4110)
   to confirm the second board renders correctly.

## Performance

- Single registry file, expected v1 volume in the dozens.
- Read API walks the full slice — O(n) per query, acceptable.
- Sync-save per write — same convention as chunks 1.3 / 1.4.
- If volume balloons (5.x consumers pump out bounties), index by
  target — but YAGNI v1.

## Out of scope (v1)

| Item                                  | Why deferred                                          |
|---------------------------------------|-------------------------------------------------------|
| Player-funded bounties                | Issuer enum doesn't include player v1; future v2.     |
| Capture / deliver-item conditions     | Substrate stores Condition enum but only `kill` works in v1. |
| Bounty hunter behavior                | That's chunk 5.2.                                     |
| Town-justice escalation (crime → bounty) | Chunk 5.1 wires the crime substrate into bounty issuance. |
| Bounty refunds                        | Withdraw doesn't refund the issuer — issuer was always going to pay. Refund only matters if/when player-funded bounties land. |
| Private / hidden bounties             | All bounties are visible v1. Future `Visible bool` field if needed. |
| NPC dialogue declaration mechanism    | API exists for NPC issuance but no dialogue/btree action surface yet. Comes when content needs it. |
| Background expiry sweep               | Manual `bounty prune-expired` only v1.                |
| Reward splits across multi-damagers   | Single-killer-takes-all v1. Splits can come if multi-damager bounty fights become a real pattern. |
| Item rewards                          | Gold + rep only v1.                                   |
| Bounty boards in zones beyond Thornwall + Stillwater | Content pass.                          |

## Open questions / deferred decisions

- **Player-death event seam for player-target bounty auto-fire.**
  Player-target bounties exist v1 but don't auto-fire on
  player-death because we don't ship a clean event listener for
  it yet. 5.2 NPC bounty hunters will explicitly call `TryClaim`
  when they kill a wanted player.
- **Bounty board rendering in non-reference zones.** Two
  reference implementations (Thornwall, Stillwater). Other
  zones get boards as content authors add them.
- **Reward shape extension.** Items, quest-completion, etc., are
  out of scope. When a consumer needs them, extend the schema
  with `omitempty` fields rather than retrofitting.

## Migration

None. New package, new files. Existing systems are untouched
except for the two room edits (Thornwall 473, Stillwater 4110) and
the new event listener registration in `internal/hooks/hooks.go`.
