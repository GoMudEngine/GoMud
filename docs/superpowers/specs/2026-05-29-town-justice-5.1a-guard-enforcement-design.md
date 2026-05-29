# Town Justice 5.1a — Wanted Status + Guard Warn→Attack + Citizenship

**Date:** 2026-05-29
**Roadmap:** Phase 5.1 (Town Justice), first sub-project of four.
**Size:** M • **Depends on:** factions (1.2/1.5), crimes (1.3), bounties (1.5),
patrols (3.4), Phase 4 guard goal types.

---

## Overview

The justice substrate already exists: per-player faction rep with tiers
(`factions`), crime logs with witnesses and resolution (`crimes`), and a
bounty registry (`bounties`). What's missing is the **enforcement reaction** —
today `factions.IsPeacefulToward` only makes a mob *skip* friendly players; no
mob acts on negative standing, so a criminal player walks past the city watch
untouched.

5.1a adds that reaction: a **shared "wanted" verdict** computed from the
existing substrate, and a **reactive guard enforcement path** that escalates
**warn → attack** against wanted players (with an `Arrest` rung reserved for
5.1c). It also does the concrete citizenship cleanup: migrating the two
`peacefulquest` mobs to faction-rep gating.

This is the first of four 5.1 slices. Later: crime→auto-bounty (5.1b), arrest
(5.1c), redemption (5.1d).

---

## Locked Decisions

| Decision | Value |
|----------|-------|
| Guard mechanism | Reactive per-tick enforcement (sibling to `lookfortrouble`), not a Phase-4 goal |
| Wanted signal | Faction rep tier **+** open faction/ally bounty **+** unresolved recent crime against faction/allies |
| Escalation | Tier sets starting rung; warned player still present past grace → escalate to attack |
| Crime-in-progress | In scope, handled reactively-via-tick (≤1 round), no coupling to combat/steal code |
| Severity type | **Ordered enum** with a reserved `Arrest` rung between Warn and Attack (forward-proofs 5.1c) |
| Reuse seam | Wanted decision + enforcement action live in **`internal/justice`** so a future goal/btree can call the same helper (forward-proofs goal integration + 5.2 hunters) |

---

## Components

### `internal/justice` (new package)

The shared justice substrate. Imports `factions`, `crimes`, `bounties`,
`mobs`, `rooms`, `configs` (none import `justice`, so no cycle). Mirrors the
Phase-1 package conventions (lazy reads, test seams).

**Severity (ordered enum):**
```go
type Severity int
const (
    SeverityNone   Severity = iota // citizen / not wanted — leave alone
    SeverityWarn                   // wanted, mild — verbal warning
    SeverityArrest                 // RESERVED for 5.1c; not produced by 5.1a Verdict yet
    SeverityAttack                 // wanted, severe — engage
)
```
Ordered so 5.1c inserts `Arrest` handling without reshaping call sites; "take
the most severe" comparisons use `>`.

**Decision (pure, reusable by 5.2 hunters):**
```go
// Verdict returns how a guard belonging to guardFactions should treat a
// player. Expands guardFactions with their declared allies, then takes the
// most severe of: bounty, rep-tier, and unresolved-crime signals.
func Verdict(guardFactions []string, userId int) Severity
```
Signal evaluation (max severity wins):
- **Bounty** → `SeverityAttack` if any open bounty targets the player issued by
  a faction in the guard+ally set. Uses a parameterizable issuer filter
  (`openBountyFrom(userId, issuerFactions)`) so 5.2 hunters can ask "any
  issuer" instead of the guard-scoped set.
- **Rep tier** → for each faction in the guard+ally set, `factions.TierFor`:
  `TierHostile` → `SeverityAttack`; `TierCold` → `SeverityWarn`. (Configurable;
  see Config.)
- **Crime** → `SeverityAttack` if the player has an unresolved crime against a
  faction in the set within `JusticeCrimeLookbackRounds`
  (`crimes.AllForFaction(f, false)` filtered to `perp == userId` + recent).

**Enforcement action (reusable by the mobcommand now, a btree action later):**
```go
// RunGuardEnforcement scans players in the room and applies warn/attack
// against wanted players for this guard, managing warn-grace memory. Both
// the enforcelaw mob command (now) and a future protection-faction btree
// action (later) call this — the strategic goal expresses "I protect
// Thornwall"; this is how it manifests tick-to-tick.
func RunGuardEnforcement(mob *mobs.Mob, room *rooms.Room)
```
Per visible, non-grace-protected, non-hidden player:
1. `sev := Verdict(factions.FactionsForMob(mob), userId)`.
2. `SeverityAttack` → `mob.Command("attack @<userId>")` (same mechanism
   `LookForTrouble` uses).
3. `SeverityWarn`:
   - Not yet warned (no memory, or warn expired): the guard says a warning and
     stamps `justice_warned_<userId>` → current round in `mob.Character.MiscData`.
   - Already warned and `now - warnedRound >= GuardWarnGraceRounds` (player
     stayed / returned still-wanted): escalate to `mob.Command("attack @...")`.
4. `SeverityNone`: ignore (citizen / neutral).

Skips charmed guards, `NoAggroTarget`-grace players, hidden players, and
already-in-combat guards — same guards as `LookForTrouble`.

### `enforcelaw` mob command (`internal/mobcommands/enforcelaw.go`)

Thin wrapper: `return justice.RunGuardEnforcement(mob, room)` semantics
(handled/err signature like other mob commands). Registered in the mob command
registry.

### `guard` behavior archetype (`_datafiles/world/dogmud/behaviors/archetypes/guard.yaml`)

Wires `enforcelaw` into the guard's idle command pool so it runs each idle tick
(the same idle-invocation mechanism aggressive mobs use to run
`lookfortrouble` — exact hook confirmed at plan time). Applied to existing
guard mobs: city_guard (106), city_gate_guard (92), guard_captain_velk (94),
constable_drunn (335). Guards remain `IsPeacefulToward`-friendly to citizens
(unchanged); enforcement is purely additive.

### Citizenship + `peacefulquest` migration

"Citizen" is operational, not a new data structure: a player at `TierNeutral`
or better with the guard faction yields `SeverityNone` (left alone). The
concrete cleanup:
- The warren scout (72) and warrior (73) mobs currently use the `peacefulquest`
  token (checked in `lookfortrouble.go`) to skip quest-holders. Migrate: the
  quest that granted the token instead **bumps warren faction rep** to
  `TierWarm` on completion, so `IsPeacefulToward` grants peace through the
  faction system.
- Remove the `peacefulquest` mob field, its `lookfortrouble.go` check, and the
  token grant; verify warren mobs carry the `warren` group so the rep gate
  applies.

---

## Data Flow

### Wanted player walks into a guarded room
Player enters → guard idle tick → `RunGuardEnforcement` → `Verdict` (rep
Hostile / open faction bounty → Attack; rep Cold → Warn) → warn or attack.

### Crime in progress (reactive intervention, ≤1 round)
Player assaults/steals from a citizen → existing crime call site records the
crime synchronously (**unchanged**) → guard's next idle tick → `Verdict` sees
the unresolved crime against an allied faction → `SeverityAttack` → guard
engages. No coupling to combat/steal code; the ≤1-round latency is the
deliberate tradeoff. (A future synchronous "tackle mid-crime" can add a
crime-recorded event without changing this baseline.)

### Composition with later slices (verified forward-compat)
- **5.1b** declares a bounty on threshold crimes → `Verdict`'s bounty branch
  picks it up with no new wiring.
- **5.1d** redemption calls `crimes.Resolve` / `bounties.Withdraw` /
  `factions.BumpRep` → `Verdict` stops returning Attack. (Note for 5.1d: a paid
  fine resolves the *crime* but Hostile *rep* may linger — institutional
  memory; 5.1d decides whether a fine also nudges rep.)
- **5.2** hunters reuse `Verdict` / the parameterizable bounty query (all
  issuers) for goal-driven pursuit; guard enforcement here is the reactive
  local counterpart.

---

## Config (Balance)

- `GuardWarnGraceRounds` (default 50) — rounds a warned player may linger before
  warn escalates to attack.
- `JusticeAttackTier` (default `Hostile`) and `JusticeWarnTier` (default `Cold`)
  — rep tiers mapping to Attack / Warn. Stored as the tier name/threshold.
- `JusticeCrimeLookbackRounds` (default ~1000) — recency window for the
  unresolved-crime signal.

---

## Testing

`internal/justice` (`justice_test.go`, test seams like crimes/bounties):
- `Verdict`: open faction bounty → Attack; ally-faction bounty → Attack;
  Hostile rep → Attack; Cold rep → Warn; Neutral/Warm → None; unresolved recent
  crime against guard faction → Attack; crime against an **ally** faction →
  Attack; resolved/stale crime → ignored; most-severe-wins when signals
  combine.
- `RunGuardEnforcement`: Attack verdict issues `attack @id`; Warn verdict says a
  warning + stamps MiscData; warned-past-grace escalates to attack; citizen
  (None) ignored; hidden / grace / charmed skipped.

Migration: a test (or boot check) that warren peace flows through faction rep
after the `peacefulquest` removal.

Boot smoke (instance wipe per SOP): server boots clean; new config knobs load;
guard archetype + mobs load without panic.

---

## Out of Scope (later 5.1 slices)

- Arrest/jail mechanic (5.1c) — the `Arrest` severity rung is reserved but not
  produced/handled here.
- Crime → auto-bounty declaration (5.1b).
- Redemption / fine-paying (5.1d).
- Stillwater constabulary rollout and per-zone justice tuning.
- Synchronous (same-round) crime intervention.
