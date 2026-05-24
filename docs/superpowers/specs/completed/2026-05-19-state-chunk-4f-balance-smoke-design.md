# State Chunk 4f — Position Balance + Smoke (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor chunks:** 4a-4e + 4b-fixup + 4b-fixup-2 + grapple-drift-formula-rework
**Successor chunks:** 5 (Presence), 6 (Perception). Chunk 4 (Position) concludes here.

---

## 1. Problem Statement

Chunks 4a-4e + the various fixups + the formula rework built the position state machine, the per-round drift loop, ControlLevel FSM, hit modifiers, third-party hooks, eat/drink restrictions, and a spell-disruption gap closure. Two known issues remain:

1. **Spell-disruption gates are deterministic 100% breaks.** Chunk 4e T4 closed a real gap (Mount-pinned casters could complete spells unimpeded) by adding `if IsGrappling() { 100% break }` to `processFoldRound`, mirroring the existing `IsProne() || IsSupine() { 100% break }` gates. Per user direction 2026-05-19: those Prone/Supine gates were also never supposed to be 100% — they predate the chance-based `checkConcentrationBreak` system in the same file. All three position-based break gates should be chance rolls mediated by Willpower.

2. **Wider-system tuning is unverified.** Chunk 4e shipped position hit modifiers, control degradation, sub interrupt, AI tiebreaker — all gated by config knobs and verified with a focused smoke. The full grapple-system experience needs a comprehensive smoke to surface anything that feels off in extended play.

Chunk 4f addresses both in tight scope: **spell-disruption rewrite + comprehensive smoke + react to critical findings only**. Anything else queued as a polish item goes into followup memories.

---

## 2. Design Goals

1. **Spell disruption becomes chance-based.** Replace the three deterministic `IsProne/IsSupine/IsGrappling` 100% gates in `processFoldRound` with a single chance-based check that calls the existing `CalcConcentrationChance(Wil, dmgPctEquiv)` and rolls. Reuses the Willpower curve already used by the damage-path concentration check — high-Willpower casters get meaningful protection.
2. **Per-position damage%-equivalent table.** Each grapple position + role gets a damage%-equivalent value (range 25-70). Standing returns 0 (skip check). Restrained values — a maxed-Wil caster has a real chance of holding concentration through Mount; Crucifix is still brutal but not impossible.
3. **Layered disruption.** The new position-disruption check runs each fold round at the top of `processFoldRound`. The existing damage-path `checkConcentrationBreak` continues to fire independently when damage lands during a round. Both paths can break a cast — the player faces real disruption from both their position AND incoming hits.
4. **Comprehensive smoke.** Two AI tester dispatches (feel-tester for tone/variety, feature-tester for mechanical correctness) against a goal file that covers every chunk-4 deliverable. Findings beyond the spell-disruption fix become followup memories, not in-scope work.
5. **Tight scope.** Out-of-scope explicitly listed. Polish items go into memory for future chunks.

---

## 3. Spell-Disruption Rewrite

### 3.1 The damage%-equivalent table

```go
// PositionDisruptionDmgEquiv returns the damage%-equivalent that the
// given position imposes on a caster's concentration. Fed into the
// existing CalcConcentrationChance(Wil, dmgPctEquiv) curve to produce
// a per-round disruption chance.
//
// Returns 0 for Standing (skip the check entirely). Higher values =
// more disruption. Controlled-role values are generally higher than
// controller-role values because the controlled side has hands and
// movement suppressed. Guard inverts: bottom (Controlling per our
// model) has free hands and lower disruption than top (Controlled
// per our model, in someone's guard).
```

| Position | Controller-role | Controlled-role |
|---|---|---|
| Standing | 0 (no check) | — |
| Prone | — | 30 |
| Supine | — | 25 |
| Clinch (symmetric) | 40 | 40 |
| BackStanding | 35 | 50 |
| Mount | 35 | 60 |
| SideControl | 30 | 55 |
| KneeOnBelly | 30 | 50 |
| NorthSouth | 30 | 45 |
| Crucifix | 35 | 70 |
| BackGround | 35 | 65 |
| HalfGuard | 30 | 40 |
| Guard (inverted) | 25 (bottom, free hands) | 40 (top) |
| Turtle | 35 | 45 |

### 3.2 Integration in `processFoldRound`

Current shape (chunk 4e T4):

```go
// Downed (prone or supine) → immediate concentration break.
if char.IsProne() || char.IsSupine() {
    clearCastingActivity(char, activity.TriggerConcentrationBreak)
    return FoldRoundResult{ProneBroke: true, CastingData: cs}
}

// Chunk 4e T4 — grapple-state catch-all.
if char.IsGrappling() {
    clearCastingActivity(char, activity.TriggerConcentrationBreak)
    return FoldRoundResult{GrappleBroke: true, CastingData: cs}
}
```

New shape (chunk 4f):

```go
// Position-based disruption check (chunk 4f). Replaces the
// deterministic Prone/Supine/Grapple gates with a single chance roll
// mediated by Willpower (same curve as damage-path checkConcentrationBreak).
posState := char.Position.State()
ctrlState := control.Neutral
if char.Control != nil {
    ctrlState = char.Control.State()
}
dmgPctEquiv := position.PositionDisruptionDmgEquiv(posState, ctrlState)
if dmgPctEquiv > 0 {
    chance := characters.CalcConcentrationChance(
        char.Stats.Willpower.ValueAdj, dmgPctEquiv)
    roll := util.Rand(100)
    util.LogRoll(`PositionConcentration`, roll, chance)
    if roll >= chance {
        // Concentration broke. Use whichever break-flag matches the
        // position so caller messaging routes correctly.
        clearCastingActivity(char, activity.TriggerConcentrationBreak)
        result := FoldRoundResult{CastingData: cs}
        switch {
        case char.IsProne():
            result.ProneBroke = true
        case char.IsSupine():
            result.ProneBroke = true // ProneBroke handles both (existing convention)
        case char.IsGrappling():
            result.GrappleBroke = true
        }
        return result
    }
    // Roll passed — concentration held; fold continues normally.
}
```

The `FoldRoundResult` fields stay unchanged (`ProneBroke`, `GrappleBroke`, etc.) so caller messaging routes correctly. The break-flag dispatch at the bottom mirrors the existing 100%-gate behavior for messaging purposes — caster sees the right "concentration shatters from grapple" vs "concentration shatters from being knocked prone" line.

### 3.3 Where the chance is rolled

Once per fold round, at the top of `processFoldRound` (which is called once per round per caster). This is the SAME firing cadence as the existing 100% gates — the only change is the deterministic break becomes a roll.

### 3.4 Layered with damage-path disruption

The damage-path `checkConcentrationBreak` is UNCHANGED. It fires when a caster takes damage during a round and triggers its own roll. Both paths can break a single cast in the same round — the position-disruption fires at the start of the fold round, the damage-disruption fires when a hit lands. The player faces the harder of the two each round (whichever fails first wins).

### 3.5 Messaging implications

Existing messages stay unchanged for ProneBroke / GrappleBroke (the caller already routes by which field is set). Only the flavor in the helpfile needs softening:
- **Before** (chunk 4e T11): "Spellcasting and other concentration-heavy actions are disrupted just as if you were knocked prone."
- **After** (chunk 4f): "Spellcasting becomes harder when you're on the ground or caught in a grapple — your Willpower determines how often your concentration holds."

---

## 4. Full-Stack Smoke + React

### 4.1 Comprehensive smoke goals file

Create `tools/testing/goals/chunk-4f-position-system-smoke.yaml` covering:

1. Grapple entry (player + mob initiated)
2. Position advancement (Clinch → Mount, etc.)
3. ControlLevel state changes (gradient messaging fires correctly)
4. Hit modifier verification (chunk 4e §3): Mount controller swinging at controlled lands meaningfully more often
5. Spell disruption (chunk 4f §3): caster pinned in Mount sometimes finishes spell, sometimes doesn't — chance-based
6. Eat/drink restriction (chunk 4e §4)
7. Sub windows (chunk 4d)
8. Sub interrupt (chunk 4e §7): crit damage during sub round forces Bad tier
9. Outside-damage control drift (chunk 4e §5)
10. AI tiebreaker (chunk 4e §6)
11. Helpfile reads naturally with the new disruption language

### 4.2 Two-pass smoke

**Feature-tester pass:** mechanical correctness. Verify each lever fires. Quantitative observations (hit rates, message presence, dedupe).

**Feel-tester pass:** qualitative play impressions. Variety, viscerality, coherence, pacing. Note anything that feels off without specifying mechanism.

### 4.3 React policy

After the smoke runs, classify findings:

- **Critical bug (regression from a prior chunk)** → fix in chunk 4f as a hotfix task.
- **Tuning-want (a value feels too high/low)** → log as a followup memory; don't fix in 4f unless multiple findings point to the same number.
- **New feature suggestion** → log as a memory; explicitly out of scope.
- **Helpfile gap** → log as a memory unless trivial to fix.

This keeps 4f bounded. If smoke surfaces 10 issues, 4f doesn't grow to 10 fix tasks — most become followup memories.

---

## 5. What Survives Unchanged

| Artifact | Notes |
|---|---|
| `position.TargetSideHitModifier` / `AttackerSelfHitModifier` (chunk 4e §3) | Unchanged unless smoke surfaces a critical issue |
| `position.ResolveOutcome` + outcome tables (chunk 4b-fixup) | Unchanged |
| ControlLevel FSM + transitions (chunk 4b-fixup-2) | Unchanged |
| Grapple-drift formula coefficients (grapple-drift-rework) | Unchanged |
| Chunk-4d submission system | Unchanged |
| Chunk-4e third-party hooks (outside damage, sub interrupt, AI tiebreaker) | Unchanged |
| `checkConcentrationBreak` (damage-path) | Unchanged — keeps firing layered with new position check |

---

## 6. New Artifacts

| Path | Responsibility |
|---|---|
| `internal/state/position/disruption.go` | `PositionDisruptionDmgEquiv(pos, role) int` lookup function + the damage%-equivalent table |
| `internal/state/position/disruption_test.go` | Unit tests for every (position, role) cell + Standing-returns-0 |
| `tools/testing/goals/chunk-4f-position-system-smoke.yaml` | Comprehensive smoke goals file |

### Modified files

| Path | Change |
|---|---|
| `internal/hooks/combat_shared_helpers.go` | Replace the three deterministic 100% gates in `processFoldRound` with a single chance-based check |
| `internal/state/position/context.md` | Document the new disruption lookup |
| `internal/hooks/context.md` | Document the rewrite of `processFoldRound`'s disruption gates |
| `_datafiles/world/dogmud/templates/help/grapple.template` | Soften disruption language ("disrupted just as if knocked prone" → "harder; Willpower determines hold rate") |
| `COMBAT_STATE_ROADMAP.md` | Add chunk 4f row as Done |

---

## 7. Out of Scope (Explicit)

- **Modifier value tuning** (chunk 4e position tables, grapple-drift coefficients, ControlLevel thresholds) — unless smoke surfaces critical issues. Polish items become followup memories.
- **Per-archetype AI bias variation** — chunk 4e shipped universal tiebreaker; per-archetype is polish.
- **Full helpfile rewrite** — just the one disruption-language softening.
- **Position-flavor template authoring** — chunk 4b-fixup-2 + chunk 4b-fixup already shipped ~280 outcome templates + 36 gradient templates. No new authoring in 4f.
- **Sub-tier alpha tweaks** — chunk 4d shipped with reasonable defaults; tuning is post-4f if needed.
- **`CalcConcentrationChance` rewrite** — the existing curve is reused as-is.
- **Combat damage formula changes** — chunk-4e position hit modifiers + grapple-drift-formula already addressed the math; further tuning is post-4f.

---

## 8. Testing Strategy

### Unit tests (new)

- `disruption_test.go`: every cell of the table returns the documented value; Standing returns 0; controlled-role values are >= controller-role values for every grapple position (sanity invariant); Guard inverts (bottom controller value < top controlled value).

### Integration tests

- `combat_shared_helpers_test.go` (extend): `processFoldRound` with a caster in Mount-controlled position. With a forced 0 roll (concentration always holds), the cast continues. With a forced 99 roll, the cast breaks. (Use a deterministic test fixture by mocking `util.Rand` or by setting Willpower very high/low.)

### Smoke (4f's verification deliverable)

- Two-pass AI smoke as described in §4.

---

## 9. Implementation Order

1. New `disruption.go` lookup + unit tests (pure code).
2. Refactor `processFoldRound` to call the new lookup + roll.
3. Soften helpfile disruption language.
4. context.md sweep.
5. Boot smoke + AI feature-tester smoke.
6. Feel-tester smoke pass.
7. React to findings (fix critical bugs only; log tuning-wants as followups).
8. COMBAT_STATE_ROADMAP row.

Estimated 5-7 tasks. The two smoke runs (T5 + T6) can fold into one task each, or one combined task with two dispatches.

---

## 10. Risks / Open Questions

- **The Wil curve in `CalcConcentrationChance` is shared between damage-path and position-path.** If 4f surfaces that the curve is wrong for the position context (e.g., grappled high-Wil casters complete every spell), we'd need to either retune the table values or fork the curve. Watch in smoke.
- **Standing-returns-0 means no position check.** That's correct (Standing isn't disrupted) but worth confirming in the unit test that Standing skips the entire path.
- **Layered disruption could feel over-punishing.** Caster takes a hit AND is grappled in the same round — two rolls, two chances to break. If smoke shows this is too cruel, decouple: only the layered roll fires, not both. Probably fine for v1.
- **The `ProneBroke` vs `GrappleBroke` dispatch in §3.2.** The caller (`handlePlayerFoldCasting` / `handleMobFoldCasting`) routes messaging by which boolean is true. Make sure the dispatch logic at the bottom of the new gate produces the right `FoldRoundResult` shape per the existing message routing in chunk 4e T4.

---

## 11. Success Criteria

1. Spell-disruption fires as chance-based: a 100-Wil caster pinned in Mount-controlled sometimes holds, sometimes breaks. Verified in smoke.
2. Crucifix-controlled caster almost always breaks (70% dmgPctEquiv with the existing Wil curve produces a low hold rate).
3. Guard-bottom controller-role caster (lowest disruption at 25) holds most rounds. Verified in smoke.
4. No regression in existing chunk 4a-4e functionality. All prior tests still pass.
5. Smoke surfaces 0-2 critical bugs (most findings become followup memories per §4.3).
6. Helpfile reads naturally with the softened disruption language.
7. COMBAT_STATE_ROADMAP row for 4f reflects "Done" with the deliverables shipped + count of followup memories generated by smoke.
