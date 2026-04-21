# Combat Quadrant Unification — Design Spec

**Date:** 2026-04-18
**Status:** Draft / Awaiting approval
**Related memory:** `project_pvm_mvp_parity_gaps.md`

## Background

DOGMud's combat round resolution lives in four parallel handlers in
`internal/hooks/NewRound_DoCombat_helpers.go`:

| Quadrant | Function | LOC |
|----------|----------|-----|
| PvP | `handlePlayerVsPlayer` | 152 |
| PvM | `handlePlayerVsMob` | 38 |
| MvP | `handleMobVsPlayer` | 86 |
| MvM | `handleMobVsMob` | 184 |

The Actor interface (`internal/actions/actor.go`) already exists with 14
methods and two implementors (`UserActor`, `MobActor`). Stage 1.2a
extracted some sub-phase helpers (`applyCombatDamageBonuses_PvM/_MvP`,
`handlePvMCritAndMessaging`, `handlePvMProgressionAndAggro`, etc.) but
each quadrant still owns its own inline copy of most of the loop.

Every combat refactor since the Actor work (early April) has surfaced a
new parity gap. Two were already closed inline (PvM return-damage
broadcast exclusion in `f3dd0fbf`; PvM `OnCritReceived` in `277ceb06`).
A scoping pass on 2026-04-18 found three more open gaps in MvM plus a
legacy MvP double-dip, and clarified the overall divergence shape.

## Goals

1. Close the four open parity gaps (three MvM + one MvP legacy
   double-dip) as small, separately-reviewable bug fixes with tests
   (Stage 1).
2. Collapse the four quadrant handlers into a single
   `handleCombatRound(attacker, defender Actor)` driven by the Actor
   interface (Stage 2). Behavior is unchanged from the post-Stage-1
   state — the second commit is a pure code reorganization.
3. Make future parity gaps **structurally impossible**: any new combat
   logic added inside the unified handler applies to all four quadrants
   by default. Quadrant-specific logic must be explicitly gated and
   commented with the reason.

## Non-Goals

- Changing combat math, balance, damage, or roll mechanics.
- Touching the Actor interface itself (no new methods this pass).
- Unifying spell resolution, dialogue, or behavior-tree dispatch.
- Removing intentional player-only mechanics (text dispatch, party
  assist, hostility groups, shield ConditionShield, behavior triggers).
  These are correctly asymmetric and must remain so.
- Performance work. We accept any modest CPU cost from interface
  dispatch in exchange for the parity guarantee.

## Stage 1 — Pre-fix Parity Gaps

Four small, independent fixes plus tests. Land as one commit.

### Gap 1: `OnCritReceived` not called for MvM defender

**Current:** `handleMobVsMob` (NewRound_DoCombat_helpers.go:1196–1379)
never calls `defMob.Character.OnCritReceived(...)` on a critical hit.
PvM (`277ceb06`) and MvP both do.

**Fix:** Inside `handleMobVsMob`, after the attack resolves and before
the round-resolution call, add:

```go
if roundResult.Hit && roundResult.Crit {
    defMob.Character.OnCritReceived("physical", 0)
}
```

(userId=0 because mobs have no user.)

### Gap 2: Attacker crit callbacks missing in MvM

**Current:** MvM attacker only calls `OnSkillUse` on a crit hit/fumble
(lines 1357–1360). PvM/MvP/PvP all also call `OnCriticalSuccess` /
`OnCriticalFailure` so character-level crit-streak and species
callbacks fire.

**Fix:** When the MvM attacker's roll is a crit, mirror MvP's pattern
(NewRound_DoCombat_helpers.go:287–295):

```go
if roundResult.Crit {
    if roundResult.Hit {
        mob.Character.OnCriticalSuccess("weapon-combat")
    } else {
        mob.Character.OnCriticalFailure("weapon-combat")
    }
}
```

Skill name should be derived the same way MvP derives it (whatever
weapon/unarmed split MvP uses).

### Gap 3: Stat-gain room messages missing in MvM attacker

**Current:** MvP attacker (mob) emits a room message via
`characters.MobStatGainMessages` when `OnStatUse` returns true (gained).
MvM attacker calls `OnStatUse` (lines 1355–1356) but discards the
return value; no room message.

**Fix:** Mirror MvP's pattern. When `mob.Character.OnStatUse(stat, 0)`
returns true, emit the same room message MvP does. Apply for both
`strength` and `dexterity`.

### Gap 4: ConditionShield double-dip (MvP-only, legacy)

**Current:** `handleMobVsPlayer` (NewRound_DoCombat_helpers.go:1171–1179)
subtracts an additional `GetConditionMagnitude(ConditionShield) / 2`
from post-attack damage when the defender is a player. The comment
claims "ConditionShield lives on player defenders," which is factually
wrong — the condition is a Character-level buff usable by any mob or
player (see `hooks_test.go:1493` where a mob has it).

More importantly, `ConditionShield` is **already counted** inside the
mitigation layer that runs as part of the attack calculation:
- `characters/combat.go:161` — added to standard armor reduction
- `characters/combat.go:200` — added to `GetPhysicalMitigation()`

So the inline MvP block is a **silent double-dip bonus for player
defenders only**. It's a legacy artifact from Stage 11.4, left behind
when ConditionShield got folded into the mitigation layer.

**Fix:** Delete the inline block at lines 1171–1179 entirely.
Mitigation already accounts for it via the standard armor path.

### Stage 1 Tests

Four new tests in `internal/hooks/` (likely a new file
`NewRound_DoCombat_parity_test.go` to keep the diff isolated):

1. **TestMvM_DefenderReceivesOnCritReceived** — set up two mobs in
   combat, force a crit hit, assert defender's `OnCritReceived`
   counter / state changed. Use whatever observation channel MvP's
   regression test uses.
2. **TestMvM_AttackerCritCallbacksFire** — set up two mobs, force a
   crit-hit and (separately) a crit-miss; assert the attacker's
   crit-success and crit-failure counters incremented.
3. **TestMvM_AttackerStatGainEmitsRoomMessage** — seed an attacker
   mob whose `OnStatUse` is guaranteed to return true (Training near
   threshold or mock); assert the room received the
   `MobStatGainMessages` text.
4. **TestMvP_ConditionShieldAppliedOnceNotDoubleDipped** — set up a
   player defender with `ConditionShield` at known magnitude; run one
   MvP round with deterministic dice; assert total damage reduction
   equals the mitigation-layer amount only (not mitigation + inline
   bonus). Baseline may require snapshotting today's (buggy) damage,
   deleting the inline block, and recomputing.

Existing tests must still pass.

### Stage 1 Commit Message

```
fix(combat): close four parity gaps before quadrant unification

Three MvM-side callback gaps + one legacy MvP double-dip:
  - MvM defender: OnCritReceived on crit hits (PvM/MvP/PvP already fire)
  - MvM attacker: OnCriticalSuccess/OnCriticalFailure on crit rolls
  - MvM attacker: stat-gain room messages on OnStatUse true
  - MvP: delete the inline ConditionShield damage reduction — it is
    already counted by the mitigation layer (characters/combat.go:161,
    200); the inline block was a Stage 11.4 leftover that silently
    double-counted the reduction for player defenders only.

These were noticed during combat-quadrant unification scoping and are
pre-fixed here so the unification commit can be a pure refactor.
```

## Stage 2 — Unification Refactor

Single new function:

```go
func handleCombatRound(
    attacker Actor,
    defender Actor,
    evt events.NewRound,
    /* ...other shared params... */
) RoundResult
```

Replaces the four `handle{P,M}vs{P,M}` functions. The two dispatchers
(`handlePlayerCombat`, `handleMobCombat`) collapse their per-quadrant
branches into a single call to `handleCombatRound(atk, def)` after
constructing the appropriate `Actor` wrappers.

### Sub-Phase Decomposition

The unified handler runs these phases in order. Each phase is its own
function. Quadrant-specific logic is gated explicitly inside the phase
(not at the top of the handler) and commented with the reason.

| Phase | Function | What it does | Currently |
|-------|----------|-------------|-----------|
| 0. Target resolve | `resolveCombatTarget(atk Actor) Actor` | Look up defender from atk.Aggro; validate alive/in-room | 4 inline copies |
| 1. Wait-round | `handleCombatWaitRound(...)` | Decrement RoundsWaiting, emit wait msgs | **Already unified** |
| 2. Roll attack | `rollCombatAttack(atk, def Actor) RoundResult` | Wrap `combat.Attack{P,M}vs{P,M}` polymorphically | 4 typed variants |
| 3. Damage bonuses | `applyCombatDamageBonuses(atk, def Actor, &res)` | Buffs (Adrenaline, ConvictionSurge), return damage, lifesteal | 2 helpers + 2 inlined |
| 4. Crit + messaging | `dispatchCritAndMessaging(atk, def Actor, &res)` | Apply crit effects; route messages per atk.IsPlayer() / def.IsPlayer() | 4 variants (3 helpers + 1 inlined) |
| 5. Progression | `applyCombatProgression(atk, def Actor, &res)` | Attacker stat/skill/crit callbacks; defender OnCritReceived; defender dodge/parry/block via `processDefenderProgression` | 4 inline copies |
| 6. Behavior trigger | `fireDefenderBehaviorTrigger(def Actor, ...)` | mob_hurt fires when defender is a mob. No-op if `def.IsPlayer()` | PvM + MvM only |
| 7. Aggro + assist | `handleAggroAndAssist(atk, def Actor, ...)` | Mob-aggro-on-attack with exit-walk (when atk is player) or direct Aggro (when atk is mob); companion-owner assist routing per quadrant | 4 inline copies, very divergent |
| 8. Round resolution | `resolveCombatRound(atk, def Actor, &res)` | Death handling, retarget messaging (player-only), close out | 4 inline copies |

### Quadrant Routing Strategy

We **do not** introduce a `Quadrant` enum. Routing is done via
`atk.IsPlayer()` and `def.IsPlayer()` checks at the leaf of each phase
where divergence is needed. Rationale:

1. The Actor interface already exposes `IsPlayer()` cheaply.
2. A `Quadrant` enum is redundant with `atk.IsPlayer() && def.IsPlayer()`
   and forces every caller to compute and pass it.
3. `IsPlayer()` checks make the divergence visible at exactly the line
   where it matters; a quadrant tag hides it.
4. Future Actor implementors (e.g., scripted entities) get correct
   default behavior without needing a new quadrant value.

Where a phase needs the **concrete** type (e.g., shield reduction needs
`*characters.Character.GetCondition()`, party assist needs
`user.PartyId`), use:

```go
if def.IsPlayer() {
    defChar := def.GetCharacter()
    // ...player-only path...
}
```

Type assertions on Actor (`if u, ok := def.(*UserActor); ok`) are also
allowed when the player-only path needs `*users.UserRecord` directly
(e.g., for `user.PartyId` lookups). Both patterns are acceptable; pick
whichever reads more cleanly per call site.

### Intentional Divergences (Must Preserve)

The unified handler must keep these asymmetric. Each gets a comment
explaining why.

1. **Crit message routing**:
   - Attacker text: only if `atk.IsPlayer()` (mobs have no connection)
   - Defender text: only if `def.IsPlayer()`
   - Room broadcast: always, with text-receiving combatants excluded
2. **`mob_hurt` behavior tree** fires only when `def.IsPlayer() == false`
3. **Hostility groups** (`mobs.MakeHostile`) only when atk is player and
   def is mob. Charisma-scaled duration is player-side only.
4. **Mob-aggro-on-attack with exit-walk** only when atk is player.
   When atk is mob, set Aggro directly (already in same room or pursuit
   handled elsewhere).
5. **Party auto-assist** fires for whichever defender is in a party.
   Players can form parties today, mobs cannot — so this naturally
   no-ops for mob defenders. When mob parties become a real mechanic,
   the phase helper routes through the same code path without changes.
   Implementation: inside the phase, check `def.GetCharacter().PartyId`
   (or the equivalent party-membership lookup) rather than gating on
   `def.IsPlayer()`.
6. **TrackPlayerDamage** only when both atk and def are players.
7. **`Aggro.Type == Flee`** is player-only; never check it on a mob
   attacker.
8. **Stat-gain room messages**: only emit when the entity actually
   gained (return value of `OnStatUse`). MvP/MvM use mob-flavor text
   via `MobStatGainMessages`; PvM/PvP use player-flavor text. The
   helper picks based on `actor.IsPlayer()`.
9. **Retarget-on-death "You turn your attention to..." message**
   only when the survivor is a player.
10. ~~**Shield reduction** (`ConditionShield`) only when defender is a
    player.~~ Removed in Stage 1 Gap 4 — this was a legacy double-dip.
    ConditionShield is correctly applied to all characters (player or
    mob) via the mitigation layer.

### Files Touched (Stage 2)

- `internal/hooks/NewRound_DoCombat_helpers.go` — the four
  `handle*Vs*` functions are removed; the new
  `handleCombatRound` and its phase helpers replace them. Net: file
  may shrink moderately or stay similar size; structure flattens.
- `internal/hooks/NewRound_DoCombat.go` — the two dispatchers
  (`handlePlayerCombat`, `handleMobCombat`) collapse their per-quadrant
  branches.
- `internal/hooks/NewRound_DoCombat_resolution.go` — may absorb some
  newly-extracted phase helpers.
- Possible new file: `internal/hooks/NewRound_DoCombat_unified.go`
  if the helpers file gets unwieldy.

No changes to:
- `internal/actions/actor.go` — Actor interface stays put.
- `internal/combat/` — math layer untouched.
- `internal/characters/` or `internal/mobs/` — no method additions.
- Spell, dialogue, behavior-tree code — out of scope.

### Stage 2 Tests

Strategy: leverage Stage 1's new MvM tests + existing combat tests
(`internal/combat/regression_test.go`, `integration_combat_test.go`,
`internal/hooks/predator_hooks_test.go`,
`internal/characters/godfunc_refactor_test.go`) as the parity baseline.
**No combat behavior changes**, so all of them must pass unchanged.

Add **one new structural test**:
`TestHandleCombatRound_AllQuadrantsRouteCorrectly` — feed each of the
four quadrant pairs (UU, UM, MU, MM) into `handleCombatRound` with
deterministic dice and assert: (a) right callbacks fire on the right
side, (b) right text recipients, (c) right behavior triggers.

Optional: a property-style test that asserts `for each quadrant, the
sequence of phases invoked is identical except for explicit
quadrant-gated branches`.

### Stage 2 Commit Strategy

Two commits:

1. `refactor(combat): introduce handleCombatRound + phase helpers`
   — adds the new function and phase helpers alongside the existing
   four. Dispatchers don't switch yet. Net diff is ADD-ONLY. All
   tests still pass.

2. `refactor(combat): switch dispatchers to handleCombatRound; remove
   quadrant handlers` — flips the two dispatchers, deletes the four
   `handle*Vs*` functions and their per-quadrant helpers. All tests
   still pass.

This staging is bisect-friendly: if a regression appears, we know
which commit introduced it.

## Risk Mitigations

The scoping pass flagged 11 risks. Mitigations:

| # | Risk | Mitigation |
|---|------|-----------|
| 1 | `if atk.IsPlayer()` branches balloon to 30+ sites | Group divergent logic into named sub-phase helpers; each phase has at most 2-3 branches. If a phase grows past 3, split it. |
| 2 | `Aggro.Type == Flee` accessed for mobs | Wrap with `if atk.IsPlayer()` everywhere; add unit test that a mob attacker never enters Flee branches. |
| 3 | Aggro struct semantic divergence | No change — both still use `Character.Aggro`. Comments + tests document mob = always DefaultAttack. |
| 4 | Message routing must preserve quadrant flavor | Stage 1 test suite + a new Stage 2 routing test (above) lock the routing rules. |
| 5 | OnSkillUse userId guard | Don't change call signatures; both already pass through `actor.OnSkillUse(skill)` which forwards correct userId. Safe. |
| 6 | Companion assist routing differs per quadrant | `handleAggroAndAssist` has explicit `if atk.IsPlayer() / def.IsPlayer()` branches; comments name each case. |
| 7 | MvM stat-gain msgs spam if room display logic merges | Phase 6 helper picks player vs mob flavor text by `actor.IsPlayer()`; no change to suppression rules. |
| 8 | Hostility group formula PvM-only | Phase 8 explicit `if atk.IsPlayer() && !def.IsPlayer()` guard. |
| 9 | Behavior tree double-emit (combat_start fires in two places) | Out of scope for this refactor. Document in spec; revisit only if test suite reveals an issue. |
| 10 | `combat.RecordAttack` parameter divergence | Phase 2 (rollCombatAttack) wraps it; single call site after refactor. |
| 11 | ~~Shield reduction (MvP-only)~~ | Removed in Stage 1 Gap 4. Phase 4 (`applyDefenderShield`) is no longer needed — delete from the phase list. |

## Success Criteria

- All Stage 1 fixes land with passing new tests; existing tests
  unchanged.
- After Stage 2, only one combat-round function exists
  (`handleCombatRound`); the four quadrant handlers are gone.
- Full test suite (`go test ./...`) passes.
- `go build ./...` and `go vet ./...` clean.
- Manual smoke: PvM, MvP, PvP, MvM combat all produce correct text,
  callbacks, behavior triggers (4 short scenarios).
- `project_pvm_mvp_parity_gaps.md` Open Gaps section is empty (it
  already is) and Resolved section gets the 4 Stage 1 fixes appended.
- A new feedback memory captures the design rule: "future combat
  logic goes inside `handleCombatRound` or its phase helpers; quadrant
  divergence requires `IsPlayer()` gating + reason comment."

## Out of Scope (Tracked Elsewhere)

- Behavior tree double-emit of `combat_start` (risk #9). Revisit if
  observed.
- The companion-assist routing tangle (risk #6). The unified handler
  preserves current semantics; a future pass could simplify but it's
  not parity-driven.
- Adding new methods to the Actor interface. If a phase needs
  something not on Actor today, prefer `actor.GetCharacter()` access
  or type assertion.
- Changing the dispatchers in `handlePlayerCombat` /
  `handleMobCombat` beyond the per-quadrant collapse (e.g., merging
  them into a single dispatcher loop). That's a follow-up refactor.

## Decision Log

- **2026-04-18**: Shape selected = pre-fix gaps then refactor (vs.
  one-pass unification or pair-wise incremental). Rationale: clean
  bisect, behavior-change tests lock semantics before code moves.
- **2026-04-18**: Routing strategy = `IsPlayer()` checks, no Quadrant
  enum. Rationale: redundant with Actor methods; makes divergence
  visible at the call site.
- **2026-04-18**: Stage 2 splits into add-then-switch commits.
  Rationale: bisect-friendly; reviewer can verify add-only commit is
  pure addition.
