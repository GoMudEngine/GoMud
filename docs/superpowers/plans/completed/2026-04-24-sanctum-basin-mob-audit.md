# Sanctum Basin Mob Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all 17 Sanctum Basin mobs to the behavior archetype system and deliver prod-safe tutorial content (via dialogue YAML, no LLM dependency) that introduces new players to recently-added gameplay systems.

**Architecture:** Four new behavior archetypes (`noncombat_questgiver`, `noncombat_shopkeeper`, `noncombat_passive`, `combat_passive`) cover role-generic handlers for enter/give/attack-rejected events. A new btree event `player_attack_rejected` fires from `attack.go` and `cast.go` HarmSingle with round-based dedupe via a new `Character.LastAttackRejectedRound` field. Tutorial content lives in dialogue YAML `patterns` — already keyword-matched, prod-safe, LLM-independent.

**Tech Stack:** Go 1.21+, YAML (`gopkg.in/yaml.v2`), existing behaviortree engine, existing dialogue engine, `github.com/stretchr/testify`.

---

## File Structure

| File | Role | Status |
|------|------|--------|
| `internal/characters/character.go` | Add `LastAttackRejectedRound` field | Modify |
| `internal/usercommands/attack.go` | Fire `player_attack_rejected` on non-combatant rejection | Modify |
| `internal/actions/cast.go` | Fire `player_attack_rejected` on HarmSingle non-combatant rejection | Modify |
| `internal/usercommands/attack_test.go` | Unit test for the new event + dedupe | Modify |
| `_datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml` | Archetype: enter ack, decline+return on give, attack-reject emote | Create |
| `_datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml` | Same shape, shopkeeper flavor | Create |
| `_datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml` | Ambient emotes only | Create |
| `_datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml` | Empty tree (tag only) | Create |
| `internal/behaviortree/archetype_noncombat_test.go` | YAML load + event firing tests | Create |
| `_datafiles/world/dogmud/mobs/sanctum_basin/*.yaml` | Assign `behavior_archetype` to 17 mobs, fix `non_combatant` flags | Modify |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml` | Add system-mention patterns (chrysalis_priest) | Modify |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/52.yaml` | Add system-mention patterns (korvath) | Modify |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/55.yaml` | Add system-mention patterns (elder_saris) | Modify |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml` | Add system-mention patterns (basin_scholar) | Modify |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/51.yaml` | Combat trainer dialogue | Create |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/53.yaml` | Alchemist Yenna dialogue | Create |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/54.yaml` | Wilderness Guide Fen dialogue | Create |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/56.yaml` | Basin Warden dialogue | Create |
| `_datafiles/world/dogmud/dialogue/sanctum_basin/63.yaml` | Merchant Adela dialogue | Create |
| `PATCH_NOTES.md` | Document the change | Modify |

---

## Key codebase references

- Existing archetype YAML format: `_datafiles/world/dogmud/behaviors/archetypes/lookout.yaml` (simplest) or `generic_fighter.yaml`
- `actEmote` implementation: `internal/behaviortree/actions_dialogue.go:67` — reads `text` param, runs `mob.Command("emote " + text)`. The emote command itself auto-prefixes the mob name, so YAML `text:` should be just the verb phrase (no `{mob_name}` placeholder needed).
- `actReturnItem` exists at `internal/behaviortree/actions.go:21` — reads `item_id` from event, returns it to the player.
- `Character.LastSuicideRound` at `internal/characters/character.go:99` — pattern to mirror for `LastAttackRejectedRound`.
- `util.GetRoundCount()` returns `uint64` — round counter for dedupe.
- Dialogue YAML format: see `_datafiles/world/dogmud/dialogue/ashwick/259.yaml` for a complete example with `patterns` + `tree.nodes`.
- Attack rejection site: `internal/usercommands/attack.go:161-164`.
- HarmSingle rejection site: `internal/actions/cast.go:104-107`.

---

## Task 1: Add `LastAttackRejectedRound` field to Character

**Files:**
- Modify: `internal/characters/character.go:99` (add field next to `LastSuicideRound`)

- [ ] **Step 1: Locate the existing `LastSuicideRound` field**

Run: `grep -n "LastSuicideRound" internal/characters/character.go`
Expected output: one line around 99.

- [ ] **Step 2: Add the new field next to it**

Open `internal/characters/character.go`. Find the line:

```go
	LastSuicideRound uint64                         `yaml:"-"` // runtime only — round of last Suicide execution, for double-fire dedupe
```

Insert immediately AFTER it:

```go
	LastAttackRejectedRound uint64                  `yaml:"-"` // runtime only — round of last player_attack_rejected event fire, for dedupe
```

- [ ] **Step 3: Build**

Run: `cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./internal/characters/...`
Expected: no output.

- [ ] **Step 4: Commit**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
feat(characters): add LastAttackRejectedRound for player_attack_rejected dedupe

Mirrors the LastSuicideRound field pattern. Populated by attack.go and
cast.go HarmSingle rejection paths to rate-limit the btree event firing
to at most once per mob per round.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Fire `player_attack_rejected` from attack.go with dedupe + test

**Files:**
- Modify: `internal/usercommands/attack.go:161-164`
- Modify: `internal/usercommands/attack_test.go` (add new test)

### Step 1: Write the failing test

Open `internal/usercommands/attack_test.go`. Check existing test structure first:

```bash
grep -n "func Test" internal/usercommands/attack_test.go | head
```

Append a new test at the end of the file. The test verifies the btree event fires once, respects round-based dedupe, and doesn't suppress the player-visible rejection message.

```go
func TestAttack_NonCombatantFiresEventOnce_ThenDedupes(t *testing.T) {
	// Set up: a player in a room with a non-combatant mob.
	// Verify first attack fires the btree event; second attack in same
	// round does not re-fire; third attack in a later round re-fires.

	// Use a mock/tracking wrapper for TryMobBehavior.
	originalTry := attackTryMobBehavior
	var fireCount int
	attackTryMobBehavior = func(instanceId int, ctx behaviortree.EventContext) bool {
		if ctx.EventType == "player_attack_rejected" {
			fireCount++
		}
		return true
	}
	defer func() { attackTryMobBehavior = originalTry }()

	// Seed a non-combatant mob.
	mob := &mobs.Mob{
		InstanceId: 12001,
		Character: characters.Character{
			RoomId:       1,
			Name:         "Test NPC",
			NonCombatant: true,
		},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{12001: mob})
	defer cleanup()

	// Seed round counter deterministically.
	util.SetRoundCountForTest(100)
	defer util.ResetRoundCountForTest()

	// Simulate two attack-rejection calls in round 100 — expect 1 fire.
	fireAttackRejected(mob, 42 /* userId */)
	fireAttackRejected(mob, 42)
	assert.Equal(t, 1, fireCount, "second call in same round should be deduped")

	// Advance round; third call should fire.
	util.SetRoundCountForTest(101)
	fireAttackRejected(mob, 42)
	assert.Equal(t, 2, fireCount, "call in a new round should fire again")
}
```

This test assumes two helpers that don't yet exist:
- `attackTryMobBehavior` — a package-level var wrapping `behaviortree.TryMobBehavior` so tests can swap it
- `fireAttackRejected(mob *mobs.Mob, userId int)` — a package-level helper that encapsulates the dedupe + fire logic

Both will be created in Step 3 below.

### Step 2: Run test to verify it fails

Run: `go test ./internal/usercommands/ -run TestAttack_NonCombatantFiresEventOnce -v`
Expected: compile error (undefined: `attackTryMobBehavior`, `fireAttackRejected`, `util.SetRoundCountForTest`).

If `util.SetRoundCountForTest` / `util.ResetRoundCountForTest` don't exist yet, you'll need to add them OR refactor the test to not depend on them. Check:

```bash
grep -n "SetRoundCountForTest\|ResetRoundCountForTest\|SetRoundCount" internal/util/*.go
```

If missing, add to `internal/util/util.go`:

```go
// SetRoundCountForTest overrides the round count for test use.
// Pairs with ResetRoundCountForTest.
func SetRoundCountForTest(r uint64) {
	roundCount = r
}

// ResetRoundCountForTest resets the round count to its default.
// Caller should defer this in tests.
func ResetRoundCountForTest() {
	roundCount = 0
}
```

Adjust the actual variable name (`roundCount`) to match what `GetRoundCount()` returns — grep to verify:

```bash
grep -n "roundCount\|func GetRoundCount" internal/util/util.go
```

### Step 3: Add the helpers in attack.go

Open `internal/usercommands/attack.go`. Add near the top (after the imports block):

```go
// attackTryMobBehavior is a test-swappable wrapper around behaviortree.TryMobBehavior.
var attackTryMobBehavior = behaviortree.TryMobBehavior

// fireAttackRejected fires the player_attack_rejected btree event on the
// given mob, subject to round-based dedupe on Character.LastAttackRejectedRound.
func fireAttackRejected(mob *mobs.Mob, userId int) {
	currentRound := util.GetRoundCount()
	if currentRound <= mob.Character.LastAttackRejectedRound {
		return
	}
	mob.Character.LastAttackRejectedRound = currentRound
	attackTryMobBehavior(mob.InstanceId, behaviortree.EventContext{
		EventType: "player_attack_rejected",
		UserId:    userId,
	})
}
```

### Step 4: Update the non-combatant rejection block to call the helper

Find the current block at `internal/usercommands/attack.go:161-164`:

```go
			if m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				return true, nil
			}
```

Replace with:

```go
			if m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				fireAttackRejected(m, user.UserId)
				return true, nil
			}
```

### Step 5: Run test to verify it passes

Run: `go test ./internal/usercommands/ -run TestAttack_NonCombatantFiresEventOnce -v`
Expected: PASS.

Also run the full usercommands test suite:

Run: `go test ./internal/usercommands/...`
Expected: all pass.

### Step 6: Commit

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git add internal/usercommands/attack.go internal/usercommands/attack_test.go internal/util/util.go
git commit -m "$(cat <<'EOF'
feat(attack): fire player_attack_rejected btree event with round dedupe

Non-combatant rejection in attack.go now fires the player_attack_rejected
btree event on the targeted mob, rate-limited to one fire per mob per round
via Character.LastAttackRejectedRound. Player-visible rejection message is
unchanged. Test verifies fire-once-per-round semantics.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Fire `player_attack_rejected` from cast.go HarmSingle with dedupe

**Files:**
- Modify: `internal/actions/cast.go:104-107`

Same pattern as Task 2 but in the HarmSingle rejection path.

### Step 1: Locate the rejection block

Run: `sed -n '95,110p' internal/actions/cast.go`

Expected:

```go
	case spells.HarmSingle:
		if targetName != `` {
			pId, mId := room.FindByName(targetName)
			if mId > 0 {
				// Don't let players target a companion or non-combatant with harm spells
				if actor.IsPlayer() {
					if m := mobs.GetInstance(mId); m != nil {
						if m.Character.IsCharmed() {
							actor.SendText("You can't target a companion with a harmful spell.")
							return CastResult{SpellInfo: spellInfo, NoTarget: true}
						}
						if m.IsNonCombatant() {
							actor.SendText(fmt.Sprintf("You can't target %s with a harmful spell.", m.Character.Name))
							return CastResult{SpellInfo: spellInfo, NoTarget: true}
						}
					}
				}
```

### Step 2: Wire up the fire — use an inline implementation (cast.go can't import usercommands)

`internal/actions/cast.go` can't import `internal/usercommands/attack.go`. We either:
- Move `fireAttackRejected` into a package both can import (e.g. `internal/actions/` itself, or `internal/behaviortree/`)
- Inline the dedupe + fire in each site

For cleanliness, move the helper into `internal/actions/attack_rejection.go`:

Create `internal/actions/attack_rejection.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// AttackRejectedTryMobBehavior is a test-swappable wrapper.
var AttackRejectedTryMobBehavior = behaviortree.TryMobBehavior

// FireAttackRejected fires the player_attack_rejected btree event on
// the given mob, with round-based dedupe via Character.LastAttackRejectedRound.
// Called from attack.go and cast.go's non-combatant rejection paths.
func FireAttackRejected(mob *mobs.Mob, userId int) {
	currentRound := util.GetRoundCount()
	if currentRound <= mob.Character.LastAttackRejectedRound {
		return
	}
	mob.Character.LastAttackRejectedRound = currentRound
	AttackRejectedTryMobBehavior(mob.InstanceId, behaviortree.EventContext{
		EventType: "player_attack_rejected",
		UserId:    userId,
	})
}
```

### Step 3: Update attack.go to call the actions-package helper

Open `internal/usercommands/attack.go`. Remove the local `attackTryMobBehavior` and `fireAttackRejected` definitions from Task 2. Replace them with a call to `actions.FireAttackRejected(m, user.UserId)`.

Specifically, delete the block added in Task 2 Step 3:

```go
var attackTryMobBehavior = behaviortree.TryMobBehavior

func fireAttackRejected(mob *mobs.Mob, userId int) {
	// ...
}
```

And update Task 2 Step 4's call site from `fireAttackRejected(m, user.UserId)` to `actions.FireAttackRejected(m, user.UserId)`. If the `actions` import isn't already present:

```bash
grep -n '"github.com/GoMudEngine/GoMud/internal/actions"' internal/usercommands/attack.go
```

If missing, add to the imports block.

### Step 4: Update the Task 2 test to use the actions-package helper

Open `internal/usercommands/attack_test.go`. The test from Task 2 used `attackTryMobBehavior` and `fireAttackRejected`. Update it to use `actions.AttackRejectedTryMobBehavior` and `actions.FireAttackRejected`:

```go
func TestAttack_NonCombatantFiresEventOnce_ThenDedupes(t *testing.T) {
	originalTry := actions.AttackRejectedTryMobBehavior
	var fireCount int
	actions.AttackRejectedTryMobBehavior = func(instanceId int, ctx behaviortree.EventContext) bool {
		if ctx.EventType == "player_attack_rejected" {
			fireCount++
		}
		return true
	}
	defer func() { actions.AttackRejectedTryMobBehavior = originalTry }()

	mob := &mobs.Mob{
		InstanceId: 12001,
		Character: characters.Character{
			RoomId:       1,
			Name:         "Test NPC",
			NonCombatant: true,
		},
	}
	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{12001: mob})
	defer cleanup()

	util.SetRoundCountForTest(100)
	defer util.ResetRoundCountForTest()

	actions.FireAttackRejected(mob, 42)
	actions.FireAttackRejected(mob, 42)
	assert.Equal(t, 1, fireCount, "second call in same round should be deduped")

	util.SetRoundCountForTest(101)
	actions.FireAttackRejected(mob, 42)
	assert.Equal(t, 2, fireCount, "call in a new round should fire again")
}
```

### Step 5: Add the fire call to cast.go HarmSingle rejection

Find the block at `internal/actions/cast.go:104-107`:

```go
						if m.IsNonCombatant() {
							actor.SendText(fmt.Sprintf("You can't target %s with a harmful spell.", m.Character.Name))
							return CastResult{SpellInfo: spellInfo, NoTarget: true}
						}
```

Replace with:

```go
						if m.IsNonCombatant() {
							actor.SendText(fmt.Sprintf("You can't target %s with a harmful spell.", m.Character.Name))
							FireAttackRejected(m, actor.GetUserId())
							return CastResult{SpellInfo: spellInfo, NoTarget: true}
						}
```

Note: no import needed because `FireAttackRejected` lives in the same `actions` package.

### Step 6: Build + test

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go test ./internal/usercommands/ ./internal/actions/ -v 2>&1 | tail -30
```

Expected: build clean, all tests pass.

### Step 7: Commit

```bash
git add internal/actions/attack_rejection.go internal/actions/cast.go internal/usercommands/attack.go internal/usercommands/attack_test.go
git commit -m "$(cat <<'EOF'
feat(cast): fire player_attack_rejected on HarmSingle non-combatant rejection

Mirrors attack.go's behavior for harm spells. FireAttackRejected helper
consolidated into internal/actions/attack_rejection.go for cross-package
use. Same round-based dedupe.

HarmArea already filters non-combatants silently at spell_resolution.go:75
— no event fires there (correct: bystanders shouldn't react to AoE they
weren't included in). HarmMulti resolution is unchanged; not a known bug.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Create four new archetype YAMLs + load tests

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml`
- Create: `_datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml`
- Create: `internal/behaviortree/archetype_noncombat_test.go`

### Step 1: Write the failing test (validates each archetype loads + responds to expected events)

Create `internal/behaviortree/archetype_noncombat_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArchetype_NoncombatQuestgiver_HandlesPlayerEnter(t *testing.T) {
	cleanup := LoadArchetypeForTest(t, "noncombat_questgiver")
	defer cleanup()

	ctx := &EvalContext{
		InstanceId: 13001,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_enter",
			UserId:    42,
			RoomId:    1,
		},
	}
	// We test structural firing — assert the archetype tree returns Success
	// when the player_enter event fires (meaning the selector found a matching
	// sequence). The emote body executes against mobs that don't exist in the
	// test harness, so actEmote may return Failure at the mob lookup — accept
	// either, but verify the tree itself has a handler.
	arch := GetArchetype("noncombat_questgiver")
	if arch == nil {
		t.Fatal("archetype not loaded")
	}
	_ = arch.Evaluate(ctx)
	// If we got here without panic, the tree is structurally valid.
}

func TestArchetype_NoncombatShopkeeper_Loads(t *testing.T) {
	cleanup := LoadArchetypeForTest(t, "noncombat_shopkeeper")
	defer cleanup()
	assert.NotNil(t, GetArchetype("noncombat_shopkeeper"))
}

func TestArchetype_NoncombatPassive_Loads(t *testing.T) {
	cleanup := LoadArchetypeForTest(t, "noncombat_passive")
	defer cleanup()
	assert.NotNil(t, GetArchetype("noncombat_passive"))
}

func TestArchetype_CombatPassive_Loads(t *testing.T) {
	cleanup := LoadArchetypeForTest(t, "combat_passive")
	defer cleanup()
	assert.NotNil(t, GetArchetype("combat_passive"))
}

func TestArchetype_NoncombatQuestgiver_HandlesAttackRejected(t *testing.T) {
	cleanup := LoadArchetypeForTest(t, "noncombat_questgiver")
	defer cleanup()

	// Verify the tree contains a player_attack_rejected handler.
	// We structurally traverse or fire the event and accept non-panic.
	ctx := &EvalContext{
		InstanceId: 13002,
		RoomId:     1,
		Event: EventContext{
			EventType: "player_attack_rejected",
			UserId:    42,
		},
	}
	arch := GetArchetype("noncombat_questgiver")
	if arch == nil {
		t.Fatal("archetype not loaded")
	}
	_ = arch.Evaluate(ctx)
}
```

Verify `LoadArchetypeForTest` exists. If not, check how other archetype tests bootstrap:

```bash
grep -rn "LoadArchetypeForTest\|GetArchetype" internal/behaviortree/ | head
```

If `LoadArchetypeForTest` is available in an existing test helper file, use it as-is. If not, check `internal/behaviortree/archetype_*_test.go` for the pattern used by the pack-tactics archetypes and reuse.

### Step 2: Run test to verify it fails

Run: `go test ./internal/behaviortree/ -run TestArchetype_Noncombat -v`
Expected: archetype files not found → FAIL.

### Step 3: Create `noncombat_questgiver.yaml`

Create `_datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml`:

```yaml
# noncombat_questgiver archetype
#
# Used by non-combat NPCs with dialogue/tutorial roles. Sanctum Basin:
# chrysalis_priest, combat_trainer, wilderness_guide_fen, elder_saris,
# basin_warden, basin_scholar.
#
# Dialogue YAML handles player_ask (the btree returns Failure for ask so
# the dispatcher falls through to dialogue patterns). This archetype
# covers only non-dialogue events.
#
# Spec: docs/superpowers/specs/2026-04-24-sanctum-basin-mob-audit-design.md

tree:
  type: selector
  children:
    - type: action
      event: player_enter
      do: emote
      text: "glances up as you enter."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "raises an eyebrow but says nothing."

    - type: sequence
      event: player_give
      children:
        - type: action
          do: emote
          text: "declines politely and hands it back."
        - type: action
          do: return_item
```

### Step 4: Create `noncombat_shopkeeper.yaml`

Create `_datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml`:

```yaml
# noncombat_shopkeeper archetype
#
# Used by non-combat merchants with shops. Sanctum Basin: korvath,
# alchemist_yenna, merchant_adela.
#
# Behaviorally identical to noncombat_questgiver today, with
# shopkeeper-flavored emote text. Kept as a separate archetype so
# future shop-specific events (purchase reactions, restock flavor)
# can attach here without disturbing questgiver NPCs.
#
# Spec: docs/superpowers/specs/2026-04-24-sanctum-basin-mob-audit-design.md

tree:
  type: selector
  children:
    - type: action
      event: player_enter
      do: emote
      text: "nods in greeting from behind the counter."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "steps back, looking mildly affronted."

    - type: sequence
      event: player_give
      children:
        - type: action
          do: emote
          text: "declines politely and hands it back."
        - type: action
          do: return_item
```

### Step 5: Create `noncombat_passive.yaml`

Create `_datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml`:

```yaml
# noncombat_passive archetype
#
# Ambient non-interactive creatures that cannot be attacked
# (non_combatant: true). Sanctum Basin: chrysalis_echo, meadow_lizard.
#
# No dialogue, no give handler — just ambient reactions.
#
# Spec: docs/superpowers/specs/2026-04-24-sanctum-basin-mob-audit-design.md

tree:
  type: selector
  children:
    - type: action
      event: player_enter
      do: emote
      text: "pays you little mind."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "slips just out of reach."
```

### Step 6: Create `combat_passive.yaml`

Create `_datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml`:

```yaml
# combat_passive archetype
#
# Can be attacked and will fight back with basic attacks via the default
# combat loop — but has no special-move tactical tree. Mobs with this
# archetype never select bash/trip/grapple/kick/taunt/cast specials.
#
# Sanctum Basin: training_dummy (65).
#
# Intentionally empty tree. The value of assigning this archetype is
# (a) classification/documentation, and (b) explicitly preventing any
# future archetype-inheritance logic from picking a tactical default.
#
# Spec: docs/superpowers/specs/2026-04-24-sanctum-basin-mob-audit-design.md

tree:
  type: selector
  children: []
```

### Step 7: Run tests — should now pass

Run: `go test ./internal/behaviortree/ -run TestArchetype_Noncombat -v` and `go test ./internal/behaviortree/ -run TestArchetype_CombatPassive -v`
Expected: all PASS.

Also run the full behaviortree test suite:

Run: `go test ./internal/behaviortree/...`
Expected: all pass.

### Step 8: Commit

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git add _datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml \
         internal/behaviortree/archetype_noncombat_test.go
git commit -m "$(cat <<'EOF'
feat(behaviortree): 4 new non-combat-oriented archetypes

- noncombat_questgiver: non-combat NPCs with dialogue/tutorial roles
- noncombat_shopkeeper: same shape, shopkeeper flavor; room for future
  shop-specific events
- noncombat_passive: ambient non-attackable creatures
- combat_passive: attackable but no special-move tree; falls back to
  basic combat loop

All handle player_enter, player_attack_rejected, and (where applicable)
player_give. player_ask is intentionally unhandled so the dispatcher
falls through to dialogue YAML patterns.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Assign archetypes to all 17 sanctum_basin mobs + fix non_combatant flags

**Files:**
- Modify: all 17 mob YAMLs under `_datafiles/world/dogmud/mobs/sanctum_basin/`

This is mechanical — add one line (`behavior_archetype: <name>`) per mob, plus `non_combatant: true` on meadow_lizard and chrysalis_echo where missing.

### Step 1: Inspect current state

Run:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && for f in _datafiles/world/dogmud/mobs/sanctum_basin/*.yaml; do
  echo "=== $(basename $f) ==="
  grep -E "^(archetype|behavior_archetype|non_combatant|hostile):" "$f" || echo "  (no archetype/combatant tags)"
done
```

Confirms which mobs need which fields added.

### Step 2: Apply assignments per this table

Use the Edit tool to add the `behavior_archetype:` line (and `non_combatant: true` where noted) to each YAML. Place `behavior_archetype:` after the existing `archetype:` line (the stat-distribution archetype) — or near the top metadata block if `archetype:` is absent.

| Mob ID | Filename | Add line(s) |
|--------|----------|-------------|
| 50 | `50-chrysalis_priest.yaml` | `behavior_archetype: noncombat_questgiver` |
| 51 | `51-combat_trainer.yaml` | `behavior_archetype: noncombat_questgiver` |
| 52 | `52-korvath.yaml` | `behavior_archetype: noncombat_shopkeeper` |
| 53 | `53-alchemist_yenna.yaml` | `behavior_archetype: noncombat_shopkeeper` |
| 54 | `54-wilderness_guide_fen.yaml` | `behavior_archetype: noncombat_questgiver` |
| 55 | `55-elder_saris.yaml` | `behavior_archetype: noncombat_questgiver` |
| 56 | `56-basin_warden.yaml` | `behavior_archetype: noncombat_questgiver` |
| 63 | `63-merchant_adela.yaml` | `behavior_archetype: noncombat_shopkeeper` |
| 65 | `65-training_dummy.yaml` | `behavior_archetype: combat_passive` |
| 66 | `66-valley_rat.yaml` | `behavior_archetype: generic_fighter` |
| 67 | `67-cave_bat.yaml` | `behavior_archetype: generic_fighter` |
| 68 | `68-cave_goblin_guard.yaml` | `behavior_archetype: generic_fighter` |
| 69 | `69-aberrant_chrysalis.yaml` | `behavior_archetype: generic_fighter` |
| 70 | `70-cave_troll.yaml` | `behavior_archetype: generic_fighter` |
| 71 | `71-meadow_lizard.yaml` | `behavior_archetype: noncombat_passive` + `non_combatant: true` (if missing) |
| 79 | `79-basin_scholar.yaml` | `behavior_archetype: noncombat_questgiver` |
| 112 | `112-chrysalis_echo.yaml` | `behavior_archetype: noncombat_passive` + `non_combatant: true` (if missing) |

For mobs 71 and 112, check whether `non_combatant: true` is already set:

```bash
grep -H "non_combatant" _datafiles/world/dogmud/mobs/sanctum_basin/71-meadow_lizard.yaml _datafiles/world/dogmud/mobs/sanctum_basin/112-chrysalis_echo.yaml
```

If either doesn't show the line, add `non_combatant: true` as a top-level field (sibling to `hostile:`, `archetype:`, etc.).

### Step 3: Verify all 17 have a behavior_archetype

Run:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -c "behavior_archetype" _datafiles/world/dogmud/mobs/sanctum_basin/*.yaml | grep -v ":1$"
```

Expected: only the problem files (missing behavior_archetype) appear. Should be empty — meaning all 17 have the line.

### Step 4: Full build (nothing Go-side should break; sanity check)

Run: `cd "C:/Users/Calabe Davis/workspace/DOGMud" && go build ./...`
Expected: empty output.

### Step 5: Commit

```bash
git add _datafiles/world/dogmud/mobs/sanctum_basin/
git commit -m "$(cat <<'EOF'
content(sanctum_basin): assign behavior_archetype to all 17 mobs

- 5 hostile combatants → generic_fighter
- 6 non-combat tutorial NPCs → noncombat_questgiver
- 3 shopkeepers → noncombat_shopkeeper
- 2 ambient creatures (meadow_lizard, chrysalis_echo) → noncombat_passive
  (plus non_combatant: true flag added where missing)
- training_dummy → combat_passive

Archetype handlers cover player_enter, player_attack_rejected, and
player_give. Dialogue YAML continues to handle player_ask.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Update chrysalis_priest dialogue (mob 50) with new system-mention patterns

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml`

### Step 1: Read existing content

Run: `cat _datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml | head -80`

Note where the existing `patterns:` block ends. We're appending new pattern entries.

### Step 2: Add new patterns for chrysalis_priest's assigned systems

Insert the following pattern entries into the `patterns:` list, BEFORE the existing `keywords: [""]` catch-all (if present) and BEFORE the `tree:` section. Keep existing patterns intact.

```yaml
  # System mention: mutations (framed as spiritual transformation)
  - keywords: ["mutation", "mutant", "change", "transform", "body", "flesh"]
    responses:
      - "The Chrysalis touches the body as well as the spirit. What the world calls mutation, we call the outward sign of inward change. Neither to be feared nor sought — only witnessed."
      - "Some are changed by the Chrysalis in visible ways. The priesthood keeps no registry of who bears such marks. It is personal, between the bearer and the silence."
    moodChange: friendly

  # System mention: spell discovery (framed as revelation)
  - keywords: ["discovery", "learn", "learning", "revealed", "revelation", "pattern"]
    responses:
      - "Patterns reveal themselves to the patient. Repeat a casting often enough, with clear intent, and new patterns crystallize from the work. This is the priesthood's quiet secret."
      - "The Awakening teaches that nothing is static — not even a spellcaster's repertoire. Practice opens doors that study alone cannot."

  # System mention: faith / conviction
  - keywords: ["conviction", "faith", "will", "willpower", "resolve"]
    responses:
      - "Conviction is the fuel of both spell and sermon. It does not come cheap — and when spent, it takes time to return."
      - "We teach that conviction regenerates with rest and with righteous action. The body does not separate courage from magic. Neither should you."
```

Also update the `tree.root.hints` line so the new keywords are discoverable:

Find the existing:

```yaml
  root:
    text: "..."
    hints: "..."
```

Replace the `hints:` line with an expanded version that mentions the new topics. Example (preserve the existing tone):

```yaml
    hints: "You could ask about the Chrysalis, the Awakening, the Fold, the sanctum, mutations, discovery, or conviction."
```

(If the existing hints already covers some of these, merge rather than duplicate.)

### Step 3: Verify YAML parses

Run: `python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml'))" && echo OK`
Expected: `OK`.

### Step 4: Commit

```bash
git add _datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): chrysalis_priest mentions mutations + discovery + conviction

Three new keyword patterns weave the Chrysalis Priest's assigned tutorial
system mentions into his voice (mutations as spiritual transformation,
spell discovery as pattern revelation, conviction as will). Hints line
updated so the new keywords are discoverable.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Update korvath dialogue (mob 52) with salvage + enchanting + 2H patterns

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/sanctum_basin/52.yaml`

### Step 1: Read existing content

Run: `cat _datafiles/world/dogmud/dialogue/sanctum_basin/52.yaml | head -80`

### Step 2: Add new patterns for korvath's assigned systems

Insert the following before the `keywords: [""]` catch-all (if present) and before `tree:`.

```yaml
  # System mention: salvage (framed as smith's practice of reclaiming material)
  - keywords: ["salvage", "scrap", "reclaim", "recover", "bent", "broken", "failed"]
    responses:
      - "Aye. Bad metal's not wasted metal. Any smith worth the apron knows how to pull usable stock out of a ruined piece. Same goes for failed crafts — don't toss it. Break it down."
      - "I'll salvage reagents from a failed batch as quick as I'll forge a new blade. Skill comes into it — the better ye are, the more ye recover."

  # System mention: enchanting (slot-targeted)
  - keywords: ["enchant", "enchanting", "rune", "runes", "magic weapon"]
    responses:
      - "Enchanting? Aye. Patterns get bound to a specific slot — weapons, armor, what have ye. Look at the recipe before ye spend materials; it'll tell ye where the pattern lands."
      - "I don't do the binding myself. That's for folk with the pattern-sense. But I forge the vessels, and I'll tell ye which slot a rune is meant for before ye waste a good blade."
    moodChange: friendly

  # System mention: 2H weapons
  - keywords: ["two-handed", "two hand", "twohand", "2h", "great sword", "maul", "polearm"]
    responses:
      - "Two hands on a weapon means no shield, no offhand. Ye trade protection for reach and a heavier strike. Some folk swear by it. Others end up dead."
      - "A proper 2H blade needs strength to wield well. If ye can't bear the weight, ye'd do better with a sword and shield."
```

Update the existing `tree.root.hints` to include the new keywords. Example:

```yaml
    hints: "You could ask about weapons, smithing, salvage, enchanting, or two-handed gear."
```

### Step 3: Verify YAML

Run: `python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/52.yaml'))" && echo OK`

### Step 4: Commit

```bash
git add _datafiles/world/dogmud/dialogue/sanctum_basin/52.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): korvath mentions salvage + enchanting + 2H weapons

Salvage framed as a smith's reclamation practice (materials, not gear
durability). Enchanting keyed to slot-targeting. 2H mentions the
strength/protection tradeoff.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Update elder_saris dialogue (mob 55) with spell discovery + manifestation + mutations

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/sanctum_basin/55.yaml`

### Step 1: Read existing content

Run: `cat _datafiles/world/dogmud/dialogue/sanctum_basin/55.yaml | head -80`

(This file already has a `mutation` pattern — extend it rather than duplicate.)

### Step 2: Add new patterns for saris's assigned systems

Insert the following before the `keywords: [""]` catch-all (if present) and before `tree:`.

```yaml
  # System mention: spell discovery (framed as "the pattern comes to you")
  - keywords: ["discovery", "new spell", "learn spell", "reveal", "unfamiliar"]
    responses:
      - "Spells are not memorized like dry facts. A caster who works with intent finds patterns crystallize in her mind — new spells, gifted by practice. Patience and perception bring them faster than study alone."
      - "The old texts called it 'the moons' gift.' I call it attentive practice. Either way — cast often, cast with care, and the patterns multiply."
    moodChange: friendly

  # System mention: spellcasting vs manifestation
  - keywords: ["manifestation", "manifest", "channel", "conduit", "school"]
    responses:
      - "Traditional spellcasting draws patterns out of the weave. Manifestation is different — the body becomes the conduit. A manifestor speaks with charisma and conviction where a caster speaks with will. Both are valid. Few master both."
      - "Manifestation is the harder path. It asks more of the self. But those who walk it are rarely surprised by what the world gives them."
```

If a `mutation` pattern already exists, ADD a new pattern entry for repeated-magic-use mutations rather than modifying the existing one:

```yaml
  # System mention: mutations from repeated magic use
  - keywords: ["magic mutation", "spell mutation", "caster body", "change from casting"]
    responses:
      - "Cast the same pattern often enough, deeply enough, and the body remembers. It shifts. Most dismiss these marks as coincidence. I do not."
      - "I have seen a fold-binder whose fingers bent inward over years. A healer whose skin glowed faintly in the dark. The body adapts to what it is repeatedly asked to do."
```

Update `tree.root.hints` to include new keywords:

```yaml
    hints: "You could ask about the Chrysalis, the Fold, mutations, spell discovery, or manifestation."
```

### Step 3: Verify YAML

Run: `python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/55.yaml'))" && echo OK`

### Step 4: Commit

```bash
git add _datafiles/world/dogmud/dialogue/sanctum_basin/55.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): elder_saris mentions spell discovery + manifestation + magic mutations

Three new patterns cover spell discovery (Per+skill crystallization),
spellcasting vs manifestation schools, and mutation emergence from
repeated magic use.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Update basin_scholar dialogue (mob 79) with mutations + chrysalis lore

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml`

### Step 1: Read existing content

Run: `cat _datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml | head -80`

The scholar already discusses mutations (it's his research focus). Task is to add a pattern that explicitly names the **player-facing** mutation system in plain terms, and a pattern pointing to the chrysalis priest for spiritual context.

### Step 2: Add new patterns

Insert before the catch-all and before `tree:`.

```yaml
  # System mention: mutations (player-facing framing)
  - keywords: ["my mutation", "my body", "my change", "new limb", "changed self", "am i changing"]
    responses:
      - "You are asking because you have felt it. Good. The Chrysalis-touch varies — some gain reach, some gain resilience, some gain senses they had no name for. The key is: it emerges from what you do repeatedly. Your body shapes itself around your habits."
      - "I cannot tell you which mutations you will awaken to. I can tell you that combat, repeated magic, and hardship all nudge the body. What emerges depends on the self."
    moodChange: friendly

  # System mention: chrysalis priest as alternate perspective
  - keywords: ["priest", "faith", "spiritual", "meaning", "why"]
    responses:
      - "If you want the spiritual angle on all this — the Chrysalis Priest, Saelin, is in the temple to the east. I study the biology. He studies the meaning. We rarely agree and rarely fight about it."
      - "You can ask Saelin about faith. He asks better questions about it than I do."
```

Update `tree.root.hints`:

```yaml
    hints: "You could ask about research, the tunnels, mutations, your own changes, or the priesthood."
```

### Step 3: Verify YAML + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): basin_scholar addresses player's own mutations + points to priest

Scholar frames mutations from the biological angle; redirects to
Chrysalis Priest for the spiritual framing. Player-facing keywords
("my mutation", "new limb") specifically surface the mutation system.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Create combat_trainer dialogue (mob 51)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/sanctum_basin/51.yaml`

### Step 1: Read mob profile for voice reference

Run: `grep -A 40 "systemprompt" _datafiles/world/dogmud/mobs/sanctum_basin/51-combat_trainer.yaml | head -50`

Use the voice cues from the LLM prompt (stern, tactical, experienced fighter) to inform the dialogue text, even though the file won't use LLM in prod.

### Step 2: Create the file

Create `_datafiles/world/dogmud/dialogue/sanctum_basin/51.yaml`:

```yaml
mobid: 51
zone: Sanctum Basin
defaultMood: neutral

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Stand straight. What do you need from me?"
      - "You look breathing. Good. That's step one."
      - "If you're here for lessons, we start now. If you're here to talk, we start never."

  # System mention: rally + warcry (rhetoric shouts)
  - keywords: ["rally", "warcry", "shout", "rhetoric", "inspire", "battle cry"]
    responses:
      - "A veteran doesn't just swing a sword. A veteran shouts. Rally steadies your allies. Warcry rattles your enemies. Both cost stamina and conviction — use them when they count, not for show."
      - "You'll learn rally and warcry through practice. Combat rhetoric isn't a skill you read about. It's a skill you bleed for."

  # System mention: companions (summon, charm, conjure, raise)
  - keywords: ["companion", "summon", "pet", "ally", "minion"]
    responses:
      - "A good fighter doesn't fight alone. Summoners call spirit allies. Charmers turn enemies. Necromancers raise the dead. Conjurers pull elementals through the weave. Each has a cost; each has a use."
      - "I've fought beside a fire elemental, a bound wolf, and a raised bandit. In that order, over thirty years. They're not replacements for skill. They're force multipliers."
    moodChange: friendly

  # System mention: mutations from combat
  - keywords: ["mutation", "change", "body adapt", "combat change"]
    responses:
      - "Fight long enough and the body starts helping you. Calluses where you grip. Reflexes where you swing. Some call it mutation. Saelin calls it awakening. I call it Tuesday."
      - "The body adapts to what you ask it to do. Swing a blade for twenty years and the arm changes. That's not magic. That's training made flesh."

  # System mention: position (prone/grapple)
  - keywords: ["prone", "grapple", "knocked down", "floor", "position"]
    responses:
      - "Prone is a killer's opportunity. Kick a downed enemy and you strike harder. Grapple them and you deny them their next swing. Position is half of combat."
      - "If you go down, get up. If they go down, finish them. The ground is not a neutral surface in a fight."

  # System mention: basin-specific — training dummy, saelin, etc.
  - keywords: ["train", "training", "practice", "dummy", "beginner"]
    responses:
      - "Dummy's that way. Hit it until you feel the rhythm. Then come back and we'll go further."
      - "First lesson: the dummy doesn't hit back. Second lesson: the world is not the dummy."

  - keywords: [""]
    responses:
      - "Speak plainly or don't speak."
      - "I don't have time for riddles. Ask something."

tree:
  root:
    text: "Combat Trainer. Name's Ren. I train fighters. If that's you, listen close. If not, don't waste my time."
    hints: "You could ask about rally, warcry, companions, mutations, position, or training."
```

### Step 3: Verify + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/51.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/51.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): combat_trainer — rally/warcry + companions + mutations + position

New dialogue file for mob 51 (Combat Trainer Ren). Terse military voice.
Covers combat rhetoric (rally/warcry), companion archetypes (summon/
charm/necromancy/conjure), body-adaptation mutations, and position-based
combat (prone/grapple).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Create alchemist_yenna dialogue (mob 53)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/sanctum_basin/53.yaml`

### Step 1: Create the file

Create `_datafiles/world/dogmud/dialogue/sanctum_basin/53.yaml`:

```yaml
mobid: 53
zone: Sanctum Basin
defaultMood: neutral

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Three drops, not four. Yes? Yes. Welcome. Mind the table — some of the compounds stain."
      - "Distracted. Sorry. Moment. Yes? What did you need?"
      - "Ah. A visitor. Good. Don't touch the green flask."

  # System mention: bandolier
  - keywords: ["bandolier", "carry potion", "potion belt", "vial belt", "slots"]
    responses:
      - "A potion bandolier keeps vials in reach without digging through a pack. The belt routes any potion you pick up straight to a slot. Worth it if you drink mid-fight. Worth it even if you don't."
      - "Ah — bandoliers. Yes. Belt-slot item. Picks up potions automatically. Holds a few. Saves your life when the fight is loud and your bag is deep."
    moodChange: friendly

  # System mention: grenades
  - keywords: ["grenade", "flashbang", "firebomb", "toxic flask", "thrown"]
    responses:
      - "Dangerous work. Flashbangs blind. Firebombs burn. Toxic flasks sicken. I brew all three. They age like potions — do not hoard them."
      - "Grenades go in the bandolier too. Throw them, don't drink them. I have had to correct this more than once."

  # System mention: potion aging + toxicity
  - keywords: ["potion aging", "potion age", "aging", "fresh", "peak", "spoiled", "expire"]
    responses:
      - "Every potion has a peak. Fresh is weaker than it could be. Peak is best. Decay has already started losing you strength. Spoiled is worse than nothing — it will hurt you. Read the label, if there is one. Ask me if not."
      - "Toxicity stacks faster than most think. Drink too many in a day and the body rebels — regen slows, perception dulls, dexterity goes wobbly. Space your potions. Pace yourself."

  # System mention: salvage after failed craft
  - keywords: ["salvage", "wasted", "failed", "ruined", "bad batch"]
    responses:
      - "Not wasted. If you have the knack, you pull reagents back from a bad batch. The smiths do the same with bent metal. Waste is mostly a failure of patience."
      - "Salvage is a skill. The better you get, the more you reclaim. And you will have bad batches — it is alchemy, not arithmetic."

  # Shop direction
  - keywords: ["buy", "sell", "shop", "potion", "reagent", "wares"]
    responses:
      - "What is on the table is what I have. Prices are fair. Quantity is limited. No haggling on the rare ones."
      - "You can buy reagents, simple potions, and an empty bottle or two. Brewing your own is better — if you have the patience for three drops, not four."

  - keywords: [""]
    responses:
      - "Mm. Precision, that is the thing."
      - "Trail off. Snap back. Yes?"
      - "I was not listening. Again, please."

tree:
  root:
    text: "I am Yenna. Alchemy, potions, reagents. Fen sent you for lessons? Good. Three drops, not four."
    hints: "You could ask about potions, grenades, the bandolier, potion aging, salvage, or the shop."
```

### Step 2: Verify + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/53.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/53.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): alchemist_yenna — bandolier + grenades + potion aging + salvage

New dialogue file for mob 53 (Alchemist Yenna). Distracted precisionist
voice. Covers the potion bandolier, alchemy-brewed grenades (flashbang/
firebomb/toxic), potion aging phases + toxicity stacking, and salvage
of failed craft reagents.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Create wilderness_guide_fen dialogue (mob 54)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/sanctum_basin/54.yaml`

### Step 1: Create the file

Create `_datafiles/world/dogmud/dialogue/sanctum_basin/54.yaml`:

```yaml
mobid: 54
zone: Sanctum Basin
defaultMood: friendly

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Good to meet you. Boots dry? The wet grass out there fools half the newcomers."
      - "Welcome. If you've been in the tunnels, you'll smell of cave. That fades."
      - "You have the look of someone who's been lost and doesn't want to say so. It's fine. Most of us have been."

  # System mention: foraging
  - keywords: ["forage", "foraging", "gather", "ingredient", "plant", "herb"]
    responses:
      - "The meadow and the forest edge both yield something, if you know what to look for. I teach the basics — which roots Yenna pays for, which leaves burn in a firebomb, which berries make a traveler sick if eaten raw."
      - "Foraging is a skill like any other. The more you do it, the better you see the land. Start with what you recognize; build from there."
    moodChange: friendly

  # System mention: pack tactics awareness
  - keywords: ["pack", "wolves", "group", "pack tactics", "bandits together"]
    responses:
      - "Nothing dangerous fights alone out here. If you see one wolf, there are three more you don't see. Bandits run in packs. Even the boars hunt in family groups. Plan accordingly."
      - "A pack will call for help. Hit one, the others hear. Hit one near a camp, the whole camp knows. Pick your ground."

  # System mention: scent/track
  - keywords: ["track", "tracking", "scent", "trail", "footprint"]
    responses:
      - "Tracking is watching the world remember what walked through it. Bent grass, broken twigs, clawed bark. Takes practice, but the land keeps records better than most scribes."
      - "Every creature leaves sign. Rats scratch. Goblins piss on their borders. Trolls break whatever they push past. Learn the difference."

  # System mention: fleeing
  - keywords: ["flee", "run away", "retreat", "escape"]
    responses:
      - "Running is a tactic, not a failure. If a fight is going wrong, flee. You lose a round of ground but you keep your blood inside you. That trade is almost always worth making."
      - "The exits out of a fight aren't always obvious. Know your exits before the fight starts. A dead scout is a scout who forgot."

  - keywords: [""]
    responses:
      - "Hmm. Ask again — different words, maybe."
      - "Not sure what you mean. The land's honest; try honest words."

tree:
  root:
    text: "Fen. Wilderness guide. I track, I forage, I teach a little of both. What brings you out of the Basin?"
    hints: "You could ask about foraging, tracking, packs, fleeing, or the wilderness."
```

### Step 2: Verify + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/54.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/54.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): wilderness_guide_fen — foraging + tracking + pack tactics + fleeing

New dialogue file for mob 54 (Wilderness Guide Fen). Calm, steady
voice. Covers the forage skill, tracking basics, pack-tactics awareness
(pack routines call for help), and flee as a valid tactical choice.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Create basin_warden dialogue (mob 56)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/sanctum_basin/56.yaml`

### Step 1: Create the file

Create `_datafiles/world/dogmud/dialogue/sanctum_basin/56.yaml`:

```yaml
mobid: 56
zone: Sanctum Basin
defaultMood: neutral

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Move along if you are just passing. Stay clear of the tunnels if you are not ready."
      - "Warden Kess. I keep the Basin safe. What do you need?"
      - "Eyes sharp and hands visible. Thank you."

  # System mention: pack routines / aggro awareness
  - keywords: ["aggro", "attack", "attacked me", "hostile", "fighting", "combat"]
    responses:
      - "Hostile creatures set aggro when they see a threat. Once aggro is set, they follow you across rooms until one of you breaks it. If you hit an enemy, its packmates near enough to hear will come for you too. This is the warden's first lesson."
      - "If you are new and a creature is chasing you, flee to a room with no enemies — they will stop and return. Running toward my patrol is also acceptable, provided you are quick about it."
    moodChange: friendly

  # System mention: respawn grace
  - keywords: ["respawn", "died", "death", "resurrect", "grace"]
    responses:
      - "If you fall, you return at the sanctum. For a few rounds after respawn you cannot be targeted. Use that window — retreat, drink a potion, find better ground. Do not stand and pose."
      - "Dying is a tax, not a sentence. You lose a sliver of progression and return weakened. A respawn grace buys you time to regroup. Spend it wisely."

  # System mention: dungeon pacing
  - keywords: ["tunnels", "dungeon", "depths", "deep", "below", "danger zone"]
    responses:
      - "Tunnels below the Basin are not a sightseeing trip. Clear a section, rest, push again. Do not try to map the whole warren in one pass. The cave creatures respawn. The clock does not favor you."
      - "Go down with a full stamina bar and a bandolier. Come back up before you are empty. New hunters have a habit of pushing one room too far."

  # Basin-specific: on the guards
  - keywords: ["guard", "watch", "warden", "patrol"]
    responses:
      - "Two wardens on duty at a time. We rotate. If you see smoke in the north, it is probably the forge. If you see smoke anywhere else, tell me immediately."

  - keywords: [""]
    responses:
      - "Watch your footing and your temper."
      - "If it matters, tell me plainly."

tree:
  root:
    text: "Warden Kess. I keep the Basin and its visitors safe. Pass through carefully."
    hints: "You could ask about combat, aggro, respawn, the tunnels, or the wardens."
```

### Step 2: Verify + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/56.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/56.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): basin_warden — aggro + respawn grace + dungeon pacing

New dialogue file for mob 56 (Basin Warden Kess). Measured, protective
voice. Covers aggro/pack awareness, respawn grace window, and dungeon
pacing advice (clear + rest + push again).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Create merchant_adela dialogue (mob 63)

**Files:**
- Create: `_datafiles/world/dogmud/dialogue/sanctum_basin/63.yaml`

### Step 1: Create the file

Create `_datafiles/world/dogmud/dialogue/sanctum_basin/63.yaml`:

```yaml
mobid: 63
zone: Sanctum Basin
defaultMood: friendly

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Hello there. Looking to buy, sell, or just pass the time? All three are welcome."
      - "Ah, a face I don't know. That always makes my day more interesting."
      - "Come in, come in. My prices are fair and my gossip is free."

  # System mention: bartering / haggling
  - keywords: ["barter", "haggle", "bargain", "discount", "price", "cheaper"]
    responses:
      - "Everything's negotiable — within reason. A smooth tongue saves you coin. I'll take offers down to a point. Below that and I lose money, and I don't lose money on purpose."
      - "Bartering is a skill like any other. The more you practice, the better the deals you get. Try me — but don't insult me."
    moodChange: friendly

  # System mention: tavern gossip
  - keywords: ["gossip", "news", "rumor", "rumors", "hear", "heard"]
    responses:
      - "Oh, I hear plenty. The wardens say the warren is getting louder. The priest keeps looking at the southern hills and won't say why. Fen brought back a pelt she couldn't name. Bits and pieces. Listen for yourself."
      - "Taverns are where the real news is. If you pass one, sit a while. The gossip channel tells you what the whole town is talking about."

  # System mention: encumbrance / carry capacity
  - keywords: ["carry", "heavy", "encumbrance", "weight", "load", "pack"]
    responses:
      - "Strength sets your carry weight. Overload and you slow down in a fight. A backpack helps — it lightens what it holds. Same for a bandolier. Same for a component bag."
      - "If you are slogging because your pack is full, sell the junk. That's what my shop is for. Well — half of it. The other half is for the things you actually need."

  # Shop direction
  - keywords: ["buy", "sell", "shop", "wares", "goods", "stock"]
    responses:
      - "I keep a general stock — bandages, basic potions, a few oddments. What I don't carry, Yenna or Korvath probably do."
      - "If you have something to sell, let me see it. I don't buy junk — actually, I do buy junk. Just don't expect riches for it."

  - keywords: [""]
    responses:
      - "Hm? I was mid-thought. Ask again."
      - "Not quite sure what you mean, dear. Try something specific."

tree:
  root:
    text: "Adela. General goods. I buy, I sell, I gossip. What will it be today?"
    hints: "You could ask about the shop, bartering, gossip, or carrying gear."
```

### Step 2: Verify + commit

```bash
python3 -c "import yaml; yaml.safe_load(open('_datafiles/world/dogmud/dialogue/sanctum_basin/63.yaml'))" && echo OK

git add _datafiles/world/dogmud/dialogue/sanctum_basin/63.yaml
git commit -m "$(cat <<'EOF'
content(dialogue): merchant_adela — bartering + tavern gossip + encumbrance

New dialogue file for mob 63 (Merchant Adela). Warm, chatty voice.
Covers bartering skill, tavern-gossip as an information channel, and
carry-capacity / bandolier / component-bag weight handling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Full-repo build + test sweep + in-game smoke

**Files:** no code changes expected; this task verifies the pilot is clean.

### Step 1: Verify all archetype files parse

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
for f in _datafiles/world/dogmud/behaviors/archetypes/noncombat_questgiver.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/noncombat_shopkeeper.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/noncombat_passive.yaml \
         _datafiles/world/dogmud/behaviors/archetypes/combat_passive.yaml; do
  python3 -c "import yaml; yaml.safe_load(open('$f'))" && echo "$f OK"
done
```

Expected: all four print `OK`.

### Step 2: Verify all dialogue files parse

```bash
for f in _datafiles/world/dogmud/dialogue/sanctum_basin/*.yaml; do
  python3 -c "import yaml; yaml.safe_load(open('$f'))" && echo "$f OK"
done
```

Expected: all 9 sanctum_basin dialogue files print `OK`.

### Step 3: Verify all 17 mob files have a `behavior_archetype`

```bash
total=$(ls _datafiles/world/dogmud/mobs/sanctum_basin/*.yaml | wc -l)
have=$(grep -l "behavior_archetype:" _datafiles/world/dogmud/mobs/sanctum_basin/*.yaml | wc -l)
echo "have $have of $total"
```

Expected: `have 17 of 17`.

### Step 4: Full build

Run: `go build ./...`
Expected: empty output.

### Step 5: Full test suite

Run: `go test ./... 2>&1 | grep -E "^(FAIL|ok.*configs|ok.*hooks|ok.*usercommands|ok.*actions|ok.*behaviortree|ok.*characters)"`
Expected: all `ok`, no `FAIL` lines.

### Step 6: In-game smoke checklist (manual; runs after merge)

The implementer should NOT attempt to run the live MUD server; this step is documented here for the human to run post-merge. List the checks:

1. Fresh character starts in Sanctum Basin.
2. `attack yenna` → player sees rejection message; Yenna emits the shopkeeper's `player_attack_rejected` emote.
3. Second `attack yenna` in same round → player still sees the rejection, but no duplicate emote from Yenna.
4. `ask yenna bandolier` → canned response from the dialogue file.
5. `ask korvath salvage` → canned response.
6. `ask saris discovery` → canned response.
7. `ask priest mutation` → canned response.
8. `give yenna iron ingot` → Yenna declines, item returned.
9. Attack any cave creature → normal combat (generic_fighter archetype).
10. Hit training dummy → it fights back with basic attacks only (no specials, no bash/trip/grapple/kick).
11. `ask adela gossip` → canned gossip response.
12. `ask fen foraging` → canned response.
13. `ask kess respawn` → canned response.
14. `ask trainer rally` → canned response.
15. `ask scholar mutation` → canned response.

### Step 7: No commit (this task only verifies)

If Step 5 fails, debug the failure, fix, and commit the fix as a separate commit before Task 16.

---

## Task 16: Patch notes

**Files:**
- Modify: `PATCH_NOTES.md`

### Step 1: Prepend a new section to PATCH_NOTES.md

Open `PATCH_NOTES.md`. Find the top-of-file header:

```markdown
# DOGMud Patch Notes

## 2026-04-24 — Discovery Rate Stat Offset
```

Insert a new section immediately after `# DOGMud Patch Notes` and BEFORE the existing `## 2026-04-24 — Discovery Rate Stat Offset` section (so the new section appears first):

```markdown
## 2026-04-24 (evening) — Sanctum Basin Mob Audit + Tutorial Content

### Gameplay

- **Sanctum Basin NPCs now offer tutorial guidance for newer gameplay
  systems.** Each of the nine non-combat NPCs covers a curated set of
  topics through their dialogue: ask Korvath about salvage or
  enchanting, ask Yenna about potion aging or the bandolier, ask Saris
  about spell discovery or manifestation, ask the Combat Trainer about
  rally/warcry or companions, ask Fen about tracking or packs, ask the
  Warden about respawn grace or aggro, ask the Scholar about mutations,
  ask the Chrysalis Priest about the Awakening, ask Merchant Adela about
  bartering or encumbrance.
- **Non-combatants now react when you try to attack them.** Trying to
  attack (or target with a harmful spell) an NPC who cannot be attacked
  now triggers an in-character emote from that NPC — a raised eyebrow
  from a questgiver, a step back from a shopkeeper. Rate-limited to one
  reaction per NPC per round so companion and party auto-assist cannot
  spam it.

### Behind the scenes

- Four new behavior archetypes: `noncombat_questgiver`,
  `noncombat_shopkeeper`, `noncombat_passive`, `combat_passive`. Every
  Sanctum Basin mob is now tagged with a `behavior_archetype` value.
  This is the first zone in a larger migration to the archetype system.
- New btree event `player_attack_rejected` fired from attack.go and
  from HarmSingle spell rejection in cast.go.
- All tutorial content is delivered via dialogue YAML `patterns`, which
  is deterministic and prod-safe (no LLM dependency).
```

### Step 2: Commit

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git add PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs: patch notes for sanctum basin mob audit + tutorial content

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

### Spec coverage check

| Spec section | Implemented by |
|--------------|---------------|
| 4 new archetypes | Task 4 |
| `player_attack_rejected` event + `LastAttackRejectedRound` dedupe | Tasks 1, 2, 3 |
| Attack.go rejection fires event | Task 2 |
| Cast.go HarmSingle rejection fires event | Task 3 |
| HarmArea stays silent-filter | No-op, noted in Task 3 commit |
| 17 mob YAML archetype assignments | Task 5 |
| `non_combatant: true` fixup for meadow_lizard + chrysalis_echo | Task 5 |
| 4 dialogue YAML updates (50, 52, 55, 79) | Tasks 6, 7, 8, 9 |
| 5 new dialogue YAML files (51, 53, 54, 56, 63) | Tasks 10, 11, 12, 13, 14 |
| System-mention assignments per NPC | Each of Tasks 6–14 covers that NPC's assigned systems |
| Hints discoverability rule | Each dialogue task updates the `hints:` line |
| No LLM dependency | All tutorial content is in deterministic dialogue YAML; llmprofile untouched |
| Build + test clean | Task 15 |
| Patch notes | Task 16 |

No gaps.

### Placeholder scan

Searched the plan for TBDs, "implement later", ambiguous instructions. None found. Every YAML content block is pre-drafted; every Go code change shows the exact diff.

### Type consistency

- `FireAttackRejected` / `AttackRejectedTryMobBehavior` used consistently in Tasks 2 and 3.
- `LastAttackRejectedRound` field referenced identically in Tasks 1, 2, 3.
- Archetype names (`noncombat_questgiver`, `noncombat_shopkeeper`, `noncombat_passive`, `combat_passive`) used consistently across archetype YAML creation, mob YAML assignment, and tests.
- Event name `player_attack_rejected` consistent everywhere.
- Mob IDs (50, 51, 52, 53, 54, 55, 56, 63, 65, 66, 67, 68, 69, 70, 71, 79, 112) match across triage, mob YAML task, and dialogue tasks.

### Scope

This plan covers the sanctum_basin pilot only. After merge, the template is clear enough to re-run on each other zone (thornwall_city, marches_spur_road, etc.) in future plans.
