package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ReloadResult holds the outcome of a reload attempt.
type ReloadResult struct {
	WeaponName    string // display name of the weapon involved
	AmmoTag       string // ammo type involved (set for NoAmmo messaging too)
	AmmoName      string // display name of the bundle consumed from
	BundleEmptied bool   // the bundle's last Use was consumed

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
// special-move cooldown and one Use from a matching ammo bundle. The
// cooldown is only consumed once every other precondition has passed —
// a reload that fails for no-ammo never burns it.
// Callers handle all messaging and progression events.
func ExecuteReload(actor Actor) ReloadResult {
	char := actor.GetCharacter()

	// Don't interrupt any active activity (cast/craft/salvage) to reload.
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
	bundleIdx := -1
	for idx := range char.Items {
		spec := char.Items[idx].GetSpec()
		if spec.Type == items.Ammo && spec.AmmoTag == ammoTag {
			bundleIdx = idx
			break
		}
	}
	if bundleIdx < 0 {
		return ReloadResult{WeaponName: weapon.DisplayName(), AmmoTag: ammoTag, NoAmmo: true}
	}

	// Shared special-move cooldown — the cost of the reload. Last gate, so a
	// reload that would fail for any earlier reason never burns the cooldown.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return ReloadResult{WeaponName: weapon.DisplayName(), OnCooldown: true}
	}

	result := ReloadResult{
		WeaponName: weapon.DisplayName(),
		AmmoTag:    ammoTag,
		AmmoName:   char.Items[bundleIdx].DisplayName(),
		Loaded:     true,
	}

	// Consume one Use; remove the bundle when emptied. RemoveItem matches by
	// ItemId+UUID (Item.Equals), so the post-decrement value still matches.
	char.Items[bundleIdx].Uses--
	if char.Items[bundleIdx].Uses <= 0 {
		result.BundleEmptied = true
		char.RemoveItem(char.Items[bundleIdx])
	}

	weapon.Loaded = true
	return result
}
