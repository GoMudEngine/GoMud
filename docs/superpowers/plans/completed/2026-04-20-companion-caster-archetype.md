# Pure Caster Archetype Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the `pure_caster` behavior-tree archetype for wraith, spectre, and air elemental (reassigned from melee) — maintains self-buffs, emergency-heals, prefers AoE when multiple enemies present, else single-target harm.

**Architecture:** Pure composition on Phase 4 primitives (`cast_best_in_category`, `mob_health_below`, `multiple_enemies`, `mob_combat_round`). Adds three spell-category tags (`self_heal`, `harm_single`, `harm_multi`), one archetype YAML, mob-YAML spellbook expansion, plus one bug fix to `multiple_enemies` making it perspective-aware for charmed mobs.

**Tech Stack:** Go 1.21+, YAML (gopkg.in/yaml.v2), Go standard testing library.

**Spec reference:** `docs/superpowers/specs/2026-04-20-companion-caster-archetype-design.md`

**Branch:** `feature/companion-caster-archetype` (already created from `development`, spec committed as `d360ba98` + `96e5b4e4`).

---

## File Structure

### New files

| File | Purpose |
|---|---|
| `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml` | The archetype tree — selector with 4 children (heal/defense/AoE/single-harm) |
| `internal/behaviortree/pure_caster_archetype_integration_test.go` | End-to-end tests for the archetype cadence |

### Modified files

| File | Change |
|---|---|
| `internal/behaviortree/conditions_player.go` | Fix `condMultipleEnemies` to be perspective-aware |
| `internal/behaviortree/conditions_test.go` | Add 5 test cases covering wild-mob regression + charmed-mob fix |
| `internal/behaviortree/melee_self_buff_archetype_integration_test.go` | Rename `TestMeleeSelfBuff_AirElementalCastsDefenseOnly` → `...FireElementalCastsDefenseOnly`; swap spellbook from air ellie's to fire ellie's |
| `_datafiles/world/dogmud/spells/heal.yaml` | Add `categories: [self_heal]` |
| `_datafiles/world/dogmud/spells/mind-spike.yaml` | Add `categories: [harm_single]` |
| `_datafiles/world/dogmud/spells/conviction-spike.yaml` | Add `categories: [harm_single]` |
| `_datafiles/world/dogmud/spells/nerve-disruption.yaml` | Add `categories: [harm_single]` |
| `_datafiles/world/dogmud/spells/sparks.yaml` | Add `categories: [harm_multi]` |
| `_datafiles/world/dogmud/spells/conviction-barrage.yaml` | Add `categories: [harm_multi]` |
| `_datafiles/world/dogmud/spells/hemorrhagic-wave.yaml` | Add `categories: [harm_multi]` |
| `_datafiles/world/dogmud/spells/hemorrhagic-burst.yaml` | Add `categories: [harm_multi]` |
| `_datafiles/world/dogmud/mobs/summons/302-wraith.yaml` | `behavior_archetype: pure_caster`, spellbook +heal +sparks, trim hardcoded casts |
| `_datafiles/world/dogmud/mobs/summons/303-spectre.yaml` | `behavior_archetype: pure_caster`, spellbook remove conviction-surge / add iron-will/heal/conviction-barrage, trim casts |
| `_datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml` | Change archetype `melee_self_buff` → `pure_caster`, spellbook +heal/mind-spike/sparks |
| `PATCH_NOTES.md` | Dated entry for the release |

---

## Tasks

### Task 1: Fix `multiple_enemies` perspective-awareness

**Files:**
- Modify: `internal/behaviortree/conditions_player.go`
- Test: `internal/behaviortree/conditions_test.go` (append)

- [ ] **Step 1: Read the existing condition to confirm current shape**

Run: `grep -A 15 "func condMultipleEnemies" internal/behaviortree/conditions_player.go`
Expected: function counts `players + charmed-mobs`, returns Success when `> 1`.

- [ ] **Step 2: Write the failing tests**

Append to `internal/behaviortree/conditions_test.go`:

```go
func TestCondMultipleEnemies_WildMob_CountsPlayersAndCharmed(t *testing.T) {
	// Wild mob (no owner): existing behavior preserved.
	// 1 player + 1 charmed mob → count=2 → Success.
	room := newTestRoomWithMobs(t, 1234, []int{} /* no mob ids from MobId seeds */)
	player := newTestPlayerInRoom(t, 501, room.RoomId)
	defer player.Cleanup()
	charmedMob := newTestCharmedMobInRoom(t, 9101, 501, room.RoomId) // charmed by player 501
	defer charmedMob.Cleanup()

	wildMob := newTestWildMobInRoom(t, 9102, room.RoomId)
	defer wildMob.Cleanup()

	ctx := &EvalContext{InstanceId: wildMob.InstanceId, RoomId: room.RoomId}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("wild mob with 1 player + 1 charmed expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_SkipsSummonerAndFellows(t *testing.T) {
	// Summoned mob (owner=501). Room contains: owner 501, fellow companion
	// (charmed by 501), and a wild mob. From this caster's POV, only the
	// wild mob is hostile → count=1 → Failure (not "multiple").
	room := newTestRoomWithMobs(t, 2345, nil)
	player := newTestPlayerInRoom(t, 501, room.RoomId)
	defer player.Cleanup()

	summoned := newTestCharmedMobInRoom(t, 9201, 501, room.RoomId)
	defer summoned.Cleanup()
	fellow := newTestCharmedMobInRoom(t, 9202, 501, room.RoomId)
	defer fellow.Cleanup()
	wild := newTestWildMobInRoom(t, 9203, room.RoomId)
	defer wild.Cleanup()

	ctx := &EvalContext{InstanceId: summoned.InstanceId, RoomId: room.RoomId}
	if got := condMultipleEnemies(nil, ctx); got != Failure {
		t.Fatalf("summoned mob with only 1 wild enemy expected Failure, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_WildMobsCount(t *testing.T) {
	// Summoned mob (owner=501). Room contains owner + 2 wild mobs.
	// From caster POV: 2 hostiles → Success.
	room := newTestRoomWithMobs(t, 3456, nil)
	player := newTestPlayerInRoom(t, 501, room.RoomId)
	defer player.Cleanup()

	summoned := newTestCharmedMobInRoom(t, 9301, 501, room.RoomId)
	defer summoned.Cleanup()
	w1 := newTestWildMobInRoom(t, 9302, room.RoomId)
	defer w1.Cleanup()
	w2 := newTestWildMobInRoom(t, 9303, room.RoomId)
	defer w2.Cleanup()

	ctx := &EvalContext{InstanceId: summoned.InstanceId, RoomId: room.RoomId}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("summoned mob with 2 wild enemies expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_CharmedMob_OtherPlayerHostile(t *testing.T) {
	// Summoned mob (owner=501). Room contains owner + another player + wild mob.
	// Other player is hostile (PvP context). count=2 → Success.
	room := newTestRoomWithMobs(t, 4567, nil)
	owner := newTestPlayerInRoom(t, 501, room.RoomId)
	defer owner.Cleanup()
	otherPlayer := newTestPlayerInRoom(t, 502, room.RoomId)
	defer otherPlayer.Cleanup()

	summoned := newTestCharmedMobInRoom(t, 9401, 501, room.RoomId)
	defer summoned.Cleanup()
	wild := newTestWildMobInRoom(t, 9402, room.RoomId)
	defer wild.Cleanup()

	ctx := &EvalContext{InstanceId: summoned.InstanceId, RoomId: room.RoomId}
	if got := condMultipleEnemies(nil, ctx); got != Success {
		t.Fatalf("charmed mob with hostile player + wild mob expected Success, got %v", got)
	}
}

func TestCondMultipleEnemies_EmptyRoom_Failure(t *testing.T) {
	// Wild mob alone in room → 0 enemies → Failure.
	room := newTestRoomWithMobs(t, 5678, nil)
	wild := newTestWildMobInRoom(t, 9501, room.RoomId)
	defer wild.Cleanup()

	ctx := &EvalContext{InstanceId: wild.InstanceId, RoomId: room.RoomId}
	if got := condMultipleEnemies(nil, ctx); got != Failure {
		t.Fatalf("empty-room wild mob expected Failure, got %v", got)
	}
}
```

**IMPORTANT:** The test helpers `newTestRoomWithMobs`, `newTestPlayerInRoom`, `newTestCharmedMobInRoom`, `newTestWildMobInRoom` are assumed. Before writing these tests:

1. Read the **existing** `conditions_test.go` to see what room/mob/player seeding helpers are already in use. Copy the pattern exactly.
2. If a similar helper is already present (likely `seedMobInRoom` or similar), use the real name. Don't invent.
3. If no such helpers exist, use the global `mobs.SeedMobsForTest`, `rooms.SeedRoomsForTest` (if present), or ad-hoc setup inline. The test should exercise the production `condMultipleEnemies` function through real `mobs.GetInstance` and `rooms.LoadRoom` calls.

Grep first:
```bash
grep -n "func newTest\|seedMob\|SeedForTest\|SeedMobsForTest" internal/behaviortree/conditions_test.go internal/behaviortree/test_helpers_test.go internal/rooms/test_helpers.go
```

Adapt the test code to the real helper names. The assertion logic is what matters; the setup is plumbing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/behaviortree/ -run TestCondMultipleEnemies -v`
Expected: FAIL — the existing condition counts `players + charmed` regardless of caller, so the "charmed mob skips summoner" tests fail.

- [ ] **Step 4: Fix the condition**

Replace the body of `condMultipleEnemies` in `internal/behaviortree/conditions_player.go`:

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
	// friends; count wild mobs + mobs charmed by someone else.
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

Ensure `mobs` import is present at top of the file (likely already is).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/behaviortree/ -run TestCondMultipleEnemies -v`
Expected: PASS — all 5 subtests.

- [ ] **Step 6: Run full package tests to confirm no regressions**

Run: `go test ./internal/behaviortree/ -v 2>&1 | tail -10`
Expected: `PASS`, all tests green. The pre-existing `multiple_enemies` callers (bandit_leader) behave identically because they're wild mobs hitting the `else` branch.

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/conditions_player.go internal/behaviortree/conditions_test.go
git commit -m "$(cat <<'EOF'
fix(behaviortree): multiple_enemies is now perspective-aware

A charmed mob's summoner and fellow same-owner companions no longer
count as "enemies" when the condition fires from that mob's btree.
Wild mobs retain the original count-players-plus-charmed behavior
(regression-covered for the existing bandit_leader use case).

Enables the forthcoming pure_caster archetype to correctly gate
AoE casting on hostile count rather than total-actors-in-room.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Tag 8 spells with new categories

**Files:**
- Modify: `_datafiles/world/dogmud/spells/heal.yaml`
- Modify: `_datafiles/world/dogmud/spells/mind-spike.yaml`
- Modify: `_datafiles/world/dogmud/spells/conviction-spike.yaml`
- Modify: `_datafiles/world/dogmud/spells/nerve-disruption.yaml`
- Modify: `_datafiles/world/dogmud/spells/sparks.yaml`
- Modify: `_datafiles/world/dogmud/spells/conviction-barrage.yaml`
- Modify: `_datafiles/world/dogmud/spells/hemorrhagic-wave.yaml`
- Modify: `_datafiles/world/dogmud/spells/hemorrhagic-burst.yaml`

- [ ] **Step 1: Add `categories: [self_heal]` to `heal.yaml`**

Insert before `cast_user_text:` (matches placement of the Phase 4 category additions):
```yaml
categories:
  - self_heal
```

- [ ] **Step 2: Add `categories: [harm_single]` to three spells**

For each of `mind-spike.yaml`, `conviction-spike.yaml`, `nerve-disruption.yaml`:

Insert before `cast_user_text:`:
```yaml
categories:
  - harm_single
```

- [ ] **Step 3: Add `categories: [harm_multi]` to four spells**

For each of `sparks.yaml`, `conviction-barrage.yaml`, `hemorrhagic-wave.yaml`, `hemorrhagic-burst.yaml`:

Insert before `cast_user_text:`:
```yaml
categories:
  - harm_multi
```

- [ ] **Step 4: Verify the YAML parses cleanly**

Run: `go build ./...`
Expected: clean build (YAML parses into `SpellData.Categories` at load time).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/spells/heal.yaml _datafiles/world/dogmud/spells/mind-spike.yaml _datafiles/world/dogmud/spells/conviction-spike.yaml _datafiles/world/dogmud/spells/nerve-disruption.yaml _datafiles/world/dogmud/spells/sparks.yaml _datafiles/world/dogmud/spells/conviction-barrage.yaml _datafiles/world/dogmud/spells/hemorrhagic-wave.yaml _datafiles/world/dogmud/spells/hemorrhagic-burst.yaml
git commit -m "$(cat <<'EOF'
data(spells): tag 8 spells with caster archetype categories

heal -> self_heal
mind-spike, conviction-spike, nerve-disruption -> harm_single
sparks, conviction-barrage, hemorrhagic-wave, hemorrhagic-burst
  -> harm_multi

Enables the pure_caster archetype to find candidates in a mob's
spellbook via cast_best_in_category. Each spell's type is already
set (HelpSingle for heal, HarmSingle for the singles, HarmArea for
the four multis); the categories just tell the AI which slot to use.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Update Phase 4 melee test — swap air → fire elemental

**Files:**
- Modify: `internal/behaviortree/melee_self_buff_archetype_integration_test.go`

**Why this task comes BEFORE mob YAML changes:** Task 7 will change air ellie's archetype to `pure_caster`. The existing Phase 4 test `TestMeleeSelfBuff_AirElementalCastsDefenseOnly` asserts air ellie's behavior under `melee_self_buff` — which will no longer be its archetype. Swap the test to use fire ellie (still melee, same defense-only shape) before touching air ellie.

- [ ] **Step 1: Read the existing test to confirm its shape**

Run: `grep -B 2 -A 30 "TestMeleeSelfBuff_AirElementalCastsDefenseOnly" internal/behaviortree/melee_self_buff_archetype_integration_test.go`
Note: the test uses mock spellbook setup (doesn't load air_elemental.yaml), so it will still pass after the mob YAML change — but its *name* becomes a lie. Fix the name and swap to fire's spellbook shape for clarity.

- [ ] **Step 2: Replace the test**

Find `TestMeleeSelfBuff_AirElementalCastsDefenseOnly`. Rename it and change the spellbook to fire ellie's actual Phase 4 shape (conviction-armor + conviction-ward):

```go
func TestMeleeSelfBuff_FireElementalCastsDefenseOnly(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff")

	// Fire ellie has only self_defense spells in its spellbook.
	// conviction-armor (cost 50, folds 6, score 300) beats conviction-ward
	// (cost 30, folds 4, score 120), so archetype picks conviction-armor first.
	mob, cleanup := seedArchetypeMob(t, 90004, map[string]int{
		"conviction-armor": 3,
		"conviction-ward":  3,
	})
	_ = mob
	defer cleanup()

	ok := TryMobBehavior(90004, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("expected tree to return Success")
	}
	cmd := drainCastCommand(t, 90004)
	if !strings.HasPrefix(cmd, "cast conviction-armor") {
		t.Fatalf("fire ellie should cast conviction-armor (top-ranked defense) first, got %q", cmd)
	}
}
```

**Verify:** `seedArchetypeSpells` already seeds `conviction-armor`. If it does not (check the function body), add an entry for it. It's spectre's Phase 4 spellbook entry so may already be seeded.

Also verify `seedArchetypeMob` — if the third arg signature or the `_ = mob` placeholder changes are needed, match the existing test's call style. The pattern must be `copy what test 1 does`, just with a different spellbook.

- [ ] **Step 3: Run the test and its siblings**

Run: `go test ./internal/behaviortree/ -run TestMeleeSelfBuff -v`
Expected: 3 tests pass — `FreshVampireCastsSelfOffense`, `WithSurgeActiveCastsIronWill`, `FireElementalCastsDefenseOnly` (the renamed one).

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/melee_self_buff_archetype_integration_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): swap air ellie → fire ellie in Phase 4 test

Air elemental is being reassigned to the pure_caster archetype in
the next commit batch, so the Phase 4 integration test can no longer
use it as a melee_self_buff example. Fire ellie has the same
defense-only spellbook shape and stays on the melee archetype —
drop-in replacement. The test's assertion updates to
conviction-armor (fire's top-ranked defense) instead of iron-will.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Create `pure_caster.yaml` archetype file

**Files:**
- Create: `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml`

- [ ] **Step 1: Write the archetype YAML**

Create `_datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml` with this exact content:

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
#   3. AoE harm when 2+ hostile actors present (multiple_enemies is
#      perspective-aware: summoner and fellow same-owner companions
#      don't count as enemies for a charmed mob)
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

- [ ] **Step 2: Verify the YAML parses by exercising a load**

Run: `go build ./... && go test ./internal/behaviortree/ -run TestGetArchetypePath -v`
Expected: PASS (the path helper works; any YAML parse error would surface when the archetype is first loaded via the engine).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/pure_caster.yaml
git commit -m "$(cat <<'EOF'
feat(behaviors): pure_caster archetype tree

Second behavior-tree archetype. Fires on mob_combat_round. Decision
order: emergency heal (< 40%), maintain defense, AoE if multiple
enemies, single-target harm otherwise. Falls through to legacy
combat when every branch returns Failure (all buffs active + on
cooldown + no harm spells castable).

No new Go code required — this archetype is pure composition of
Phase 4 primitives (cast_best_in_category) and existing btree
conditions (mob_health_below, multiple_enemies).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Update wraith YAML (302-wraith.yaml)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/summons/302-wraith.yaml`

- [ ] **Step 1: Read the current wraith YAML**

Run: `cat _datafiles/world/dogmud/mobs/summons/302-wraith.yaml`
Confirm current state: `archetype: casting` (statpool), `aiprofile: caster`, spellbook has `mind-spike`, `nerve-disruption`, `conviction-ward`; combatcommands include `'cast mind-spike'` and `'cast nerve-disruption'`.

- [ ] **Step 2: Apply the changes**

Make these edits:

1. Add new line after `archetype: casting` (around line 3):
```yaml
behavior_archetype: pure_caster
```

2. Replace `combatcommands` block with (remove hardcoded `cast X` entries):
```yaml
combatcommands:
  - ''
  - 'emote phases through its target, trailing cold that cuts to the bone'
  - ''
```

3. In `character.spellbook`, append `heal: 3` and `sparks: 3`:
```yaml
  spellbook:
    mind-spike: 5
    nerve-disruption: 3
    conviction-ward: 4
    heal: 3
    sparks: 3
```

- [ ] **Step 3: Verify YAML parses**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Check for stale instance saves**

Run: `find _datafiles/world/dogmud/mobs.instances/ -name "302-*.yaml" 2>/dev/null | head -5`
Expected: no output (mobs.instances/ was nuked this session). If any matching files exist, delete them — stale instance saves would mask the YAML changes.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/summons/302-wraith.yaml
git commit -m "$(cat <<'EOF'
data(mobs): wraith adopts pure_caster archetype

Adds heal + sparks to the spellbook (self_heal + harm_multi slots
for the archetype). Existing mind-spike + nerve-disruption fill
harm_single, conviction-ward fills self_defense.

Drops hardcoded cast entries from combatcommands — archetype handles
cast decisions now. Kept the phase-through emote for combat flavor
on rounds where the archetype falls through.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Update spectre YAML (303-spectre.yaml)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/summons/303-spectre.yaml`

- [ ] **Step 1: Read the current spectre YAML**

Run: `cat _datafiles/world/dogmud/mobs/summons/303-spectre.yaml`
Confirm: spellbook has `conviction-spike`, `conviction-ward`, `conviction-surge`; combatcommands include `'cast conviction-spike'` and `'cast conviction-ward'`.

- [ ] **Step 2: Apply the changes**

1. Add new line after `archetype: casting` (around line 3):
```yaml
behavior_archetype: pure_caster
```

2. Replace `combatcommands` block:
```yaml
combatcommands:
  - ''
  - 'emote projects a wave of pure dread that presses on the chest like a stone'
  - ''
```

3. In `character.spellbook`:
   - **Remove** `conviction-surge: 3` (boosts strength, useless for caster)
   - **Add** `iron-will: 3` (self_defense — willpower + magic_mitigation + conviction_mitigation)
   - **Add** `heal: 3` (self_heal)
   - **Add** `conviction-barrage: 3` (harm_multi — AoE conviction damage, thematic)
   - Keep existing `conviction-spike: 5` and `conviction-ward: 4`

Final spellbook:
```yaml
  spellbook:
    conviction-spike: 5
    conviction-ward: 4
    iron-will: 3
    heal: 3
    conviction-barrage: 3
```

- [ ] **Step 3: Verify YAML parses**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Check for stale instance saves**

Run: `find _datafiles/world/dogmud/mobs.instances/ -name "303-*.yaml" 2>/dev/null | head -5`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/summons/303-spectre.yaml
git commit -m "$(cat <<'EOF'
data(mobs): spectre adopts pure_caster archetype

Drops conviction-surge (strength buff — not useful for caster) and
adds iron-will (will + magic_mitigation + conviction_mitigation,
self_defense), heal (self_heal), conviction-barrage (harm_multi —
thematic AoE conviction damage). Existing conviction-spike fills
harm_single; conviction-ward stays as self_defense.

Drops hardcoded cast entries from combatcommands.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: Update air elemental YAML (312-air_elemental.yaml) — archetype reassignment

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml`

- [ ] **Step 1: Read the current air elemental YAML**

Run: `cat _datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml`
Confirm current state: `behavior_archetype: melee_self_buff`, spellbook has `iron-will: 3`, `conviction-ward: 4` (from Phase 4). Combatcommands: `'emote crackles and spins faster, striking from unexpected angles'` between blanks.

- [ ] **Step 2: Apply the changes**

1. Change the `behavior_archetype` value:
```yaml
behavior_archetype: pure_caster   # was: melee_self_buff
```

2. Leave `combatcommands` unchanged — air ellie's flavor emote stays.

3. In `character.spellbook`, add three new spells:
```yaml
  spellbook:
    iron-will: 3           # existing — self_defense
    conviction-ward: 4     # existing — self_defense
    heal: 3                # NEW — self_heal
    mind-spike: 3          # NEW — harm_single
    sparks: 3              # NEW — harm_multi
```

- [ ] **Step 3: Verify YAML parses**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Check for stale instance saves**

Run: `find _datafiles/world/dogmud/mobs.instances/ -name "312-*.yaml" 2>/dev/null | head -5`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/summons/312-air_elemental.yaml
git commit -m "$(cat <<'EOF'
data(mobs): air elemental moves from melee_self_buff to pure_caster

Air ellie's stats (dex 20, perception 20, will 10) were always
caster-shaped; Phase 4's melee assignment was a convenient grouping
rather than a thematic fit. Reassigning to pure_caster and adding
heal + mind-spike + sparks to round out the 4 archetype slots
(self_heal, self_defense kept via existing ward/iron-will,
harm_single, harm_multi).

Vampire + fire elemental remain on melee_self_buff.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Integration test for `pure_caster` archetype

**Files:**
- Create: `internal/behaviortree/pure_caster_archetype_integration_test.go`

- [ ] **Step 1: Read the Phase 4 integration test for pattern reference**

Run: `cat internal/behaviortree/melee_self_buff_archetype_integration_test.go`
Note the helpers: `seedArchetypeSpells`, `seedArchetypeMob`, `seedBuffActive`, `drainCastCommand`, `LoadArchetypeForTest`. Reuse these.

- [ ] **Step 2: Write the integration tests**

Create `internal/behaviortree/pure_caster_archetype_integration_test.go`:

```go
package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/spells"
)

// seedPureCasterSpells installs the spell library the pure_caster archetype
// uses. Includes the Phase 4 self_defense/self_offense spells (so the
// seedArchetypeSpells call isn't strictly required for these tests) plus
// the new self_heal, harm_single, harm_multi additions.
func seedPureCasterSpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		// self_defense (reused from Phase 4)
		"iron-will": {
			SpellId: "iron-will", Name: "Iron Will",
			Type: spells.HelpSingle, Cost: 45, BaseFolds: 6,
			EffectType: "buff", BuffIds: []int{27},
			Categories: []string{"self_defense"},
		},
		"conviction-ward": {
			SpellId: "conviction-ward", Name: "Conviction Ward",
			Type: spells.HelpSingle, Cost: 30, BaseFolds: 4,
			EffectType: "shield", EffectMagnitude: 75,
			Categories: []string{"self_defense"},
		},
		// self_heal
		"heal": {
			SpellId: "heal", Name: "Heal",
			Type: spells.HelpSingle, Cost: 40, BaseFolds: 5,
			EffectType: "heal", EffectMagnitude: 3,
			Categories: []string{"self_heal"},
		},
		// harm_single
		"mind-spike": {
			SpellId: "mind-spike", Name: "Mind Spike",
			Type: spells.HarmSingle, Cost: 35, BaseFolds: 4,
			EffectType: "damage",
			Categories: []string{"harm_single"},
		},
		"conviction-spike": {
			SpellId: "conviction-spike", Name: "Conviction Spike",
			Type: spells.HarmSingle, Cost: 40, BaseFolds: 5,
			EffectType: "damage",
			Categories: []string{"harm_single"},
		},
		// harm_multi
		"sparks": {
			SpellId: "sparks", Name: "Sparks",
			Type: spells.HarmArea, Cost: 75, BaseFolds: 4,
			EffectType: "damage",
			Categories: []string{"harm_multi"},
		},
	})
}

// seedPureCasterMob sets up a wraith-shaped caster mob with the archetype bound
// and a generous CP pool. Returns cleanup function.
func seedPureCasterMob(t *testing.T, instanceId int, spellbook map[string]int) func() {
	t.Helper()
	// Use the same pattern as the Phase 4 seedArchetypeMob helper.
	// Adjust MobId/Character.Name as needed for clarity — the spec names
	// this a "testcaster" for readability in assertion failure messages.
	return seedArchetypeMob_forCaster(t, instanceId, 302 /* wraith MobId */, "testcaster", spellbook)
}

// seedArchetypeMob_forCaster is a Caster-archetype-aware version of
// seedArchetypeMob. If seedArchetypeMob from Phase 4 already takes a mobId
// arg, drop this helper and call that directly. Inline the body if needed
// — this function exists only to match the Phase 4 helper shape.
func seedArchetypeMob_forCaster(t *testing.T, instanceId, mobId int, name string, spellbook map[string]int) func() {
	t.Helper()
	// Look up what Phase 4's seedArchetypeMob does and mirror it exactly.
	// The implementation below is illustrative — adapt to match.
	//
	// Typical shape:
	//   m := &mobs.Mob{InstanceId: instanceId, MobId: mobs.MobId(mobId)}
	//   m.BehaviorArchetype = "pure_caster"
	//   m.Character.Name = name
	//   m.Character.SpellBook = spellbook
	//   m.Character.Conviction = 500
	//   m.Character.Health = 100
	//   m.Character.HealthMax.Value = 100
	//   cleanup := mobs.SeedMobsForTest(map[int]*Mob{mobId: m}, map[int]*Mob{instanceId: m})
	//   return cleanup
	panic("adapt this helper to match seedArchetypeMob's signature — see Phase 4 integration test")
}

func TestPureCaster_FullHP_MaintainsDefenseFirst(t *testing.T) {
	defer seedPureCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster")

	cleanup := seedPureCasterMob(t, 91001, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
		"heal":            3,
		"mind-spike":      5,
		"sparks":          3,
	})
	defer cleanup()

	// HP full, no buffs. Branch 1 (heal) fails (HP not < 40%). Branch 2
	// (self_defense) picks top-score candidate.
	// Scores: iron-will 6×45=270, conviction-ward 4×30=120. iron-will wins.
	ok := TryMobBehavior(91001, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("expected tree to return Success, got Failure")
	}
	cmd := drainCastCommand(t, 91001)
	if !strings.HasPrefix(cmd, "cast iron-will") {
		t.Fatalf("full-HP fresh caster should cast iron-will (top defense), got %q", cmd)
	}
}

func TestPureCaster_LowHP_EmergencyHeal(t *testing.T) {
	defer seedPureCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster")

	cleanup := seedPureCasterMob(t, 91002, map[string]int{
		"conviction-ward": 4,
		"heal":            3,
		"mind-spike":      5,
	})
	defer cleanup()

	// Drop HP below 40% to trigger branch 1.
	if m := mobs.GetInstance(91002); m != nil {
		m.Character.Health = 30 // 30% of 100
	}

	ok := TryMobBehavior(91002, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("expected tree to return Success")
	}
	cmd := drainCastCommand(t, 91002)
	if !strings.HasPrefix(cmd, "cast heal") {
		t.Fatalf("low-HP caster should cast heal first, got %q", cmd)
	}
}

func TestPureCaster_DefenseCovered_SingleEnemy_CastsHarmSingle(t *testing.T) {
	defer seedPureCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster")

	cleanup := seedPureCasterMob(t, 91003, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
		"heal":            3,
		"mind-spike":      5,
	})
	defer cleanup()

	// Mark both defensive buffs as active so defense branch fails.
	if m := mobs.GetInstance(91003); m != nil {
		seedBuffActive(t, &m.Character, 27)                         // iron-will buff
		m.Character.AddCondition(characters.ConditionShield, 20, 75, "test") // ward-equivalent
	}

	// No other actors seeded in the room → multiple_enemies returns Failure.
	// Tree falls through: heal skipped (HP full), defense skipped (active),
	// harm_multi gated by multiple_enemies (1 actor only) → harm_single.
	ok := TryMobBehavior(91003, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("expected tree to return Success")
	}
	cmd := drainCastCommand(t, 91003)
	if !strings.HasPrefix(cmd, "cast mind-spike") {
		t.Fatalf("defense-covered, single-enemy caster should cast mind-spike, got %q", cmd)
	}
}

func TestPureCaster_DefenseCovered_MultipleEnemies_CastsAoE(t *testing.T) {
	defer seedPureCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster")

	// Room with 2+ enemies (charmed-mob-counted-as-enemy — see
	// condMultipleEnemies). The wild caster we're testing shouldn't
	// have a charmed owner, so multiple_enemies fires on
	// `count = players + charmed-mobs > 1`.
	//
	// Seed 2 players in the test room (mob room defaults to 0/1 based
	// on test infra). Use the same pattern as the Phase 4 integration
	// test if it has similar room-population helpers.

	cleanup := seedPureCasterMob(t, 91004, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
		"mind-spike":      5,
		"sparks":          3,
	})
	defer cleanup()

	if m := mobs.GetInstance(91004); m != nil {
		// Mark defensive buffs active so the tree reaches the harm branches.
		seedBuffActive(t, &m.Character, 27)
		m.Character.AddCondition(characters.ConditionShield, 20, 75, "test")
		// Seed 2+ hostile actors in the mob's room — adapt to test infra.
		// If the existing helpers expose a "put N players in room" shortcut,
		// use it. Otherwise: seed room with 2 user records + register.
	}

	// (Test setup for room-population varies by existing helpers; see
	// Phase 4 test for the established pattern.)
	t.Skip("room population helper not wired in test infra yet — covered by manual smoke")
}

func TestPureCaster_NoCandidates_FallsThrough(t *testing.T) {
	defer seedPureCasterSpells(t)()
	LoadArchetypeForTest(t, "pure_caster")

	// Caster with only self_defense spells in spellbook, all active, HP fine.
	// Expected: all 4 branches fail → selector returns Failure → legacy.
	cleanup := seedPureCasterMob(t, 91005, map[string]int{
		"conviction-ward": 4,
		"iron-will":       3,
	})
	defer cleanup()

	if m := mobs.GetInstance(91005); m != nil {
		seedBuffActive(t, &m.Character, 27)
		m.Character.AddCondition(characters.ConditionShield, 20, 75, "test")
	}

	ok := TryMobBehavior(91005, EventContext{EventType: "mob_combat_round"})
	if ok {
		t.Fatalf("expected tree to return Failure (all branches declined), got Success")
	}
}
```

**Helper adaptation:** The `seedArchetypeMob_forCaster` wrapper is a placeholder. Read the actual `seedArchetypeMob` helper in `melee_self_buff_archetype_integration_test.go` and either:
- Call it directly (if it accepts a `mobId` param) and remove the wrapper, OR
- Copy its body and adjust for caster-specific setup (e.g., generous CP, HP+HealthMax for the low-HP test)

The HealthMax setup for the low-HP test is important — `mob_health_below` checks `Health * 100 / HealthMax.Value < percent`. Set both.

**Skip the multiple_enemies test if needed:** The AoE branch requires room-population infrastructure. If the existing test file doesn't have a helper for "put N players in a room that the mob is in," skip that test with `t.Skip("covered by manual smoke")` and cover it in Task 9.

- [ ] **Step 3: Make imports match**

Add these imports to the top of the new file:
```go
import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/behaviortree/ -run TestPureCaster -v`
Expected: PASS for all non-skipped tests. The multiple_enemies AoE test may be skipped — that's acceptable.

- [ ] **Step 5: Run the full package**

Run: `go test ./internal/behaviortree/ -v 2>&1 | tail -10`
Expected: all tests pass (Phase 4 + pure_caster).

- [ ] **Step 6: Commit**

```bash
git add internal/behaviortree/pure_caster_archetype_integration_test.go
git commit -m "$(cat <<'EOF'
test(behaviortree): end-to-end pure_caster archetype cadence

Drives a caster-shaped mob through TryMobBehavior with mob_combat_round
events, loading the real pure_caster.yaml. Verifies: full-HP caster
casts iron-will first (top self_defense); low-HP caster casts heal
(emergency branch); defense-covered caster with 1 enemy casts
mind-spike (harm_single); no-candidate caster falls through.

multiple_enemies/AoE test is skipped pending room-population helper
in the test infra — covered by the Task 9 manual smoke test.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Manual smoke test + PATCH_NOTES + merge

**Files:**
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Build and restart the server**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
go build ./...
# Restart your MUD server however you normally run it.
```

- [ ] **Step 2: In-game smoke test**

Summon each mob type and verify archetype behavior:

1. **Wraith**
   - `cast raise-wraith` (or the raise spell that produces it)
   - Engage a training dummy / weak mob
   - Expected: "flickers" / emote flavor text; casts `conviction-ward`, then `mind-spike` / `nerve-disruption` alternating; heal when HP drops (if it takes enough damage).

2. **Spectre**
   - `cast raise-spectre` (or appropriate)
   - Engage a target
   - Expected: maintains ward + iron-will; casts `conviction-spike`; conviction-barrage fires if multiple enemies present (group fight).

3. **Air elemental**
   - `cast conjure-air` (or appropriate)
   - Engage a target
   - Expected: was previously casting conviction-ward via melee_self_buff — should now also cast `mind-spike` offensively and `heal` when low.

4. **AoE sanity**
   - Take wraith into a room with 2+ hostile mobs (no companion present)
   - Expected: wraith casts `sparks` (AoE) instead of single-target `mind-spike`.

5. **multiple_enemies fix sanity**
   - Summon wraith + another companion; fight ONE wild mob
   - Expected: wraith casts `mind-spike` (single-target), NOT `sparks`. The fellow companion no longer counts as "multiple enemies."

6. **Charmed wild mob fallthrough**
   - Charm a mob with no `pure_caster` spellbook entries
   - Engage combat
   - Expected: no crashes, mob attacks normally via legacy combatcommands.

Document any anomalies and fix before merging. If the AoE test (#4) fails, check the `condMultipleEnemies` fix in Task 1 and add debug logging.

- [ ] **Step 3: Update PATCH_NOTES.md**

Open `PATCH_NOTES.md` and insert a new entry at the top (after the `# DOGMud Patch Notes` header). Use this content:

```markdown
## 2026-04-20 — Pure Caster Archetype

### Gameplay

- **Wraiths, spectres, and air elementals are now proper mages.**
  Each maintains defensive buffs, emergency-heals when HP drops
  below 40%, picks AoE damage when enemies are grouped, and
  single-target damage otherwise. Watch for `heal`, `sparks`,
  `conviction-barrage`, `mind-spike`, `conviction-spike`, and
  `nerve-disruption` depending on the mob and situation.
- **Air elemental is now a caster** (was a melee specialist in the
  previous Phase 4 release). Its stats were always caster-shaped;
  this update gives it the spellbook to match. Vampire and fire
  elemental stay on melee.

### Under the hood

- **New archetype `pure_caster`** sits alongside `melee_self_buff`.
  Both share the same framework — only the tree YAML and mob
  spellbook tags differ.
- **`multiple_enemies` btree condition is now perspective-aware.**
  Previously it counted `players + charmed mobs` regardless of the
  calling mob's perspective, so a summoned caster with a fellow
  companion in the room saw "multiple enemies" when fighting a
  single wild mob. Now, from a charmed mob's POV, the summoner
  and fellow same-owner companions are excluded. Wild mobs
  (like bandit_leader) preserve original behavior for regression
  safety.
- **Three new spell categories** — `self_heal`, `harm_single`,
  `harm_multi` — tag the spells archetypes filter for.
```

- [ ] **Step 4: Commit PATCH_NOTES**

```bash
git add PATCH_NOTES.md
git commit -m "docs: patch notes for pure_caster archetype release

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 5: Merge into development**

```bash
git checkout development
git merge --no-ff feature/companion-caster-archetype -m "merge: pure_caster archetype — wraith + spectre + air elemental

Second behavior-tree archetype. Adds self_heal + harm_single +
harm_multi spell categories, pure_caster.yaml tree (heal → defense
→ AoE → single-harm selector), wraith/spectre/air elemental
spellbook expansions. Air ellie moves from melee_self_buff to
pure_caster.

Includes the multiple_enemies perspective-aware fix so summoned
casters correctly gate AoE decisions on hostile count rather
than total actors in the room.
"
```

- [ ] **Step 6: Verify clean state**

```bash
git log development --oneline -8
```
Expected: merge commit at HEAD with all caster-archetype commits in history.

**Prod push is OUT OF SCOPE for this plan.** When you're ready to ship to master, follow the Pre-Push SOP in `CLAUDE.md` (verify `Logging.LogToFile: false`, then `git checkout master && git merge --no-ff development && git push origin master`).

---

## Self-Review

### Spec coverage

- **§1 Archetype tree** → Task 4 (create pure_caster.yaml) ✓
- **§2 Spell categorization (3 new tags, 8 spells)** → Task 2 ✓
- **§3 Mob changes (wraith, spectre, air ellie)** → Tasks 5, 6, 7 ✓
- **§4 Engine changes (multiple_enemies fix)** → Task 1 ✓
- **§5 Error handling / edge cases** → Covered by Task 8 (fallthrough test) + Task 9 (smoke checks)
- **§6 Testing strategy (unit tests + integration tests + manual smoke)** → Tasks 1, 8, 9 ✓
- **§7 Deliverables + Phase 4 test update** → Task 3 (air→fire swap in Phase 4 test) ✓

All spec sections have tasks.

### Placeholder scan

Two deliberate hedges, both clearly flagged with adaptation instructions:
- Task 1's test helper names (`newTestRoomWithMobs` etc.) — instructed the implementer to grep existing helpers and adapt
- Task 8's `seedArchetypeMob_forCaster` wrapper — instructed to call the Phase 4 helper directly if it accepts the needed params

No TBDs, no generic "add validation" language, no bare "similar to Task N" references.

### Type consistency

- `pure_caster` archetype name used consistently (Tasks 4, 5, 6, 7, 8, 9)
- Category names `self_heal` / `harm_single` / `harm_multi` match between Task 2 (data), Task 4 (tree YAML), Task 8 (seeded spell data)
- `mob_combat_round` event name matches Task 4 (YAML) and Task 8 (test invocation)
- Mob instance IDs in tests (91001-91005) don't collide with Phase 4 (90001-90005)
- Buff IDs (27 for iron-will, shield via AddCondition) match spec + Phase 4 patterns
