# Chunk 3 documentation + helpfile audit

Produced 2026-05-15 to feed Task 13 of the chunk-3 Activity machine plan.

Scope: identify docs, helpfiles, and YAML lore mentioning concepts
changed (CastingState/CraftingState → Activity machine, salvage hijack
cleanup, IsActing gate, per-activity interrupt policy) or added
(Activity machine, cancel_activity btree primitive, mob cancel parity)
by chunk 3 (Activity state machine).

Survey date: 2026-05-15. Branch: feature/mob-aliveness-1.3-crimes, commit after T11 sunset.

---

## Context.md files

### Updates needed (mechanic changed, references outdated)

- `internal/combat/context.md` — Lines 370, 619: references to `CastingState` field. Section 2c "Fold Casting Check" describes legacy behavior reading `CastingState` directly. Update to describe Activity machine query instead. Section 4b "Fold Casting Check" (mob side) has same issue. Both should reference `Character.IsCasting()` predicate, which now queries the Activity machine internally.

- `internal/hooks/context.md` — Line 522: `RegisterActivityCheck` table entry states the check reads `c.CastingState == nil && c.CraftingState == nil`. Update to describe it reading `c.IsActing()` (negated). Lines 609, 658-659: Similar references in Awareness_Vetoes and Life_Cascades sections. Line 658 notes "Nils `CastingState` and `CraftingState`" in the Alive→Dead cascade; update to describe the Activity machine transitioning to Free state instead.

- `internal/state/awareness/context.md` — Lines 117-118: RegisterActivityCheck section describes closure reading `CastingState` and `CraftingState` fields directly. Update to describe reading Activity machine state via `Character.IsActing()`. Lines 253-254: Dependencies section lists `CastingState` and `CraftingState` as things the package reads. Update to say Activity machine (`Character.Activity`) instead.

- `internal/state/life/context.md` — Line 206: Alive→Dead cascade section states "Nils `CastingState` and `CraftingState`". Update to describe the cascade transitioning Activity machine to Free state.

### Keep as-is (keyword match but meaning unchanged)

- `internal/items/context.md` — Full scan performed; no CastingState/CraftingState/Activity references found. Content safe.

### Net new files needed

- `internal/state/activity/context.md` — Chunk 3's new state machine package needs its own context.md documenting the machine types (Free, Casting, Crafting, Salvaging), transitions, predicates, and integration with Character/hooks/btree. T13 will create this based on the chunk-3 implementation.

---

## Helpfiles

No helpfiles directory found at `_datafiles/helpfiles/`. User help is embedded in command handlers and templates (e.g., `_datafiles/world/dogmud/templates/help/craft.template`).

### Help templates searched for chunk-3 concepts

- `_datafiles/world/dogmud/templates/help/craft.template` — Lines 55-58: describes crafting being interrupted ("If attacked mid-craft, your work is interrupted"). This is a feature description, not an implementation detail, and remains accurate post-chunk-3. No change needed.

- Config file `_datafiles/config.yaml` — Line ~: comment "Concentration: % chance to maintain casting when struck mid-cast." This is a feature description, accurate post-chunk-3. No change needed.

**No helpfiles to update.**

---

## Top-level docs

### Updates needed (stale forward-references or mechanic changes)

- `COMBAT_STATE_ROADMAP.md` — Lines 138, 260-262, 283: References to Activity machine. Line 138: "the callback to the proper Activity machine." Line 260-262: "Activity machine (chunk 3) will repoint the Activity pre-wire in `Life_Cascades.go` (currently clears `CastingState`/`CraftingState` directly) to the proper Activity machine query." These are forward-references that are now satisfied; update to mark them as completed or remove the forward-reference language. Line 283: "Next: chunk 3 — Activity machine..." This is historical. Update to mark activity machine as done and point to the next chunk.

- `DEVELOPMENT_PLAN.md` — Lines ~2738-2750: Describes CastingState struct definition and implementation. Lines ~2750-2759: "Create `internal/combat/casting.go` — CastingState struct..." and "Add `CastingState *CastingState` to Character struct". These are historical records of stage 1 design (pre-chunk-3 Activity machine). Lines ~2959: "Mob fold casting — mobs use CastingState instead of legacy SpellCast/onMagic path." Line ~3149: "Add CraftingState (similar to CastingState)". Line ~6741: "clear Aggro/CastingState". All are historical records of stage-1 design; no changes strictly needed, but if DEVELOPMENT_PLAN is maintained as a living doc, these should be marked as "Stage 1 design (superseded by chunk-3 Activity machine)".

### Keep as-is (historical or unaffected)

- `PATCH_NOTES.md` — Line 326: "set `CastingState` and froze forever." This is a historical bug report (fold-recall in stage 1). Kept as historical record. Accurate context shows this was fixed by direct teleport; current Activity machine does not exhibit this bug by design.

- `docs/superpowers/specs/2026-05-15-state-chunk-3-activity-design.md` — Chunk 3 design spec. References Activity machine as the solution being implemented. Accurate and intentional. Kept.

- `docs/superpowers/plans/2026-05-15-state-chunk-3-activity.md` — Chunk 3 implementation plan. References CastingState/CraftingState as things being replaced by Activity machine. Accurate historical record of the work. Kept.

---

## YAML lore / descriptions

### Keep as-is (feature descriptions, accurate)

- `_datafiles/world/dogmud/templates/help/craft.template` — Line 55-58: "Crafting takes several rounds... If attacked mid-craft, your work is interrupted." Feature description, accurate post-chunk-3. Content safe.

- No NPC dialogue, room descriptions, or quest lore found mentioning CastingState/CraftingState or the old hijack pattern (`salvage:<itemid>`). No changes needed.

---

## Templates

Searched `_datafiles/world/default/templates/` for CastingState/CraftingState references — none found.

**No template changes needed.**

---

## Tools / testing fixtures

Searched `tools/testing/goals/`, `tools/testing/roles/`, and `tools/` broadly for CastingState/CraftingState references — none found.

**No test fixture updates needed.**

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Context.md files | 4 hits | 4 need updates |
| Helpfiles | 0 | N/A (no helpfiles dir) |
| Top-level docs | 2 hits | 1 needs update (COMBAT_STATE_ROADMAP), 1 for awareness (DEVELOPMENT_PLAN historical note) |
| YAML lore | 1 | Keep as-is (feature description) |
| Templates | 30+ | 0 updates needed |
| Test fixtures | N/A | 0 updates needed |

**Files requiring edits by T13:**

1. `internal/combat/context.md` — Update lines 370, 619: Replace `CastingState` references with `Character.IsCasting()` predicate description.
2. `internal/hooks/context.md` — Update lines 522, 609, 658-659: Replace `CastingState == nil && CraftingState == nil` references with Activity machine equivalents.
3. `internal/state/awareness/context.md` — Update lines 117-118, 253-254: Replace field-read references with Activity machine predicates.
4. `internal/state/life/context.md` — Update line 206: Replace field-nilling description with Activity machine transition description.
5. `internal/state/activity/context.md` — **Create new** (document Activity machine types, transitions, predicates, integration).
6. `COMBAT_STATE_ROADMAP.md` — Update lines 138, 260-262, 283: Mark forward-references as satisfied or remove; update next-chunk pointer.

**Notes for T15 (out of T13 scope):**

- Mark chunk 3 Done in `COMBAT_STATE_ROADMAP.md` and `MOB_ALIVENESS_ROADMAP.md`.
- Update `DEVELOPMENT_PLAN.md` with historical markers if maintaining as a living doc (e.g., "Stage 1 design, superseded by chunk-3 Activity machine").

---

## Surprising findings

1. **No user helpfiles directory.** Confirms chunk-2 finding: DOGMud does not use `_datafiles/helpfiles/`. Help is embedded in command handlers and served via templates.

2. **Feature descriptions (mid-cast/mid-craft) are accurate post-chunk-3.** The Activity machine preserves the user-facing concept of "in-flight activities that can be interrupted." No prose changes needed.

3. **Activity machine is the sole source of truth going forward.** Post-chunk-3, predicates like `Character.IsCasting()` / `Character.IsCrafting()` / `Character.IsActing()` query the Activity machine internally. Docs should consistently reference these predicates rather than raw field access.

4. **CastingState/CraftingState struct definitions remain in DEVELOPMENT_PLAN as historical.** They are marked as stage-1 work; keeping them for historical context is reasonable. If pruning old designs, they can be removed, but T13 can leave as-is.

5. **Salvage hijack (`MiscData["salvage_item_uuid"]`, `RecipeId = "salvage:<itemid>"`) has no doc references.** The hijack was internal implementation detail; no docs describe it. Cleanup in T11 required no doc updates.

---

## Post-audit notes for T13 execution

When updating context.md files, use this pattern:

- **Before (chunk 2):** "reads `c.CastingState == nil && c.CraftingState == nil`"
- **After (chunk 3):** "reads `c.IsActing()` (negated) to check if the character is free"

This preserves the semantic meaning (activity check) while pointing to the current API (Activity machine predicates).

For `internal/state/activity/context.md`, model it after the existing `internal/state/awareness/context.md` or `internal/state/life/context.md` structure:
- Overview (machine states, transitions)
- Key Components / Files
- Key Functions (construction, predicates, transitions, cascades)
- Integration (btree events, Character field, hooks)
- Dependencies
- Sunrise notes (what chunk-3 adds)
- Sunset notes (what chunk-3 removes: CastingState/CraftingState struct files, salvage hijack)

---

## Post-T13 additions

T13 execution date: 2026-05-15.

### Files created

- `internal/state/activity/context.md` — NEW. Documents Activity machine
  types, transitions, predicates, per-activity interrupt policy table,
  intentional asymmetries (no Foraging/Tracking state, mob forager FSM
  boundary, no IsForaging predicate, salvage own state), cascade
  integration, persistence, test notes, sunset list.

### Files updated

- `internal/combat/context.md` — Lines 370, 619: Updated `CastingState`
  is non-nil → `c.IsCasting()` is true (Activity machine is in Casting).
- `internal/hooks/context.md` — Lines 522, 609: Updated
  `RegisterActivityCheck` table entries from `CastingState == nil &&
  CraftingState == nil` to `c.IsActing()` (negated). Line 658: Updated
  "Nils CastingState and CraftingState" to describe Activity machine
  transitioning to Free via `activity_life_dead` observer. Added new
  section "Activity Machine Cascade + Observers (chunk 3)" documenting
  `Activity_Cascades.go` and all call-site wirings.
- `internal/state/awareness/context.md` — Lines 117-118: Updated
  `RegisterActivityCheck` description to reference `c.IsActing()`.
  Lines 253-254: Updated Dependencies to reference `Character.Activity`
  via `IsActing()`.
- `internal/state/life/context.md` — Line 206: Updated "Nils
  CastingState and CraftingState" to "Transitions Activity machine to
  Free via activity_life_dead observer in Activity_Cascades.go".
- `internal/characters/context.md` — Added "Activity Machine Integration
  (chunk 3)" section with Activity field, all five predicates, new
  OnCharacterCreated registration, and sunset notes.

### Skipped (file does not exist)

- `internal/forager/context.md` — Does not exist. The intentional-
  asymmetry rationale (mob forager FSM stays separate from
  Character.Activity) is documented in `internal/state/activity/
  context.md` "Notes on intentional asymmetries" bullet 2. No cross-
  reference file to update.
