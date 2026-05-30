# Justice Package Context

## Purpose / Overview

`internal/justice` is the town-justice substrate: it decides whether a
player is "wanted" by a faction's guards (`Verdict`), runs the per-round
guard enforcement tick (`RunGuardEnforcement`), auto-posts faction
kill-bounties at crime sites (`MaybeDeclareBounty`), and executes the
full arrest/jail lifecycle (`ExecuteArrest`, `ResolveDetention`).

The verdict logic is deliberately pure and self-contained so that the
same decision function can be reused by future bounty-hunter NPCs (5.2)
without pulling guard enforcement along.

---

## Key Files

### `justice.go` — Severity enum + `Verdict`

Defines the `Severity` ordered enum:

```go
type Severity int

const (
    SeverityNone   Severity = iota // citizen / not wanted
    SeverityWarn                   // wanted, mild — verbal warning
    SeverityArrest                 // wanted, arrest-level
    SeverityAttack                 // wanted, severe — engage on sight
)
```

**Ordering matters** — callers compare with `>` to find the worst signal.
`SeverityAttack` is never produced by `Verdict`; it is only emitted by
`RunGuardEnforcement` when the resist fork fires (see enforce.go).

`Verdict(guardFactions []string, userId int) Severity` is the pure
decision function:
- Builds the full faction set (guard factions + their declared allies).
- Returns `SeverityArrest` if the player has any open faction bounty,
  any Hostile rep tier, or any unresolved crime within the lookback
  window.
- Returns `SeverityWarn` if the player has a Cold rep tier with any
  faction in the set.
- Returns `SeverityNone` otherwise.

Config knob: `JusticeCrimeLookbackRounds` (default 1000 if unset).

Read seams (`repTierFn`, `alliesFn`, `openFactionBountyFn`,
`unresolvedCrimeFn`, `nowRoundFn`, `lookbackFn`) are package-level
function vars; tests override them to avoid standing up live faction/
bounty/crimes stacks.

---

### `enforce.go` — Guard enforcement tick + speech seam

`RunGuardEnforcement(mob *mobs.Mob, room *rooms.Room, nowRound uint64) []EnforceAction`

Called once per round from `internal/hooks/NewRound_MobRoundTick.go` for
every `guard`-group mob. Skips mobs already in combat or charmed, and
players who are hidden, dead, or aggro-flagged immune.

**Three-branch dispatch on `Verdict` result:**

| Verdict | Outcome |
|---------|---------|
| `SeverityAttack` | `mob.Command("attack @uid")` immediately |
| `SeverityWarn` | On first sight: `guardSayFn` + stamp `justice_warned_<uid>` in guard MiscData. After `GuardWarnGraceRounds` elapse: escalate to attack |
| `SeverityArrest` | Fork on player's `ArrestPolicy` (see below) |

**Arrest branch** (`SeverityArrest`, from pure helper `resolveArrest`):

| Player policy | Outcome |
|---------------|---------|
| Resist | `mob.Command("attack @uid")` immediately — `SeverityAttack` action emitted |
| Surrender — first sight | `guardSayFn` warning + stamp `justice_arrest_pending_<uid>` in guard MiscData |
| Surrender — within `ArrestResistGraceRounds` | No-op (waiting) |
| Surrender — grace expired | Call `executeArrestFn` (→ `ExecuteArrest`), clear pending stamp |

The `EnforceAction` return value carries `{UserId, Severity, Escalated}`
for tests and future telemetry.

**`guardSayFn` seam**: The package-level var `guardSayFn` is a no-op by
default. `internal/hooks/justice_wiring.go` installs the real broadcaster
at `init()` via `justice.SetGuardSay(fn)`. This keeps `justice` free of
`internal/actions` imports, breaking the import cycle that would otherwise
arise because crime sites in `internal/actions` call `MaybeDeclareBounty`.

**`executeArrestFn` seam**: Wraps `ExecuteArrest`; tests override to
intercept without a live mob/world.

---

### `bounty.go` — Auto-bounty declaration

`MaybeDeclareBounty(faction string, userId int, triggerKind crimes.Kind)`

Posts a faction kill-bounty on the player when warranted. Idempotent per
(faction, player): skips if an open faction bounty already exists.

Declares on an identified murder (`crimes.KindMurder`) OR Hostile rep
tier — same two signals that produce `SeverityArrest` in `Verdict`.

**Fine/bounty gold formula** (`bountyGold`, pure):
```
gold = powerBase × max(crimeMult, repMult)
```
- `powerBase` = `bounties.DefaultGoldFor(PlayerSubject(userId))`
- `crimeMult` = `JusticeBountyMurderMult` on murder, else 1.0
- `repMult` ramps 1.0 (rep −50, Hostile floor) → `JusticeBountyRepMultMax`
  (rep −100); 1.0 if not yet Hostile

Config knobs: `JusticeBountyExpiryRounds` (default 5000),
`JusticeBountyMurderMult`, `JusticeBountyRepMultMax`.

**Where it is called:**
- `internal/hooks/MobDeath_FactionRep` (identified murder)
- `internal/actions/attack.go` (assault rep hit)
- `internal/actions/steal.go` and `internal/actions/plant.go` (theft)

All seams (`bDefaultGoldFn`, `bRepFn`, `bTierFn`, `bDeclareFn`,
`bExistingFn`, `bExpiryRoundsFn`, `bMurderMultFn`, `bRepMultMaxFn`,
`bNowFn`) are package-level function vars.

---

### `arrest.go` — Jail lifecycle

Holds the fine math, the jail record type, and the two public lifecycle
functions. Does NOT import `internal/actions` or `internal/usercommands`;
player messages go through `users.GetByUserId(uid).SendText` directly.

**MiscData jail record keys** (stamped on `characters.Character`):

| Key | Type | Meaning |
|-----|------|---------|
| `jail_until_round` | uint64 | Round when sentence expires |
| `jail_fine_original` | int | Fine at time of arrest |
| `jail_decay_per_round` | int | Gold deducted per round served |
| `jail_faction` | string | Arresting faction slug |
| `jail_cell_room` | int | Holding-cell room ID |
| `jail_crime_ids` | string | Comma-separated unresolved crime IDs |

**Holding-cell registry**: `jailCellFor` maps faction slug → cell room ID.
Currently: `"thornwall_guards"` → room 5105. `ExecuteArrest` returns
`false` (no-op) for factions without a registered cell.

**`JailRecord` struct** (exported for commands):
```go
type JailRecord struct {
    FineOriginal  int
    DecayPerRound int
    UntilRound    uint64
    Faction       string
    CellRoom      int
}
```

`JailInfo(player *characters.Character) (JailRecord, bool)` — accessor
used by the `fine` command to compute the current decaying fine.

**`computeFine`** reuses `bountyGold` (same formula as auto-bounty gold)
so the fine is proportional to the bounty the player attracted.

**`sentenceRounds(fine, decayPerRound int) int`** — `fine / decay`, min 1.

**`currentFine(original, roundsServed, decayPerRound int) int`** — pure
decay math; floors at zero.

#### `ExecuteArrest(player *characters.Character, userId int, faction string, isMurder bool) bool`

1. Looks up the cell room for `faction`; returns `false` if none.
2. Computes fine + sentence rounds.
3. Collects unresolved crime IDs for the faction via
   `crimeIdsForFactionPlayer`.
4. Stamps all six jail MiscData keys.
5. Applies buff 88 (Jailed) via `player.AddBuffScaled(88, float64(rounds))`
   — `TriggersLeft` is scaled to `rounds` so the buff expires naturally
   at sentence end.
6. Moves the player to the cell via `rooms.MoveToRoom`.
7. Sends arrest-flavor text to the player.

The Jailed buff (id 88, `no-go` / `NoMovement` flag) prevents movement
for the duration. Its `end_user_text` fires automatically when the buff
expires; `ExecuteArrest` also sends an additional arrest-context line.

#### `ResolveDetention(player *characters.Character, userId int) bool`

Ends detention on timer expiry or fine payment:
1. Guards on `jail_until_round` presence; returns `false` if not jailed.
2. Resolves each stamped crime via `crimes.Resolve(faction, id, "served
   sentence")`.
3. Withdraws all open faction bounties against the player.
4. Resets rep to `JusticeArrestRepReset` floor only if currently below it
   (default −10).
5. Removes buff 88 via `player.RemoveBuff(88)`.
6. Clears all jail MiscData keys.
7. Moves player to barracks (room 473).
8. Sends release-flavor text.

Release seams: `aResolveCrimeFn`, `aOpenBountiesFn`, `aWithdrawFn`,
`aGetRepFn`, `aSetRepFn`, `aMoveFn`, `aDecayFn`, `aRepResetFn`.

---

## Enforcement + Jail Flow (end-to-end)

```
crime site (actions/hooks)
  └─ MaybeDeclareBounty          → bounties registry
  └─ rep hit (factions.BumpRep)

guard mob's per-round tick
  └─ RunGuardEnforcement
       └─ Verdict(guardFactions, uid) → SeverityArrest
       └─ resolveArrest(resist, pending, pendingRound, now, grace)
            ├─ resist=true  → attack @uid
            ├─ !pending     → guardSayFn + stamp pending key
            ├─ within grace → no-op
            └─ grace done   → executeArrestFn
                               └─ ExecuteArrest
                                    ├─ buff 88 (Jailed, no-go)
                                    ├─ jail MiscData record
                                    └─ MoveToRoom(cell)

per-round player tick (hooks/Jail_ExpiryRelease.go)
  └─ releaseIfSentenceServed → jailExpired? → ResolveDetention
                                               ├─ crimes resolved
                                               ├─ bounty withdrawn
                                               ├─ rep floor restored
                                               ├─ buff 88 removed
                                               └─ MoveToRoom(barracks 473)

player commands
  └─ fine    → currentJailFine → JailInfo → display decaying fine
  └─ payfine → currentJailFine → deduct gold/bank → ResolveDetention
```

---

## Cross-Package Wiring

| Caller / site | What it calls |
|---------------|---------------|
| `internal/hooks/NewRound_MobRoundTick.go` | `RunGuardEnforcement` (guard-group mobs) |
| `internal/hooks/Jail_ExpiryRelease.go` | `JailInfo`, `ResolveDetention` (per-player tick) |
| `internal/hooks/PlayerDeath_BountyResolve.go` | `bounties.Withdraw` (on wanted player's death) |
| `internal/hooks/justice_wiring.go` | `justice.SetGuardSay(fn)` at init |
| `internal/hooks/MobDeath_FactionRep.go` | `MaybeDeclareBounty` (murder) |
| `internal/actions/attack.go` | `MaybeDeclareBounty` (assault) |
| `internal/actions/steal.go`, `plant.go` | `MaybeDeclareBounty` (theft) |
| `internal/usercommands/jail.go` | `JailInfo`, `ResolveDetention` (fine/payfine) |

`justice` itself imports only: `bounties`, `configs`, `crimes`,
`factions`, `knowledge`, `opinions`, `util`, `characters`, `rooms`,
`mobs`, `users`, `messaging`. It never imports `internal/actions` or
`internal/usercommands`.

---

## Test Seam Convention

All external calls (faction rep, crime lookups, bounty queries, room
moves, etc.) are wrapped in package-level `var` function pointers with
descriptive names (e.g. `repTierFn`, `executeArrestFn`, `aMoveFn`).
Tests replace these vars with stubs that return controlled values without
standing up live subsystems. Production never sets them; the zero-value
wires to the real implementation.

Test files: `justice_test.go`, `enforce_test.go`, `bounty_test.go`,
`arrest_test.go`.

---

## Config Knobs (Balance)

| Knob | Default | Purpose |
|------|---------|---------|
| `JusticeCrimeLookbackRounds` | 1000 | How far back unresolved crimes count toward Verdict |
| `GuardWarnGraceRounds` | 50 | Rounds a Warn-severity player has before escalation to attack |
| `ArrestResistGraceRounds` | 3 | Rounds a guard waits after declaring arrest before hauling |
| `JusticeFineDecayPerRound` | 5 | Gold the fine drops per served round |
| `JusticeArrestRepReset` | -10 | Rep floor restored to on release (only if below) |
| `JusticeBountyExpiryRounds` | 5000 | TTL on auto-declared faction bounties |
| `JusticeBountyMurderMult` | — | Fine/bounty multiplier for murder |
| `JusticeBountyRepMultMax` | — | Max rep-based multiplier at rep -100 |
