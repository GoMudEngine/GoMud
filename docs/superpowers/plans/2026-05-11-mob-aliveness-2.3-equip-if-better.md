# Mob Aliveness 2.3 — Equip-If-Better Behavior Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make NPCs visibly smart about gear. Push path: rewrite the existing `gearup` mobcommand's gold-value heuristic to use `itemvalue.IsUpgrade` from chunk 2.2. Pull path: add an idle-tick floor-loot scan via new `EquipBestFloorItem` function in `internal/itemvalue/`. Two gate helpers (`CanEquipFromGive`, `CanScanFloorLoot`) keep animals, non-combat archetypes, and (for pull only) charmed mobs out of the loop.

**Architecture:** All logic for equip-if-better lives in `internal/itemvalue/equip_eligibility.go`: the two gate helpers plus the floor-scan function. The `gearup` mobcommand body is rewritten to use `itemvalue.IsUpgrade` + the gate helper. `MobIdle_HandleIdleMobs.go` gets one new line calling `itemvalue.EquipBestFloorItem(mob, room)` alongside the existing per-mob idle behaviors. Existing `actions.EquipItem` handles the actual swap mechanics; existing `gearup` semantics (PermaGear blocking, charmed-drop convention, emote phrasing) preserved.

**Tech Stack:** Go 1.21+, existing `internal/itemvalue` package (chunk 2.2), existing `internal/mutations` package (chunk 2.2a gear-effectiveness multiplier already integrated into `ItemValueDelta`), existing `internal/actions.EquipItem` for swap mechanics.

**Spec:** `docs/superpowers/specs/2026-05-11-mob-aliveness-2.3-equip-if-better-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `internal/itemvalue/equip_eligibility.go` | NEW | `CanEquipFromGive`, `CanScanFloorLoot`, `EquipBestFloorItem` |
| `internal/itemvalue/equip_eligibility_test.go` | NEW | Tests for the three new functions |
| `internal/mobcommands/gearup.go` | REWRITE | Replace gold-value heuristic with `itemvalue.IsUpgrade`; add `CanEquipFromGive` gate |
| `internal/mobcommands/gearup_test.go` | NEW or expand | Tests for gearup behavior |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | MODIFY | Add `itemvalue.EquipBestFloorItem(mob, room)` call |
| `internal/itemvalue/context.md` | MODIFY | Document the new helpers + push/pull behaviors |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Mark 2.3 Done, roll-up to 11/41 |

---

## Task 1: Gate helpers + tests

**Files:**
- Create: `internal/itemvalue/equip_eligibility.go`
- Create: `internal/itemvalue/equip_eligibility_test.go`

The two gate helpers are pure functions over `*mobs.Mob`. Tests construct synthetic mobs inline and verify each gate condition.

- [ ] **Step 1: Create `internal/itemvalue/equip_eligibility.go` with the two gates**

```go
package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// CanEquipFromGive returns true if this mob is the kind of
// creature that can rationally equip gear from a player give.
// Returns false for:
//   - Animal species (Species.DisabledSlots contains "Weapon")
//   - Non-combat archetypes (noncombat_*, prey, combat_passive)
//
// Note: incorporeal-rank-4 mobs are NOT explicitly skipped here.
// Their IsUpgrade naturally returns false because all gear
// scores 0 via mutations.GearEffectivenessMultiplier (chunk 2.2a).
func CanEquipFromGive(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}

	// Animal-species gate: species with Weapon slot disabled.
	if speciesInfo := species.GetSpecies(mob.Character.SpeciesId); speciesInfo != nil {
		for _, slot := range speciesInfo.DisabledSlots {
			if slot == "Weapon" {
				return false
			}
		}
	}

	// Non-combat archetypes silently skip equip-if-better.
	switch mob.BehaviorArchetype {
	case "noncombat_passive", "noncombat_questgiver",
		"noncombat_shopkeeper", "prey", "combat_passive":
		return false
	}

	return true
}

// CanScanFloorLoot returns true if this mob should scan floor
// loot for upgrades on idle ticks. Equals CanEquipFromGive plus
// !mob.Character.IsCharmed(). Companions and mercs accept pushes
// from their owner but skip pull (owner has dibs on floor loot).
func CanScanFloorLoot(mob *mobs.Mob) bool {
	if !CanEquipFromGive(mob) {
		return false
	}
	return !mob.Character.IsCharmed()
}
```

- [ ] **Step 2: Create `internal/itemvalue/equip_eligibility_test.go` with gate tests**

```go
package itemvalue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCanEquipFromGive_NilMob(t *testing.T) {
	if CanEquipFromGive(nil) {
		t.Errorf("expected false for nil mob")
	}
}

func TestCanEquipFromGive_RegularBruiser(t *testing.T) {
	// Default mob with no special archetype, no species → true.
	m := &mobs.Mob{
		BehaviorArchetype: "generic_fighter",
	}
	m.Character = characters.Character{}
	if !CanEquipFromGive(m) {
		t.Errorf("expected true for generic_fighter")
	}
}

func TestCanEquipFromGive_NonCombatArchetypes(t *testing.T) {
	cases := []string{
		"noncombat_passive",
		"noncombat_questgiver",
		"noncombat_shopkeeper",
		"prey",
		"combat_passive",
	}
	for _, arch := range cases {
		m := &mobs.Mob{BehaviorArchetype: arch}
		m.Character = characters.Character{}
		if CanEquipFromGive(m) {
			t.Errorf("expected false for archetype %q", arch)
		}
	}
}

func TestCanScanFloorLoot_CharmedSkipsScan(t *testing.T) {
	// Charmed (companion) bruiser — should skip floor-loot scan
	// even though they CAN equip from give.
	m := &mobs.Mob{
		BehaviorArchetype: "generic_fighter",
	}
	m.Character = characters.Character{}
	m.Character.Charm(1, -1, "")
	if !CanEquipFromGive(m) {
		t.Errorf("charmed bruiser should still pass CanEquipFromGive (push)")
	}
	if CanScanFloorLoot(m) {
		t.Errorf("charmed bruiser should NOT pass CanScanFloorLoot (pull)")
	}
}

func TestCanScanFloorLoot_NonCharmedBruiser(t *testing.T) {
	m := &mobs.Mob{
		BehaviorArchetype: "generic_fighter",
	}
	m.Character = characters.Character{}
	if !CanScanFloorLoot(m) {
		t.Errorf("non-charmed bruiser should pass CanScanFloorLoot")
	}
}
```

- [ ] **Step 3: Build and run tests**

```
go build ./internal/itemvalue/...
go test ./internal/itemvalue/ -run 'TestCanEquipFromGive|TestCanScanFloorLoot' -v
```

Expected: all PASS. Animal-species test deferred to integration since constructing a synthetic species with DisabledSlots requires species-package fixture setup; the non-mob cases cover the archetype gates.

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/equip_eligibility.go internal/itemvalue/equip_eligibility_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): CanEquipFromGive + CanScanFloorLoot gate helpers

Two gates for chunk 2.3 equip-if-better behavior.
CanEquipFromGive skips animal species (Weapon slot disabled)
and non-combat archetypes (noncombat_*, prey, combat_passive).
CanScanFloorLoot adds charmed-status exclusion for pull —
companions and mercs accept pushes from their owner but skip
pull (owner has dibs on floor loot).

Incorporeal-rank-4 mobs are not explicitly skipped — chunk
2.2a's gear-effectiveness multiplier already makes IsUpgrade
return false for them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: EquipBestFloorItem function + tests

**Files:**
- Modify: `internal/itemvalue/equip_eligibility.go`
- Modify: `internal/itemvalue/equip_eligibility_test.go`

Add the floor-scan function to the same file. It's a thin orchestrator: gate-check, combat-check, room-check, then iterate `room.Items` scoring via `ItemValueDelta`, equip best upgrade via `actions.EquipItem`.

- [ ] **Step 1: Append `EquipBestFloorItem` to `equip_eligibility.go`**

Add imports to the existing import block: `"fmt"`, `"github.com/GoMudEngine/GoMud/internal/actions"`, `"github.com/GoMudEngine/GoMud/internal/items"`, `"github.com/GoMudEngine/GoMud/internal/rooms"`.

Append:

```go
// EquipBestFloorItem scans the floor items in mob's room, scores
// each via ItemValueDelta, and equips the best positive-scoring
// upgrade if any. Returns true if a swap occurred.
//
// No-op (returns false) if any of:
//   - !CanScanFloorLoot(mob)
//   - mob is in combat (Character.Aggro != nil)
//   - room is nil or has no floor items
//   - no floor item scores as an upgrade for this mob+profile
//
// On successful pickup, emits a room broadcast distinct from
// give-equip phrasing ("picks up X and dons it" / "wields it").
// Displaced items go to mob's backpack per actions.EquipItem
// default (charmed mobs don't reach this path).
func EquipBestFloorItem(mob *mobs.Mob, room *rooms.Room) bool {
	if !CanScanFloorLoot(mob) {
		return false
	}
	if mob.Character.Aggro != nil {
		return false // busy fighting
	}
	if room == nil || len(room.Items) == 0 {
		return false // nothing on floor
	}

	profile := ProfileFor(mob.Archetype, mob.BehaviorArchetype)

	// Find the floor item with the highest positive delta score.
	var bestItem items.Item
	bestScore := 0.0
	for _, floorItem := range room.Items {
		delta := ItemValueDelta(&mob.Character, profile, floorItem)
		if delta.Score > bestScore {
			bestScore = delta.Score
			bestItem = floorItem
		}
	}
	if bestItem.ItemId == 0 {
		return false // nothing was an upgrade
	}

	// Remove from floor and into backpack so EquipItem can find it.
	room.RemoveItem(bestItem, false)
	mob.Character.StoreItem(bestItem)

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.EquipItem(actor, bestItem.Name())
	if !result.Equipped {
		// Edge case: ItemValueDelta thought slot was compatible
		// but EquipItem refused (rare). Item is still in backpack;
		// mob effectively "picked it up" without equipping.
		return false
	}

	// Room broadcast: distinct from give-equip phrasing to signal
	// the loot-pickup origin.
	spec := result.Item.GetSpec()
	if spec.Subtype == items.Wearable {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and dons it.`,
			mob.Character.Name, result.Item.DisplayName()))
	} else {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and wields it.`,
			mob.Character.Name, result.Item.DisplayName()))
	}

	return true
}
```

- [ ] **Step 2: Verify the `room.RemoveItem` signature**

The plan assumes `room.RemoveItem(item items.Item, isDeath bool)`. Quick check via Grep:

```
grep -n 'func .* RemoveItem' internal/rooms/rooms.go
```

If the signature differs, adapt the call. The two-arg form (item + a bool flag) is the most common pattern.

- [ ] **Step 3: Append tests to `equip_eligibility_test.go`**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestEquipBestFloorItem_NilRoom(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	if EquipBestFloorItem(m, nil) {
		t.Errorf("expected false for nil room")
	}
}

func TestEquipBestFloorItem_EmptyRoom(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	r := &rooms.Room{RoomId: 99999}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for empty room")
	}
}

func TestEquipBestFloorItem_NonEligibleMob(t *testing.T) {
	// Non-combat archetype mob → CanScanFloorLoot false → skip.
	m := &mobs.Mob{BehaviorArchetype: "noncombat_shopkeeper"}
	m.Character = characters.Character{}
	r := &rooms.Room{RoomId: 99999}
	// Even with items on the floor, should skip.
	r.Items = []items.Item{{ItemId: 1}}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for non-eligible mob")
	}
}

func TestEquipBestFloorItem_InCombatSkips(t *testing.T) {
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	m.Character.Aggro = &characters.Aggro{UserId: 1}
	r := &rooms.Room{RoomId: 99999}
	r.Items = []items.Item{{ItemId: 1}}
	if EquipBestFloorItem(m, r) {
		t.Errorf("expected false for in-combat mob")
	}
}
```

The "real swap" case (room with items + eligible mob + at least one item scores as upgrade) needs fixture-loaded item specs to compute meaningful scores. Add a skip-with-reason for completeness:

```go
func TestEquipBestFloorItem_RealSwap_FixtureLimited(t *testing.T) {
	t.Skip("requires loaded item specs to score IsUpgrade meaningfully; covered by Task 6 smoke")
}
```

- [ ] **Step 4: Build and run tests**

```
go build ./internal/itemvalue/...
go test ./internal/itemvalue/ -v
```

Expected: all PASS or SKIP. New tests pass for the gate paths; the fixture-dependent test skips cleanly.

- [ ] **Step 5: Commit**

```bash
git add internal/itemvalue/equip_eligibility.go internal/itemvalue/equip_eligibility_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): EquipBestFloorItem for idle floor-loot scan

Scans room.Items for the best ItemValueDelta-positive upgrade
for the mob, calls actions.EquipItem to perform the swap, emits
distinct "picks up X and dons/wields it" room broadcast text.
Gated by CanScanFloorLoot + combat-state + non-empty room.
Real-swap test deferred to smoke (needs fixture-loaded items).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Rewrite gearup mobcommand

**Files:**
- Modify: `internal/mobcommands/gearup.go` (full rewrite)

The current body uses `item.Value` (gold value) as the upgrade heuristic. Replace with `itemvalue.IsUpgrade` and add the `CanEquipFromGive` gate.

- [ ] **Step 1: Read the current gearup.go to understand existing behavior**

```
Read internal/mobcommands/gearup.go
```

Note the existing patterns to preserve:
- `PermaGear` buff check (boss equipment lockdown)
- Charmed-mob convention: drop displaced items
- Wild mob convention: keep displaced items in backpack
- Emote text: "puts on" for wearable, "wields" for weapon

- [ ] **Step 2: Replace the file with the new body**

Overwrite `internal/mobcommands/gearup.go` with:

```go
package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Gearup attempts to equip items from the mob's backpack that
// would be upgrades over the currently-equipped gear. Heuristic:
// itemvalue.IsUpgrade (replacing the prior gold-value heuristic).
//
// Bare "gearup" iterates the backpack, equipping every item
// that scores as an upgrade. "gearup <item>" or "gearup !<id>"
// considers only the specified item.
//
// Non-combat archetypes, animal species, and Incorporeal rank-4
// mobs are gated out: non-combat + animal via CanEquipFromGive,
// Incorporeal via gear scoring 0 (no special skip path needed).
func Gearup(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	// PermaGear buff blocks any equip change (boss equipment lockdown).
	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	// Animal species, non-combat archetypes, prey, etc. silently skip.
	if !itemvalue.CanEquipFromGive(mob) {
		return true, nil
	}

	profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
	actor := &actions.MobActor{Mob: mob, Room: room}

	if rest != "" {
		// Specific-item case: "gearup !12345" or "gearup sword name"
		candidate, found := mob.Character.FindInBackpack(rest)
		if !found {
			return true, nil
		}
		if !itemvalue.IsUpgrade(&mob.Character, profile, candidate) {
			return true, nil
		}
		equipAndDisplay(actor, candidate, mob, room)
		return true, nil
	}

	// Bare "gearup": scan backpack, equip each item that's an upgrade.
	// Iteration order doesn't matter — equipping changes the baseline,
	// so subsequent IsUpgrade calls reflect the new loadout.
	backpackItems := mob.Character.GetAllBackpackItems()
	for _, itm := range backpackItems {
		if !itemvalue.IsUpgrade(&mob.Character, profile, itm) {
			continue
		}
		equipAndDisplay(actor, itm, mob, room)
	}
	return true, nil
}

// equipAndDisplay invokes actions.EquipItem, emits the room
// broadcast text, and (for charmed mobs) drops displaced items
// to the floor so the owner can reclaim them. Wild mobs keep
// displaced items in their backpack (actions.EquipItem default).
func equipAndDisplay(actor *actions.MobActor, itm items.Item, mob *mobs.Mob, room *rooms.Room) {
	isCharmed := mob.Character.IsCharmed()
	oldEquipped := mob.Character.Equipment.GetAllItems()

	result := actions.EquipItem(actor, itm.Name())
	if !result.Equipped {
		return
	}

	spec := result.Item.GetSpec()
	if spec.Subtype == items.Wearable {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> puts on <ansi fg="item">%s</ansi>.`,
			mob.Character.Name, result.Item.DisplayName()))
	} else {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> wields <ansi fg="item">%s</ansi>.`,
			mob.Character.Name, result.Item.DisplayName()))
	}

	// Charmed-mob convention: drop displaced items to the floor.
	if isCharmed {
		newEquipped := mob.Character.Equipment.GetAllItems()
		for _, oldItm := range oldEquipped {
			if oldItm.ItemId < 1 {
				continue
			}
			stillEquipped := false
			for _, newItm := range newEquipped {
				if oldItm.ItemId == newItm.ItemId {
					stillEquipped = true
					break
				}
			}
			if !stillEquipped {
				mob.Command(fmt.Sprintf(`drop !%d`, oldItm.ItemId))
			}
		}
	}
}
```

- [ ] **Step 3: Build the whole module**

```
go build ./...
```

Expected: clean compile.

- [ ] **Step 4: Run existing mobcommands tests for regressions**

```
go test ./internal/mobcommands/ -v
```

Expected: all PASS. Note: if any pre-existing test asserted specific behavior from the old gold-value heuristic, it may need to be updated. Read failing test output carefully.

- [ ] **Step 5: Add at least one new test for the rewritten behavior**

Create or extend `internal/mobcommands/gearup_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestGearup_PermaGearBuffBlocks(t *testing.T) {
	// Mobs with PermaGear buff should emit struggle emote and no-op.
	m := &mobs.Mob{BehaviorArchetype: "generic_fighter"}
	m.Character = characters.Character{}
	m.Character.Buffs.AddBuff(buffs.PermaGear, "test")
	r := &rooms.Room{RoomId: 99999}

	handled, err := Gearup("", m, r)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handled {
		t.Errorf("expected handled=true")
	}
	// No assertion on emote (would require capturing room broadcasts);
	// the no-error + handled-true confirms the early-return path fired.
}

func TestGearup_NonEligibleArchetypeSkips(t *testing.T) {
	// Shopkeeper archetype should no-op silently.
	m := &mobs.Mob{BehaviorArchetype: "noncombat_shopkeeper"}
	m.Character = characters.Character{}
	r := &rooms.Room{RoomId: 99999}

	handled, err := Gearup("any item name", m, r)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handled {
		t.Errorf("expected handled=true")
	}
}
```

The "real upgrade-equip" cases require fixture-loaded items + a configured Equipment slot — defer those to Task 6 smoke.

- [ ] **Step 6: Run new tests**

```
go test ./internal/mobcommands/ -run TestGearup -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/mobcommands/gearup.go internal/mobcommands/gearup_test.go
git commit -m "$(cat <<'EOF'
refactor(mobcommands): rewrite gearup to use itemvalue.IsUpgrade

Replaces the old gold-value upgrade heuristic with
itemvalue.IsUpgrade from chunk 2.2 — properly archetype-aware
(via WeightProfile), slot-conflict-aware (2H weapons displace
both Weapon and Offhand), and incorporeal-aware (chunk 2.2a's
gear-effectiveness multiplier naturally zeros gear scores for
rank-4 ethereal mobs).

Adds CanEquipFromGive gate so animal species (Weapon slot
disabled) and non-combat archetypes silently skip. Charmed-
mob drop convention for displaced items preserved.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: MobIdle hook integration

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`

Add one line that calls `itemvalue.EquipBestFloorItem(mob, room)` alongside the existing per-mob idle behaviors.

- [ ] **Step 1: Read MobIdle_HandleIdleMobs.go around lines 100-200**

```
Read internal/hooks/MobIdle_HandleIdleMobs.go offset 100 limit 100
```

Find the section after the TickMobCraft block (around line 140) and before the gossiper-mob tick (around line 132). The exact insertion point: after the existing crafter/shop logic, before the gossip tick, so floor-loot pickup happens before gossip decisions but after combat-readiness behaviors.

- [ ] **Step 2: Add the import and the call**

Add to the import block: `"github.com/GoMudEngine/GoMud/internal/itemvalue"`.

Add this block somewhere in `HandleIdleMobs` AFTER the existing TickMobCraft section and BEFORE the gossip tick. A safe insertion point is right after the crafter block ends (around line 130) and before the `if mobHasGroup(mob, "gossiper")` line:

```go
	// Floor-loot scan: wild non-charmed combat mobs pick up
	// gear upgrades they find on the room floor (chunk 2.3).
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		itemvalue.EquipBestFloorItem(mob, room)
	}
```

Use the existing `rooms.LoadRoom(mob.Character.RoomId)` pattern that other sections in this file use — same way they get a `*rooms.Room` from the mob.

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: clean compile.

- [ ] **Step 4: Run tests**

```
go test ./internal/hooks/ -v
go test ./internal/itemvalue/ -v
```

Expected: all PASS. No new tests added here — the hook integration is just a one-line call to an already-tested function.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "$(cat <<'EOF'
feat(hooks): wire EquipBestFloorItem into MobIdle handler

Every idle tick, eligible mobs (non-charmed, combat archetype,
not in combat) scan their room's floor items via
itemvalue.EquipBestFloorItem. The function is a fast no-op for
the common case (no eligible mob OR no floor items OR no
upgrades) — gates exit in ~5 operations. Cost only paid when
all gates pass AND items exist on the floor.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: context.md update

**Files:**
- Modify: `internal/itemvalue/context.md`

Document the new helpers + push/pull behaviors in the package context.

- [ ] **Step 1: Read the existing context.md**

```
Read internal/itemvalue/context.md
```

Locate the public API section and the integration notes section.

- [ ] **Step 2: Add the three new public functions to the API section**

After the existing `IsUpgrade` entry, add:

```markdown
- `CanEquipFromGive(mob *mobs.Mob) bool` — gates push-equip
  (skips animal species, non-combat archetypes).
- `CanScanFloorLoot(mob *mobs.Mob) bool` — gates pull-equip
  (CanEquipFromGive + !IsCharmed).
- `EquipBestFloorItem(mob *mobs.Mob, room *rooms.Room) bool` —
  scans floor items, equips best ItemValueDelta-positive
  upgrade, emits room broadcast.
```

- [ ] **Step 3: Add a "Equip-If-Better Integration" section**

Append (or place near the existing algorithm descriptions):

```markdown
## Equip-If-Better Integration (Chunk 2.3)

Two consumer paths use the IsUpgrade primitive to drive
automatic equip behavior on NPCs:

**Push (give-equip):** When a player gives a mob an item,
`give.go` falls through to `gearup` (after btree handling),
which calls `IsUpgrade` per backpack item and equips upgrades.
Non-combat archetypes and animal species skip via
`CanEquipFromGive`. Charmed mobs (companions, mercs) accept
pushes from their owner.

**Pull (idle floor-loot scan):** Every idle tick, eligible
mobs call `EquipBestFloorItem(mob, room)` which scans
`room.Items`, scores each via `ItemValueDelta`, and equips the
best positive-scoring upgrade. Combat-state gated. Charmed
mobs are additionally excluded via `CanScanFloorLoot` (owner
has dibs on floor loot).

Incorporeal mobs (chunk 2.2a) are handled automatically: their
gear scores 0 via `GearEffectivenessMultiplier`, so `IsUpgrade`
returns false and no swap occurs. No special incorporeal skip
path is needed at the eligibility-gate layer.
```

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/context.md
git commit -m "$(cat <<'EOF'
docs(itemvalue): document chunk 2.3 equip-if-better integration

Adds CanEquipFromGive, CanScanFloorLoot, EquipBestFloorItem to
the public API section. Adds Equip-If-Better Integration
section documenting the push (gearup rewrite) and pull (idle
floor-scan) paths, plus the natural incorporeal handling via
chunk 2.2a's gear-effectiveness scaling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Smoke + roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- (No source changes; smoke verification only)

Final verification + roadmap closeout.

- [ ] **Step 1: Build + tests**

```
go build -o dogmud.exe .
go test ./... 2>&1 | tail -50
```

Expected: clean build; all packages PASS (only pre-existing fixture-dependent SKIPs OK). No FAILs.

- [ ] **Step 2: Server boot smoke**

Start the server in the background using bash run_in_background:
```
./dogmud.exe
```

Wait 8-12 seconds. Read background output. Confirm:
- `mobs.LoadDataFiles()` and `mutations.LoadMutationFiles()` both report unchanged counts vs prior chunk
- No `panic:` or `fatal error:` lines
- Server reaches "Server Ready" state

Kill the server, remove `dogmud.exe`.

- [ ] **Step 3: Optional admin spot-check**

If feasible to do interactively, exercise the push + pull paths via admin:
- **Push positive:** spawn a bandit, give it a stronger weapon than its current → expect "wields" emote
- **Push negative:** give it an inferior weapon → no equip (stays in backpack)
- **Push animal:** give a sword to a wolf → no-op
- **Pull positive:** drop a stronger-than-equipped item in a wild bandit's room → next idle tick, bandit picks up and wields, emits "picks up X and wields it"
- **Pull charmed:** drop item near your companion → companion ignores it

Document any findings; if anything misbehaves, capture details for follow-up.

- [ ] **Step 4: Update the progress tracker**

In `MOB_ALIVENESS_ROADMAP.md`, find the Progress tracker table. Change the row for chunk 2.3 from `Not started` to `Done`.

The 2.3 row currently reads:
```
| 2.3 | Tactical | Equip-if-better behavior | S | 2.2 | Not started |
```

Change to:
```
| 2.3 | Tactical | Equip-if-better behavior | S | 2.2 | Done |
```

- [ ] **Step 5: Update the chunk 2.3 mini-brief**

Find the section `### 2.3 Equip-if-better behavior`. Change the Status line from `Not started • **Size:** S` to `Done (2026-05-11) • **Size:** S`.

After the existing bullet list, append a `**Shipped:**` paragraph:

```
- **Shipped:** Two gate helpers in `internal/itemvalue/`:
  `CanEquipFromGive` (skips animal species via
  Species.DisabledSlots + non-combat archetypes) and
  `CanScanFloorLoot` (above + charmed-status). New
  `EquipBestFloorItem(mob, room) bool` function for idle-tick
  floor-loot scan; wired into
  `internal/hooks/MobIdle_HandleIdleMobs.go`. Existing
  `internal/mobcommands/gearup.go` rewritten to use
  `itemvalue.IsUpgrade` instead of the gold-value heuristic;
  PermaGear / charmed-drop / emote phrasing all preserved.
  Push and pull broadcast emotes are distinct ("puts on" /
  "wields" for push; "picks up X and dons it" / "wields it"
  for pull). Incorporeal mobs (chunk 2.2a) skip naturally via
  gear-effectiveness scoring — no special path. Per-archetype
  configurability is satisfied by chunk 2.2's WeightProfile
  system (no new knobs). Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.3-equip-if-better-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.3-equip-if-better.md`.
```

- [ ] **Step 6: Bump the roll-up**

Find `**Roll-up:** 10 / 41 done • ...` and update to:

```
**Roll-up:** 11 / 41 done • 0 in progress • 30 not started.
```

- [ ] **Step 7: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark chunk 2.3 (equip-if-better behavior) as Done

Ships the two gate helpers (CanEquipFromGive, CanScanFloorLoot),
the EquipBestFloorItem floor-scan function wired into MobIdle,
and the gearup mobcommand rewrite that swaps the gold-value
heuristic for itemvalue.IsUpgrade. Incorporeal handling is
automatic via chunk 2.2a's gear-effectiveness scoring. Roll-up
moves to 11/41.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run before declaring done)

- [ ] `go build ./...` passes clean
- [ ] `go test ./internal/itemvalue/ ./internal/mobcommands/ ./internal/hooks/ -v` all green
- [ ] `internal/itemvalue/equip_eligibility.go` has all three new functions
- [ ] `internal/mobcommands/gearup.go` uses `itemvalue.IsUpgrade` (no `item.Value` upgrade comparison remaining)
- [ ] `internal/hooks/MobIdle_HandleIdleMobs.go` calls `itemvalue.EquipBestFloorItem`
- [ ] Server boots cleanly past data load
- [ ] Roadmap roll-up updated to 11/41
