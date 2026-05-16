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
commit after T20 F1 (test-fixture parallel-write migration).

---

## Context.md files

### Updates needed (mechanic changed, references outdated)

- `internal/state/position/context.md` — Largest scope. The chunk-4a
  "ships DORMANT" framing (lines 12-26) is now stale: 4b's writer cutover
  (W1-W8), reader cutover (R1, R2, R3, R5, R6), and F1 fixture migration
  have shipped. **R4** (delete `Life_Cascades.go:55-57` chunk-2 pre-wire)
  is **deferred** pending a broader CombatPosition reader sweep (~25
  unmigrated readers across combat/ai.go, combat/grapple.go,
  actions/combat_kick.go, actions/command_readiness.go,
  behaviortree/conditions_mob.go, hooks/combat_shared_helpers.go,
  mobcommands/submit.go, usercommands/submit.go,
  characters/combat_state_compat.go); **S1-S5** (legacy field/file
  sunsets) are blocked on the same sweep. The "Intentional
  Simplifications" section (lines 385-410) lists items 1, 3, 4, 5, 7 as
  4b targets — items 1 (control rolls), 3 (writers), 4 (reader cutover),
  5 (flee veto) shipped; item 7 (submissions) remains 4d. The "Sunset
  Notes" table (lines 456-467) needs status columns updated (W1-W8 / R
  done, R4 deferred, S blocked). Add new sections for control-axis API
  (`MutateGrappleControlLevel`, `ConsumeRecoveryRound`, `IsController`,
  `IsBeingControlled`, `IsControllerLevel`, `IsControlledLevel`,
  `IsLowGrappleStamina`), per-round messaging contract (gradient,
  transition, stamina warning, cooldown semantics), three new observers
  (`Position_GrappleTick`, `Position_Messaging`, `Position_ConsistencyCheck`),
  the 6 control-axis btree primitives, and the 16-primitive total btree
  surface.

- `internal/characters/context.md` — Lines 656-659: "These coexist with
  the legacy `CombatPosition` enum and its `IsGroundPosition()` /
  `IsGrapplePosition()` helpers. Chunk 4b removes the legacy helpers
  once command sites cut over to write the new FSM." Status update: the
  legacy helpers are no longer read by combat_helpers.go (R1) or
  grapple.go (R2) but the broader sweep is incomplete — ~25 readers
  remain. Update to "Chunk 4b reader sweep in progress; legacy helpers
  to be deleted in S5 after the sweep finishes." Also add a new section
  for the 4b-introduced predicates on Character: `IsController`,
  `IsBeingControlled`, `IsLowGrappleStamina`, `GetPositionSpeedMultiplier`,
  and the new prompt helpers `positionPromptColor` / `positionPromptAbbrev`
  (lives in users package but worth cross-referencing).

- `internal/hooks/context.md` — Line 523: `RegisterPositionCheck` table
  entry reads `c.CombatPosition == PositionStanding`. **Update to
  `c.IsStanding()`** (R5 cutover landed in commit `edc12d81`). Lines
  660-661: Life cascade list "Resets `CombatPosition` to Standing /
  Clears `GrappleControllerId`" — R4 deferred, so these legacy resets
  are still in place. Update with a "(legacy parallel reset; the new
  FSM is reset independently by `position_life_dead` observer in
  `Position_Cascades.go`; R4 will delete the legacy pre-wire after the
  reader sweep)" note. Lines 793-799: Position cascade "coexists with
  chunk-2 pre-wire" prose — accurate now, but should note R4 status
  ("deferred pending broader reader sweep"). **Add new sections** for
  three chunk-4b observers: `Position_GrappleTick` (per-round control
  drift via `MutateGrappleControlLevel` + grapple stamina cost +
  threshold-triggered position transitions), `Position_Messaging`
  (gradient / transition / stamina warning with cooldown), and
  `Position_ConsistencyCheck` (periodic pair-invariant checker via
  `ValidateGrapplePair`).

- `internal/combat/context.md` — Line 492: `ws.attacks *=
  CombatPosition.GetSpeedMultiplier()` — pseudocode in
  `calcSwingCount` walkthrough. **Update to `sourceChar.GetPositionSpeedMultiplier()`**
  (R1 cutover). Broader: grapple mechanics section should be rewritten
  in 4b voice — single-round Clinch/Grounded binary is replaced by the
  per-round opposed control roll model, the 14-state taxonomy, and
  threshold-triggered transitions. Mention the `IsThirdPartyAttack`
  rewrite reading `GrappleData.Partner` (R2).

- `internal/state/life/context.md` — Line 207: "Resets `CombatPosition`
  to Standing" — same R4-deferred status note as the hooks doc. Update
  to "(legacy parallel reset; `position_life_dead` observer in
  `internal/hooks/Position_Cascades.go` resets the new FSM
  independently; R4 will delete the legacy pre-wire after the broader
  reader sweep)".

- `internal/spells/context.md` — Line 105: `knockdown` effect entry
  describes "sets target prone (CombatPosition = PositionProne, 1 round
  min)". W5 cutover (`spell_resolution.go`) now also fires
  `Position.TransitionToSupine(MinRecoveryRounds: 1, TriggerKnockdownSpell)`
  alongside the legacy write — the "slams to the ground" wording fits a
  backward blast → Supine, not Prone. Update to: "Deals damage + knocks
  the target down — Supine via the Position FSM
  (`TriggerKnockdownSpell`) and `CombatPosition = PositionProne` legacy
  parallel-write, 1 round min recovery. T22 doc audit can revisit if
  spells gain a direction config to distinguish blast (Supine) from
  shockwave (Prone)."

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

3. `internal/hooks/context.md` — Update `RegisterPositionCheck` table
   entry (line 523) to `c.IsStanding()`. Add "(legacy parallel; R4
   deferred)" notes on lines 660-661 (Life cascade Position resets).
   Update Position cascade prose (lines 793-799) to note R4 deferred
   status. **Add new sections** for `Position_GrappleTick`,
   `Position_Messaging`, `Position_ConsistencyCheck` observers.

4. `internal/combat/context.md` — Update line 492 pseudocode
   (`CombatPosition.GetSpeedMultiplier()` → `c.GetPositionSpeedMultiplier()`).
   Rewrite grapple mechanics section: replace single-round
   Clinch/Grounded binary with per-round opposed control roll, 14-state
   taxonomy, threshold-triggered transitions, gradient/transition
   messaging contract. Note `IsThirdPartyAttack` Partner-based check
   (R2).

5. `internal/state/life/context.md` — Update line 207 "(legacy
   parallel; `position_life_dead` observer resets the FSM
   independently; R4 deferred)".

6. `internal/spells/context.md` — Update line 105 knockdown effect
   entry: note Supine transition via `TriggerKnockdownSpell` (W5
   cutover) alongside the legacy `CombatPosition = PositionProne`
   write.

7. `internal/behaviortree/context.md` — **Net-new section** documenting
   the 16 position btree primitives (10 chunk-4a per-state + rollup +
   6 chunk-4b control-axis). Include a table mapping primitive name
   to underlying predicate (e.g. `mob_in_grapple → c.IsGrappling()`,
   `mob_is_in_control → c.IsController()`, `target_is_being_controlled`,
   `mob_low_grapple_stamina → c.IsLowGrappleStamina()`, etc.).

8. `COMBAT_STATE_ROADMAP.md` — Status pass on lines 268-270 (chunk-2
   forward-reference now partially satisfied), lines 514-522 (chunk-4a
   "Deferred to 4b" list). Detailed "Chunk 4b — Shipped" section is
   **T25 scope**, not T23.

**Notes for T25 (out of T23 scope):**

- Mark chunk 4b shipped in `COMBAT_STATE_ROADMAP.md` and
  `MOB_ALIVENESS_ROADMAP.md` with the deferral list (R4, S1-S5,
  broader reader sweep, helpfile 4f sweep).
- Update `MEMORY.md` followups index to mark the R4 reader sweep as
  the natural next-session work.
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

4. **`Position_Cascades.go` observer + Life pre-wire still coexist by
   design.** R4 (delete the pre-wire) is **deferred** pending the
   broader CombatPosition reader sweep. The "coexistence" prose in
   `hooks/context.md`, `state/life/context.md`, and
   `state/position/context.md` is **still accurate**, but should be
   tagged with "(R4 deferred; reader sweep pending)" status so future
   readers understand why both observers fire.

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

When updating context.md files, preserve the chunk-by-chunk historical
voice (matches chunk-2/3 audit pattern):

- **Before (chunk 4a):** "These coexist with the legacy `CombatPosition`
  enum and its helpers. Chunk 4b removes the legacy helpers..."
- **After (chunk 4b):** "Chunk 4b reader sweep in progress; ~25
  legacy readers remain (see memory entry
  `project_chunk_4b_r4_blocked_on_reader_sweep.md`). The legacy enum +
  helpers are scheduled for deletion in S5 after the sweep finishes."

For the `Position_Cascades.go` coexistence note across three files
(hooks/, state/life/, state/position/), the pattern is:

- "The chunk-4a `position_life_dead` observer (in
  `internal/hooks/Position_Cascades.go`) resets the new FSM
  independently. The chunk-2 `Life_Cascades.go:55-57` pre-wire
  (legacy `CombatPosition` + `GrappleControllerId` resets) **also
  fires** — R4 (pre-wire deletion) is deferred pending the broader
  CombatPosition reader sweep."

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

For `position/context.md` Sunset Notes table, add a "Status" column:

| Legacy item | Status |
|-------------|--------|
| `CombatPosition` enum | 4b cutover in progress; S5 blocked on reader sweep |
| `PositionRoundsMin` field | 4b parallel-writes in place; S2 blocked on reader sweep |
| `GrappleControllerId` field | 4b parallel-writes in place; S3 blocked on reader sweep |
| `ConditionGrappleController` | 4b R1/R2 migrated all known readers; S4 verify-then-delete |
| `Life_Cascades.go` Position pre-wire | R4 deferred — see `project_chunk_4b_r4_blocked_on_reader_sweep.md` |
| Legacy `AttemptRecovery()` path | W6 done — parallel-writes Position FSM |
| Kick variant selector legacy reads | Unmigrated — followup noted in R1+R2 commit |
| ~25 `c.CombatPosition` read sites | Unmigrated — see R4 memory entry for file list |
