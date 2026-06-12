package actions

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Test item constructors for reload
// ---------------------------------------------------------------------------

func reloadRangedWeapon(id int, ammoTag string) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId:  id,
			Name:    "longbow",
			Type:    items.Weapon,
			Subtype: items.Shooting,
			AmmoTag: ammoTag,
			Hands:   2,
		},
	}
}

func reloadMeleeWeapon(id int) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId:  id,
			Name:    "sword",
			Type:    items.Weapon,
			Subtype: items.Slashing,
			Hands:   1,
		},
	}
}

func reloadAmmoBundle(id int, ammoTag string, uses int) items.Item {
	return items.Item{
		ItemId: id,
		Uses:   uses,
		Spec: &items.ItemSpec{
			ItemId:  id,
			Name:    "arrows",
			Type:    items.Ammo,
			AmmoTag: ammoTag,
		},
	}
}

// specialMoveCooldownPeriod is the cooldown string ExecuteReload uses; tests
// reuse it to probe whether the cooldown was consumed.
func specialMoveCooldownPeriod() string {
	return fmt.Sprintf("%d rounds", configs.GetBalanceConfig().SpecialMoveCooldown)
}

// ---------------------------------------------------------------------------
// 1. No ranged weapon equipped → NoWeapon
// ---------------------------------------------------------------------------

func TestReload_NoRangedWeapon(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadMeleeWeapon(2)
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.NoWeapon, "expected NoWeapon when no ranged weapon equipped")
	assert.False(t, res.Loaded)
}

// ---------------------------------------------------------------------------
// 2. Ranged weapon already loaded → AlreadyLoaded
// ---------------------------------------------------------------------------

func TestReload_AlreadyLoaded(t *testing.T) {
	char := characters.New()
	w := reloadRangedWeapon(1, "arrows")
	w.Loaded = true
	char.Equipment.Weapon = w
	char.Items = append(char.Items, reloadAmmoBundle(3, "arrows", 20))
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.AlreadyLoaded, "expected AlreadyLoaded")
	assert.False(t, res.Loaded)
	// Ammo must not be consumed.
	assert.Equal(t, 20, char.Items[0].Uses, "ammo should be untouched")
}

// ---------------------------------------------------------------------------
// 3. No matching ammo bundle → NoAmmo, cooldown NOT consumed
// ---------------------------------------------------------------------------

func TestReload_NoAmmo_DoesNotBurnCooldown(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadRangedWeapon(1, "arrows")
	// Wrong tag bundle present; should not match.
	char.Items = append(char.Items, reloadAmmoBundle(4, "bolts", 20))
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.NoAmmo, "expected NoAmmo")
	assert.Equal(t, "arrows", res.AmmoTag, "NoAmmo result should report the needed tag")
	assert.False(t, res.Loaded)
	assert.False(t, char.Equipment.Weapon.Loaded, "weapon must stay unloaded")

	// Proof the cooldown was never consumed: a fresh Try succeeds.
	assert.True(t, char.Cooldowns.Try("special-move", specialMoveCooldownPeriod()),
		"special-move cooldown must not have been consumed on a no-ammo reload")
}

// ---------------------------------------------------------------------------
// 4. Cooldown busy → OnCooldown, ammo NOT consumed, weapon stays unloaded
// ---------------------------------------------------------------------------

func TestReload_OnCooldown(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadRangedWeapon(1, "arrows")
	char.Items = append(char.Items, reloadAmmoBundle(3, "arrows", 20))

	// Pre-occupy the shared special-move cooldown slot.
	assert.True(t, char.Cooldowns.Try("special-move", specialMoveCooldownPeriod()))

	actor := newStubActor(char, newTestRoom())
	res := ExecuteReload(actor)

	assert.True(t, res.OnCooldown, "expected OnCooldown when special-move is busy")
	assert.False(t, res.Loaded)
	assert.False(t, char.Equipment.Weapon.Loaded, "weapon must stay unloaded")
	assert.Equal(t, 20, char.Items[0].Uses, "ammo must not be consumed on cooldown")
}

// ---------------------------------------------------------------------------
// 5. Success: bundle decremented, weapon Loaded persisted, cooldown consumed
// ---------------------------------------------------------------------------

func TestReload_Success(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadRangedWeapon(1, "arrows")
	char.Items = append(char.Items, reloadAmmoBundle(3, "arrows", 20))
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.Loaded, "expected success")
	assert.False(t, res.BundleEmptied, "bundle had plenty left")
	assert.Equal(t, "arrows", res.AmmoTag)
	assert.Equal(t, 19, char.Items[0].Uses, "bundle Uses should drop by one")
	// Writeback proof: the equipment slot itself is now loaded.
	assert.True(t, char.Equipment.Weapon.Loaded, "weapon Loaded must persist on the slot")

	// Cooldown consumed: an immediate second Try fails.
	assert.False(t, char.Cooldowns.Try("special-move", specialMoveCooldownPeriod()),
		"special-move cooldown should have been consumed by the reload")
}

// ---------------------------------------------------------------------------
// 6. Bundle at Uses==1 → success + bundle removed from backpack
// ---------------------------------------------------------------------------

func TestReload_LastUse_RemovesBundle(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadRangedWeapon(1, "arrows")
	char.Items = append(char.Items, reloadAmmoBundle(3, "arrows", 1))
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.Loaded, "expected success")
	assert.True(t, res.BundleEmptied, "expected BundleEmptied on last use")
	assert.True(t, char.Equipment.Weapon.Loaded)
	assert.Empty(t, char.Items, "emptied bundle should be removed from backpack")
}

// ---------------------------------------------------------------------------
// 7. Offhand ranged + main melee → reload targets the offhand
// ---------------------------------------------------------------------------

func TestReload_OffhandRanged(t *testing.T) {
	char := characters.New()
	char.Equipment.Weapon = reloadMeleeWeapon(2)
	char.Equipment.Offhand = reloadRangedWeapon(1, "arrows")
	char.Items = append(char.Items, reloadAmmoBundle(3, "arrows", 20))
	actor := newStubActor(char, newTestRoom())

	res := ExecuteReload(actor)

	assert.True(t, res.Loaded, "expected success reloading the offhand ranged weapon")
	assert.True(t, char.Equipment.Offhand.Loaded, "offhand should be loaded")
	assert.False(t, char.Equipment.Weapon.Loaded, "main-hand melee should be untouched")
	assert.Equal(t, 19, char.Items[0].Uses)
}
