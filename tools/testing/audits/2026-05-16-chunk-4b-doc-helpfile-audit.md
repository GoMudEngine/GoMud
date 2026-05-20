# Chunk 4b documentation + helpfile audit

Produced 2026-05-16 to feed Task 23 of the chunk-4b Position control-axis plan.

Scope: identify docs, helpfiles, and YAML lore mentioning concepts changed
(legacy `CombatPosition` enum, `PositionRoundsMin`, `GrappleControllerId`,
`ConditionGrappleController`, `IsGround*Position`, `IsGrapple*Position`)
or added (per-round control rolls, 14-state predicate API, new
`IsController` / `IsBeingControlled` / `GetPositionSpeedMultiplier`
helpers, three new Position observers, 16 btree position primitives,
Supine state distinction, gradient/transition/stamina messaging) by
chunk 4b (Position FSM cutover + control axis).

Survey date: 2026-05-16. Branch: `feature/mob-aliveness-1.3-crimes`,
commit after T21 S1-S5 sunset (legacy field/file deletes fully shipped,
including R4 Life-cascade pre-wire removal).

---

## Context.md files

### Updates needed (mechanic changed, references outdated)

- `internal/state/position/context.md` — Largest scope. **All of
  W1-W8, R1-R6 (including R4 pre-wire deletion), F1, and S1-S5 sunset
  have shipped.** UPDATE: rewrite Status section from "R4 deferred /
  S1-S5 blocked" to "fully shipped". Update Cascade Integration to
  remove "coexists with chunk-2 pre-wire" framing. Update Intentional
  Simplifications item 8 from "in progress" to shipped. Update Sunset
  Notes table to mark all items deleted. Update Persistence note to
  remove "identical to how CombatPosition behaves today". Update
  "What 4b brings" to remove the deferred-work list.

- `internal/characters/context.md` — UPDATE: remove "R4 deferred /
  S1-S5 blocked" and "remain in place until S1-S5 land" language.
  Replace with "fully removed (T21 sunset)". Update `IsController()`
  description from "sunset target S4" to "S4 shipped". Update
  `GetPositionSpeedMultiplier()` from "sunset S5" to "S5 shipped".
  Rename "Legacy enum coexistence" block to historical-reference table.

- `internal/hooks/context.md` — UPDATE: Life_Cascades Alive→Dead list
  still mentions "Resets CombatPosition to Standing" and "Clears
  GrappleControllerId" — both removed by R4 + T21. Delete those bullet
  points and replace with a note that R4 removed the pre-wire and the
  `position_life_dead` observer in `Position_Cascades.go` is now the
  sole position reset. Similarly update the Position_Cascades section
  to remove "coexists with chunk-2 pre-wire / R4 deferred" framing.
  Note: `RegisterPositionCheck` (line 523) already reads `c.IsStanding()`
  — no change needed there.

- `internal/combat/context.md` — UPDATE: remove "plus legacy
  parallel-write of CombatPosition / PositionRoundsMin" from the
  stand-command recovery section (T21 sunset). Remove "unmigrated
  readers of the legacy enum" paragraph (R-sweep complete). Update
  IsThirdPartyAttack note to say "deleted" fields rather than
  "legacy". Update Down state "legacy parallel" bullet.

- `internal/state/life/context.md` — UPDATE: Life_Cascades section
  still mentions "Resets CombatPosition to Standing (legacy parallel;
  R4 deferred)". Replace with a note that those lines were removed in
  R4 and the `position_life_dead` observer in `Position_Cascades.go`
  is now sole owner of the Position death cascade.

- `internal/spells/context.md` — UPDATE: knockdown entry still
  mentions "alongside the legacy `CombatPosition = PositionProne`
  parallel-write". Remove the legacy reference (T21 sunset). The
  canonical outcome is `TransitionToSupine` (face-up, Supine) via
  `TriggerKnockdownSpell`. The legacy write is gone.

### Keep as-is (keyword match but meaning unchanged)

- None — every chunk-4b-affected context.md entry surveyed requires
  some adjustment, either content or status-language.

### Net new files needed

- None. The Position FSM was scaffolded in chunk 4a;
  `internal/state/position/context.md` already exists and just needs
  extension for the 4b additions.

---

## Helpfiles

No `_datafiles/helpfiles/` directory found (confirms chunk-2 and
chunk-3 audit findings). User help is embedded in command handlers and
served via templates under `_datafiles/world/dogmud/templates/help/`.

### Help templates touching position / grapple concepts (27 files)

Grep on `grapple|prone|knockdown|standing` (case-insensitive) returned:
`chameleon-skin.template`, `submit.template`, `grapple.template`,
`unarmed-combat.template`, `tail.template`, `stomp.template`,
`spellcasting.template`, `set-prompt.template`, `kinetic-shove.template`,
`knee.template`, `kick.template`, `fold-recall.template`,
`defense.template`, `conjure-magma.template`, `conjure-earth.template`,
`combat.template`, `assess.template`, `trip.template`,
`strength.template`, `stand.template`, `stamina.template`,
`special.template`, `prone.template`, `cooldowns.template`,
`attack.template`, `bash.template`, `about.template`.

Sampled the most relevant: `prone.template`, `stand.template`,
`grapple.template`, `submit.template`, `bash.template`,
`set-prompt.template`. **None mention internal field names**
(`CombatPosition`, `GrappleControllerId`, etc.). All describe
player-facing mechanics in prose. Player-visible behavior is mostly
unchanged or strictly additive in 4b:

- `prone.template` / `stand.template`: describe "prone" as the
  knocked-down state. Post-4b, Supine is a sibling face-up knockdown
  with the same penalty profile in the legacy enum sense — prose still
  reads correctly, but does not yet teach the Prone-vs-Supine
  distinction or its submission/back-take consequences. Strict 4b
  accuracy: KEEP. 4f content enhancement: ADD Prone/Supine split
  description, recovery-curve nuance, and pull-guard option from
  Supine.
- `grapple.template`: 4-line generic description — "lock your opponent
  in a grappling hold." Doesn't describe the per-round control drift,
  threshold-triggered transitions, or the 11-state grapple taxonomy.
  Strict 4b accuracy: KEEP. 4f content enhancement: ADD per-round
  drift mechanics, control gradient ("in control → losing →
  neutral → becoming controlled → controlled"), submission gating
  (4d), and the 11-state position list.
- `submit.template`: describes "you must be in control of a grounded
  grapple" — semantically still correct under the FSM (`IsController()`
  + `IsGroundGrapple()`). KEEP.
- `bash.template`: "your target becomes prone for a minimum of 1 full
  round" — accurate (bash → Supine via FSM, but legacy enum maps to
  PositionProne and player-visible "prone" copy is the same). 4f could
  add the face-direction distinction.
- `set-prompt.template`: `{pos}` token doc reads "Your combat position
  (empty when standing)." Still accurate post-R6 — the FSM-driven
  implementation hides Standing and renders abbreviated 14-state names
  in tier colors. KEEP. 4f could add a sample listing of the 13
  non-Standing renderings (e.g. `Prone`, `Supine`, `Clinch`, `B.Std`,
  `Mount`, `SC`, `KOB`, `N-S`, `Crucifix`, `B.Gnd`, `H.Gd`, `Guard`,
  `Turtle`) for player reference.

**No 4b helpfile content changes required.** All 27 grapple/prone/
knockdown/standing-touching templates are factually correct in their
player-facing prose. The Supine distinction, per-round drift, and
14-state taxonomy are deferred to chunk 4f's "Helpfile + full doc
sweep" — explicitly listed in `position/context.md` line 483.

---

## Top-level docs

### Updates needed (stale forward-references or status changes)

- `COMBAT_STATE_ROADMAP.md` — Lines 268-270 (Chunk 2 deferred section):
  "Position machine (chunk 4) will repoint the Position pre-wire in
  `Life_Cascades.go` (currently clears `CombatPosition` /
  `GrappleControllerId` directly)." Status: **partially satisfied** —
  4a created the cascade observer (`Position_Cascades.go`) that
  coexists with the pre-wire; 4b R4 (delete the pre-wire) is **deferred**
  pending the broader reader sweep (see memory entry
  `project_chunk_4b_r4_blocked_on_reader_sweep.md`). Update line to
  "Position machine (chunk 4a) created a cascade observer that
  coexists with the legacy pre-wire; chunk 4b R4 (pre-wire deletion)
  is deferred until the broader CombatPosition reader sweep finishes."
  Lines 514-522 (Chunk 4a Deferred-to-4b list): all items listed as
  "deferred to 4b" need status pass — W1-W8, R1/R2/R3/R5/R6, F1 shipped;
  R4, S1-S5, broader reader sweep, and broader doc sweep are still
  outstanding. T25 will mark chunk 4b shipped and rewrite this section.

- `DEVELOPMENT_PLAN.md` — Lines 1295-1387 (Stage 8.1/8.2 design),
  Lines 2435/2461 (`ConditionGrappleController` references). These are
  **historical records** of the pre-chunk-4 legacy design. Following
  the chunk-3 audit pattern: KEEP as-is. If maintaining as a living
  doc, could be marked "Stage 8 design (superseded by chunk 4 Position
  FSM)" but not strictly needed.

### Keep as-is (historical or unaffected)

- `PATCH_NOTES.md` — Surveyed for CombatPosition/GrappleControllerId
  references; none found. KEEP.

- `docs/superpowers/specs/2026-05-16-state-chunk-4b-position-control-axis-design.md`
  — Chunk 4b's own design spec. Self-reference. KEEP.

- `docs/superpowers/plans/2026-05-16-state-chunk-4b-position-control-axis.md`
  — Chunk 4b's own plan. Self-reference. KEEP.

- `docs/superpowers/specs/2026-05-16-state-chunk-4a-position-fsm-design.md`
  + `docs/superpowers/plans/2026-05-16-state-chunk-4a-position-fsm.md`
  — Chunk 4a's spec/plan. Historical record. KEEP.

- `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md`
  — Master spec for the six-machine vision. Lines 7, 24, 326-328,
  411, 745-747 reference CombatPosition / PositionRoundsMin /
  GrappleControllerId / ConditionGrappleController as items being
  replaced by the Position machine. All references describe the
  intentional sunset path — accurate. KEEP.

- `docs/superpowers/plans/2026-05-15-state-chunk-2-life.md` +
  `docs/superpowers/specs/2026-05-15-state-chunk-2-life-design.md`
  — Chunk-2 plan/spec referencing the Position pre-wire as an
  in-cascade reset. Historical record. KEEP.

- `docs/superpowers/plans/2026-05-13-state-chunk-0-framework-and-combat-phase.md`
  — Chunk-0 plan referencing the original `RegisterPositionCheck`
  closure. Historical. KEEP.

- 4 completed plan files
  (`2026-04-21-tank-and-generic-archetypes-{plan,design}.md`,
  `2026-04-21-command-readiness-drift-plan.md`,
  `2026-04-17-code-cleanup-1.2a-combat-spell-refactor.md`,
  `2026-04-07-mob-ai-framework.md`,
  `2026-04-02-command-unification-substage3.md`) — historical records;
  references are accurate at writing time. KEEP.

---

## YAML lore / descriptions

### Keep as-is (no implementation references found)

- Grep across `_datafiles/` for
  `CombatPosition|GrappleControllerId|ConditionGrappleController|PositionRoundsMin`
  returned **zero matches**. No NPC dialogue, room descriptions,
  quest lore, or behavior YAML mentions internal position field names.
  Player-facing prose in dialogue/quest text is unaffected.

- `_datafiles/config.yaml` — Already contains in-line documentation for
  the chunk-4b Balance knobs (lines 563-574:
  `GrappleStaminaPenaltyMax`, `GrappleStaminaPenaltyCurve`,
  `GrappleEncumbrancePenaltyMax`, `GrappleEncumbrancePenaltyCurve`,
  `GrappleStaminaCostPerRound`, `GrappleControllerCostMultiplier`,
  `GrappleControlledCostMultiplier`, `GrappleStaminaLowThreshold`).
  Self-contained reference. KEEP.

**No YAML lore changes needed.**

---

## Schema docs

### Keep as-is

- `docs/schemas/mob.md` — Lines 46, 67, 195, 198, 200, 223: mentions of
  `grapple` / `grappler` / `target_grappled` / `target_prone` as
  archetype names and trigger keywords. These are persistent feature
  identifiers, not implementation references. KEEP.

- `docs/schemas/item.md` — Grep match on `grapple` was a stat-mod
  feature reference. KEEP.

**No schema doc changes needed.**

---

## Templates (non-help)

Searched `_datafiles/world/default/templates/` and
`_datafiles/world/dogmud/templates/` for CombatPosition /
GrappleControllerId references — none found.

**No template changes needed.**

---

## Tools / testing fixtures

Searched `tools/testing/` for CombatPosition / GrappleControllerId
references — none found.

Test fixtures themselves (under `internal/**/*_test.go`) were migrated
in chunk-4b T20 F1 (commits `b4815fe3`, `858492ff`, `2ba0d8bd`). All
`c.CombatPosition = characters.PositionX` direct writes have been
replaced with the `setCombatPositionParallel` helper that lockstep-
writes both the legacy field and the FSM.

**No test-fixture doc updates needed.**

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Context.md files | 6 hits | 6 need updates (all listed above) |
| Context.md additions | 1 | behaviortree/context.md needs new position-primitives section |
| Helpfiles | 27 templates | 0 strict 4b updates needed; 4-5 deferred to 4f for content enhancement |
| Top-level docs | 2 hits | 1 needs status update (COMBAT_STATE_ROADMAP); 1 KEEP historical (DEVELOPMENT_PLAN) |
| YAML lore | 0 | KEEP — no internal field references in data |
| Schema docs | 2 hits | KEEP — feature identifiers, not implementation refs |
| Templates (non-help) | 0 | KEEP |
| Test fixtures | 9 files | Already migrated in T20 F1; no doc changes |
| Config docs | 1 (config.yaml in-line) | KEEP — self-contained |

**Files requiring edits by T23:**

1. `internal/state/position/context.md` — Largest scope. Status rewrite
   for the 4a "DORMANT" framing → "post-4b cutover with R4/S1-S5
   deferred." Add control-axis API section (4 new helpers + 3
   predicates), per-round messaging contract section, three new
   observers section, expanded btree primitives table (16 total),
   updated Sunset Notes table with status columns, updated Intentional
   Simplifications status.

2. `internal/characters/context.md` — Update coexistence language (line
   656-659) to "reader sweep in progress." Add 4b predicates section:
   `IsController`, `IsBeingControlled`, `IsLowGrappleStamina`,
   `GetPositionSpeedMultiplier`. Cross-reference users-package prompt
   helpers (`positionPromptColor`, `positionPromptAbbrev`).

3. `internal/hooks/context.md` — DELETE "Resets CombatPosition to
   Standing" and "Clears GrappleControllerId" bullets from Life cascade
   list (R4 removed them). UPDATE Position_Cascades section to remove
   "coexists with chunk-2 pre-wire / R4 deferred" framing.
   (`RegisterPositionCheck` already reads `c.IsStanding()` — no change.)

4. `internal/combat/context.md` — DELETE "plus legacy parallel-write
   of CombatPosition / PositionRoundsMin" from stand-command section.
   DELETE "unmigrated readers of the legacy enum" paragraph. UPDATE
   IsThirdPartyAttack note to say "deleted" fields. UPDATE Down state
   "legacy parallel" bullet to reflect T21 sunset.

5. `internal/state/life/context.md` — DELETE "Resets CombatPosition
   to Standing (legacy parallel; R4 deferred)" and "Clears grapple
   controller (same legacy-parallel + R4-deferred status)". Replace
   with note that R4 deleted those lines.

6. `internal/spells/context.md` — UPDATE knockdown entry: remove
   "alongside the legacy `CombatPosition = PositionProne` parallel-
   write". The FSM `TransitionToSupine` is now the sole write.

7. `internal/behaviortree/context.md` — UPDATE: `mob_is_in_control`
   entry says "sunset target S4" — replace with "S4 shipped". Update
   the preamble from "never the legacy CombatPosition enum" to note the
   legacy enum is now deleted (T21). The 16-primitive table is already
   present and accurate; only the status language needs correction.

8. `COMBAT_STATE_ROADMAP.md` — Status pass on lines 268-270 (chunk-2
   forward-reference now partially satisfied), lines 514-522 (chunk-4a
   "Deferred to 4b" list). Detailed "Chunk 4b — Shipped" section is
   **T25 scope**, not T23.

**Notes for T25 (out of T23 scope):**

- Mark chunk 4b fully shipped in `COMBAT_STATE_ROADMAP.md` and
  `MOB_ALIVENESS_ROADMAP.md` (all R4, S1-S5, and reader sweep work
  landed before T23; helpfile sweep remains 4f scope).
- Update `MEMORY.md` followups index to remove the R4 reader sweep
  entry (shipped) and note that the next chunk is 4c.
- `DEVELOPMENT_PLAN.md` Stage 8 historical entries can be marked
  "superseded by chunk 4b Position FSM" if maintaining as a living
  doc; not strictly required.

---

## Surprising findings

1. **No user helpfiles directory.** Confirms chunk-2 and chunk-3 audit
   findings: DOGMud does not use `_datafiles/helpfiles/`. Help is
   embedded in command handlers and templates.

2. **All 27 grapple/prone/knockdown/standing help templates are
   factually correct post-4b** — they describe mechanics in
   player-facing prose without citing internal field names. The
   Supine distinction, per-round control drift, and 14-state taxonomy
   are intentionally deferred to chunk 4f's "Helpfile + full doc
   sweep" — listed in `position/context.md` line 483.

3. **Zero YAML lore references** to CombatPosition / GrappleControllerId /
   ConditionGrappleController / PositionRoundsMin. Behavior trees,
   archetypes, mob YAML, quest dialogue, room descriptions are all
   shielded from implementation churn — they reference high-level
   feature names (e.g. `target_prone` trigger, `grappler` archetype)
   that survive the rename.

4. **`Position_Cascades.go` observer is now the sole Position death
   cascade.** R4 shipped (`a481797f`) — the chunk-2 `Life_Cascades.go`
   pre-wire that reset `c.CombatPosition` and `c.GrappleControllerId`
   is deleted. T21 removed the legacy fields entirely. All three docs
   (`hooks/context.md`, `state/life/context.md`, `state/position/
   context.md`) still contain "coexists with chunk-2 pre-wire / R4
   deferred" language — that is what T23 corrects.

5. **Btree primitive surface tripled in 4b** (10 chunk-4a per-state
   primitives + 6 chunk-4b control-axis primitives), but
   `internal/behaviortree/context.md` has zero position references.
   This is the single largest new doc surface in T23 — a fresh table
   mapping each primitive name to its underlying Character predicate
   is needed.

6. **W5 spell-knockdown cutover changed the default knockdown direction
   from Prone to Supine.** The `spell_resolution.go` writer fires
   `TransitionToSupine` (face-up, "slams to the ground") not
   `TransitionToProne` (face-down). `spells/context.md` line 105 still
   describes the legacy `PositionProne` outcome. Update needed to
   teach the Supine default and note the future T22 revisit if spells
   gain a direction config.

7. **R6 prompt token migration added two new helper functions**
   (`positionPromptColor`, `positionPromptAbbrev`) inside the `users`
   package. These have no context.md home in `internal/users/` (no
   such file exists). Cross-reference from `internal/characters/context.md`
   (the predicate doc) is the natural place to mention them so future
   contributors can find them.

8. **Config knobs are self-documenting.** All 8 new chunk-4b Balance
   knobs (lines 563-574 of `_datafiles/config.yaml`) ship with in-line
   comments. No separate config doc file needs updating.

---

## Post-audit notes for T23 execution

When updating context.md files, use present-tense "fully shipped" voice
(T21 + R4 both landed before T23 runs):

- **Before (stale):** "coexists with the legacy `CombatPosition` enum
  and its helpers; R4 deferred; S1-S5 blocked"
- **After (current):** "The legacy `CombatPosition` enum and all
  supporting fields are deleted (T21 sunset). The Position FSM is the
  sole source of truth."

For the `Position_Cascades.go` section across three files
(hooks/, state/life/, state/position/):

- "The chunk-4a `position_life_dead` observer (in
  `internal/hooks/Position_Cascades.go`) is the sole Position reset on
  death. The chunk-2 `Life_Cascades.go` pre-wire that reset legacy
  `CombatPosition` + `GrappleControllerId` was deleted in R4. Those
  fields no longer exist (T21 sunset)."

For `internal/behaviortree/context.md` (net new section), model it
after the existing structure (overview + table + integration notes):

```
### Position primitives (chunks 4a + 4b)

**Per-state predicates (10 — chunk 4a):**
| Primitive | Underlying check |
|-----------|------------------|
| `mob_is_standing`      | `c.IsStanding()` |
| `mob_is_prone`         | `c.IsProne()` |
| `mob_is_supine`        | `c.IsSupine()` |
| `mob_in_grapple`       | `c.IsGrappling()` |
| `mob_in_standing_grapple`  | `c.IsStandingGrapple()` |
| `mob_in_ground_grapple`    | `c.IsGroundGrapple()` |
| `mob_in_top_dominant`  | `c.IsTopDominant()` |
| `mob_on_floor`         | `c.IsOnFloor()` |
| `target_is_prone`      | aggro target IsProne |
| `target_in_grapple`    | aggro target IsGrappling |

**Control-axis primitives (6 — chunk 4b):**
| Primitive | Underlying check |
|-----------|------------------|
| `mob_is_in_control`            | `c.IsController()` |
| `mob_is_being_controlled`      | `c.IsBeingControlled()` |
| `target_is_in_control`         | target IsController |
| `target_is_being_controlled`   | target IsBeingControlled |
| `mob_low_grapple_stamina`      | `c.IsLowGrappleStamina()` |
| `target_low_grapple_stamina`   | target IsLowGrappleStamina |
```

(Verify primitive names against `internal/behaviortree/conditions_position.go`
before committing — names above are the working set; actual
identifiers may differ in case/underscore convention.)

For `position/context.md` Sunset Notes table — all items now fully
shipped (update "Status" column accordingly):

| Legacy item | Status |
|-------------|--------|
| `CombatPosition` enum | **Deleted** — T21 S5 (reader sweep complete) |
| `PositionRoundsMin` field | **Deleted** — T21 S2 |
| `GrappleControllerId` field | **Deleted** — T21 S3 |
| `ConditionGrappleController` | **Deleted** — T21 S4 |
| `Life_Cascades.go` Position pre-wire | **Deleted** — R4 (`a481797f`) |
| `internal/characters/combatposition.go` | **Deleted** — T21 |
| Legacy `AttemptRecovery()` path | **Shipped** W6 |
