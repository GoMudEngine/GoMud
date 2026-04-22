package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// Helper: fresh character with ExtraArms set and all extra arm slots
// initialized to empty (ItemId 0). characters.New() calls Validate() which
// sets unused extra arm slots to ItemDisabledSlot (ItemId -1); tests that
// want "enabled but empty" must reset explicitly.
func newCharWithArms(extraArms int) *characters.Character {
	c := characters.New()
	c.ExtraArms = extraArms
	empty := items.Item{ItemId: 0}
	if extraArms >= 1 {
		c.Equipment.ExtraArm1 = empty
	}
	if extraArms >= 2 {
		c.Equipment.ExtraArm2 = empty
	}
	if extraArms >= 3 {
		c.Equipment.ExtraArm3 = empty
	}
	if extraArms >= 4 {
		c.Equipment.ExtraArm4 = empty
	}
	return c
}

func makeSword() items.Item {
	return items.Item{ItemId: 10, Spec: &items.ItemSpec{Type: items.Weapon}}
}

func makeShield(id int) items.Item {
	return items.Item{ItemId: id, Spec: &items.ItemSpec{
		Type:            items.Offhand,
		DamageReduction: 5,
	}}
}

// Baseline: sword + empty offhand + ExtraArms=1 + empty ExtraArm1 = 3 attacks.
func TestCollectAttackWeapons_EmptyExtraArm_ProducesFist(t *testing.T) {
	c := newCharWithArms(1)
	c.Equipment.Weapon = makeSword()

	got := collectAttackWeapons(c)
	assert.Equal(t, 3, len(got), "sword + 2 empty slots = 3 attacks; got: %+v", got)
}

// Repro for project_extra_arms_shield_bonus_attacks.md — shield in
// ExtraArm1 must NOT generate a fist attack from that arm.
// Baseline: sword + empty offhand + ExtraArms=1 + shield in ExtraArm1.
// Expected: sword + offhand-fist = 2 attacks. Shield arm contributes 0.
func TestCollectAttackWeapons_ShieldInExtraArm1_NoFistAttack(t *testing.T) {
	c := newCharWithArms(1)
	c.Equipment.Weapon = makeSword()
	c.Equipment.ExtraArm1 = makeShield(20)

	got := collectAttackWeapons(c)
	assert.Equal(t, 2, len(got),
		"shield in ExtraArm1 must not add a fist; got: %+v", got)
}

// ExtraArms=2 with shields in BOTH extra arms must not contribute fist
// attacks. sword + empty offhand + shield ExtraArm1 + shield ExtraArm2.
// Expected: sword + offhand-fist = 2 attacks.
func TestCollectAttackWeapons_ShieldInExtraArm1AndExtraArm2_NoFists(t *testing.T) {
	c := newCharWithArms(2)
	c.Equipment.Weapon = makeSword()
	c.Equipment.ExtraArm1 = makeShield(20)
	c.Equipment.ExtraArm2 = makeShield(21)

	got := collectAttackWeapons(c)
	assert.Equal(t, 2, len(got),
		"two shields in extra arms must not add fists; got: %+v", got)
}

// ExtraArms=4 with shields in ALL four extra arms — worst case.
// sword + empty offhand + 4 shields in extras.
// Expected: sword + offhand-fist = 2 attacks.
func TestCollectAttackWeapons_ShieldsInAllExtraArms_NoFists(t *testing.T) {
	c := newCharWithArms(4)
	c.Equipment.Weapon = makeSword()
	c.Equipment.ExtraArm1 = makeShield(20)
	c.Equipment.ExtraArm2 = makeShield(21)
	c.Equipment.ExtraArm3 = makeShield(22)
	c.Equipment.ExtraArm4 = makeShield(23)

	got := collectAttackWeapons(c)
	assert.Equal(t, 2, len(got),
		"four shields in extra arms must not add fists; got: %+v", got)
}

// Sanity: mixed — sword main, sword ExtraArm1, shield ExtraArm2.
// Expected: main sword + offhand-fist + ExtraArm1 sword = 3 attacks.
// Shield in ExtraArm2 contributes nothing.
func TestCollectAttackWeapons_MixedShieldAndWeaponInExtras(t *testing.T) {
	c := newCharWithArms(2)
	c.Equipment.Weapon = makeSword()
	c.Equipment.ExtraArm1 = items.Item{ItemId: 11, Spec: &items.ItemSpec{Type: items.Weapon}}
	c.Equipment.ExtraArm2 = makeShield(20)

	got := collectAttackWeapons(c)
	assert.Equal(t, 3, len(got),
		"sword + offhand-fist + extra sword; got: %+v", got)
}

// 2H weapon + shields in extra arms. Equipping a 2H weapon leaves the
// partner slot (offhand) cleared to items.Item{} (ItemId 0). That slot is
// physically occupied by the 2H weapon — it must NOT generate a fist.
// Regression for project_extra_arms_shield_bonus_attacks.md.
func TestCollectAttackWeapons_TwoHandedWeapon_NoOffhandFist(t *testing.T) {
	twoHander := items.Item{ItemId: 30, Spec: &items.ItemSpec{
		Type:  items.Weapon,
		Hands: 2,
	}}

	c := newCharWithArms(4)
	c.Equipment.Weapon = twoHander
	// Offhand is the partner of a 2H main hand — cleared to ItemId 0
	// by the equip code (not disabled).
	c.Equipment.Offhand = items.Item{ItemId: 0}
	c.Equipment.ExtraArm1 = makeShield(20)
	c.Equipment.ExtraArm2 = makeShield(21)
	c.Equipment.ExtraArm3 = makeShield(22)
	c.Equipment.ExtraArm4 = makeShield(23)

	got := collectAttackWeapons(c)
	assert.Equal(t, 1, len(got),
		"2H weapon + 4 shields = 1 attack (the 2H); got: %+v", got)
}

// 2H weapon in ExtraArm1 blocks ExtraArm2. Shield in ExtraArm2 slot's
// position means the pair-partner check must also apply to extra arms.
func TestCollectAttackWeapons_TwoHandedInExtraArm1_NoExtraArm2Fist(t *testing.T) {
	sword := makeSword()
	twoHander := items.Item{ItemId: 30, Spec: &items.ItemSpec{
		Type:  items.Weapon,
		Hands: 2,
	}}

	c := newCharWithArms(2)
	c.Equipment.Weapon = sword
	c.Equipment.ExtraArm1 = twoHander
	// ExtraArm2 cleared by 2H in ExtraArm1 — must NOT generate a fist.
	c.Equipment.ExtraArm2 = items.Item{ItemId: 0}

	got := collectAttackWeapons(c)
	// sword + offhand-fist + 2H in ExtraArm1 = 3 attacks.
	// ExtraArm2 is occupied by the 2H → no fist from ExtraArm2.
	assert.Equal(t, 3, len(got),
		"sword + offhand-fist + 2H; got: %+v", got)
}

// 2H weapon in ExtraArm3 (arm 5) blocks ExtraArm4 (arm 6). Mirrors the
// pair-C case of the 2H block check.
func TestCollectAttackWeapons_TwoHandedInExtraArm3_NoExtraArm4Fist(t *testing.T) {
	sword := makeSword()
	twoHander := items.Item{ItemId: 30, Spec: &items.ItemSpec{
		Type:  items.Weapon,
		Hands: 2,
	}}

	c := newCharWithArms(4)
	c.Equipment.Weapon = sword
	c.Equipment.ExtraArm1 = makeShield(20)
	c.Equipment.ExtraArm2 = makeShield(21)
	c.Equipment.ExtraArm3 = twoHander
	// ExtraArm4 cleared by 2H in ExtraArm3 — must NOT generate a fist.
	c.Equipment.ExtraArm4 = items.Item{ItemId: 0}

	got := collectAttackWeapons(c)
	// sword + offhand-fist + 2H in ExtraArm3 = 3 attacks.
	// Shields in ExtraArm1/2 and the 2H-blocked ExtraArm4 contribute 0.
	assert.Equal(t, 3, len(got),
		"sword + offhand-fist + 2H in arm5; got: %+v", got)
}
