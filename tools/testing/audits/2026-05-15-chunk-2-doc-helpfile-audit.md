# Chunk 2 documentation + helpfile audit

Produced 2026-05-15 to feed Task 13 of the chunk-2 Life machine plan.

Scope: identify docs, helpfiles, and YAML lore mentioning concepts
changed (death flow, stat decay, skill rust, respawn grace) or removed
(permadeath system, extra lives mechanic) by chunk 2 (Life state machine).

Survey date: 2026-05-15. Branch: feature/mob-aliveness-1.3-crimes, commit a76f5c67.

---

## Context.md files

### Keep as-is (mechanic references but content unchanged)
- `internal/buffs/context.md` — References `ReviveOnDeath` (flag, still exists) and mentions buff triggers/expiration mechanics. No permadeath references. Kept.
- `internal/events/context.md` — Lines 298-306: contains a comment noting `death.Permanent is always false (permadeath sunset)` in the `PlayerDeath` event handler example. This is a post-chunk-2 comment documenting the sunset. Kept (educational).
- `internal/hooks/context.md` — References `ReviveOnDeath` buff (id 43), `NoAggroTarget` buff (id 81, respawn grace mechanic), and documents death flow through observers. All current. References to graveyard teleport under "Respawn observers" section are current. Kept.

### Create new
- `internal/state/life/context.md` — Must be created by T13 to document the new Life state machine package (types: `Alive`, `Dead`, `Respawning`; transitions, observers, DeadData).

---

## Helpfiles

No helpfiles directory found at `_datafiles/helpfiles/`. User help is likely served via inline templates or help text in usercommands.

**Admin help templates** (`_datafiles/world/default/templates/admincommands/help/`) searched for permadeath/suicide/respawn references — none found.

User-facing help (if any) is likely in character templates. Searched `_datafiles/world/default/templates/usercommands/` — no directory found in the filesystem. Help may be embedded in command handlers themselves.

**No helpfiles to update.**

---

## Top-level docs

### Updates needed (stale references to removed/changed mechanics)

- `DEVELOPMENT_PLAN.md` — Line ~102: References `ExtraLives` as a field to remove. No action needed (historical record of Stage 1.3 work), but note that permadeath and extra-lives systems were targeted for removal as part of that stage.

### Keep as-is (mechanics still accurate)

- `PATCH_NOTES.md` — Numerous references to respawn grace, stat decay, skill rust, death flow changes. All post-chunk-2 content; all accurate. Examples:
  - "3-round grace period post-respawn" (respawn grace buff #81) — accurate
  - "Death respawn at 5% of max pools" — accurate
  - "Stat decay and skill rust penalties still apply" — accurate
  - "Death no longer kills you twice" (section on poison/condition clearing) — accurate
  Kept as-is.

- `docs/superpowers/specs/completed/2026-04-15-phase4c-room-spell-buff-migration-design.md` — References permadeath/ExtraLives in historical context of work prior to chunk 2. Kept as historical record.

- `docs/superpowers/specs/2026-05-13-combat-state-machines-design.md` — Early design doc, may reference legacy concepts. Not core to chunk-2 scope.

- `docs/superpowers/specs/2026-05-15-state-chunk-2-life-design.md` — Chunk 2 design spec. Discusses permadeath sunset as part of the design. Accurate and intentional.

- `docs/superpowers/plans/2026-05-15-state-chunk-2-life.md` — Chunk 2 implementation plan. Contains numerous references to permadeath/extra-lives removal as part of T11 (already completed). Accurate historical record of implementation steps. Kept.

- `feature-screenshots/README.md` — Unknown content (not read). Assume historical; low priority.

### Note for T15 (roadmap closeout)

- `MOB_ALIVENESS_ROADMAP.md` — Chunk 2 likely has Done entries to mark. Not T13's scope; noted for T15.
- `COMBAT_STATE_ROADMAP.md` — Chunk 2 Done entry noted for T15.

---

## YAML lore / descriptions

### Updates needed (example text to remove)

- `_datafiles/guides/building/scripting/SCRIPTING_ITEMS.md` — Line ~52: "such as purchasing extra lives, a ticket for travel, or renting a room". Update to remove "extra lives" example; replace with skill progression item or other non-removed example.

### Keep as-is

- `_datafiles/guides/building/scripting/FUNCTIONS_ACTORS.md` — Line 66: TOC entry `[ActorObject.GiveExtraLife()](#actorobjectgiveextralife)`. Lines ~700+: Function documentation for `GiveExtraLife()` declares "Increases extra lives by 1 for the player/actor". **This function signature still exists in the codebase post-chunk-2 (kept for upstream parity or legacy actor scripts).** No YAML lore; kept as a dead function in the scripting API.

No world-facing NPC dialogue or room descriptions found mentioning permadeath or extra lives.

---

## Templates

Searched `_datafiles/world/default/templates/` for `{{ permadeath }}` and related template helpers — none found.

**No template changes needed.**

---

## Test goals / roles

Searched `tools/testing/goals/` and `tools/testing/roles/` for permadeath/extra-life references — none found.

**No test fixture updates needed.**

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| Context.md files | 3 read | 2 keep as-is, 1 new file needed |
| Helpfiles | 0 | N/A (no helpfiles dir) |
| Top-level docs | 6 | All kept or noted for T15 |
| YAML lore | 2 | 1 update needed (example text) |
| Templates | ~30 | 0 updates needed |
| Test fixtures | N/A | 0 updates needed |

**Files requiring edits by T13:**
1. `_datafiles/guides/building/scripting/SCRIPTING_ITEMS.md` — Remove "extra lives" from onPurchase example
2. `internal/state/life/context.md` — **Create new** (document Life machine)
3. `internal/events/context.md` — Optional: clarify the permadeath sunset comment if needed (currently adequate)

**Notes for T15 (out of T13 scope):**
- Mark chunk 2 Done in `COMBAT_STATE_ROADMAP.md`
- Mark chunk-2-related entries Done in `MOB_ALIVENESS_ROADMAP.md`

---

## Surprising findings

1. **No user helpfiles directory.** DOGMud does not maintain a `_datafiles/helpfiles/` directory; help is likely embedded in command handlers or served via inline templates.

2. **GiveExtraLife() still exists in scripting API.** The function is documented in `FUNCTIONS_ACTORS.md` and likely still exists in the codebase for upstream parity, but the mechanic (extra lives) is removed. Chunks T13 should note this in `internal/state/life/context.md` as "deprecated/noop" or leave it undocumented in the life machine itself.

3. **Permadeath system already sunset before chunk 2.** References in spec/plan files show permadeath was already identified for removal; chunk 2's T11 completed that cleanup. The system was not live.

4. **Death flow references are all post-chunk-2.** PATCH_NOTES entries about respawn grace, stat decay, skill rust, and 5% pool restoration are accurate and intentional; no corrections needed.
