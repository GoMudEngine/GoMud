package items

import (
	"testing"
)

func TestItemPower_ZeroSpec(t *testing.T) {
	spec := ItemSpec{}
	if got := ItemPower(spec); got != 0.0 {
		t.Errorf("expected 0.0 for empty ItemSpec, got %f", got)
	}
}

func TestItemPower_DamageMultiplier(t *testing.T) {
	base := ItemSpec{ItemId: 1, Type: Weapon}
	boosted := ItemSpec{ItemId: 2, Type: Weapon, DamageMultiplier: 1.5}

	if ItemPower(boosted) <= ItemPower(base) {
		t.Errorf("weapon with DamageMultiplier should score higher than one without")
	}
}

func TestItemPower_SpellDamageMultiplier(t *testing.T) {
	base := ItemSpec{ItemId: 1, Type: Weapon}
	caster := ItemSpec{ItemId: 2, Type: Weapon, SpellDamageMultiplier: 1.3}

	if ItemPower(caster) <= ItemPower(base) {
		t.Errorf("caster weapon with SpellDamageMultiplier should score higher than one without")
	}
}

func TestItemPower_Mitigation(t *testing.T) {
	base := ItemSpec{ItemId: 1, Type: Body}
	armored := ItemSpec{
		ItemId:               2,
		Type:                 Body,
		PhysicalMitigation:   20,
		MagicalMitigation:    10,
		ConvictionMitigation: 5,
	}

	if ItemPower(armored) <= ItemPower(base) {
		t.Errorf("armored item should score higher than unarmored")
	}

	// Verify the exact expected score
	expected := 20.0 + 10.0 + 5.0
	if got := ItemPower(armored); got != expected {
		t.Errorf("expected power %f for mitigation-only armor, got %f", expected, got)
	}
}

func TestItemPower_StatMods(t *testing.T) {
	base := ItemSpec{ItemId: 1, Type: Ring}
	statRing := ItemSpec{
		ItemId: 2,
		Type:   Ring,
		StatMods: map[string]int{
			"strength":  10,
			"dexterity": 5,
		},
	}

	if ItemPower(statRing) <= ItemPower(base) {
		t.Errorf("ring with stat mods should score higher than empty ring")
	}

	expected := 15.0
	if got := ItemPower(statRing); got != expected {
		t.Errorf("expected power %f for stat-only ring, got %f", expected, got)
	}
}

func TestItemPower_NegativeStatModsIgnored(t *testing.T) {
	cursed := ItemSpec{
		ItemId: 1,
		Type:   Ring,
		StatMods: map[string]int{
			"strength": -10,
		},
	}

	if got := ItemPower(cursed); got != 0.0 {
		t.Errorf("negative stat mods should not reduce power below 0, got %f", got)
	}
}

func TestIsUpgrade_TrueWhenHigherPower(t *testing.T) {
	current := ItemSpec{ItemId: 1, Type: Body}
	better := ItemSpec{ItemId: 2, Type: Body, PhysicalMitigation: 30}

	if !IsUpgrade(current, better) {
		t.Errorf("expected IsUpgrade to return true for higher-power item")
	}
}

func TestIsUpgrade_FalseWhenLowerPower(t *testing.T) {
	current := ItemSpec{ItemId: 1, Type: Body, PhysicalMitigation: 30}
	weaker := ItemSpec{ItemId: 2, Type: Body, PhysicalMitigation: 10}

	if IsUpgrade(current, weaker) {
		t.Errorf("expected IsUpgrade to return false for lower-power item")
	}
}

func TestIsUpgrade_FalseForDifferentType(t *testing.T) {
	current := ItemSpec{ItemId: 1, Type: Head}
	body := ItemSpec{ItemId: 2, Type: Body, PhysicalMitigation: 100}

	if IsUpgrade(current, body) {
		t.Errorf("expected IsUpgrade to return false when types differ")
	}
}

func TestIsUpgrade_FalseForZeroItemId(t *testing.T) {
	current := ItemSpec{ItemId: 1, Type: Weapon}
	empty := ItemSpec{ItemId: 0, Type: Weapon, DamageMultiplier: 9.9}

	if IsUpgrade(current, empty) {
		t.Errorf("expected IsUpgrade to return false when candidate has no ItemId")
	}
}
