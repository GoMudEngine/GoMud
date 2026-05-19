# State Chunk 4e — Position Third-Party + Defense Degradation (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes`
**Predecessor chunks:** 4a (Position FSM), 4b (sunset), 4c (reach), 4d (submissions), 4b-fixup, 4b-fixup-2 (ControlLevel FSM), grapple-drift-formula-rework
**Successor chunks:** 4f (balance + smoke), 5 (Presence), 6 (Perception)

---

## 1. Problem Statement

After chunks 4a-4d + the 4b-fixup line + the grapple-drift-formula rework, grappling itself works: positions advance, ControlLevel state drives gradient flavor + sub eligibility, drift z-buckets produce earned outcomes. But the per-round COMBAT MATH doesn't yet respect position dominance:

**Bug observed in production smoke (2026-05-19):** quester0 mounts a steppe boar. Mount-strike-apex flavor fires correctly ("knees driving into their biceps — their arms can't lift to defend"). But the boar still dodges incoming attacks at roughly normal rate. Position dominance isn't translating to hit-rate or damage advantage. The narrative says "you have them pinned"; the dice say "they're fine."

Five related gaps in how grappling integrates with the rest of combat:

1. **Defense degradation isn't position-aware.** A defender under Mount dodges as well as a standing fighter. Real grappling: defender's defensive options collapse the moment they're pinned.

2. **Offense restrictions are incomplete.** Movement and grapple-incompatible commands are blocked. But the player can quaff a potion or eat food mid-grapple — both hands busy, neither realistic.

3. **Outside damage doesn't disrupt the grappler.** A third party clobbering the controller has no mechanical effect on the grapple. Realistic: a kick to the ribs while you're holding mount makes you lose your grip.

4. **Mob AI is grapple-blind.** Mobs don't pile on a pinned enemy when picking targets. A predator should sense weakness.

5. **Subs aren't interruptible by outsiders.** A submitter who's getting clobbered mid-armbar still finishes the sub. Realistic: outside damage breaks concentration on a sub setup.

The roadmap entry for chunk 4e captures these five gaps: "Symmetric defense degradation (controlled severely, controller moderately); offense restrictions; outside-damage → control degradation; mob AI bias toward grappled targets; submission-interrupt risk." This spec turns those bullets into a design.

---

## 2. Design Goals

1. **Position-aware hit math.** The attacker's hit roll picks up position-tiered multipliers based on both the target's position+role and the attacker's own. Mount controller swinging at controlled = significant bonus. Mount controlled swinging back = significant penalty.
2. **Symmetric attacker math.** Modifiers live on the ATTACK side and apply uniformly to first-party grapplers AND third-party intruders. Walking up on two grapplers makes both easier to bop.
3. **Restrained values.** Modifiers in the 0.50-1.25 range; even small shifts (1.10 → 1.20) are gameplay-significant. No 2× or 0.2× extremes — those make positions feel auto-win or auto-lose.
4. **Outside damage matters to grapples.** Third-party hits on the controller disrupt their grip. Defender unaffected (already pinned).
5. **AI behaves like a pack predator — gently.** Mobs prefer grappled targets as a TIEBREAKER, not as a primary factor. A mob already focused on a high-threat target doesn't switch off them.
6. **Sub interrupt is meaningful but not punishing.** Crit damage OR significant damage (>10% max HP) during a sub-firing round forces Bad outcome; small damage ignored.
7. **YAGNI on per-archetype tuning.** Single universal modifier tables for v1; per-archetype variation belongs in 4f.

---

## 3. Position Hit Modifiers (two-table system)

### 3.1 Lookup model

When a hit roll fires:
```
final_hit_modifier = AttackerSelfModifier(attacker.Position.State(), attacker.Control.State())
                   × TargetSideModifier(target.Position.State(), target.Control.State())
```

The product multiplies the attacker's existing hit roll output (whatever the current hit calc returns). Both modifiers default to 1.0 for Standing, so no behavior change outside grapples.

The lookup is keyed by `(position State, control State)` — the position machine for geometry + the control machine for role within that geometry. We use `Control.State() == Controlling` / `Controlled` to dispatch into the controller-role vs controlled-role columns of each table. Neutral (mid-drift) defaults to the controller-role row in symmetric positions (Clinch/HalfGuard/Turtle); to the asymmetric-role row otherwise.

Both tables live in a new file `internal/state/position/modifiers.go`. Exposed via:

```go
func TargetSideHitModifier(pos State, role control.State) float64
func AttackerSelfHitModifier(pos State, role control.State) float64
```

Pure functions; trivially testable.

### 3.2 Target-side modifier (bonus to anyone attacking a target in this position)

| Target Position | Controller-role | Controlled-role |
|---|---|---|
| Standing / Prone / Supine | 1.00 / — / — | — / 1.10 / 1.15 |
| Clinch (symmetric) | 1.08 | 1.08 |
| BackStanding | 1.10 | 1.15 |
| Mount | 1.12 | 1.20 |
| SideControl | 1.10 | 1.15 |
| KneeOnBelly | 1.10 | 1.13 |
| NorthSouth | 1.10 | 1.12 |
| Crucifix | 1.10 | 1.25 |
| BackGround | 1.12 | 1.22 |
| HalfGuard | 1.08 | 1.10 |
| Guard (inverted) | 1.08 (bottom) | 1.10 (top, exposed to legs) |
| Turtle | 1.10 | 1.12 |

### 3.3 Attacker-self modifier (your own position helps/hurts YOUR swings)

| Your Position | Controller-role | Controlled-role |
|---|---|---|
| Standing | 1.00 | — |
| Prone / Supine | — | 0.75 / 0.70 |
| Clinch (symmetric) | 1.00 | 1.00 |
| BackStanding | 1.05 | 0.85 |
| Mount | 1.10 (apex) | 0.70 |
| SideControl | 1.05 | 0.75 |
| KneeOnBelly | 1.05 | 0.80 |
| NorthSouth | 1.00 | 0.80 |
| Crucifix | 1.10 | 0.50 (arm trapped) |
| BackGround | 1.10 (free hands) | 0.65 |
| HalfGuard | 1.05 | 0.85 |
| Guard (inverted) | 1.05 (bottom uses legs) | 0.90 (top has frames) |
| Turtle | 1.00 | 0.85 |

### 3.4 Net effect (sample composition)

| Scenario | Math | Net | Plain English |
|---|---|---|---|
| Mount controller swinging at controlled | 1.10 × 1.20 | **1.32** | The bug-report fix: controller swings land much more |
| Mount controlled trying to hit controller back | 0.70 × 1.05 | **0.74** | Big penalty — barely able to swing out |
| Third-party (Standing) attacks mount-controlled | 1.00 × 1.20 | **1.20** | Easy bop |
| Third-party (Standing) attacks mount-controller | 1.00 × 1.12 | **1.12** | Easier than standing, but not free |
| Crucifix victim attacking back | 0.50 × 1.10 | **0.55** | Brutal — can hardly swing at all |
| Two equal Clinch fighters trading | 1.00 × 1.08 | **1.08** | Mild edge to whoever swings first |

Modifiers ship as code constants (not config) for v1. The tables are tunable surfaces for chunk 4f if smoke surfaces over- or under-tuning.

---

## 4. Offense Restrictions While Grappled

Two restrictions added; two already-handled paths confirmed:

### 4.1 Eat / Drink / Quaff potions — NEW

When a character is in any grapple state (`IsGrappling() == true`), reject the command with a flavor message: "Your hands are committed to the grapple — you can't reach for that." Applies to:

- `eat <food>` (existing usercommand)
- `drink <drink>` (existing usercommand)
- `quaff <potion>` (existing usercommand)

Hook into each command's pre-check. If the character is grappling, return early with the rejection message. Simple, single-point check.

### 4.2 Spell casting — INVESTIGATE LATER, NOT 4e SCOPE

Spell disruption already exists in the engine (chunk-3 Activity machine + various interrupt paths). Per user direction, leave it alone for chunk 4e. Logged as a future polish item: re-evaluate whether grapple-specific disruption (different from generic damage-interrupts-cast) is wanted.

### 4.3 Crafting / salvaging — CONFIRMED NO-OP

Already correctly blocked: the Activity machine forbids transitioning to Crafting/Salvaging once combat is entered. Grappling implies combat; no additional check needed.

### 4.4 Movement (flee, go) — CONFIRMED NO-OP

Already blocked by chunk-0 / chunk-1 / chunk-4a logic. The Position FSM doesn't permit `TransitionToStanding` while in a grapple except via the escape outcomes.

---

## 5. Outside-Damage → Control Degradation

### 5.1 Trigger

When the combat damage pipeline applies damage to a target, check:
1. Is the target currently a grapple controller? (`target.Control.State() == Controlling` AND `target.Position.IsGrappling()`)
2. Is the attacker NOT the target's grapple partner? (`attacker.ActorRef() != target.GrappleData().Partner`)

If both true, fire one Control state shift toward Neutral on the target. The shift uses the same machinery as the per-round drift roll's `applyControlShift` (chunk 4b-fixup-2 T9): 1-step shift, traverses transient state (LosingControl), fires gradient messaging via the registered boundary-cross callback.

### 5.2 Granularity — once per round, not per hit

Multiple hits in one round shouldn't multiply the shift (would compound too fast). Use a per-character round-marker:

- `Character.OutsideHitDisruptedRound int64` — round number of last outside-hit disruption
- If `OutsideHitDisruptedRound == util.GetRoundCount()`, skip (already disrupted this round)
- Otherwise apply the shift and mark the round

### 5.3 Magnitude

Each disrupted round = 1 step toward Neutral. Three outside-attack-rounds in a row turns a Controlling controller into Controlled (via LosingControl → Neutral → BecomingControlled → Controlled). Then escape becomes possible on the next defender-favorable drift roll.

### 5.4 Defender unaffected

The defender (controlled side) is already at the bottom of the gradient. Third-party damage doesn't help them escape mechanically; if they want out, they earn it through their own drift roll. Avoids the perverse incentive of "punch my pinned ally to help them."

### 5.5 New config knob

```yaml
Balance:
  ControlDegradeOnOutsideHit: true  # set false to disable for tuning
```

Default true. Disable in 4f if it turns out to be too punishing for grapplers fighting in groups.

---

## 6. Mob AI Bias Toward Grappled Targets

### 6.1 Tiebreaker-only model

Per user direction: AI bias does NOT override a clear primary preference. Only kicks in when the top candidates are at SIMILAR priority. Pseudocode:

```go
candidates := /* existing target picker — sorted by primary priority */
topPriority := candidates[0].priority
tieBand := topPriority * 0.10  // within 10% = "tied"

tied := candidates where priority >= (topPriority - tieBand)

if len(tied) > 1 {
    // Tiebreaker: prefer a grappled-controlled target.
    for _, t := range tied {
        if t.Control.State() == control.Controlled && t.Position.IsGrappling() {
            return t
        }
    }
}
return candidates[0]
```

### 6.2 What "priority" means

The existing mob target picker uses some scoring function (likely `aggro` accumulators, recent-damage memory, distance, etc.). T1 of the plan verifies the actual function and finds the right hook point — the tie-band tweak goes in there.

### 6.3 Universal across archetypes

For chunk 4e, every hostile archetype uses the same tiebreaker. Per-archetype tuning (predators 2×, tanks 1×, etc.) belongs in chunk 4f if needed.

### 6.4 No new config knob

The 10% tie band is a code constant for v1. If 4f finds it too tight (mobs never swap onto grappled targets) or too loose (mobs flip-flop), expose as a knob.

---

## 7. Submission Interrupt Risk

### 7.1 Trigger window

Chunk 4d's `Position_SubmissionTick` fires once per round. When a sub attempt is determined to fire (drift |z| ≥ 1.5 + position eligibility check passes), the submitter is "in flight" for that round.

If the submitter has taken disruptive third-party damage in the SAME round, the sub outcome is forced to **Bad** tier (the worst possible outcome — sub fails AND the submitter loses position).

### 7.2 Damage threshold

Two ways third-party damage qualifies as "disruptive":

1. **Crit damage** — any third-party hit that crit (combat z ≥ 2.0) interrupts.
2. **Above-threshold damage** — any third-party hit that did damage exceeding `SubInterruptDamageThresholdPct × submitter.HealthMax`. Default 0.10 (10%).

Either condition triggers the override. Small damage (a single jab) doesn't break the sub setup.

### 7.3 Implementation hook

Add a per-character per-round tracker:
- `Character.SubInterruptDamageThisRound float64` — accumulated qualifying damage from third parties this round
- Damage pipeline writes to it when conditions match (third-party attacker + crit or > threshold)
- Position_SubmissionTick reads it before resolving sub outcome; if > 0, overrides the outcome to Bad tier
- Reset to 0 at round-end (or just compare against round counter)

### 7.4 New config knob

```yaml
Balance:
  SubInterruptDamageThresholdPct: 0.10  # 10% of max HP triggers; 0 to disable
```

Default 0.10. Set to 0 to disable the threshold path (crit-only).

### 7.5 Why Bad tier vs Coin-flip

Forcing Bad is predictable and punishing — players know "don't try a sub while someone is hammering you." A coin-flip would feel random.

---

## 8. Cross-cutting: combat damage pipeline hook

The damage pipeline (likely `internal/combat/damage_pipeline.go`) is the single integration point for §5 (control degrade) and §7 (sub interrupt damage tracking). Both fire from "damage was applied to target" — after the existing damage application, check:

1. Is the attacker a third party (not the target's grapple partner)?
2. If yes:
   - For §5: was target a controller? If so, mark this round as a disrupted round on the target.
   - For §7: did the hit crit OR exceed `SubInterruptDamageThresholdPct × target.HealthMax`? If so, add to `target.SubInterruptDamageThisRound`.

Both writes are O(1), no perf concern. Single check + dispatch at one pipeline site.

---

## 9. What Survives Unchanged

| Artifact | Notes |
|---|---|
| Position FSM (chunks 4a, 4b-fixup, 4b-fixup-2) | Unchanged |
| ControlLevel FSM (chunk 4b-fixup-2) | Unchanged — new modifiers read State() but don't modify it (except §5's drift shift via existing applyControlShift) |
| Outcome resolver (chunk 4b-fixup) | Unchanged |
| Drift roll formula (grapple-drift-rework 2026-05-19) | Unchanged |
| Sub eligibility (chunk 4b-fixup-2 T14) | Unchanged |
| Sub outcome resolution (chunk 4d) | Output unchanged except for the new §7 force-Bad override path |
| All messaging (chunk 4b-fixup + chunk 4b-fixup-2) | Unchanged |
| Reach utility (chunk 4c) | Unchanged |

---

## 10. New Artifacts

| Path | Responsibility |
|---|---|
| `internal/state/position/modifiers.go` | Two lookup functions (TargetSideHitModifier, AttackerSelfHitModifier) + the tables |
| `internal/state/position/modifiers_test.go` | Unit tests for all (position, role) cells in both tables |
| `Character.OutsideHitDisruptedRound int64` field | Per-round dedupe for §5 |
| `Character.SubInterruptDamageThisRound float64` field | Per-round damage accumulator for §7 |
| Config: `ControlDegradeOnOutsideHit` bool | §5 panic button |
| Config: `SubInterruptDamageThresholdPct` float64 | §7 threshold knob |

---

## 11. Modified Files Preview

| Path | Change |
|---|---|
| `internal/combat/combat_helpers.go` or wherever the hit roll is computed | Multiply final hit modifier by the two position lookups |
| `internal/combat/damage_pipeline.go` | Fire §5 and §7 hooks after damage application |
| `internal/usercommands/eat.go`, `drink.go`, `quaff.go` (or `potion.go`) | Pre-check `IsGrappling()`, reject with flavor |
| `internal/mobcommands/<same set>` | Same restriction on mob side |
| `internal/mobs/target_picker.go` (or wherever target selection lives) | Add tiebreaker logic |
| `internal/hooks/Position_SubmissionTick.go` | Read `SubInterruptDamageThisRound`; override outcome if > 0 |
| `internal/characters/character.go` | Add two new round-tracking fields |
| `internal/configs/config.balance.go` | Add two new knobs |
| `_datafiles/config.yaml` | Set the two knobs to default values |
| `internal/state/position/context.md` | Document modifier tables |
| `internal/hooks/context.md` | Document the new damage-pipeline hooks |
| `internal/combat/context.md` | Document the hit-roll modifier integration |
| `_datafiles/world/dogmud/templates/help/grapple.template` | Add brief note about offense restrictions + outside hits (descriptive, no hard numbers) |
| `_datafiles/world/dogmud/templates/help/combat.template` (if it discusses to-hit) | Cross-reference |
| `COMBAT_STATE_ROADMAP.md` | Add chunk 4e row as Done |

---

## 12. Testing Strategy

### Unit tests

- `modifiers_test.go`: every (position, role) cell in both tables returns the documented value. Caught by table-driven test fixtures.
- Composition test: `final = AttackerSelf × TargetSide` for representative scenarios — Mount controller hitting controlled = 1.32 exactly; etc.
- Defaults: Standing × Standing = 1.00 × 1.00 = 1.00 (no behavior change outside grapples).

### Integration tests

- **§5 control degradation:** simulate a controller in Mount, fire third-party damage, verify Control state shifts to LosingControl (and through to Neutral after subsequent rounds).
- **§5 partner-doesn't-disrupt:** simulate the controller's own grapple partner damaging them (not a third party); verify no shift.
- **§5 once-per-round:** two third-party hits in same round = one shift, not two.
- **§7 sub interrupt by crit:** force a sub to fire, deliver a crit from a third party that round; verify outcome forced to Bad.
- **§7 small damage ignored:** sub fires, third party deals 1% damage; sub resolves normally.
- **§7 threshold respected:** sub fires, third party deals exactly threshold-1; resolves normally. Threshold+1 forces Bad.

### Smoke tests

- Boot smoke: server starts cleanly, new config knobs read correctly, no panics.
- AI feature-tester smoke (reuse chunk-4b-fixup-2 goal file): verify Mount-controller hit rate is now meaningfully higher (the bug you reported). Add a new feel-tester goal file for the third-party scenarios (let a mob group attack a mounted grappler and observe Control degradation + AI piling on).

---

## 13. Implementation Order

1. `modifiers.go` + unit tests (pure code, no integration).
2. Hook hit modifiers into the combat hit roll. Boot smoke + targeted manual test (mount a boar, verify hit rate jumped).
3. Eat/drink/quaff pre-checks (user + mob commands).
4. Config knobs + character fields.
5. Damage-pipeline hooks: §5 control degrade + §7 sub interrupt damage tracking.
6. Position_SubmissionTick sub-interrupt override.
7. Mob target picker tiebreaker.
8. context.md + helpfile updates.
9. Boot smoke + AI smoke.

Order is dependency-driven: §1 must precede §5/§7 hooks (which need the helper plumbing). Helpfile/docs at the end.

---

## 14. Out of Scope

- **Per-archetype AI bias tuning** — single universal tiebreaker only; per-archetype is 4f.
- **Per-mob modifier overrides** — no mob YAML can override the position modifier tables.
- **Position-specific damage bonuses on top of hit modifiers** — hit-only per user direction.
- **Spell disruption improvements** — already in game; revisit later.
- **Sub interrupt that DEGRADES the outcome tier instead of forcing Bad** — picked the simpler force-Bad model.
- **Outside damage helping the defender** — defender unaffected; avoids the "punch my pinned ally" exploit.
- **Crit defense from grappled state** — not a chunk 4e deliverable; crit logic stays as-is.

---

## 15. Risks / Open Questions

- **Modifier composition might compound past intended limits in extreme cases.** Crucifix attacker self 1.10 × target 1.25 = 1.38 — large but in line with the design intent. Could push past 1.40 if we accidentally combine with another bonus. Smoke catches it.
- **§5 once-per-round in PvP groups.** If two third parties hit the controller in one round, only one disruption fires (per the dedupe). Players might feel like the second hit "didn't count." Trade-off accepted — chunk 4f can revisit if it feels wrong.
- **§7 threshold tuning.** 10% max HP is a reasonable starting threshold but might be too low for tanky submitters (a small hit on a 500-HP submitter = 50 damage, easy to hit). Watch in smoke; adjust to 0.15 or higher if subs are too easily interrupted.
- **Mob AI tiebreaker scope.** If the existing mob picker doesn't return ranked candidates, the tiebreaker logic needs a different shape. T1 investigates the actual function before specing the implementation. Worst case: the picker is replaced wholesale, which would balloon the chunk.
- **Eat/drink/quaff edge cases.** Should the rejection also apply to throwables (grenades, flasks)? Probably yes, but outside scope. If a player wants to throw a flash-bang while pinned, that's a different design discussion.

---

## 16. Success Criteria

1. **Bug fix verified:** Mount controller's hit rate against the controlled is meaningfully higher than against a standing target (visible in AI smoke).
2. **Symmetry verified:** third-party Walker attacking a mounted-controlled gets the same 1.20 bonus a first-party controller gets.
3. **Offense restrictions verified:** `eat`/`drink`/`quaff` fail with the rejection message during a grapple; success after escape.
4. **§5 verified:** third-party damage on a mounted controller produces an "your control slips" gradient message (the existing chunk-4b-fixup-2 message firing via the new drift shift).
5. **§7 verified:** sub attempt that takes a crit from a third party produces a Bad-tier outcome; small damage doesn't.
6. **AI bias verified:** in a 3-vs-1 scenario with one ally grappled, mobs not previously focused on a specific target prefer the grappled ally.
7. **No regressions:** all chunk-4a-through-grapple-rework tests still pass; chunk 4b-fixup-2 smoke goals still pass.
8. **Helpfile reads cleanly:** no hard numbers, descriptive language only.
