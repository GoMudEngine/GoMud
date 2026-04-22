# Companion AI — Pure Caster Archetype

**Date:** 2026-04-20
**Scope:** Second archetype in the companion-AI series. Pure Caster: maintains self-buffs, emergency-heals when low HP, prefers AoE when enemies are grouped, else single-target harm.
**Status:** Design — awaiting user review before plan.

**Depends on:** Companion Phase 4 (`2026-04-20-companion-melee-self-buff-archetype-design.md`) — reuses the archetype framework, `cast_best_in_category` action, `mob_combat_round` event, and `self_defense` category.

---

## Summary

Phase 4 shipped the archetype framework and the first archetype (`melee_self_buff`) covering vampire, air elemental, fire elemental. This spec adds the second archetype — `pure_caster` — for wraiths, spectres, and **air elemental reassigned from melee**.

The archetype is a pure composition of existing btree primitives: both required conditions (`mob_health_below`, `multiple_enemies`) are already registered, `cast_best_in_category` already does the work, the `mob_combat_round` event is already wired. The only new content is three spell-category tags, a per-mob spellbook pass, and the archetype tree file itself.

## Goals

- Second archetype proves the framework scales — multiple archetypes sharing the same infrastructure
- Wraith / spectre / air elemental behave like actual mages: maintain wards + iron-will, heal self in emergencies, AoE when grouped, single-target otherwise
- Zero new btree framework code (no new actions, no new conditions)
- Air elemental moves from `melee_self_buff` to `pure_caster` (its stats — dex 20, perception 20, will 10 — are caster-shaped anyway)

## Non-goals

- **Not** rolling out to non-summon casters in the world (deferred to the legacy-deprecation project)
- **Not** adding a perception-boost buff spell (none exists today; iron-will's willpower boost covers the caster stat slot)

---

## 1. Archetype Tree

**File location:** `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml`

```yaml
# pure_caster archetype
#
# A spellcaster who maintains self-buffs, emergency-heals, and prefers
# AoE when multiple enemies are present, else single-target harm.
# Reused by wraith, spectre, air elemental, plus any future caster mobs
# that opt in via behavior_archetype: pure_caster on their mob YAML.
#
# Decision order per mob_combat_round:
#   1. Emergency heal if HP < 40%
#   2. Maintain defensive buffs (shields / will+mitigation)
#   3. AoE harm when 2+ actors in room (multiple_enemies)
#   4. Single-target harm on aggro target
#   5. Fall through to legacy combat (combatcommands / default attack)
#
# Every cast_best_in_category action self-gates on shared special-move
# cooldown / CP / component / summon / already-active, so mobs naturally
# alternate between cast rounds and attack-filler rounds.

tree:
  type: selector
  event: mob_combat_round
  children:
    # 1. Emergency heal — survival priority
    - type: sequence
      children:
        - type: condition
          check: mob_health_below
          percent: 40
        - type: action
          do: cast_best_in_category
          category: self_heal
          target: self

    # 2. Maintain defensive buffs (reuses melee archetype's category)
    - type: action
      do: cast_best_in_category
      category: self_defense
      target: self

    # 3. AoE harm when multiple enemies in room
    - type: sequence
      children:
        - type: condition
          check: multiple_enemies
        - type: action
          do: cast_best_in_category
          category: harm_multi

    # 4. Single-target harm (default offensive)
    - type: action
      do: cast_best_in_category
      category: harm_single
```

### Decision flow per round

1. `mob_combat_round` fires for a caster mob with `behavior_archetype: pure_caster`.
2. Selector tries child 1: if mob HP < 40% AND a `self_heal`-tagged spell is castable → initiate heal. Success.
3. Else tries child 2: if any `self_defense` buff is missing and castable → initiate it. Success.
4. Else tries child 3: if 2+ actors in room AND a `harm_multi` spell is castable → initiate it. Success.
5. Else tries child 4: cast best `harm_single` spell on aggro target. Success.
6. If none of the above fire (on cooldown / no spellbook matches / CP-starved) → selector Failure → legacy combat runs (default swing, emote).

### Why offense-second to defense

Casters are fragile (statpool 70-90, low physical stats). Burning through defense before attacking keeps them alive long enough to deal damage. For melee (vampire/fire), offense-first made sense because they can take hits. Different archetype, different priority — exactly the design flexibility archetypes-as-first-class enables.

## 2. Spell Categorization

Three new categories; all existing self-buff tags from Phase 4 stay as-is.

| Category | Existing/New | Meaning | Spells to tag |
|---|---|---|---|
| `self_heal` | **new** | Self-cast regen effect | `heal` |
| `self_defense` | existing | Shields / protection buffs | (iron-will, conviction-ward, conviction-armor — already tagged in Phase 4) |
| `harm_single` | **new** | Single-target damage or debuff | `mind-spike`, `conviction-spike`, `nerve-disruption`, `hemorrhagic-burst`* |
| `harm_multi` | **new** | AoE damage | `sparks`, `conviction-barrage`, `hemorrhagic-wave` |

\* Note: `hemorrhagic-burst` is `type: harmarea` despite its name — it's an AoE. Tagging as `harm_multi`, not `harm_single`.

### Revised table (correcting for spell-type verification)

| Category | Spells (verified by `type:` field) |
|---|---|
| `self_heal` | `heal` (HelpSingle, effect_type: heal) |
| `harm_single` | `mind-spike`, `conviction-spike`, `nerve-disruption` (all HarmSingle) |
| `harm_multi` | `sparks`, `conviction-barrage`, `hemorrhagic-wave`, `hemorrhagic-burst` (all HarmArea) |

## 3. Mob Changes

### Wraith (302-wraith.yaml)

```yaml
behavior_archetype: pure_caster   # NEW

character:
  spellbook:
    conviction-ward: 4             # existing — self_defense
    mind-spike: 5                  # existing — harm_single
    nerve-disruption: 3            # existing — harm_single
    heal: 3                        # NEW — self_heal
    sparks: 3                      # NEW — harm_multi

combatcommands:                    # trim hardcoded cast entries
  - ''
  - 'emote phases through its target, trailing cold that cuts to the bone'
  - ''
  # removed: 'cast mind-spike' (archetype handles it)
  # removed: 'cast nerve-disruption' (archetype handles it)
```

### Spectre (303-spectre.yaml)

```yaml
behavior_archetype: pure_caster   # NEW

character:
  spellbook:
    conviction-ward: 4             # existing — self_defense
    conviction-spike: 5            # existing — harm_single
    iron-will: 3                   # NEW — self_defense
    heal: 3                        # NEW — self_heal
    conviction-barrage: 3          # NEW — harm_multi
    # removed: conviction-surge (boosts strength, useless for caster)

combatcommands:                    # trim hardcoded cast entries
  - ''
  - 'emote projects a wave of pure dread that presses on the chest like a stone'
  - ''
  # removed: 'cast conviction-spike'
  # removed: 'cast conviction-ward'
```

### Air elemental (312-air_elemental.yaml) — **archetype change**

Currently assigned `behavior_archetype: melee_self_buff` in Phase 4. Reassigning to `pure_caster` because the ellie's actual stats (dex 20, perception 20, willpower 10) are caster-shaped, not melee-shaped.

```yaml
behavior_archetype: pure_caster   # CHANGED from melee_self_buff

character:
  spellbook:
    conviction-ward: 4             # existing — self_defense
    iron-will: 3                   # existing — self_defense
    heal: 3                        # NEW — self_heal
    mind-spike: 3                  # NEW — harm_single
    sparks: 3                      # NEW — harm_multi

combatcommands:                    # unchanged from Phase 4
  - ''
  - 'emote crackles and spins faster, striking from unexpected angles'
  - ''
```

### Coverage matrix

| Mob | self_heal | self_defense | harm_single | harm_multi |
|---|---|---|---|---|
| Wraith | heal | conviction-ward | mind-spike, nerve-disruption | sparks |
| Spectre | heal | conviction-ward, iron-will | conviction-spike | conviction-barrage |
| Air ellie | heal | conviction-ward, iron-will | mind-spike | sparks |

All four archetype children have at least one matching spell per mob — no empty-category fallthroughs this phase (which the melee archetype exercised via air/fire's empty self_offense).

## 4. Engine Changes

### One bug fix: `multiple_enemies` condition perspective-awareness

**File:** `internal/behaviortree/conditions_player.go`

**Current behavior:** `condMultipleEnemies` counts `players + charmed-mobs` in the room. This was correct for the condition's original callers (wild hostile mobs like `bandit_leader`), where players and charmed companions ARE the enemies. But it's wrong from a charmed/summoned mob's perspective — the summoner (a player) and fellow companions (charmed mobs of same owner) are *friendly*, not hostile.

**Fix:** Make the condition perspective-aware. For a charmed mob, skip the summoner from the player count and fellow same-owner companions from the mob count. For a wild mob, preserve today's behavior (count all players + all charmed mobs).

```go
func condMultipleEnemies(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	charmedByUserId := 0
	if mob != nil {
		charmedByUserId = mob.Character.GetCharmedUserId()
	}

	count := 0

	// Players — skip the summoner if this is a charmed mob
	for _, pId := range room.GetPlayers() {
		if charmedByUserId > 0 && pId == charmedByUserId {
			continue
		}
		count++
	}

	// Mobs — from a charmed mob's POV, fellow same-owner companions are
	// friends; count only wild mobs + mobs charmed by someone else.
	// From a wild mob's POV, preserve original behavior (count charmed
	// companions; wild mobs don't count other wild mobs as enemies).
	for _, mId := range room.GetMobs() {
		if mob != nil && mId == mob.InstanceId {
			continue // don't count self
		}
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		if charmedByUserId > 0 {
			// Charmed mob: skip fellow companions of same owner
			if m.Character.IsCharmed(charmedByUserId) {
				continue
			}
			count++
		} else {
			// Wild mob: original behavior — only charmed companions count
			if m.Character.IsCharmed() {
				count++
			}
		}
	}

	if count > 1 {
		return Success
	}
	return Failure
}
```

**APIs used** (all already exist):
- `Character.GetCharmedUserId() int` — 0 if not charmed, else owner's userId
- `Character.IsCharmed(userId ...int) bool` — with userId param: matches specific owner; without: any-owner check

**Regression safety:** Wild mobs (where `charmedByUserId == 0`) fall into the `else` branch which replicates the original `len(players) + len(GetMobs(FindCharmed))` semantics. Existing users (`bandit_leader`) see zero behavior change.

### Verified, no changes needed

- `mob_health_below` condition exists (`internal/behaviortree/conditions_mob.go`), param name is `percent`
- `cast_best_in_category` already handles `target: self` and implicit target (mob.Command with no target → aggro for HarmSingle, room-wide for HarmArea)
- `mob_combat_round` event firing at `NewRound_DoCombat.go:276` is shared with `melee_self_buff` — no new fire point needed
- Phase 4's `applyMobSelfEffect` already handles `buff`, `heal`, `shield` effect types — heal-on-self for caster works via the existing `heal` case

## 5. Error Handling & Edge Cases

### Load-time errors

| Failure | Behavior |
|---|---|
| `pure_caster.yaml` missing | Archetype negative-cached, Warn logged, mobs fall through to legacy (per Phase 4 resolution) |
| YAML malformed | Error logged, negative-cached, legacy fallthrough |
| `percent: 40` param missing on `mob_health_below` node | Condition returns false (never fires) — heal branch never triggers; caster still casts defense/harm |

### Runtime edge cases

| Case | Behavior |
|---|---|
| Mob has no `self_heal` spells in spellbook but HP < 40% | Child 1 returns Failure at cast-time → selector tries child 2 (defense) |
| All buffs active AND CP < any harm spell cost | Selector fails → legacy combat → default melee (feeble for these mobs, but flavor via emotes) |
| Mob is a summoned caster, `multiple_enemies` over-counts (pre-fix) | **Fixed in this spec** (see §4). Condition is now perspective-aware — summoner and fellow same-owner companions are excluded from the count for charmed mobs. |

### Deliberately not added

- Target-selection smarts beyond "default from cast pipeline" — the existing InitiateCast for HarmSingle picks the mob's aggro target, HarmArea hits the room; that's enough for this phase
- Heal-target other characters (owner, fellow companions) — reserved for a future healer archetype
- CP-reserve logic (don't burn all CP on harm so heal is affordable) — not worth the complexity for this phase; heal priority ordering is the simpler safeguard

## 6. Testing Strategy

### Unit tests

**New test coverage for `multiple_enemies` fix** — extend `internal/behaviortree/conditions_test.go`:
- Wild mob (not charmed) + 1 player + 1 charmed mob → Success (2 enemies) — regression check for existing callers
- Wild mob + 0 players + 0 charmed mobs → Failure (0 enemies)
- Charmed mob (owner = user 5) + user 5 + fellow companion charmed-by-5 → Failure (summoner + fellow are friends; count = 0)
- Charmed mob (owner = user 5) + user 5 + 2 wild mobs → Success (2 wild enemies; summoner excluded)
- Charmed mob (owner = user 5) + user 6 + wild mob → Success (1 hostile player + 1 wild mob = 2)

`cast_best_in_category` and `mob_health_below` already have coverage from Phase 4 and earlier work.

### Integration test

Extend `internal/behaviortree/melee_self_buff_archetype_integration_test.go` OR create a sibling `pure_caster_archetype_integration_test.go`. Either way, the pattern is identical to Phase 4's T11:

1. Seed the full spell library (self_heal, self_defense, harm_single, harm_multi)
2. Load `pure_caster.yaml` via `LoadArchetypeForTest`
3. For each of these scenarios, fire `mob_combat_round` via `TryMobBehavior` and assert the queued command:
   - Fresh wraith (HP full, no buffs active, 1 enemy) → casts `conviction-ward` (child 2 — defense first since HP is fine)
   - Ward active, HP full, 1 enemy → casts `mind-spike` (child 4 — top-ranked harm_single)
   - Ward active, HP < 40% → casts `heal` (child 1 — emergency priority)
   - Ward active, HP full, 2+ hostiles → casts `sparks` (child 3 — AoE)
   - All buffs active, HP full, no harm spells → tree returns Failure (fallthrough)

### Manual smoke (pre-merge)

1. Summon wraith, engage a training dummy → should see ward, then mind-spike/nerve-disruption alternating
2. Summon spectre, take damage until HP < 40% → should see heal text
3. Summon wraith + enter a room with 2+ hostile mobs → should see AoE casts
4. Summon air elemental (now on caster archetype) → should cast instead of swinging
5. Charmed wild caster mob with no spells → falls through cleanly, no crashes

### Regression risk

- Air elemental behavior changes from Phase 4's melee-defense-only to caster-with-harm. Existing Phase 4 integration test `TestMeleeSelfBuff_AirElementalCastsDefenseOnly` references air ellie as a melee_self_buff example — **must be updated** to either use fire ellie instead, or removed. Fire ellie is still `melee_self_buff` and has the same defense-only spellbook shape (no self_offense), so it's a drop-in replacement.

## 7. Deliverables

### New files

- `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml` — the archetype tree
- `internal/behaviortree/pure_caster_archetype_integration_test.go` — integration tests (or extend existing file)

### Modified files

- `internal/behaviortree/conditions_player.go` — perspective-aware `multiple_enemies` fix (§4)
- `internal/behaviortree/conditions_test.go` — extended coverage for the fix (§6)
- `_datafiles/world/dogmud/spells/heal.yaml` — add `categories: [self_heal]`
- `_datafiles/world/dogmud/spells/mind-spike.yaml` — add `categories: [harm_single]`
- `_datafiles/world/dogmud/spells/conviction-spike.yaml` — add `categories: [harm_single]`
- `_datafiles/world/dogmud/spells/nerve-disruption.yaml` — add `categories: [harm_single]`
- `_datafiles/world/dogmud/spells/sparks.yaml` — add `categories: [harm_multi]`
- `_datafiles/world/dogmud/spells/conviction-barrage.yaml` — add `categories: [harm_multi]`
- `_datafiles/world/dogmud/spells/hemorrhagic-wave.yaml` — add `categories: [harm_multi]`
- `_datafiles/world/dogmud/spells/hemorrhagic-burst.yaml` — add `categories: [harm_multi]`
- `_datafiles/world/dogmud/mobs/summons/302-wraith.yaml` — `behavior_archetype: pure_caster` + spellbook expansion + combatcommands trim
- `_datafiles/world/dogmud/mobs/summons/303-spectre.yaml` — same pattern
- `_datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml` — **archetype reassignment** + spellbook expansion
- `internal/behaviortree/melee_self_buff_archetype_integration_test.go` — update or remove the air-elemental test case (fire ellie substitute)
- `PATCH_NOTES.md` — dated entry for the release

### Explicitly not touched

- `internal/behaviortree/` — only `conditions_player.go` + `conditions_test.go` (the fix). No other framework changes.
- `internal/hooks/` — zero changes
- `internal/spells/` Go code — zero changes
- `internal/mobs/` Go code — zero changes

## Out of Scope / Future Work

- Tank/Taunter archetype (earth/magma elemental, flesh golem) — next spec
- Legacy-system deprecation project — audit all 187 mobs, migrate to archetypes, delete `mobai` + `preferredSpell` + `ChooseCastAction`
- Perception-boosting buff spell — none exists today; would need a new buff definition + spell
- Healer archetype (heals allies, not just self)
- Per-companion-type spell accumulation cleanup — players' `CompanionInfo.SpellBook` retains discovered spells across summons, which pollutes the archetype's category-filtered selection. Fix is a UI/admin tool to reset companion spellbook to template; not urgent.

## Summary

One new archetype file, three new spell-category tags, three mob YAMLs edited, one Phase 4 test file updated, plus one small Go fix (perspective-aware `multiple_enemies` condition) with test coverage. Rides almost entirely on the framework built in Phase 4.
