# Ironwind Steppe Mob Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Complete mob archetype coverage for ironwind_steppe (23 uncovered + 2 reassignments), introduce two new archetypes (`ambusher`, `prey`), and ship custom boss btrees for the stone beetle queen and the windscour wyrm.

**Architecture:** Reuse existing behaviortree primitives (`mob_has_buff`, `mob_health_below`, `add_buff`, `flee`, `attack`, `command`, `command_best_of`, `callforhelp`). No engine changes.

---

## Scope + file map

| Scope | Files |
|-------|-------|
| New archetype YAML | `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml` |
| New archetype YAML | `_datafiles/world/dogmud/behaviors/archetypes/prey.yaml` |
| New per-mob btree | `_datafiles/world/dogmud/behaviors/ironwind_steppe/228-stone_beetle_queen.yaml` |
| New per-mob btree | `_datafiles/world/dogmud/behaviors/ironwind_steppe/229-windscour_wyrm.yaml` |
| Mob YAML edits | 25 files (23 archetype assigns + 2 reassigns + 2 vitality bumps — overlap) |
| Archetype tests | `internal/behaviortree/archetype_ambusher_test.go`, `_prey_test.go` |
| Patch notes | `PATCH_NOTES.md` |

---

## Task 1: Create `ambusher` archetype + load test

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml`
- Create: `internal/behaviortree/archetype_ambusher_test.go`

### Step 1: Create `ambusher.yaml`

Create `_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml`:

```yaml
# ambusher archetype
#
# Hit-and-fade predator. Cycle:
#   1. Mob starts hidden (buff 9) via buffids in mob YAML.
#   2. player_enter while Hidden → surprise strike (attack).
#   3. Combat cancels buff 9 via CancelIfCombat flag.
#   4. mob_hurt → flee to adjacent room.
#   5. In new room (not in combat) → mob_idle fires → re-apply buff 9.
#   6. Cycle back to step 1 when next player enters.
#
# Spec: docs/superpowers/plans/2026-04-24-ironwind-steppe-mob-audit.md

tree:
  type: selector
  children:
    # Surprise strike when hidden and a player arrives
    - type: sequence
      event: player_enter
      children:
        - type: condition
          check: mob_has_buff
          buff_id: 9
        - type: action
          do: attack

    # Flee after being hit (the ambush has cost us our cover)
    - type: action
      event: mob_hurt
      do: flee

    # Re-apply Hidden when idle (mob_idle only fires out-of-combat)
    - type: action
      event: mob_idle
      do: add_buff
      buff_id: 9
```

### Step 2: Create load test

Create `internal/behaviortree/archetype_ambusher_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const ambusherYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml"

func TestArchetype_Ambusher_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "ambusher", ambusherYAML)
	assert.NotNil(t, GetEngine().GetArchetype("ambusher"))
}

func TestArchetype_Ambusher_HandlesMobIdle(t *testing.T) {
	LoadArchetypeForTest(t, "ambusher", ambusherYAML)
	arch := GetEngine().GetArchetype("ambusher")
	if arch == nil {
		t.Fatal("archetype not loaded")
	}
	ctx := &EvalContext{
		InstanceId: 14001,
		RoomId:     1,
		Event: EventContext{
			EventType: "mob_idle",
		},
	}
	// Structural pass — expect no panic. Mob isn't in the test harness so
	// actAddBuff will return Failure at the mobs.GetInstance lookup; that's
	// fine, we're just asserting tree shape.
	_ = arch.Evaluate(ctx)
}
```

### Step 3: Build + test

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build ./internal/behaviortree/...
go test ./internal/behaviortree/ -run TestArchetype_Ambusher -v
```

### Step 4: Commit

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/ambusher.yaml \
         internal/behaviortree/archetype_ambusher_test.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): ambusher archetype — hit-and-fade predator

Cycle: strike from hidden on player_enter → flee on mob_hurt → re-apply
Hidden buff on mob_idle. Reuses existing primitives (mob_has_buff, attack,
flee, add_buff). Maximum-nuisance design for cave stalkers and similar
hit-and-run predators.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create `prey` archetype + load test

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/prey.yaml`
- Create: `internal/behaviortree/archetype_prey_test.go`

### Step 1: Create `prey.yaml`

```yaml
# prey archetype
#
# Flee-on-hurt wildlife. Complements the existing `hates` mechanic
# (which drives "predator attacks on sight"); this gives the reverse
# — "prey flees when attacked." Small animals huntable for food/
# materials: hares, grouse, squirrels, lizards, etc.
#
# Spec: docs/superpowers/plans/2026-04-24-ironwind-steppe-mob-audit.md

tree:
  type: action
  event: mob_hurt
  do: flee
```

### Step 2: Create load test

Create `internal/behaviortree/archetype_prey_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const preyYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/prey.yaml"

func TestArchetype_Prey_Loads(t *testing.T) {
	LoadArchetypeForTest(t, "prey", preyYAML)
	assert.NotNil(t, GetEngine().GetArchetype("prey"))
}
```

### Step 3: Build + test + commit

```bash
go build ./internal/behaviortree/...
go test ./internal/behaviortree/ -run TestArchetype_Prey -v

git add _datafiles/world/dogmud/behaviors/archetypes/prey.yaml \
         internal/behaviortree/archetype_prey_test.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): prey archetype — flee on hurt

One-line btree: mob_hurt → flee. Gives small wildlife (hares, grouse,
lizards) a sensible response to being attacked without making them
non-combatant (so they remain valid hunting targets for food/materials).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Stone beetle queen custom btree + vitality bump

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/ironwind_steppe/228-stone_beetle_queen.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml` (vitality bump — stays on `leader` archetype as fallback, but per-mob file overrides)

### Step 1: Create per-mob btree

Create `_datafiles/world/dogmud/behaviors/ironwind_steppe/228-stone_beetle_queen.yaml`:

```yaml
# Stone Beetle Queen (228) — Maternal Tank Boss
#
# Matriarch of the cave beetle colony. Heavy defensive tank. Calls
# swarm (via callforhelp → packmates from adjacent rooms) aggressively
# once wounded. Basic attacks handle DPS.
#
# Tuning: vitality bumped to support the tank role (see mob YAML).

tree:
  type: selector
  children:
    # Packmate hit? Rally the swarm and pile in
    - type: sequence
      event: packmate_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    # Mob_hurt: call swarm once wounded
    - type: sequence
      event: mob_hurt
      children:
        - type: action
          do: command
          cmd: callforhelp
        - type: action
          do: attack

    # Combat round: at 60% HP, call swarm again (desperation)
    - type: sequence
      event: mob_combat_round
      children:
        - type: condition
          check: mob_health_below
          percent: 60
        - type: action
          do: command
          cmd: callforhelp

    # heard_callforhelp: rally toward the caller (mirrors lookout)
    - type: action
      event: heard_callforhelp
      do: go_to_caller_room
```

### Step 2: Bump queen vitality

Edit `_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml`. Find the existing `stats:` block (if any) or add one. Add a vitality training bump of +60:

Locate the existing `character:` → `stats:` block, or if absent add one at the character level. Example:

```yaml
character:
  ...
  stats:
    vitality:
      training: 60
```

If a `stats:` block already exists with vitality, ADD 60 to whatever training value is there. If vitality isn't present, add the training line above.

Verify the file still parses:
```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml'))" && echo OK
```
(Skip if python/pyyaml unavailable — the go build/test after mob YAML changes covers it.)

### Step 3: Commit

```bash
git add _datafiles/world/dogmud/behaviors/ironwind_steppe/228-stone_beetle_queen.yaml \
         _datafiles/world/dogmud/mobs/ironwind_steppe/228-stone_beetle_queen.yaml
git commit -m "$(cat <<'EOF'
feat(ironwind_steppe): stone beetle queen — maternal tank boss btree + vit bump

Per-mob btree: packmate_hurt / mob_hurt → callforhelp + attack.
mob_combat_round at HP<60% → call swarm again (desperation).
heard_callforhelp → rally toward caller (mirrors lookout).

Vitality training bumped +60 to pay off the tank role. Mob still has
behavior_archetype: leader as YAML-level fallback but per-mob btree
REPLACES archetype, so the leader handlers don't run here.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Windscour wyrm custom btree + vitality bump

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/ironwind_steppe/229-windscour_wyrm.yaml`
- Modify: `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml` (vitality bump + reassign archetype to solitary)

### Step 1: Create per-mob btree

Create `_datafiles/world/dogmud/behaviors/ironwind_steppe/229-windscour_wyrm.yaml`:

```yaml
# Windscour Wyrm (229) — Two-Phase Apex Boss
#
# Phase 1 (HP ≥ 50%): Default combat — slow, devastating base attacks.
#   Handled entirely by the combat loop's default behavior.
# Phase 2 (HP < 50%): Enraged — rotates tail-sweep knockdown moves
#   (trip + bash) on every combat round.
#
# No pack behavior — the wyrm is solitary.
#
# Tuning: vitality bumped significantly to support the two-phase pacing.

tree:
  type: selector
  children:
    # Phase 2: enraged below 50% HP — tail-sweep / knockdown rotation
    - type: sequence
      event: mob_combat_round
      children:
        - type: condition
          check: mob_health_below
          percent: 50
        - type: action
          do: command_best_of
          commands: ["trip", "bash"]
```

### Step 2: Bump wyrm vitality + note archetype

Edit `_datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml`:

1. Vitality training +80 (similar pattern to queen: add to existing `stats.vitality.training` or add new).
2. Archetype assignment stays at whatever is currently set (`generic_fighter`). The per-mob btree replaces it anyway.

### Step 3: Commit

```bash
git add _datafiles/world/dogmud/behaviors/ironwind_steppe/229-windscour_wyrm.yaml \
         _datafiles/world/dogmud/mobs/ironwind_steppe/229-windscour_wyrm.yaml
git commit -m "$(cat <<'EOF'
feat(ironwind_steppe): windscour wyrm — two-phase apex boss + vit bump

Phase 1 (HP ≥ 50%): default combat loop handles slow, devastating
base attacks. Phase 2 (HP < 50%): per-mob btree fires command_best_of
[trip, bash] on every combat round — tail-sweep knockdown rotation.

Vitality training bumped +80 to pay off the two-phase pacing. No pack
behavior — the wyrm is solitary.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Assign archetypes to 23 uncovered mobs + reassign 2 cave stalkers

**Files:** 25 mob YAMLs under `_datafiles/world/dogmud/mobs/ironwind_steppe/`

### Step 1: Apply this table

| Mob file | `behavior_archetype` | Notes |
|----------|---------------------|-------|
| 200-steppe_rat.yaml | `generic_fighter` | |
| 201-dust_crow.yaml | `generic_fighter` | |
| 202-feral_cat.yaml | `generic_fighter` | |
| 203-scavenger_dog.yaml | `generic_fighter` | |
| 204-grass_snake.yaml | `generic_fighter` | |
| 211-prairie_viper.yaml | `generic_fighter` | |
| 212-burrowing_beetle.yaml | `generic_fighter` | |
| 213-dust_hare.yaml | `prey` | |
| 214-sage_grouse.yaml | `prey` | |
| 220-coulee_spider.yaml | `generic_fighter` | |
| 221-rock_viper.yaml | `generic_fighter` | |
| 225-pale_lurker.yaml | **`ambusher`** | REASSIGN from generic_fighter. Also needs `buffids: [9]` in mob YAML to start hidden. |
| 227-blind_stalker.yaml | **`ambusher`** | REASSIGN from generic_fighter. Also needs `buffids: [9]`. |
| 230-steppe_lizard.yaml | `prey` | |
| 231-tumble_beetle.yaml | `prey` | |
| 232-wind_scorpion.yaml | `generic_fighter` | |
| 233-carrion_crow.yaml | `generic_fighter` | |
| 234-ground_squirrel.yaml | `prey` | |
| 235-horned_toad.yaml | `prey` | |
| 236-steppe_fox.yaml | `generic_fighter` | |
| 237-dust_moth.yaml | `prey` | |
| 238-dry_creek_crayfish.yaml | `prey` | |
| 240-hermit_kael.yaml | `noncombat_questgiver` | |
| 241-windwarden_sylara.yaml | `noncombat_questgiver` | |
| 242-geomancer_rhett.yaml | `noncombat_questgiver` | |

### Step 2: Add `buffids: [9]` to ambushers

For 225-pale_lurker.yaml and 227-blind_stalker.yaml, add a top-level field:

```yaml
buffids: [9]
```

If the mob YAML already has a `buffids:` list, append 9 to it (avoid duplicates). Place the field near other top-level metadata like `archetype:`, `hostile:`, etc.

This makes them spawn already hidden so the ambusher loop's first strike works on their first encounter.

### Step 3: Verify coverage

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
total=$(ls _datafiles/world/dogmud/mobs/ironwind_steppe/*.yaml | wc -l)
have=$(grep -l "behavior_archetype:" _datafiles/world/dogmud/mobs/ironwind_steppe/*.yaml | wc -l)
echo "have $have of $total"
```
Expected: `have 43 of 43`.

Verify ambusher reassigns worked:
```bash
grep "behavior_archetype:" _datafiles/world/dogmud/mobs/ironwind_steppe/225-pale_lurker.yaml \
     _datafiles/world/dogmud/mobs/ironwind_steppe/227-blind_stalker.yaml
```
Expected: both show `behavior_archetype: ambusher`.

### Step 4: Build

```bash
go build ./...
```

### Step 5: Commit

```bash
git add _datafiles/world/dogmud/mobs/ironwind_steppe/
git commit -m "$(cat <<'EOF'
content(ironwind_steppe): complete archetype coverage (23 new + 2 reassigns)

Assignments by category:
- 12 solitary hostile wildlife (rat, crow, cat, dog, snake, vipers,
  beetle, spider, scorpion, carrion crow, fox) → generic_fighter
- 8 passive prey (hare, grouse, lizard, tumble beetle, squirrel,
  toad, moth, crayfish) → prey (new archetype)
- 3 named NPCs (Hermit Kael, Windwarden Sylara, Geomancer Rhett) →
  noncombat_questgiver
- 2 cave stalkers reassigned from generic_fighter → ambusher:
  pale_lurker (225), blind_stalker (227). Both now spawn with
  buffids: [9] (Hidden) for the first-strike ambush.

Coverage: 43/43 ironwind_steppe mobs have behavior_archetype.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Full-repo build + test sweep

Run:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go test ./... 2>&1 | grep -E "^FAIL"  # should be empty
```

If FAIL lines appear, investigate and fix before proceeding to Task 7. Most likely failure is an archetype test pointing to a broken YAML.

No commit unless a fix is needed.

---

## Task 7: Patch notes

**Files:**
- Modify: `PATCH_NOTES.md`

### Step 1: Prepend a new section

Open `PATCH_NOTES.md`. Find the top-of-file header:

```markdown
# DOGMud Patch Notes

## 2026-04-24 (evening) — Sanctum Basin Mob Audit + Tutorial Content
```

Insert AFTER the header and BEFORE the existing 2026-04-24 (evening) entry:

```markdown
## 2026-04-24 (late evening) — Ironwind Steppe Audit + Boss Behaviors

### Gameplay

- **Cave stalkers now ambush from the dark.** Pale lurkers and blind
  stalkers spawn hidden, open with a surprise strike when a player
  enters their room, then flee the moment they take damage and
  re-hide in an adjacent room. Maximum-nuisance hit-and-fade cycle.
- **Stone Beetle Queen calls her swarm.** Boss behavior: when wounded
  or when one of her brood is hurt, she calls for help — pulling
  cave beetles from adjacent rooms. Vitality bumped to match her
  tank role.
- **Windscour Wyrm goes two-phase.** Above 50% HP the wyrm fights
  its slow, devastating baseline rotation. Below 50% HP it rages —
  tail-sweep knockdown rotations on every round. Vitality bumped to
  support the pacing.
- **Prey animals flee when hit.** Hares, grouse, lizards, squirrels,
  toads, moths, tumble beetles, and dry creek crayfish now retreat
  to an adjacent room when attacked instead of standing and dying.
  They remain attackable for hunting.

### Behind the scenes

- Two new behavior archetypes: `ambusher` and `prey`.
- Custom per-mob btrees for the Stone Beetle Queen (228) and
  Windscour Wyrm (229).
- Ironwind Steppe now has 43/43 archetype coverage.
- No engine changes — all behaviors reuse existing primitives.
```

### Step 2: Commit

```bash
git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs: patch notes for ironwind steppe audit

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

**Spec coverage:** Every item from conversation captured — 2 new archetypes, queen custom btree with vit bump, wyrm two-phase btree with vit bump, 23 uncovered assignments, 2 cave stalker reassigns, hidden buff seeding for ambushers. ✓

**Placeholder scan:** No TBDs. Every YAML content block is complete. Every grep/verify step has exact expected output.

**Type consistency:** `ambusher`, `prey` used consistently. `behaviors/archetypes/*.yaml` path consistent. Event names (`player_enter`, `mob_hurt`, `mob_idle`, `mob_combat_round`, `packmate_hurt`, `heard_callforhelp`) consistent with existing usage.
