# Phase 3: Item Cleanup + Charm Migration Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete 11 dead item JS files, migrate 1 item script to YAML, port charm spell to Go. Eliminates 13 JS files total.

**Architecture:** Dead code deletion (no changes needed), one YAML field set on ItemSpec for use-triggered skill training, one Go function for charm spell resolution.

**Tech Stack:** Go, YAML, existing combat/dice packages

**Spec:** `docs/superpowers/specs/2026-04-11-phase3-items-charm-cleanup-design.md`

---

### Task 1: Delete 11 Dead Default Item JS Files

**Files:**
- Delete: 11 files in `_datafiles/world/default/items/other-0/`

- [ ] **Step 1: Delete all 11 files**

```bash
cd _datafiles/world/default/items/other-0
git rm 4-winterfire_crystal.js 6-sleeping_bag.js 10-history_of_frostfang.js \
      19-the_shadow_herbarium.js 21-stat_coupon.js 22-training_coupon.js \
      24-spellbound_projectiles.js 26-broom.js 100-newbie_kit.js \
      101-arcane_flute.js 102-room_rental.js
```

- [ ] **Step 2: Verify build still passes**

Run: `go build ./...`

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: delete 11 unused default item JS scripts

All are either shadowed by DOGMud items at the same ID or not
referenced anywhere in the DOGMud world.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add Item Use Fields to ItemSpec + Migrate Recipe Page

**Files:**
- Modify: `internal/items/itemspec.go`
- Modify: `internal/hooks/` or `internal/scripting/item.go` (wherever item use is handled)
- Modify: `_datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.yaml`
- Delete: `_datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.js`

- [ ] **Step 1: Add fields to ItemSpec**

In `internal/items/itemspec.go`, find the `ItemSpec` struct. Add these
fields (group them with a comment, similar to how spell/buff text fields
were added):

```go
	// YAML-driven use effects — replaces JS onUse/onCommand_use
	OnUseTrainSkill  string `yaml:"on_use_train_skill,omitempty"`
	OnUseTrainAmount int    `yaml:"on_use_train_amount,omitempty"`
	OnUseUserText    string `yaml:"on_use_user_text,omitempty"`
	OnUseRoomText    string `yaml:"on_use_room_text,omitempty"`
```

- [ ] **Step 2: Add YAML-driven item use handler**

Find where item use events are dispatched. The item script system uses
`TryItemScriptEvent` in `internal/scripting/item.go`. The herbalism
recipe page uses an `onUse` hook (not `onCommand_use`).

Search for where `onUse` is called — it may be in `internal/scripting/item.go`
or in the item event dispatch in hooks. Read the code to find the exact
call site.

BEFORE the JS `onUse` call (same pattern as spells/buffs — YAML first,
then JS), add:

```go
	// Handle YAML-driven use effects
	itemSpec := items.GetItemSpec(item.ItemId)
	if itemSpec != nil && itemSpec.OnUseTrainSkill != "" {
		user.Character.TrainSkill(itemSpec.OnUseTrainSkill, itemSpec.OnUseTrainAmount)
		// Send YAML text
		if itemSpec.OnUseUserText != "" {
			msg := textutil.SubstituteTokens(itemSpec.OnUseUserText, textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			})
			user.SendText(msg)
		}
		if itemSpec.OnUseRoomText != "" {
			room := rooms.LoadRoom(user.Character.RoomId)
			if room != nil {
				msg := textutil.SubstituteTokens(itemSpec.OnUseRoomText, textutil.TokenContext{
					SourceName:      user.Character.GetCharacterName(true),
					SourcePlainName: user.Character.GetCharacterName(false),
				})
				room.SendText(msg, user.UserId)
			}
		}
		// Consume a use
		item.AddUsesLeft(-1)
	}
```

**IMPORTANT:** Read the actual code to understand:
- How `TrainSkill` is called in Go (check `internal/characters/` for the method signature)
- How `AddUsesLeft` works on items (check `internal/items/`)
- Whether `onUse` returns a boolean that prevents further processing
- Adjust variable names to match what's in scope

If `OnUseTrainAmount` is 0, default it to 1.

- [ ] **Step 3: Add fields to recipe page YAML**

Edit `_datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.yaml`, append:

```yaml
on_use_train_skill: search
on_use_train_amount: 1
on_use_user_text: "You study the recipe page, tracing the careful handwriting and the teacher's corrections. The technique for fever-bark extraction is elegant in its simplicity -- lower heat, longer time, patience as an ingredient. Something clicks in your understanding of how plants yield their properties."
```

- [ ] **Step 4: Delete recipe page JS**

```bash
git rm _datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.js
```

- [ ] **Step 5: Verify build**

Run: `go build ./...`

- [ ] **Step 6: Commit**

```bash
git add internal/items/itemspec.go internal/scripting/item.go _datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.yaml
git rm _datafiles/world/dogmud/items/materials-40000/40042-herbalism_recipe_page.js
git commit -m "feat: add YAML-driven item use effects, migrate recipe page

New fields: on_use_train_skill, on_use_train_amount, on_use_user_text,
on_use_room_text. Herbalism recipe page migrated, JS deleted.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Write resolveCharmSpell Go Function

**Files:**
- Create: `internal/hooks/charm_spell.go`

- [ ] **Step 1: Create charm_spell.go**

The function replaces charm.js's `onMagic` logic. Read the existing
charm.js (in the worktree, it still exists until we delete it) to
understand the exact formulas. Here's what the JS does:

```
Attack score:
  charisma + manifestation_skill * 25

Defense score:
  target_willpower + round(target_stat_training_total * 0.10)

Aggro penalties (applied to ATTACK score, not defense):
  - Target fighting caster: attack *= 0.75
  - Target fighting someone else: attack *= 0.85

Opposed roll: RollOpposed(attack, defense)

On success:
  - CharmSet(caster_user_id, 99999)  (permanent charm)
  - AddCompanion
  - Send success text to caster and room

On failure:
  - Send failure text
  - Random selection from 5 failure messages
```

Function signature:
```go
func resolveCharmSpell(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room) bool
```

Key Go APIs to use:
- `user.Character.Stats.Charisma.ValueAdj` — charisma stat
- `user.Character.GetSkillLevel(skills.Manifestation)` — manifestation rank
- `mob.Character.Stats.Willpower.ValueAdj` — target willpower
- `mob.Character.GetStatTrainingTotal()` — target total stat pool
- `mob.Character.IsCharmImmune()` or check species flags — charm immunity
- `user.Character.GetMaxCompanions()` / `len(user.Character.Companions)` �� cap check
- `dice.OpposedRollStat(attack, defense)` — the opposed roll
- `mob.Character.Charm` — charm state (find the Set method)
- `user.Character.AddCompanion(...)` — register companion

Error messages should be descriptive, sent to caster only. Success
messages sent to caster and room.

Read `internal/hooks/companion_summon.go` for reference on how the
companion registration pattern works — charm follows the same post-success
flow (charm + add companion).

- [ ] **Step 2: Verify build**

Run: `go build ./internal/hooks/...`

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/charm_spell.go
git commit -m "feat: add resolveCharmSpell Go function

Opposed roll with charisma+manifestation vs willpower+statpool.
Aggro penalties, charm immunity check, companion registration.
Replaces charm.js onMagic logic.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Wire Charm into Spell Resolution + Delete JS

**Files:**
- Modify: `internal/hooks/spell_resolution.go`
- Modify: `_datafiles/world/dogmud/spells/charm.yaml`
- Delete: `_datafiles/world/dogmud/spells/charm.js`

- [ ] **Step 1: Add effect_type to charm.yaml**

Edit `_datafiles/world/dogmud/spells/charm.yaml`, add:

```yaml
effect_type: charm
```

- [ ] **Step 2: Wire charm resolution in spell_resolution.go**

In `spell_resolution.go`, find the `resolveSpell` function. In the
section where spell effects are applied (near the companion summon
check added in Phase 2), add a check for charm:

```go
	// Resolve charm spell (if configured)
	if spellData.EffectType == "charm" {
		if len(cs.TargetMobInstanceIds) > 0 {
			if targetMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); targetMob != nil {
				resolveCharmSpell(user, targetMob, room)
			}
		}
	}
```

This should go near the companion summon check, before the JS `onMagic`
call. Read the existing code to find the right insertion point and
verify variable names.

- [ ] **Step 3: Delete charm.js**

```bash
git rm _datafiles/world/dogmud/spells/charm.js
```

- [ ] **Step 4: Verify build**

Run: `go build ./internal/hooks/...`

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/spell_resolution.go _datafiles/world/dogmud/spells/charm.yaml
git rm _datafiles/world/dogmud/spells/charm.js
git commit -m "feat: wire charm spell resolution, delete charm.js

Charm now resolved via Go resolveCharmSpell when effect_type=charm.
Last non-mob/room spell JS file eliminated.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Update Schema Docs + Patch Notes

**Files:**
- Modify: `docs/schemas/item.md` (if it exists)
- Modify: `docs/schemas/spell.md`
- Modify: `PATCH_NOTES.md`

- [ ] **Step 1: Update item schema**

If `docs/schemas/item.md` exists, add the 4 new item use fields to the
field reference table:

| `on_use_train_skill` | string | no | Skill trained when item is used |
| `on_use_train_amount` | int | no | Amount to train (default 1) |
| `on_use_user_text` | string | no | Text sent to user on use |
| `on_use_room_text` | string | no | Text sent to room on use |

- [ ] **Step 2: Update spell schema**

In `docs/schemas/spell.md`, add `charm` to the list of valid
`effect_type` values:

```
| `charm` | Charm a mob into becoming a companion |
```

- [ ] **Step 3: Update patch notes**

Add Phase 3 entry to `PATCH_NOTES.md`.

- [ ] **Step 4: Commit**

```bash
git add docs/schemas/item.md docs/schemas/spell.md PATCH_NOTES.md
git commit -m "docs: update schemas and patch notes for Phase 3

New item use fields, charm effect_type, dead code cleanup notes.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
