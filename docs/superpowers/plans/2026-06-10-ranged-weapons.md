# Ranged Weapons System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the loaded-weapon ranged combat system per `docs/superpowers/specs/2026-06-10-ranged-weapons-design.md` — instant `shoot`, `reload` on the shared special-move cooldown, ammo bundles, revived Perception-governed `ranged-combat` skill, crime/justice integration, and full archer AI.

**Architecture:** Loaded state + ammo tags live on items. Firing resolves immediately through the existing `combat.ExecuteSkillMove` deliberate-attack path (same machinery as kick/trip/bash) with Perception substituted for Strength, gated on a loaded weapon and consuming the loaded state; `reload` consumes the shared special-move cooldown + one ammo Use. The legacy continuous remote-shoot auto-attack is REPLACED by this model. Crime/witness/justice flows fire on the victim's room. Archer AI rides the behavior-tree special-move delegation pattern (beast-move precedent).

**Tech Stack:** Go, `internal/items`, `internal/actions` (actor-parity pattern), `internal/combat` (ExecuteSkillMove pipeline), `internal/skills`, `internal/crimes`/`factions`/`justice`/`knowledge`, `internal/behaviortree`, YAML content.

**Branch:** `feature/ranged-weapons` off `master`.

---

## Verified API facts (do not re-derive)

- **Immediate-attack model:** `combat.ExecuteSkillMove(p combat.SkillMoveParams) SkillMoveResult` (internal/combat/skill_moves.go:51) — opposed roll `(AttackSkill+AttackStat) vs (DefenseSkill+DefenseStat)` via `dice.OpposedRollStat`, damage `CalcRawDamage(DamageStat, SkillRank, DamagePercent, ChannelPhysical)` → `ApplyMitigation(…, GetPhysicalMitigation()×MitigationMultiplier, cap)` → `dice.RollStat`, applies HP loss + optional knockdown directly. `SkillMoveResult{Hit, Damage, KnockedDown, TargetMaxHP}`.
- **Special-move pattern:** `actions.ExecuteKick(actor Actor) KickResult` (internal/actions/combat_kick.go:64) — `char.IsActing()` guard, `char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown))`, `ResolveAggroTarget(char.Aggro)`, ExecuteSkillMove, `RecordAndWait(char, moveName, sourceType, target.Char, targetType, hit, dmg, util.GetRoundCount())`, `actor.OnSkillUse(...)`. Player wrapper internal/usercommands/kick.go (resolves target + SetAggro when out of combat, formats messages); mob wrapper internal/mobcommands/kick.go. Wiring documented in `docs/superpowers/plans/2026-06-09-nonhuman-attacks-phase3-beast-moveset.md` lines 22-23.
- **Cooldowns:** `Character.Cooldowns.Try(key, "N rounds")` — shared by players and mobs (Character is common). `configs.GetBalanceConfig().SpecialMoveCooldown` (ConfigInt, default 5, config.balance.go:110).
- **Existing shoot:** `actions.ExecuteShoot(actor, rest) ShootResult` (internal/actions/combat_shoot.go:78) — parses `<target words...> <direction>`, resolves exit + adjacent-room target, charm/non-combatant guards, then `char.SetAggroRemote(exitName, userId, mobInstanceId, characters.Shooting)`. The round loop then attacks the remote target every round: cross-room is allowed when `Aggro.ExitName` resolves to the defender's room (NewRound_DoCombat_unified.go:188-219). Player command internal/usercommands/shoot.go, mob command internal/mobcommands/shoot.go (check exact name), `attack` sets `attkType = characters.Shooting` for shooting-subtype weapons (usercommands/attack.go:113).
- **Items:** `Item` struct (internal/items/items.go:40) — `Uses int yaml:"uses,omitempty"`, instance fields persist via yaml; `Spec *ItemSpec yaml:"overrides,omitempty"`; `GetSpec()`. `ItemSubType` consts at internal/items/itemspec.go:155+ (`Shooting ItemSubType = "shooting"` line 159). `BlockRating int yaml:"blockrating,omitempty"` exists on ItemSpec (itemspec.go:249) — shield block bonus. Item TYPE list also in itemspec.go (grep `ItemType =` for the enum block when adding `Ammo`).
- **Character:** `Equipment.Weapon` / `Equipment.Offhand` are `items.Item` VALUES — mutations must be written back to the field (see how `UseItem` handles Uses writeback). `Stats.Perception.ValueAdj`. `HealthMax.Value`. `IsHidden()`.
- **Skills:** SkillTag consts internal/skills/skills.go:27-46 (no RangedCombat currently); `allSkillNames`; progression multipliers map ~line 295 (`UnarmedCombat: 0.3`); skill list for display ~line 371. Retirement strip: internal/characters/validate.go:286 `for _, dead := range []string{"cast", "ranged-combat", "first-aid"}` with tests in internal/characters/godfunc_refactor_test.go:355-367. `CombatSkillTagForItem` (internal/characters/skills.go:253) returns WeaponCombat for everything but Claws/Fist.
- **Crime flow:** `recordAssaultCrime(user, mob, room)` (internal/usercommands/attack.go:333) — `factions.FactionsForMob`, `crimes.WitnessesInRoom(factionIds, room, excludeInstanceId)`, `crimes.IdentifiedPerp(userId, witnesses)`, `crimes.Record(...)`, `factions.BumpRep`, `justice.MaybeDeclareBounty`, `knowledge.RecordCrimeWitnessed`. The `room` parameter is where witnesses are sought — for cross-room shots pass the VICTIM's room.
- **Btree actions:** registry `actionRegistry map[string]ActionFunc` in internal/behaviortree/actions.go:11, registered in init(). Beast-move delegation pattern (predator/generic_fighter btrees delegating special-move selection) shipped in commits 807dc5d9 / cb0863c6; the wiring recipe is documented in `docs/superpowers/plans/2026-06-09-nonhuman-attacks-phase3-beast-moveset.md` — implementers of Task 9 MUST read that file.
- **Balance config:** fields in internal/configs/config.balance.go, validation/defaults in config.balance.combat.go / config.balance.misc.go; live values in `_datafiles/config.yaml`.
- **ID allocation:** run `python tools/id_inventory.py --type items` before creating item YAML; allocate from the reported next-free IDs (or `--alloc items N` for a block). Same for mobs.
- **Verbosity interaction:** `CategoryHitRanged` is in the light-suppression table — at light verbosity a participant's own fire would be suppressed-and-tallied. The fire path sends its result through `user.SendText`/room sends directly (not the AttackResult drain), so it is NOT gated; that matches the spec note (deliberate actions stay visible, like kick).

---

### Task 0: Branch

- [ ] **Step 1:**
```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/ranged-weapons master
```
(Unrelated runtime artifacts exist in the tree — never `git add -A` in any task.)

---

### Task 1: Item plumbing — ammo tags, ammo type, loaded state

**Files:**
- Modify: `internal/items/itemspec.go` (ItemSpec fields + ItemType/const blocks)
- Modify: `internal/items/items.go` (Item instance field + helpers)
- Test: `internal/items/ranged_test.go` (create)

- [ ] **Step 1: Write the failing tests**

Create `internal/items/ranged_test.go`:

```go
package items

import "testing"

func TestItemSpec_RangedFields(t *testing.T) {
	spec := ItemSpec{Subtype: Shooting, AmmoTag: "bolts", MinStrength: 90}
	if spec.AmmoTag != "bolts" || spec.MinStrength != 90 {
		t.Fatalf("ranged spec fields not settable: %+v", spec)
	}
}

func TestItem_LoadedState(t *testing.T) {
	itm := Item{ItemId: 1, Loaded: true}
	if !itm.Loaded {
		t.Error("Loaded field must persist on the instance")
	}
}

func TestItem_IsRangedWeapon(t *testing.T) {
	// Training bow (10004) is subtype shooting in live data; use a spec
	// override to stay registry-independent.
	ranged := Item{ItemId: 1, Spec: &ItemSpec{ItemId: 1, Type: Weapon, Subtype: Shooting}}
	if !ranged.IsRangedWeapon() {
		t.Error("shooting-subtype weapon must report IsRangedWeapon")
	}
	melee := Item{ItemId: 2, Spec: &ItemSpec{ItemId: 2, Type: Weapon, Subtype: Slashing}}
	if melee.IsRangedWeapon() {
		t.Error("slashing weapon must not report IsRangedWeapon")
	}
	var none Item
	if none.IsRangedWeapon() {
		t.Error("zero item must not report IsRangedWeapon")
	}
}

func TestAmmoType(t *testing.T) {
	bundle := Item{ItemId: 3, Uses: 20, Spec: &ItemSpec{ItemId: 3, Type: Ammo, AmmoTag: "arrows"}}
	if bundle.GetSpec().Type != Ammo || bundle.GetSpec().AmmoTag != "arrows" {
		t.Errorf("ammo bundle spec: %+v", bundle.GetSpec())
	}
}
```

NOTE: verify how `GetSpec()` resolves the `Spec` override pointer (read items.go) and adapt the fixtures if overrides don't shortcut the registry — registering a test spec via whatever seam the package's existing tests use is the fallback (read an existing items test first).

- [ ] **Step 2: Run to verify failure**
`go test ./internal/items/ -run 'TestItemSpec_RangedFields|TestItem_Loaded|TestItem_IsRanged|TestAmmoType' -v` — FAIL (AmmoTag/MinStrength/Loaded/Ammo undefined).

- [ ] **Step 3: Implement**

In `internal/items/itemspec.go`:
- Find the `ItemType` const block and add: `Ammo ItemType = "ammo"` (and add it to any validation list of legal types — grep how existing types are validated).
- Add to `ItemSpec` (near BlockRating, matching tag style):
```go
	AmmoTag     string `yaml:"ammo_tag,omitempty"`     // Ranged weapons: ammo type required (arrows/bolts/shot). Ammo items: type provided.
	MinStrength int    `yaml:"min_strength,omitempty"` // Minimum Strength to wield (heavy bows/arbalest)
```

In `internal/items/items.go`:
- Add to `Item` after `Uses`:
```go
	Loaded        bool           `yaml:"loaded,omitempty"`        // Ranged weapons: projectile chambered/nocked
```
- Add helpers (near the existing small predicate methods — find one and colocate):
```go
// IsRangedWeapon reports whether this item is a shooting-subtype weapon
// (bow, crossbow, pistol, sling).
func (i *Item) IsRangedWeapon() bool {
	if i.ItemId == 0 {
		return false
	}
	spec := i.GetSpec()
	return spec != nil && spec.Type == Weapon && spec.Subtype == Shooting
}
```

- [ ] **Step 4: Verify pass**
`go test ./internal/items/ -count=1` — green (full package; existing tests must not regress).

- [ ] **Step 5: Min-Strength wield enforcement**

Find where weapons are equipped/wielded and validated (grep `func CanEquip|Wear|Wield` in internal/characters and internal/usercommands — the equip path that already rejects e.g. two-handers with full hands). Add a Strength gate: if `spec.MinStrength > 0 && char.Stats.Strength.ValueAdj < spec.MinStrength` → reject with a message ("You aren't strong enough to handle <item>." — no numbers). Add a unit test at whatever seam that function has (mirror an existing equip-rejection test).

- [ ] **Step 6: Commit**
```bash
git add internal/items/ internal/characters/ internal/usercommands/
git commit -m "feat(items): ammo type + ammo_tag/min_strength specs + loaded state for ranged weapons"
```
(Adjust the add list to the files actually touched in Step 5.)

---

### Task 2: `reload` — actions core + player/mob commands

**Files:**
- Create: `internal/actions/combat_reload.go`
- Create: `internal/usercommands/reload.go`, `internal/mobcommands/reload.go`
- Modify: `internal/usercommands/usercommands.go`, `internal/mobcommands/mobcommands.go` (registration maps)
- Test: `internal/actions/combat_reload_test.go`

- [ ] **Step 1: Write the failing test**

Read 1-2 existing tests in internal/actions (e.g. how forage_test.go/consider_test.go fake an Actor — they stub SendText etc.) and mirror that style. Behaviors to pin:

```go
// Reload behavior matrix:
// 1. No ranged weapon equipped (main or offhand) → NoWeapon.
// 2. Ranged weapon already loaded → AlreadyLoaded.
// 3. No ammo bundle with matching AmmoTag in backpack → NoAmmo (result
//    includes the needed tag for messaging).
// 4. Cooldown busy → OnCooldown (use Character.Cooldowns to pre-occupy
//    "special-move").
// 5. Success: bundle Uses decremented by 1; weapon Loaded=true written
//    back to the equipment slot; cooldown consumed (a second immediate
//    reload of a second weapon returns OnCooldown).
// 6. Bundle at Uses==1 → consumed/removed from backpack on success.
// 7. Offhand ranged + main melee: reload targets the offhand weapon.
```

Write these as table-driven subtests against `actions.ExecuteReload` (defined in Step 3), constructing a Character with Equipment.Weapon/Offhand and Items directly (the actions tests construct characters; follow their pattern).

- [ ] **Step 2: Run to verify failure** — compile error (ExecuteReload undefined).

- [ ] **Step 3: Implement `internal/actions/combat_reload.go`**

```go
package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ReloadResult holds the outcome of a reload attempt.
type ReloadResult struct {
	// WeaponName is the display name of the weapon reloaded (or found).
	WeaponName string
	// AmmoTag is the ammo type involved (set for NoAmmo messaging too).
	AmmoTag string
	// AmmoName is the display name of the bundle consumed from.
	AmmoName string
	// BundleEmptied is true when the bundle's last Use was consumed.
	BundleEmptied bool

	Loaded        bool // success
	NoWeapon      bool // no ranged weapon equipped
	AlreadyLoaded bool
	NoAmmo        bool
	OnCooldown    bool
	Crafting      bool
}

// findRangedWeaponSlot returns a pointer to the equipped ranged weapon
// (main hand first, then offhand) so Loaded can be written back, or nil.
func findRangedWeaponSlot(actor Actor) *items.Item {
	char := actor.GetCharacter()
	if char.Equipment.Weapon.IsRangedWeapon() {
		return &char.Equipment.Weapon
	}
	if char.Equipment.Offhand.IsRangedWeapon() {
		return &char.Equipment.Offhand
	}
	return nil
}

// ExecuteReload chambers/nocks a projectile into the actor's equipped
// ranged weapon (main hand first, then offhand): consumes the shared
// special-move cooldown and one Use from a matching ammo bundle.
// Callers handle all messaging and progression events.
func ExecuteReload(actor Actor) ReloadResult {
	char := actor.GetCharacter()

	if char.IsActing() {
		return ReloadResult{Crafting: true}
	}

	weapon := findRangedWeaponSlot(actor)
	if weapon == nil {
		return ReloadResult{NoWeapon: true}
	}
	if weapon.Loaded {
		return ReloadResult{WeaponName: weapon.DisplayName(), AlreadyLoaded: true}
	}

	ammoTag := weapon.GetSpec().AmmoTag

	// Find a matching ammo bundle in the backpack.
	var bundle *items.Item
	for idx := range char.Items {
		spec := char.Items[idx].GetSpec()
		if spec != nil && spec.Type == items.Ammo && spec.AmmoTag == ammoTag {
			bundle = &char.Items[idx]
			break
		}
	}
	if bundle == nil {
		return ReloadResult{WeaponName: weapon.DisplayName(), AmmoTag: ammoTag, NoAmmo: true}
	}

	// Shared special-move cooldown — the cost of the reload.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return ReloadResult{WeaponName: weapon.DisplayName(), OnCooldown: true}
	}

	result := ReloadResult{
		WeaponName: weapon.DisplayName(),
		AmmoTag:    ammoTag,
		AmmoName:   bundle.DisplayName(),
		Loaded:     true,
	}

	// Consume one Use; remove the bundle when emptied.
	bundle.Uses--
	if bundle.Uses <= 0 {
		result.BundleEmptied = true
		char.RemoveItem(*bundle)
	}

	weapon.Loaded = true
	return result
}
```

IMPORTANT verification while implementing: `char.Items` element pointers and `char.RemoveItem(itm)` semantics — read how `UseItem`/`RemoveItem` work (internal/characters) and adapt (if RemoveItem matches by UUID/value, taking `*bundle` by value before mutation may be required; get the order right and pin it with test #6). Same for `Equipment.Weapon` writeback — taking `&char.Equipment.Weapon` mutates in place, which persists; confirm saves serialize Equipment items (they do — items carry yaml tags).

- [ ] **Step 4: Player + mob commands**

`internal/usercommands/reload.go` — model on kick.go's shape:
```go
func Reload(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	result := actions.ExecuteReload(&actions.UserActor{User: user, Room: room})

	switch {
	case result.Crafting:
		user.SendText(messaging.CategorySystem, "You're too busy to reload right now.")
	case result.NoWeapon:
		user.SendText(messaging.CategorySystem, "You don't have a ranged weapon equipped.")
	case result.AlreadyLoaded:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="itemname">%s</ansi> is already loaded.`, result.WeaponName))
	case result.NoAmmo:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You have no <ansi fg="item">%s</ansi> left to load your <ansi fg="itemname">%s</ansi> with.`, result.AmmoTag, result.WeaponName))
	case result.OnCooldown:
		user.SendText(messaging.CategorySystem, "You need a moment before you can reload.")
	case result.Loaded:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You ready your <ansi fg="itemname">%s</ansi>.`, result.WeaponName))
		room.SendText(messaging.CategoryRoomDescription, fmt.Sprintf(`<ansi fg="username">%s</ansi> readies their <ansi fg="itemname">%s</ansi>.`, user.Character.Name, result.WeaponName), user.UserId)
		if result.BundleEmptied {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`That was the last of your <ansi fg="itemname">%s</ansi>.`, result.AmmoName))
		}
	}
	return true, nil
}
```
(Verify `actions.UserActor{User:, Room:}` field names against kick.go's actual usage and the exact registration mechanism in internal/usercommands/usercommands.go — add `"reload": {...}` mirroring "kick"'s entry incl. help category. Same for mobcommands/reload.go mirroring mobcommands/kick.go: silent failures, registered in mobcommands.go.)

- [ ] **Step 5: Run everything**
`go build ./... && go test ./internal/actions/ ./internal/items/ -count=1` — green.

- [ ] **Step 6: Commit**
```bash
git add internal/actions/ internal/usercommands/ internal/mobcommands/
git commit -m "feat(combat): reload command — special-move cooldown + ammo bundle consumption"
```

---

### Task 3: Balance knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (+ validation in config.balance.combat.go)
- Modify: `_datafiles/config.yaml`
- Test: extend whatever config test pattern exists (grep TestBalance in internal/configs; if none for defaults, the validation function test style)

- [ ] **Step 1: Add fields** to the Balance struct (config.balance.go, near SpecialMoveCooldown):
```go
	RangedShotScale          ConfigFloat `yaml:"RangedShotScale"`          // Global multiplier on all ranged shot damage (default 1.0)
	RangedShieldDefenseBonus ConfigInt   `yaml:"RangedShieldDefenseBonus"` // Flat defense-score bonus vs ranged when defender has a shield (default 15)
```
Validation in config.balance.combat.go (mirror SpecialMoveCooldown's style): RangedShotScale `<=0 → 1.0`; RangedShieldDefenseBonus `<0 → 15` (allow 0 = disabled deliberately? choose: `<0 → 15`).

- [ ] **Step 2: config.yaml** — add under the Balance section near SpecialMoveCooldown with comments:
```yaml
  # - RangedShotScale - Global multiplier on ranged shot damage (tuning knob)
  RangedShotScale: 1.0
  # - RangedShieldDefenseBonus - Defense bonus vs ranged shots when holding a shield
  RangedShieldDefenseBonus: 15
```
(Match the file's exact comment/indent conventions in that section.)

- [ ] **Step 3: Test** — add to the configs package test that exercises balance validation (find it; if validation tests exist per-field, add the two; otherwise a small test calling the validate func with zero values asserting defaults).

- [ ] **Step 4:** `go build ./... && go test ./internal/configs/ -count=1` then commit:
```bash
git add internal/configs/ _datafiles/config.yaml
git commit -m "feat(configs): RangedShotScale + RangedShieldDefenseBonus balance knobs"
```

---

### Task 4: Skill revival — ranged-combat, Perception-governed

**Files:**
- Modify: `internal/skills/skills.go`
- Modify: `internal/characters/validate.go:286` + `internal/characters/godfunc_refactor_test.go:355-367`
- Modify: `internal/characters/skills.go:253` (`CombatSkillTagForItem`)
- Test: extend internal/skills + internal/characters tests

- [ ] **Step 1: Failing tests first.** In internal/characters (mirror existing test style):
```go
// CombatSkillTagForItem must return ranged-combat for shooting weapons.
func TestCombatSkillTagForItem_Shooting(t *testing.T) {
	bow := items.Item{ItemId: 1, Spec: &items.ItemSpec{ItemId: 1, Type: items.Weapon, Subtype: items.Shooting}}
	if got := CombatSkillTagForItem(bow); got != skills.RangedCombat {
		t.Errorf("shooting weapon skill tag = %v, want ranged-combat", got)
	}
}

// ranged-combat must no longer be stripped by Validate.
func TestValidate_RangedCombatSurvives(t *testing.T) {
	c := New()
	c.Skills["ranged-combat"] = 5
	c.Validate()
	if _, ok := c.Skills["ranged-combat"]; !ok {
		t.Error("ranged-combat must survive Validate (revived skill)")
	}
}
```
(Adapt constructor/Validate invocation to the file's existing tests — godfunc_refactor_test.go:355 shows the current strip test to MODIFY: remove "ranged-combat" from its retired-skill fixtures/assertions, keep "cast"/"first-aid".)

- [ ] **Step 2: Run to verify failure** (RangedCombat undefined; strip test currently asserts removal).

- [ ] **Step 3: Implement**
- skills.go const block: `RangedCombat SkillTag = \`ranged-combat\` // Bows, crossbows, pistols — aimed shots (Perception)` placed with the combat skills.
- Add RangedCombat everywhere the other combat skills are enumerated in skills.go: the registration/allSkillNames init, the progression-multiplier map (~line 295 — set `RangedCombat: 0.5` as a starting multiplier between UnarmedCombat 0.3 and crafting rates; confirm neighboring values and pick consistently), the display list (~line 371), and Professions: add a `"hunter"` entry `{RangedCombat, Search}` ONLY IF Professions affect anything load-bearing (read how Professions is consumed first; if it's chargen-only flavor, still fine).
- validate.go:286: remove `"ranged-combat"` from the dead list.
- characters/skills.go CombatSkillTagForItem:
```go
	if spec.Subtype == items.Shooting {
		return skills.RangedCombat
	}
```
before the WeaponCombat fallthrough.
- Update godfunc_refactor_test.go strip assertions accordingly.

- [ ] **Step 4: Sweep for "9 skills" assumptions** — `grep -rn "9 total\|nine skills\|len(allSkillNames)" internal/ _datafiles/world/dogmud/templates/` and fix anything that hardcodes the count (skill sheet templates usually iterate — verify the `skills` command renders the 10th without layout breakage by reading its template).

- [ ] **Step 5:** `go build ./... && go test ./internal/skills/ ./internal/characters/ -count=1` green, commit:
```bash
git add internal/skills/ internal/characters/
git commit -m "feat(skills): revive ranged-combat (10th skill), Perception-governed, shooting weapons train it"
```

---

### Task 5: Fire resolution — `actions.ExecuteFire`

**Files:**
- Create: `internal/actions/combat_fire.go`
- Modify: `internal/actions/combat_shoot.go` (reuse its target-resolution pieces; the SetAggroRemote behavior moves out of the action — see Step 3 notes)
- Test: `internal/actions/combat_fire_test.go`

- [ ] **Step 1: Failing tests.** Pin (actions-test style, direct Characters):

```go
// Fire behavior matrix:
// 1. No ranged weapon → NoWeapon.
// 2. Ranged weapon unloaded → NotLoaded (message names reload).
// 3. Loaded + same-room target resolved → shot resolves: weapon.Loaded
//    becomes false; result carries Hit/Damage/TargetMaxHP from the
//    SkillMove; attack uses Perception (verify by stacking the test
//    character's Perception very high vs a 1-Dex dummy → always hits;
//    and Strength 1 to prove damage doesn't collapse).
// 4. Shield defense: defender with offhand BlockRating>0 gets the
//    config bonus folded into their defense score (probe: identical
//    rolls seed not possible — instead assert the computed defenderScore
//    via a small exported-for-test helper or by giving the bonus an
//    extreme test value (e.g. set RangedShieldDefenseBonus high via
//    configs test override if the configs package supports overrides;
//    otherwise unit-test the score-builder helper directly).
// 5. Cross-room: loaded + valid exit + target in adjacent room →
//    resolves, Loaded=false, result flags CrossRoom + ExitName.
// 6. Firing does NOT consume the special-move cooldown (fire then an
//    immediate kick attempt must not be cooldown-blocked by the fire).
```

Structure the implementation so #4 is testable: extract `rangedDefenseScore(defender *characters.Character) float64` as a small pure helper.

- [ ] **Step 2: Run to verify failure.**

- [ ] **Step 3: Implement `internal/actions/combat_fire.go`**

```go
package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// FireResult holds the outcome of a shoot/fire attempt.
type FireResult struct {
	WeaponName string
	TargetName string
	// Target identities for the caller's retaliation/crime bookkeeping.
	TargetUserId        int
	TargetMobInstanceId int
	TargetRoomId        int
	IsTargetMob         bool
	CrossRoom           bool
	ExitName            string
	IsSneaking          bool

	MoveResult combat.SkillMoveResult
	Executed   bool

	// Early exits.
	NoWeapon       bool
	NotLoaded      bool
	BadSyntax      bool
	NoExit         bool
	ExitLocked     bool
	NoTarget       bool
	IsCharmed      bool
	IsNonCombatant bool
	Crafting       bool
}

// rangedDefenseScore computes the defender's score against a ranged
// shot: Dexterity + combat skill, plus the configured shield bonus when
// an offhand with a block rating is equipped. Parry contributes nothing
// to ranged defense by design (you can't parry a bolt).
func rangedDefenseScore(defender *characters.Character) float64 {
	score := float64(defender.Stats.Dexterity.ValueAdj) + float64(defender.GetCombatSkillLevel())
	if defender.Equipment.Offhand.ItemId > 0 {
		if spec := defender.Equipment.Offhand.GetSpec(); spec != nil && spec.BlockRating > 0 {
			score += float64(configs.GetBalanceConfig().RangedShieldDefenseBonus)
		}
	}
	return score
}

// ExecuteFire resolves a ranged shot immediately. rest is either
// "<target>" (same room) or "<target words...> <direction>" (adjacent
// room). The weapon must be loaded; firing unloads it. Firing does NOT
// consume the special-move cooldown — reloading does.
//
// Callers are responsible for: messaging, OnSkillUse/OnStatUse
// progression, retaliation aggro on the target, crime recording, and
// combat-initiation aggro for same-room shots.
func ExecuteFire(actor Actor, rest string) FireResult {
	char := actor.GetCharacter()

	if char.IsActing() {
		return FireResult{Crafting: true}
	}

	weapon := findRangedWeaponSlot(actor)
	if weapon == nil {
		return FireResult{NoWeapon: true}
	}
	if !weapon.Loaded {
		return FireResult{WeaponName: weapon.DisplayName(), NotLoaded: true}
	}

	args := strings.Fields(rest)
	if len(args) < 1 {
		return FireResult{BadSyntax: true}
	}

	// Try to interpret the last word as a direction for a cross-room
	// shot; fall back to a same-room target if it isn't an exit.
	room := actor.GetRoom()
	crossRoom := false
	exitName := ""
	targetRoom := room
	targetWords := args

	if len(args) >= 2 {
		if name, roomId := room.FindExitByName(args[len(args)-1]); name != "" {
			exitInfo, _ := room.GetExitInfo(name)
			if exitInfo.Lock.IsLocked() {
				return FireResult{WeaponName: weapon.DisplayName(), ExitName: name, ExitLocked: true}
			}
			if adj := rooms.LoadRoom(roomId); adj != nil {
				crossRoom = true
				exitName = name
				targetRoom = adj
				targetWords = args[:len(args)-1]
			}
		}
	}

	targetUserId, targetMobInstanceId := targetRoom.FindByName(strings.Join(targetWords, " "))
	if targetUserId == 0 && targetMobInstanceId == 0 {
		// Cross-room parse may have misfired (target name ended in an
		// exit-like word) — retry same-room with the full args.
		if crossRoom {
			crossRoom, exitName, targetRoom = false, "", room
			targetUserId, targetMobInstanceId = room.FindByName(strings.Join(args, " "))
		}
		if targetUserId == 0 && targetMobInstanceId == 0 {
			return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
		}
	}

	result := FireResult{
		WeaponName:          weapon.DisplayName(),
		TargetUserId:        targetUserId,
		TargetMobInstanceId: targetMobInstanceId,
		TargetRoomId:        targetRoom.RoomId,
		CrossRoom:           crossRoom,
		ExitName:            exitName,
		IsSneaking:          char.IsHidden(),
	}

	var defChar *characters.Character
	if targetMobInstanceId > 0 {
		m := mobs.GetInstance(targetMobInstanceId)
		if m == nil {
			return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
		}
		charmerKey := actor.GetUserId()
		if charmerKey == 0 {
			charmerKey = actor.GetMobInstanceId()
		}
		if m.Character.IsCharmed(charmerKey) {
			result.IsCharmed = true
			result.TargetName = m.Character.Name
			return result
		}
		if m.IsNonCombatant() || m.PlayerAttackImmune {
			result.IsNonCombatant = true
			result.TargetName = m.Character.Name
			return result
		}
		result.IsTargetMob = true
		result.TargetName = m.Character.Name
		defChar = &m.Character
	} else {
		u := users.GetByUserId(targetUserId)
		if u == nil {
			return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
		}
		result.TargetName = u.Character.Name
		defChar = u.Character
	}

	// The shot: unload first (fires even on a miss), then resolve.
	weapon.Loaded = false

	cfg := configs.GetBalanceConfig()
	shotMult := weapon.GetSpec().DamageMultiplier * float64(cfg.RangedShotScale)
	rangedRank := char.GetSkillLevel(skills.RangedCombat)

	result.MoveResult = combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:             char,
		Defender:             defChar,
		AttackStat:           char.Stats.Perception.ValueAdj,
		AttackSkill:          rangedRank,
		DefenseStat:          0, // folded into DefenseSkill via rangedDefenseScore
		DefenseSkill:         int(rangedDefenseScore(defChar)),
		DamagePercent:        shotMult,
		KnockdownChance:      0,
		SkillRank:            rangedRank,
		DamageStat:           char.Stats.Perception.ValueAdj,
		MitigationMultiplier: 1.0,
	})
	result.Executed = true

	// Analytics + round consumption, mirroring other deliberate moves.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if result.IsTargetMob {
		targetType = combat.Mob
	}
	dmg := 0
	if result.MoveResult.Hit {
		dmg = result.MoveResult.Damage
	}
	RecordAndWait(char, "shoot", sourceType, defChar, targetType, result.MoveResult.Hit, dmg, util.GetRoundCount())

	return result
}
```

Implementation verifications while writing: `DamageMultiplier` field name on ItemSpec (the YAML key is `damage_multiplier` — confirm the Go field); `room.FindByName` signature (combat_shoot.go used `FindByName(...)` returning (playerId, mobInstanceId) — confirm); whether `RecordAndWait` consumes the round appropriately for a non-cooldown move (read it — if it sets RoundsWaiting, decide with the spec in hand: firing is deliberate, consuming the attacker's round is correct and matches kick); `weapon.Loaded = false` persists via the pointer from findRangedWeaponSlot.

NOTE on `DefenseStat: 0` — folding stat+skill+shield into DefenseSkill keeps `rangedDefenseScore` independently testable; ExecuteSkillMove just sums them.

- [ ] **Step 4:** Tests green: `go test ./internal/actions/ -count=1`.

- [ ] **Step 5: Balance pin test** — add to combat_fire_test.go:
```go
// Spec balance target: a 2h ranged shot at stat 100 / rank 0 / arbalest
// mult 7.0 must produce raw damage in the 180-220 band BEFORE mitigation:
// CalcRawDamage(100, 0, 7.0*1.0, ChannelPhysical) = 100×1.0×7.0×0.30 = 210.
func TestRangedShotRawDamage_BalanceBand(t *testing.T) {
	raw := combat.CalcRawDamage(100, 0, 7.0, combat.ChannelPhysical)
	if raw < 180 || raw > 220 {
		t.Errorf("arbalest baseline raw %v outside 180-220 spec band", raw)
	}
}
```
(Verify ChannelPhysical's exported name + CalcRawDamage signature; the arithmetic shows mult 7.0 lands exactly on the spec's 60-75%-of-melee-cycle target at 210.)

- [ ] **Step 6: Commit**
```bash
git add internal/actions/
git commit -m "feat(combat): ExecuteFire — immediate Perception-based ranged shot, shield defense, balance pin"
```

---

### Task 6: `shoot` command rewrite + retaliation + crime integration

**Files:**
- Modify: `internal/usercommands/shoot.go` (full rewrite to the new model)
- Modify: `internal/mobcommands/shoot.go` (check the actual filename; create if mobs lack one)
- Modify: `internal/usercommands/usercommands.go` (alias `fire` → shoot's entry)
- Modify: `internal/usercommands/attack.go` (lift `recordAssaultCrime` so shoot can call it — move to a shared location in the usercommands package or export-within-package; it's the same package, so shoot.go can call it directly — verify and just call it)
- Test: hook-level/integration tests (see Step 4)

- [ ] **Step 1: Read first.** `internal/usercommands/shoot.go` (current messaging + SetAggroRemote call), `internal/mobcommands/` for the mob shoot entry, and how mobs pursue out-of-room aggro (grep `UpdateCombatMemoryLocation` / `CombatMemory` in internal/mobs + hooks — this is the retaliation pathing you'll lean on).

- [ ] **Step 2: Rewrite the player command** around `actions.ExecuteFire`:

Flow (player):
1. `result := actions.ExecuteFire(&actions.UserActor{User: user, Room: room}, rest)`
2. Early-exit messages for NoWeapon ("You don't have a ranged weapon equipped."), NotLoaded (`Your <weapon> isn't loaded. Try <ansi fg="command">reload</ansi>.`), BadSyntax (usage line: `shoot <target> [direction]`), NoExit/ExitLocked/NoTarget/IsCharmed/IsNonCombatant — mirror the tone of the existing shoot.go messages (read them; reuse where they fit).
3. On Executed:
   - Messages (all no-hard-numbers; damage tier via `combat.GetDamageDescription(result.MoveResult.Damage, result.MoveResult.TargetMaxHP)`):
     - shooter (hit): `Your <weapon> bolt takes <target> (<tier>)!` — craft per-hit/miss lines; CategoryHitRanged for hits, CategoryDodge for the miss line ("...but <target> twists aside!").
     - shooter's room (cross-room: mention direction): `<name> fires <weapon> <direction>ward.` via room.SendTextVisual, exclude shooter. SUPPRESSED when result.IsSneaking (mirror existing shoot.go's sneak handling).
     - target's room (cross-room): impact line naming the arrival direction (reverse exit if cheaply derivable, else "from somewhere beyond the <exitName> exit" phrasing — read how the old shoot messaging handled it and reuse).
     - target (player targets): direct SendText describing being shot (CategoryHitRanged).
   - **Aggro semantics:**
     - Same-room: if shooter has no aggro, `user.Character.SetAggro(targetUserId, targetMobInstanceId, characters.DefaultAttack)` (melee fallback rounds follow — verify SetAggro's exact signature from attack.go usage).
     - Cross-room: do NOT set shooter aggro (no continuous remote auto-attack in the new model — this intentionally retires the legacy behavior).
   - **Retaliation (the spec requirement):** for mob targets, after a HIT:
     - `mob.Character.TrackPlayerDamage(user.UserId, dmg)` if not already done inside the fire path (ExecuteSkillMove does NOT call it — do it here).
     - Fire the `mob_hurt` behavior event exactly as `fireDefenderBehaviorTrigger` does (NewRound_DoCombat_unified.go:645 — replicate: behaviortree.TryMobBehavior with EventContext{EventType: "mob_hurt", RoomId, UserId}).
     - Aggro the mob onto the shooter: `mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)` and, for cross-room, ensure the mob's pursuit kicks in (set combat memory location via the same call the flee-pursuit path uses — from your Step-1 reading; the mob must path to the shooter's room).
   - **Crime recording:** for mob targets when `!result.IsSneaking`: `recordAssaultCrime(user, mob, targetRoom)` — note the third arg is the VICTIM's room (`rooms.LoadRoom(result.TargetRoomId)`); witnesses are sought there. When sneaking/hidden, SKIP the call (unattributed assault — matches melee's hidden behavior: verify how attack.go gates recordAssaultCrime on visibility and mirror it; if melee always records and lets `IdentifiedPerp` handle attribution, do the SAME — call it and let IdentifiedPerp decide. Read crimes.IdentifiedPerp to learn whether identification depends on the perp being in the witnesses' room; report findings in the task report — this is the one open judgment point and the live smoke covers it).
   - Progression: `user.Character.OnSkillUse(string(skills.RangedCombat), user.UserId)` and `user.Character.OnStatUse("perception", user.UserId)` on Executed (hit or miss — using the skill is using the skill; match how AttackPlayerVsMob handles OnSkillUse on hit only — mirror melee: skill use on HIT, stat use always).
4. Register `fire` as an alias for the shoot command (check how aliases are declared — keywords.yaml command aliases or the command-map; mirror an existing alias).

- [ ] **Step 3: Mob command parity** — mobcommands shoot: same ExecuteFire flow, silent early exits, mob-flavored messages; mob shooters skip crime recording entirely. Mobs reload via Task 2's mob reload; the btree (Task 8) drives the sequencing.

- [ ] **Step 4: Tests.** In internal/usercommands (or hooks, wherever an integration seam exists — read how usercommands tests construct users/rooms/mobs, e.g. TestSetSubCommands fixtures and mobcommands_test.go's TestAttackMob):
```go
// Pin (adapt to available fixtures):
// 1. shoot with unloaded weapon → "isn't loaded" message, no damage.
// 2. same-room loaded shot at mob → mob HP dropped, weapon unloaded,
//    shooter has aggro, mob has aggro on shooter.
// 3. cross-room loaded shot → mob HP dropped, shooter has NO aggro,
//    mob aggro set on shooter (retaliation).
// 4. crime: shooting a faction mob with a guard witness in the MOB's
//    room records an assault crime (assert via crimes package query —
//    find its test helpers) — and NOT when the shooter is hidden
//    (or document IdentifiedPerp's verdict per Step 2 findings).
```
These are the spec's explicit test requirements — do not skip; if the fixture cost is high, build the minimal room/mob/user scaffolding the packages' existing tests already demonstrate.

- [ ] **Step 5:** `go build ./... && go test ./internal/usercommands/ ./internal/mobcommands/ ./internal/actions/ -count=1` green. Commit:
```bash
git add internal/usercommands/ internal/mobcommands/
git commit -m "feat(combat): shoot rewritten to loaded-weapon model — instant shot, retaliation, crime integration"
```

---

### Task 7: Legacy remote-aggro cleanup sweep

**Files:**
- Modify: `internal/actions/combat_shoot.go` (retire ExecuteShoot or reduce it to the resolution helpers ExecuteFire reuses)
- Audit: `internal/usercommands/attack.go:113`, `internal/mobcommands/attack.go:51`, `internal/usercommands/target.go:48`, NewRound_DoCombat_unified.go cross-room chase block

- [ ] **Step 1:** Grep all `SetAggroRemote` and `characters.Shooting` callsites. Determine what still needs them:
  - `attack` with a ranged weapon (same room): keep working — it's the melee-fallback auto-attack; the Shooting aggro type can remain for flavor/compat, but verify the round loop doesn't do anything ranged-special with it that contradicts the loaded model (read the swing path for shooting-subtype weapons — the existing weak-mult melee swings are exactly the desired fallback).
  - The unified-loop cross-room attack allowance driven by `Aggro.ExitName`: with no shooter ever holding remote aggro from `shoot` anymore, the remaining ExitName users are flee-chase paths — verify, and leave those intact.
  - `ExecuteShoot`/its ShootResult: now dead if Task 6 fully replaced it — delete it and its direct callers' dead code, or keep only shared helpers. Prefer deletion (YAGNI) with its tests updated.
- [ ] **Step 2:** `go build ./... && go test ./internal/... -count=1` (full internal tree — this sweep is the riskiest regression point; budget the time).
- [ ] **Step 3:** Commit: `git commit -m "refactor(combat): retire continuous remote-shoot aggro path (replaced by loaded-weapon fire)"`

---

### Task 8: Archer AI

**Files:**
- Create: btree action(s) in `internal/behaviortree/actions_archer.go`
- Create: `_datafiles/world/dogmud/behaviors/...` archer archetype (path per existing archetypes — find predator/generic_fighter/tank_taunter files and mirror)
- Modify: `internal/behaviortree/actions.go` (registry init)
- Test: `internal/behaviortree/actions_archer_test.go` + an end-to-end guard mirroring `test(behaviortree): end-to-end guard — predator archetype fires a beast move` (commit f3e9ba1b — read that test)

- [ ] **Step 0: REQUIRED READING:** `docs/superpowers/plans/2026-06-09-nonhuman-attacks-phase3-beast-moveset.md` (the delegation wiring recipe), the predator btree YAML, and `internal/behaviortree/actions_forager.go` (action implementation style + ActionFunc signature).

- [ ] **Step 1: Actions** (names registered in actionRegistry):
  - `try_fire` — mob has a loaded ranged weapon + a target (in-room aggro target, OR a remembered cross-room target one exit away): issue the mob `shoot` command (mob actions enqueue commands — see how existing actions make mobs act, e.g. how forager actions queue `forage`/movement; use the same mechanism so messaging/cooldowns flow through the real command). Success when fired.
  - `try_reload` — unloaded ranged weapon + matching ammo in mob inventory + NOT melee-engaged (no same-room aggro on the mob): enqueue `reload`. If the mob has skullduggery skill and isn't hidden, enqueue `hide` first when available (check the mob hide command exists — grep mobcommands; if no hide command, drop the hide step and note it).
  - `keep_distance` — mob melee-engaged + healthy (HP above a threshold arg, default 50%): pick the exit toward... simplest correct v1: any passable exit (prefer the one leading toward its home/anchor if cheaply available), enqueue movement through it; the next ticks let try_fire shoot back through the remembered exit. Store the chosen exit + prior room in mob MiscData for the return-fire targeting.
- [ ] **Step 2: Archetype** — `archer.yaml` btree: selector ordering `keep_distance → try_fire → try_reload → (fallback melee via existing combat behavior)`. Goal-weights: read how predator/tank archetypes declare 4.2 goal weights and give archer ones that don't fight the kiting (mirror an existing combat archetype's weights).
- [ ] **Step 3: Tests** — unit-test each action's gating with the behaviortree test fixtures (read existing actions tests); end-to-end guard: an archer-archetype mob with loaded crossbow + aggro fires (mirrors the beast-move e2e test's structure).
- [ ] **Step 4:** `go build ./... && go test ./internal/behaviortree/ -count=1`, commit:
```bash
git add internal/behaviortree/ _datafiles/world/dogmud/behaviors/
git commit -m "feat(behaviortree): archer archetype — try_fire/try_reload/keep_distance kiting AI"
```

---

### Task 9: Content — weapons, ammo, mobs, vendors

**Files:**
- Create: 5 weapon YAMLs + 3 ammo YAMLs under `_datafiles/world/dogmud/items/...`
- Modify: `_datafiles/world/dogmud/items/weapons-10000/10004-training_bow.yaml`
- Create: 2 archer mob YAMLs under `_datafiles/world/dogmud/mobs/...`
- Modify: vendor/shop data for ammo stocking

- [ ] **Step 1: Allocate IDs:** `python tools/id_inventory.py --type items` and `--type mobs`; use the reported next-free IDs for the 8 new items and 2 mobs. All filenames follow `{id}-{ConvertForFilename(name)}.yaml`.

- [ ] **Step 2: Author the weapons.** Template (fill allocated IDs; follow 10004's field layout):

| Name | hands | subtype | ammo_tag | damage_multiplier | min_strength | notes |
|---|---|---|---|---|---|---|
| sling | 1 | shooting | shot | 2.0 | — | newbie, cheap |
| hand crossbow | 1 | shooting | bolts | 3.0 | — | offhand hybrid |
| primitive pistol | 1 | shooting | shot | 3.5 | — | loud flavor text |
| hunting bow | 2 | shooting | arrows | 5.5 | — | mainline |
| arbalest | 2 | shooting | bolts | 7.0 | 110 | heavy hitter |

Each weapon: `speedmultiplier: 0.5`, `grapplemodifier: 0.2`, low `parryrating` (0-2), sensible `weight`, `vendor_categories: [blacksmithing]`, immersive description (≤80-char lines). Training bow 10004: set `ammo_tag: arrows` and RAISE `damage_multiplier` to `4.0` (starter-tier under hunting bow). NOTE: the unloaded melee-fallback swings use the same damage_multiplier — that makes an arbalest club hit like a maul. Counterbalance: verify how melee swings read the multiplier (combat_helpers weapon setup); if the fallback can cheaply use a floor (e.g. reach/bludgeon narration already handles ranged-as-blunt — combat_helpers.go:979), add a clamp: shooting-subtype melee swings use `min(damage_multiplier, 0.30)` — implement in the weapon-setup path with a unit test, and document it in the YAML comments.

- [ ] **Step 3: Ammo bundles** (type `ammo`, no equip slot):
```yaml
itemid: <alloc>
name: Quiver of Arrows
namesimple: quiver
description: Two dozen straight-shafted arrows, fletched and ready.
type: ammo
ammo_tag: arrows
uses: 24
weight: 1.5
value: 60
vendor_categories:
- blacksmithing
```
Same shape for "Case of Bolts" (`bolts`, uses 20) and "Pouch of Shot" (`shot`, uses 30). Check `uses:` is honored at item creation (items.go:70 copies spec Uses → instance) and that type `ammo` survives the spec validator (Task 1 added the type).

- [ ] **Step 4: Archer mobs** (2): a Thornwall crossbowman (guard-flavored: stillwater/thornwall faction per zone conventions — read a neighboring guard mob YAML) and a marsh hunter (hostile wilderness). Each: `archetype: fighting` stats, the archer btree archetype reference (field per the behaviors convention found in Task 8), equipped ranged weapon (loaded loadout — check how mob YAML declares equipment + items; give 1 ammo bundle in items), reasonable HP. Run the room-spawn additions ONLY if trivially safe; otherwise leave spawn placement to a followup and note it (avoid zone-balance surprises).

- [ ] **Step 5: Boot check** — wipe instance saves, boot, confirm `items.LoadDataFiles`/`mobs.LoadDataFiles` counts rose by the right amounts, no panics. Commit:
```bash
git add _datafiles/world/dogmud/items/ _datafiles/world/dogmud/mobs/ <vendor files>
git commit -m "content(ranged): 5 weapons + 3 ammo bundles + 2 archer mobs"
```

---

### Task 10: Help, docs, polish

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/reload.template`, `ranged-combat.template`
- Modify: `_datafiles/world/dogmud/templates/help/shoot.template` (exists? check; create if not)
- Modify: `_datafiles/world/dogmud/keywords.yaml`
- Modify: `CLAUDE.md` (damage-pipeline table: weapon/unarmed/**ranged-combat** is now accurate again; "Skills (9 total)" → 10)
- Audit: context.md files in touched packages (items, actions, skills, characters, behaviortree, usercommands, mobcommands, configs)

- [ ] **Step 1:** Helpfiles mirroring the house conventions (weather.template/combatverbosity.template headers): `shoot` (usage incl. direction form, loaded requirement), `reload` (cooldown cost, ammo bundles), `ranged-combat` (skill description, Perception governance). keywords.yaml entries (combat category for shoot/reload; skills category if one exists for skill helpfiles — mirror how other skills are listed).
- [ ] **Step 2:** CLAUDE.md edits (the two stale-count spots). context.md audit: for each touched package, surgically update sections that now lie (items context.md item-type list; actions context.md action inventory; skills count; usercommands command list; behaviortree actions list) — same drill as the verbosity Task 6 audit.
- [ ] **Step 3:** PATCH_NOTES.md entry (player-facing, no numbers): loaded-weapon ranged combat, shoot/reload, new weapons + ammo at smiths, the revived skill, archer enemies that keep their distance.
- [ ] **Step 4:** `go build ./... && go test ./internal/... -count=1` full sweep, boot smoke, commit:
```bash
git add _datafiles/world/dogmud/templates/help/ _datafiles/world/dogmud/keywords.yaml CLAUDE.md PATCH_NOTES.md <context.md files>
git commit -m "docs(ranged): helpfiles, keywords, CLAUDE.md skill-count fixes, context.md sync, PATCH_NOTES"
```

---

### Task 11: Live smoke (incl. the crime/justice requirement)

Verification only, scripted over AI port 55555 (smoketester/smoke123test is admin; pace ~3s; RoundSeconds=4). Boot with wiped instance saves.

- [ ] 1. **Loadout:** create/give a hunting bow + quiver (admin item spawn — `item give`-style admin command; discover via help). `reload` → "You ready your..."; `reload` again → already loaded.
- [ ] 2. **Same-room shot:** `shoot <training dummy>` in Test Arena → immediate hit line with wound tier; weapon unloaded (`reload` works again after cooldown); dummy aggroes.
- [ ] 3. **Cooldown economy:** `reload` then immediately `kick` → kick blocked by shared cooldown; after ~5 rounds it works.
- [ ] 4. **Cross-room shot:** stand one room away, `shoot <mob> <direction>` → shot resolves; shooter has no aggro; MOB COMES TO YOU (retaliation pathing) — wait and verify arrival.
- [ ] 5. **Ammo depletion:** fire/reload until the quiver empties → "last of your" message; next reload → no-ammo message naming arrows.
- [ ] 6. **Crime/justice (spec requirement):** shoot a faction-guarded NPC (e.g. a Stillwater townsperson) from the adjacent room WITH a guard in the victim's room → guard reacts per town justice (warn/arrest attempt reaches you in the next room); verify a bounty/crime record exists (`questtoken`-style admin tooling or the crimes admin command — discover). Then the hidden variant: `hide` (admin-grant skullduggery if needed), shoot → verify the unattributed path (no rep hit / unknown perp per Task 6's IdentifiedPerp findings).
- [ ] 7. **Archer mob:** spawn the marsh hunter, engage in melee → it withdraws, shoots back from the next room, reloads (watch over ~10 rounds).
- [ ] 8. **Skill progression:** repeated shots eventually tick ranged-combat (`skills` output) and Perception use.
- [ ] 9. **Offhand hybrid:** equip sword main + hand crossbow offhand + bolts; `reload` targets the offhand; `shoot` fires it; melee rounds use the sword normally.
- [ ] 10. Cleanup: kill server, wipe instance saves, no stray processes.

Record verbatim evidence per check; failures get fixed before merge.

---

### Task 12: Final review + merge

- [ ] Final whole-implementation review (integration-level), then:
```bash
git checkout master
git merge --no-ff feature/ranged-weapons -m "Merge feature/ranged-weapons: loaded-weapon ranged combat (shoot/reload, ranged-combat skill, archer AI)"
```
- [ ] On-master boot smoke. No prod push (end-of-day bundle per SOP).

---

## Self-review notes

- **Spec coverage:** loaded model + cooldown reload (T2/T5/T6), ammo bundles (T1/T2/T9), skill revival + Perception (T4/T5), balance target with pinned test + RangedShotScale (T3/T5 step 5/T9 multipliers — arbalest 7.0 ⇒ raw 210 in the 180-220 band), no-parry + shield bonus (T5 rangedDefenseScore), offhand hybrids (T1 findRangedWeaponSlot order + T11 check 9), crime/witness/justice with tests + live checks (T6 steps 2/4, T11 check 6), full archer AI incl. kiting + hide-as-cover (T8, T11 check 7), legacy remote auto-attack retirement (T6/T7), content + vendors (T9), UX/help/docs (T10), min-Strength (T1 step 5).
- **Type consistency:** `findRangedWeaponSlot` defined T2 used T5; `ReloadResult`/`FireResult` self-contained; `rangedDefenseScore` defined+tested T5; `skills.RangedCombat` defined T4 used T5/T6; `items.Ammo`/`AmmoTag`/`MinStrength`/`Loaded` defined T1 used T2/T5/T9.
- **Known judgment points (explicit, bounded):** `IdentifiedPerp` cross-room attribution semantics (T6 step 2 — investigate and report, live-smoke validated); mob movement enqueue mechanism for `keep_distance` (T8 — pattern documented in the beast-moveset plan); unloaded-melee damage clamp location (T9 step 2); mob equipment YAML shape (T9 step 4 — read a guard mob).
- **Deviation from spec to note at review:** spec said "dodge and block apply" via best-of-all exclusion; implementation uses the deliberate-move opposed roll (the same defense model as kick/trip/bash) with an explicit shield bonus — same intent (no parry, shields matter), simpler and consistent with the engine's other deliberate attacks.
