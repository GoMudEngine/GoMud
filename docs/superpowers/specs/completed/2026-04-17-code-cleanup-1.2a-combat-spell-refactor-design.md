# Code Cleanup 1.2a: Combat + Spell God-Function Refactor — Design Spec

## Goal

Decompose 3 oversized functions on combat's hottest paths — two per-combatant round handlers in `internal/hooks/NewRound_DoCombat_helpers.go` and one spell-effect applicator in `internal/hooks/spell_resolution.go` — into focused helpers. Paired substages 1.2b (character + admin) and 1.2c (`character.go` file split) have shipped on `development`; combat has had the intended settle window before this round of restructuring.

The three target functions sit on the combat round's per-tick path. They're the most test-sensitive code in the project, and 1.2a has been deliberately scheduled last among the 1.2 batches so this refactor lands against a code base that has absorbed the character and admin moves first.

## Scope

**In scope:**

1. **Combat batch** — refactor the two per-combatant round handlers. Helpers cluster naturally (pre-round target resolution, damage-bonus modifiers, post-swing progression), so they move to a new themed file `NewRound_DoCombat_resolution.go` next to the existing `combat_shared_helpers.go`. Source file `NewRound_DoCombat_helpers.go` stays at its existing 1807 lines minus the two parent bodies — no attempt to further split it in this substage.
   - `handlePlayerVsMob()` (286 lines) — decompose into target-resolution, wait-round short-circuit, damage-bonus pipeline, crit/effect dispatch, progression, and post-round aggro helpers.
   - `handleMobVsPlayer()` (236 lines) — same phase decomposition, mirrored for a mob attacker. Shared helpers from the player-vs-mob extraction get reused where the shape is symmetric (damage-bonus pipeline, wait-round short-circuit). Asymmetric pieces (charmed-mob assist, mob stat-gain messaging) stay specific.
2. **Spell batch** — refactor `applyMobEffect()` by extracting one helper per `EffectType` case. Helpers stay inline in `spell_resolution.go` (the switch cases don't form a standalone theme — they're branches of one dispatcher).
   - `applyMobEffect()` (246 lines) — extract `applyMobEffect_damage`, `applyMobEffect_dot`, `applyMobEffect_knockdown`, `applyMobEffect_buff`, `applyMobEffect_tame`, `applyMobEffect_default`. Parent becomes a thin dispatcher (~50 lines) plus shared pre-case setup (`critTag`, `viewerId`, `mName`) and shared post-case aggro write-in.

**Out of scope:**

- `applyPlayerEffect()` — not on the 1.2a target list from the overview. Leave untouched even if the case-per-EffectType pattern is mirrored. A future 1.2-follow-up can apply the same pattern once 1.2a has baked.
- `applyRoomEffect()` and `resolveMobSpellAgainstMob()` — ditto, out of scope.
- `handleMobVsMob()` — not on the target list, not in the overview. Untouched.
- `handlePlayerVsPlayer()` — not on the target list; already smaller. Untouched.
- Any function under 200 lines in these files.
- Any behavior change, except clear-bug fixes during refactoring (see scope-creep policy below).
- Splitting `NewRound_DoCombat_helpers.go` further into themed files — the 1.2a remit is god-function decomposition, not a character.go-style file split.

## Decisions Locked During Brainstorming

- **Lean on existing tests + manual smoke, do NOT write new characterization tests for the combat handlers.** `handlePlayerVsMob` and `handleMobVsPlayer` are not unit-tested today (per the in-file comment in `hooks_test.go`: *"Full DoCombat with aggro requires combat.ResolveAttack which depends on items/weapons initialization — covered by integration tests"*). Writing characterization tests for them would require pulling in the entire `combat.AttackPlayerVsMob` / `combat.AttackMobVsPlayer` stack plus deterministic dice, moon mods, mutation lookups, events emission, and return-damage/species tables. The cost is days, not hours — well beyond the 5h budget. `applyMobEffect` already has 7 existing branch-covering tests in `hooks_test.go` (`TestApplyMobEffect_Damage`, `_DamageWithCrit`, `_DotEffect`, `_NilUser`, `_Knockdown`, `_Buff`, `_Tame_NotAnimal`, `_DefaultEffect`). That suite doubles as a characterization net for the spell refactor at zero extra cost. For the combat handlers, manual smoke + a test-mud AI run is the verification gate — not ideal, but honest about the cost of genuine coverage here.
- **New file for combat helpers, inline for spell helpers.** `NewRound_DoCombat_helpers.go` is already 1807 lines and holds *many* helpers that are not specific to the PvM / MvP pair. New PvM/MvP helpers cluster into a clear theme — "post-attack-roll resolution pipeline" — and have enough volume (~6 helpers × ~40 lines) to justify a themed file `NewRound_DoCombat_resolution.go`. Spell `applyMobEffect` cases, by contrast, are branches of one function — they belong together in `spell_resolution.go`. This mirrors 1.2b's reasoning (inline for character helpers because they decompose one parent's logic; new files for admin because helpers clustered into themes).
- **Combat batch first, spell batch second.** Combat is riskier (more mutation sites, more state) and larger — doing it first gets the hard work out of the way while focus is fresh. `applyMobEffect` runs second as an easier consolidation against an existing test net.
- **Within the combat batch, `handleMobVsPlayer` refactors FIRST, then `handlePlayerVsMob`.** `handleMobVsPlayer` is smaller (236 vs 286 lines) and already delegates several concerns (`handleMobDownedGrace`, `handlePartyAutoAttack`, `handleMobTargetSwitch`, `handleMobWeaponPickup`, `handlePlayerConcentrationBreak`, `handleOffhandBreakUserDef`) to helpers that predate this substage. Refactoring it first produces the shared helpers (damage-bonus pipeline, wait-round short-circuit) that the larger `handlePlayerVsMob` refactor will then reuse, avoiding a second extraction pass.
- **Scope-creep policy: fix clear bugs in a separate prior commit; pause on ambiguous cases.** Same as 1.2b. Clear bugs (nil deref, dropped error, obvious typo) get a preceding `fix:` commit so the refactor diff stays semantics-preserving. Ambiguous cases stop and ask — intentional → add a code comment explaining why, not a bug → case-by-case fix-or-TODO.
- **One commit per extraction step, not one commit per whole function.** 1.2b used one commit per parent function. For 1.2a the parents are bigger and the helpers more numerous; a wrong turn on extraction #4 shouldn't force a revert of extractions #1–3. Each commit does one named extraction and is independently revertable.

## Architecture

**Execution order — 8 commits on `feature/stage-1.2a-combat-spell-refactor`:**

| # | Commit | Description |
|---|--------|-------------|
| 1 | `refactor(hooks): extract combat damage-bonus pipeline helpers` | Pull the four consecutive damage-bonus stages (Conviction Surge, Adrenal Surge, return damage, lifesteal) into helpers. Produced by reading `handleMobVsPlayer` first since its version is cleanest, but immediately applied to both parents in this commit. New file `NewRound_DoCombat_resolution.go` created here. |
| 2 | `refactor(hooks): extract combat wait-round short-circuit helper` | The `RoundsWaiting > 0` block appears symmetrically in both parents (~20 lines each) with side-specific `SourceTarget` tags. One helper with `role` params replaces both. |
| 3 | `refactor(hooks): extract handleMobVsPlayer phase helpers` | Break remaining `handleMobVsPlayer` body into target-validation, attack-and-bonus, crit-and-messaging, and progression-and-aggro helpers. Parent shrinks to ~60 lines. |
| 4 | `refactor(hooks): extract handlePlayerVsMob phase helpers` | Same decomposition applied to `handlePlayerVsMob`, reusing helpers from commits 1–3 where symmetric. Parent shrinks to ~80 lines. |
| 5 | `refactor(hooks): extract applyMobEffect damage + dot + knockdown cases` | Three case helpers extracted. Parent's switch cases become one-liners. |
| 6 | `refactor(hooks): extract applyMobEffect buff + tame + default cases` | Remaining three cases extracted; parent becomes a ~50-line dispatcher + shared pre/post-case setup. |
| 7 | `refactor(hooks): consolidate shared applyMobEffect aggro helper` | The "set aggro on both sides immediately" block appears three times inside `applyMobEffect` (damage, dot, knockdown) — collapse into one helper reused by all three case helpers. |
| 8 | `docs: mark code cleanup 1.2a complete` | Flip status in `code_cleanup_stage_1_overview.md`. |

**Why this order:**

- Commits 1–2 extract the cross-parent shared helpers FIRST. Doing it this way keeps commits 3–4 focused on genuinely single-parent logic — commit 4 (the biggest) isn't *also* discovering shared code.
- Commit 3 (`handleMobVsPlayer`) before commit 4 (`handlePlayerVsMob`) because the smaller parent confirms the decomposition before applying it to the larger one.
- Commits 5–7 are the spell batch, isolated from combat. Commit 7 (aggro consolidation) is a distinct cleanup that only becomes visible *after* 5 and 6 have extracted the per-case helpers — trying to see the pattern before extraction is harder.
- Commit 7 is intentionally separate so if the aggro-helper consolidation turns out to have a subtle difference per case (e.g., a `PreventIdle = true` only in some branches), the earlier extractions stay clean.

**Commit cadence rule:** after each commit, `go build ./... && go vet ./... && go test ./...` must be clean. No `--no-verify`. If a hook fails, fix and create a new commit (per repo convention).

## Testing Strategy

No new characterization tests. Verification rests on three legs:

1. **Existing `applyMobEffect` coverage.** Seven tests in `internal/hooks/hooks_test.go` exercise every branch of the switch, plus the nil-user path and the default case. These run on every commit in the spell batch and must stay green. For the refactor to be correct, the tests run against the refactored code with no modifications.
2. **Full `go test ./...` per commit.** Catches cross-package fallout from import or signature changes. Commit gate.
3. **Manual smoke at end of combat batch + end of spell batch.** Script:
   - **Combat smoke (after commit 4):** Boot local server, spawn a hostile mob, attack it (PvM path), let it retaliate (MvP path), verify hit/miss/crit messages appear correctly and both parties' Health/Stamina move as expected. Repeat with a 2H weapon, with dual-wield, with a shield, with a character that's wait-rounding (use `rest` adjacent-round). Run one round with Moon Phase active if possible. Run the test-mud AI against the server for 5 minutes of autonomous combat.
   - **Spell smoke (after commit 7):** Cast one spell per `EffectType` branch at a mob: `sparks` (damage), `poison` (dot), `knockback` (knockdown), a buff spell (`weaken` or equivalent), `tame` on an animal mob. Verify effect messages, damage numbers, aggro onset, crit flagging. Verify player-side bonus-damage modifiers (Adrenaline Surge, lifesteal) still fire by attacking with low HP and with a lifesteal-enchanted weapon.

Escalation: if the test-mud AI run surfaces a divergence that isn't a known pre-existing bug, stop, diagnose, and either revert the triggering commit or fix in a follow-up commit before proceeding.

## Refactor Patterns per Function

### `handleMobVsPlayer` (236 → ~60 lines parent)

Extracted helpers (all in `NewRound_DoCombat_resolution.go`):

- `resolveMobVsPlayerTarget(mob) (defUser, defRoom, ok)` — target validity gate. Runs the `defUser == nil`, room-mismatch, `defRoom == nil` checks, calls `EndAggro` on fail. Also handles the `CancelBuffsWithFlag(CancelIfCombat)` call which happens immediately after target resolution.
- `handleCombatWaitRound(attacker, defender, atkRole, defRole, atkRoom, defRoom, viewerId)` — shared wait-round short-circuit. Called from both parents.
- `applyCombatDamageBonuses(roundResult, attacker, defender)` — damage pipeline: Conviction Surge, Adrenal Surge, return damage, lifesteal. Returns the mutated `roundResult` (or mutates in-place). Attacker/defender types are `*characters.Character`; callers pass `user.Character` / `&mob.Character` as appropriate. **Note:** the player-vs-mob version has explicit source/target-room messaging for return damage ("You recoil…"); the mob-vs-player version has the symmetric mob-source version. Abstract the message emission into a role-aware inner call or keep it inline per parent — decide during commit 1 implementation.
- `dispatchCombatRoundMessages(roundResult, atkUser, defUser, atkMob, defMob, atkRoom, defRoom, viewerId)` — the big message-emission block (messages to source, source-room, target-room, dark-room fallback). Parameters vary by role; probably simpler to keep two variants `dispatchCombatRoundMessages_PvM` / `dispatchCombatRoundMessages_MvP` rather than one union. Pre-existing `dispatchCombatMessages` helper suggests a PvP-focused variant — confirm during extraction whether it's reusable.
- `handleMvPCritAndConcentration(mob, defUser, defRoom, roundResult, mobRoom)` — runs `applyCritEffects`, dispatches `DefenderMsg` and `RoomMsg`, runs `handleCharmedMobAssist`, then `handlePlayerConcentrationBreak` and `handleOffhandBreakUserDef`.
- `handleMvPProgression(mob, defUser, roundResult, mobRoom)` — stat-gain messaging, weapon skill use, unarmed progression, defender progression.
- `handleMvPRoundResolution(mob, defUser)` — endgame: health ≤ 0 → EndAggro / RetargetOrEnd / SetAggro.

Parent flow: validate → buff cancel → downed grace → hidden check → aggro setup → auto-assist → grapple → target switch → weapon pickup → wait-round short-circuit → attack roll + moon mod → damage bonuses → analytics + crit → messaging → progression → resolution.

### `handlePlayerVsMob` (286 → ~80 lines parent)

Extracted helpers (in `NewRound_DoCombat_resolution.go`, reusing commits-1/2 shared helpers):

- `resolvePlayerVsMobTarget(user, uRoom) (defMob, defRoom, ok)` — target validity. Includes the multi-room exit check (`FindExitByName`) absent from MvP, the `combat_start` emission for first-round defenders, `CancelBuffsWithFlag`, and the `Health < 1` gate.
- (reuses `handleCombatWaitRound` from commit 2)
- (reuses `applyCombatDamageBonuses` from commit 1)
- `handlePvMAttackAndAnalytics(user, defMob, roundResult, evt.RoundNumber, uRoom)` — `RecordAttack`, `applyCritEffects`, `dispatchCritEffectsPvM`, `replaceDarknessMessages`, buff applications from `roundResult`.
- (reuses `dispatchCombatRoundMessages_PvM` from commit 3, or extracts a new PvM variant during commit 4)
- `handlePvMConcentrationAndBehavior(defMob, roundResult, uRoom, userId)` — mob concentration break check + `behaviortree.TryMobBehavior` for `mob_hurt`.
- `handlePvMProgression(user, defMob, roundResult)` — attacker stat/skill use (strength, dexterity, per-weapon, unarmed), defender progression, hostility group marking.
- `handlePvMAggroAndAssist(user, defMob, c)` — mob-aggro-on-attack logic (set aggro, issue `go <exit>` + `GetAngryCommand`, issue `attack` command) + `handleCompanionOwnerAssist`.
- `handlePvMRoundResolution(user, defMob, uRoom)` — endgame: health ≤ 0 → EndAggro / RetargetOrEnd / SetAggro, plus the retarget message emission specific to the user's perspective ("You turn your attention to…").

Parent flow: mirror of MvP flow, but with the PvM-specific target resolution, attack-and-analytics path, and retarget messaging.

### `applyMobEffect` (246 → ~50 lines parent + shared pre/post helpers)

Parent becomes a dispatcher over `spellData.EffectType`. Per-case helpers all take `(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool, critTag string, mName string) (dmgDealt int)`:

- `applyMobEffect_damage` — spell-deflection-aware damage, three messaging branches (critDeflect / partial deflect / normal), aggro setup.
- `applyMobEffect_dot` — `ConditionPoisoned` application with duration calc, aggro setup, messaging.
- `applyMobEffect_knockdown` — damage + prone + position lockout + aggro setup, two messaging branches (deflected / normal).
- `applyMobEffect_buff` — buff application with tick-pool snapshot computation, conditional aggro for `Harm*` types, messaging.
- `applyMobEffect_tame` — animal-check, charm-chain cleanup, charm application, messaging.
- `applyMobEffect_default` — fallback messaging.

Plus a consolidated helper (commit 7):

- `setMobSpellAggro(user, mob)` — the 7-line "set aggro on both sides immediately" block currently duplicated in damage/dot/knockdown.

Parent shape after all commits:

```go
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {
    critTag := ""
    if isCrit { critTag = ` <ansi fg="yellow">[CRIT!]</ansi>` }
    viewerId := 0
    if user != nil { viewerId = user.UserId }
    mName := mobDisplayName(mob, room, viewerId)

    switch spellData.EffectType {
    case "damage":    return applyMobEffect_damage(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
    case "dot":       return applyMobEffect_dot(user, mob, room, spellData, magnitude, critTag, mName)
    case "knockdown": return applyMobEffect_knockdown(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
    case "buff":      return applyMobEffect_buff(user, mob, room, spellData, critTag, mName)
    case "tame":      return applyMobEffect_tame(user, mob, room, mName)
    default:          return applyMobEffect_default(user, spellData, mName)
    }
}
```

## Verification & Rollout

**Per-commit verification:**
```bash
go build ./...
go vet ./...
go test ./internal/hooks/...     # after every commit
go test ./...                     # after commit 4 and commit 7 (batch boundaries)
```

Any failure = fix before committing. For commits 5–7, the existing `TestApplyMobEffect_*` tests must all still pass.

**After combat batch (commits 1–4):**
- `go test ./internal/hooks/...` clean.
- Full combat smoke (see Testing Strategy).
- Diff review: each commit's diff reads as purely structural — no dropped checks, no reordered mutations that observable state depends on.

**After spell batch (commits 5–7):**
- All 7 existing `TestApplyMobEffect_*` tests pass unchanged.
- Spell smoke with one cast per EffectType branch.

**Pre-push:**
- All commits pass per-commit verification.
- Combat + spell manual smoke complete.
- Test-mud AI run (5 min) with no divergences.
- `PATCH_NOTES.md` — no entry (internal-only, zero player impact).
- `code_cleanup_stage_1_overview.md` — 1.2a flipped to Complete.

**Branch strategy:**
- Branch: `feature/stage-1.2a-combat-spell-refactor` off `development`.
- Merge `--no-ff` into `development`.
- No urgent prod push — combat changes deserve a soak. Consider landing on `development` for a week of test-mud exposure before rolling to `master`.

**Rollback:**
- Each commit independently revertable. Breakage in commit 4 → revert just 4 without losing 1–3. Shared-helper commits (1–2) staying even if commits 3–4 revert is the whole reason the order is structured that way.
- Spell-batch commits (5–7) are independently revertable from combat-batch commits and from each other.

## Risk Register

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Combat handlers have no characterization tests; a subtle regression slips past manual smoke | **High** | Test-mud AI run is the primary catcher for behavior drift. Each commit diff reviewed for structural-only changes. Batch commits small (one extraction per commit) so any regression localises. |
| Damage-bonus pipeline helper unifies two slightly-different message emissions and loses a role-specific nuance | Medium | Commit 1 is the riskiest single commit; plan to diff-review it very carefully. If the PvM/MvP versions differ enough that unification hides semantics, keep them as sibling helpers `applyCombatDamageBonuses_PvM` / `_MvP` rather than forcing one. |
| Extracted helper has too many parameters and becomes unreadable | Medium | Accept ugly signatures as tolerable for private helpers — the parent readability win is the point. Where 8+ params piles up, introduce a private `combatCtx` struct for commit 3 or later. |
| `applyMobEffect_damage` and `applyMobEffect_knockdown` share deflection logic that begs consolidation | Medium | Don't consolidate in this substage — deflection details (crit-deflect vs plain-deflect, which side applies the multiplier, whether position gets set) differ enough that premature unification is its own risk. Leave duplicated; flag as a 1.2a-follow-up TODO. |
| `combat_start` mob memory emission moves during `handlePlayerVsMob` refactor and gets delayed past the attack roll | Low | The emission currently sits before the attack roll *on purpose* (inline comment flags it). Helper extraction must preserve call order; mark it explicitly in `resolvePlayerVsMobTarget` or keep it in the parent. |
| Return-damage / lifesteal block mutates attacker state before messaging — re-ordering breaks a downstream read | Low | Damage-bonus helpers take explicit in/out params; no globals. Post-extraction diff verifies state mutations are emitted in the same order. |
| New file `NewRound_DoCombat_resolution.go` introduces a circular helper reference with existing `combat_shared_helpers.go` | Low | Same package — no import cycle possible. `go build` catches any symbol-resolution issue. |
| Scope creep — "while I'm in here" fixes pull in unrelated refactors | Low | Scope-creep policy. `fix:` commits allowed only for clear bugs; ambiguous cases pause. |

## Success Criteria

- `handlePlayerVsMob` reduced from 286 → ≤80 lines parent with helpers each <80 lines.
- `handleMobVsPlayer` reduced from 236 → ≤60 lines parent with helpers each <80 lines.
- `applyMobEffect` reduced from 246 → ~50 lines parent (dispatcher + shared setup) with each case helper <80 lines.
- New file `internal/hooks/NewRound_DoCombat_resolution.go` houses the combat phase helpers; `internal/hooks/spell_resolution.go` grows by the `applyMobEffect_*` case helpers but doesn't need a split.
- `go build ./...`, `go vet ./...`, `go test ./...` clean after every commit.
- All 7 existing `TestApplyMobEffect_*` tests pass on refactored code.
- Combat manual smoke passes: PvM hit/miss/crit, MvP retaliation, wait-rounds, 2H/dual-wield/shield configurations, Moon Phase active.
- Spell manual smoke passes: one cast per EffectType (damage, dot, knockdown, buff, tame), damage numbers and messages unchanged, aggro onset correct.
- 5-minute test-mud AI autonomous combat run shows no divergences from pre-refactor behavior.
- No player-visible behavior change.
