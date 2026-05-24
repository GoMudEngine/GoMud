# Grapple Drift Formula Rework (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor work:** chunks 4a / 4b / 4c / 4d / 4b-fixup / 4b-fixup-2 (ControlLevel FSM, pair iteration, 280+ outcome templates, 36 gradient templates)
**Successor work:** chunk 4f (balance + full smoke), 4e (third-party)

---

## 1. Problem Statement

Two bugs in `internal/hooks/Position_GrappleTick.go:processGrapplePair`'s per-round drift formula:

**Bug 1 — wrong skill referenced.** The formula reads `skills.WeaponCombat` for both sides. WeaponCombat is the sword/axe/mace skill; UnarmedCombat is the documented grappling skill (its own comment at `internal/skills/skills.go:29` literally says `// Fist/body attacks & defense, grappling`). Trained grapplers (e.g. quester0 with `unarmed-combat: 30`) get zero credit for their training; they bring `weapon-combat: 1` to the roll instead.

**Bug 2 — defender bonus is structurally too large.** The defender side gets `+0.5·Dex + EscapeModifier` on top of the base `Str + WeaponCombat` formula. For a defender at Dex 91 (steppe boar), this is +45.5 to defender_score before the actual roll variance even kicks in. Against an attacker with comparable Str and no skill credit, the deterministic margin already produces z ≤ -2.0 — escape fires every round, no matter what.

**Observable result** (quester0 vs steppe boar smoke, 2026-05-19): every single `grapple boar` command produces `OutcomeEscape` on the first drift roll. The grapple never gets past round 1. Across 7+ attempts in `bug_log.txt`, the boar broke free immediately each time despite quester0 having both higher Str (effective 116 vs ~149 raw — soft-cap-adjusted similar) and 30 unarmed-combat training.

**Why prior chunks didn't catch this:**
- Chunk 4b had a "Controlled for 2 consecutive rounds → escape" gate that masked the imbalance; round 1's bad roll didn't immediately end the grapple.
- Chunk 4b-fixup removed that gate and made escape directly z-bucketed at \|z\| ≥ 2.0.
- Chunk 4b-fixup-2 inherited the same formula and added the ControlLevel FSM on top, but didn't revisit the score math.

The smoke test exposed both bugs as a single symptom: immediate escape on every grapple, regardless of player skill.

---

## 2. Design Goals

1. **Correct skill.** UnarmedCombat is the grappling skill; use it.
2. **Symmetric formula.** Both sides compute the same shape — no role-based unilateral bonus. Position bias is already captured by `ControlLevel` state initialization (chunk 4b-fixup-2); the formula shouldn't double-encode it.
3. **Skill matters but doesn't dominate.** A trained grappler should reliably beat an untrained one of comparable stats, but a maxed-skill (50) vs zero-skill matchup at otherwise-balanced stats should be `z ≈ ±2.5` (decisive but not auto-instant). Mid-range skill gaps should sit in the 1- to 2-step outcome bands.
4. **Initiative edge.** The aggressor (the side that triggered the grapple via `grapple` command or btree action) gets a small skill-scaled bonus — they chose the moment. 10% boost to their UnarmedCombat contribution.
5. **Drop the EscapeModifier read.** Field stays on ItemSpec (no content data breaks), but the formula stops reading it. Encumbrance multiplier already encodes "heavy armor hurts grappling" indirectly.
6. **Z-scores in [-2, 2] for typical pairs.** Escape (\|z\| ≥ 2) should be a notable round, not a default.

---

## 3. The Formula

For each side per round:

```
base_score = (0.7·Str + 0.3·Dex + skill_coefficient·UnarmedCombat)
             × stamina_multiplier
             × encumbrance_multiplier
```

where:
- `Str` and `Dex` are the character's `Stats.{Strength,Dexterity}.Value` (already soft-cap-adjusted via `Recalculate()`).
- `UnarmedCombat` is `Character.GetSkillLevel(skills.UnarmedCombat)`.
- `skill_coefficient = 2.2` if this side has `GrappleData.IsAggressor == true`; else `2.0`.
- `stamina_multiplier` is the existing `grappleStaminaMultiplier` (unchanged from chunk 4b).
- `encumbrance_multiplier` is the existing `grappleEncumbranceMultiplier` (unchanged from chunk 4b).

The pair's two scores feed `dice.OpposedRollStat(attacker_score, defender_score)` → margin → z. The chunk-4b-fixup-2 z-bucket outcome resolver (`position.ResolveOutcome`) is **unchanged**.

### Per-component rationale

| Component | Coefficient | Why |
|---|---|---|
| Str | 0.7 | Primary grappling stat. Muscling through positions, holding pins, lifting takedowns. |
| Dex | 0.3 | Secondary. Setups, framing, scrambling. Same on both sides (no asymmetry). |
| UnarmedCombat | 2.0 | Skill cap 50 → +100 score contribution. Comparable in magnitude to a maxed stat. Skill is heavy-weighted because grappling is a discipline. |
| Aggressor edge | +0.2·UnarmedCombat | Initiative bonus; scales with skill. Skilled grapplers set up better; unskilled lunges get nothing extra. |
| stamina_mult | (existing) | Gassed grapplers are worse. Multiplicative — drains the whole score. |
| encumbrance_mult | (existing) | Loaded-up grapplers are worse. Also multiplicative. |

### Sample z-scores

Computed with no stamina/encumbrance penalty (both at 1.0):

| Pair | atk score | def score | margin | sigma | z | Outcome tier |
|---|---|---|---|---|---|---|
| Quester0 (S116/D126/UC30) aggressor vs steppe boar (S149/D91/UC0) | 185 | 131.6 | +53.4 | 27.75 | +1.92 | 2-step advance |
| Trained (S100/D100/UC30) aggressor vs untrained (S100/D100/UC0) | 166 | 100 | +66 | 24.9 | +2.65 | 3-step advance + sub |
| Untrained (S100/D100/UC0) aggressor vs trained (S100/D100/UC30) | 100 | 160 | -60 | 15.0 | -4.00 | Escape (defender wins decisively) |
| Equal pair (S100/D100/UC15 vs S100/D100/UC15), A aggressor | 133 | 130 | +3 | 19.95 | +0.15 | Hold |
| Str-brute (S140/D80/UC20) aggressor vs Dex-glass (S80/D140/UC20) | 166 | 140 | +26 | 24.9 | +1.04 | 2-step (Str-leaning fight) |

**Reads:**
- Quester0 (the bug reporter's scenario): now wins by 2-step margins instead of losing by 4. The boar's raw Str advantage no longer auto-escapes.
- Trained vs untrained at equal stats: decisive but not insta-escape (+2.65 z is just into 3-step / sub-window territory).
- Untrained aggressor vs trained defender: escapes immediately (z = -4). Realistic — a beginner can't hold a black belt.
- Equal grapplers: Hold band (small drift either way). Most rounds will inch the gradient back and forth; outcomes are earned through stamina + cumulative shifts.

---

## 4. Implementation Scope

**File:** `internal/hooks/Position_GrappleTick.go`

### Changes
1. **Score formula** at lines ~261-265: replace `Str + WeaponCombat` body with the new shape.
2. **Aggressor coefficient lookup** at the top of `processGrapplePair`: read `GrappleData.IsAggressor` for each side; use `2.2` if true, `2.0` if false.
3. **Comment block** at lines ~258-260: update to reflect new formula.
4. **Delete `escapeModifierFromBody`** helper (~lines 660-680) — no longer called.

### What stays unchanged
- `grappleStaminaMultiplier` and `grappleEncumbranceMultiplier` (chunk-4b).
- The `dice.OpposedRollStat` call, margin/z calculation, and `LastDriftRoll` snapshot.
- The z-bucket outcome resolver (chunk 4b-fixup-2).
- The pair iteration in `processGrappleTick` (chunk 4b-fixup-2 T8).
- The ControlLevel shift logic (chunk 4b-fixup-2 T9).
- All messaging — outcome, gradient, hold, mount-strike-apex.
- Sub eligibility predicates.

### What's preserved but unused
- `ItemSpec.EscapeModifier` field stays in place. Existing YAML content (cotton-shirt EM=1.0, chainmail-vest EM=-0.05, etc.) is preserved. Formula stops reading it; the field becomes vestigial unless future work re-purposes it (e.g., for sub-eligibility, defender skill bonus on specific positions, etc.).

---

## 5. Testing Strategy

### Unit tests (`internal/hooks/Position_GrappleTick_test.go` — extend existing)

Validate the score formula directly:

- `TestGrappleScore_StrWeight07`: a character with Str=100, Dex=0, UC=0 → score 70 (verify Str coefficient).
- `TestGrappleScore_DexWeight03`: a character with Str=0, Dex=100, UC=0 → score 30.
- `TestGrappleScore_SkillWeight2_Defender`: defender-side, Str=100, Dex=100, UC=50 → score = 100 + 100 = 200.
- `TestGrappleScore_SkillWeight22_Aggressor`: aggressor-side (IsAggressor=true), Str=100, Dex=100, UC=50 → score = 100 + 110 = 210.
- `TestGrappleScore_StaminaMultiplierApplies`: same as above but Stamina at penalty band — score is scaled down by stamina_mult.
- `TestGrappleScore_EscapeModifierIgnored`: defender wears body armor with EM=1.0; score is unchanged from EM=0.0 case.

### Integration regression test

- `TestProcessGrapplePair_QuesterVsBoarSurvives`: simulate quester0's stats + steppe boar's stats, run 10 drift rolls. Assert at least 8 of them produce `OutcomeHold` or `OutcomeAdvance` (not Escape). Catches the original bug.

### Smoke (T-1 boot smoke + AI feature-tester rerun)

Re-run the chunk-4b-fixup-2 feature-tester smoke goal file (`tools/testing/goals/chunk-4b-fixup-position-advancement-smoke.yaml`). Expected change: grapples don't immediately escape; position advances; gradient messages still fire per chunk-4b-fixup-2 wiring.

---

## 6. Migration / Implementation Order

Single self-contained change; one or two commits is sufficient.

1. Modify `processGrapplePair` score formula (the two lines computing `ctrlBase` and `cdBase`).
2. Add the aggressor-coefficient lookup helper.
3. Delete `escapeModifierFromBody` (and the comment chain referencing EscapeModifier).
4. Update the comment block above the score computation.
5. Add unit tests covering the new formula's coefficients.
6. Add the integration regression test for the quester-vs-boar scenario.
7. Boot smoke + AI feature-tester smoke rerun.

---

## 7. Risks / Open Questions

- **Aggressor reads `IsAggressor` from `GrappleData`.** Chunk 4b-fixup-2 T5 set this field via `markAggressor(attacker)` in `ApplyGrappleResult`. Verify it's actually populated for both player-initiated (`grapple` command) and mob-initiated (btree `grapple` primitive) entries. If either path skips the mark, that side never gets the aggressor bonus. Smoke catches it.
- **Symmetric formula means SOME grapples won't progress.** Two equal grapplers will mostly Hold round after round; stamina drain eventually breaks the tie. Verify in smoke that this isn't tedious — Hold flavor (sparse, every ~4 rounds) helps but if grapples routinely last 15+ rounds with no advancement, players may feel stuck. Consider adding small per-round entropy if smoke surfaces it. Out of scope for this fix.
- **EscapeModifier becomes vestigial content.** The field stays valid on ItemSpec; future systems (sub eligibility, armor-specific resistance, etc.) can re-claim it. Not a regression in functionality — the field's prior contribution was so small it barely mattered. No content authors will notice.
- **Stat soft cap interaction.** `Stats.Strength.Value` is already soft-cap-adjusted via `Recalculate()`. The formula sees post-cap values; high-stat characters don't get runaway bonuses past 150. Verify nobody bypasses the cap when reading.

---

## 8. Out of Scope

- **ControlLevel-gated escape outcomes.** Chunk 4b-fixup-2 §16 explicitly said escape is z-bucketed, not state-gated. The original bug discussion floated this as an alternative fix, but the formula fix addresses the root cause cleanly; gating escape on state would be a separate design decision.
- **Re-tuning chunk 4b-fixup-2 z-buckets.** The 0.5 / 1.0 / 2.0 thresholds stay. The fix is upstream — produce sensible z-scores in the first place.
- **Mob archetype-specific grapple bias.** A "wrestler" archetype that gets extra UC at spawn could be a content design later. Out of scope for the formula fix.
- **Species-gated grappling.** Tracked separately as `project_species_gated_grappling.md`. Wolf BJJ is still goofy regardless of formula tuning.
- **The combatstats Grapple Controller % bug.** Logged at `project_combatstats_grapple_controller_pct.md`. Separate concern.

---

## 9. Success Criteria

1. `quester0 grapple boar` produces a grapple lasting 3+ rounds in a fresh smoke test (not immediate escape).
2. Untrained quester0 grappling a black-belt-equivalent mob still escapes immediately — skill gap matters.
3. Equal-stat / equal-skill pairs sit in Hold for most rounds (z near 0).
4. All chunk-4b-fixup-2 messaging (outcomes, gradients, hold flavor, mount strike apex) still fires per the existing wiring.
5. Unit tests cover all formula coefficients (Str weight, Dex weight, skill weight, aggressor coefficient, stamina/encumbrance multiplication).
6. Regression test catches the original "immediate escape against boar" symptom.
7. No surviving reads of `body.EscapeModifier` from the grapple formula.
8. AI feature-tester smoke rerun passes the chunk-4b-fixup-2 goals without regression.
