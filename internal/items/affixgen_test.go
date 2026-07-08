package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/statmods"
)

func TestAffixValue(t *testing.T) {
	base := ItemSpec{Value: 100}

	tests := []struct {
		name    string
		affixed ItemSpec
		gpp     float64
		want    int
	}{
		{"no affixes", ItemSpec{Value: 100}, 3.0, 100},
		{"stat +3 strength", ItemSpec{Value: 100, StatMods: statmods.StatMods{"strength": 3}}, 3.0, 100 + 9*3},
		{"skill +1 weapon-combat", ItemSpec{Value: 100, StatMods: statmods.StatMods{"weapon-combat": 1}}, 3.0, 100 + 12*3},
		{"phys mit +5", ItemSpec{Value: 100, PhysicalMitigation: 5}, 3.0, 100 + 25*3},
		{"damage phys +0.10", ItemSpec{Value: 100, DamageMultiplier: 0.10}, 3.0, 100 + 16*3},
		{"damage both +0.05", ItemSpec{Value: 100, DamageMultiplier: 0.05, SpellDamageMultiplier: 0.05}, 3.0, 100 + 12*3},
		{"gpp scales linearly", ItemSpec{Value: 100, StatMods: statmods.StatMods{"strength": 3}}, 2.0, 100 + 9*2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AffixValue(tt.affixed, base, tt.gpp)
			if got != tt.want {
				t.Errorf("AffixValue(%s) = %d; want %d", tt.name, got, tt.want)
			}
		})
	}
}

// A base with its own damage/stats must only count the POSITIVE delta above base.
func TestAffixValue_OnlyCountsDeltaAboveBase(t *testing.T) {
	base := ItemSpec{Value: 80, DamageMultiplier: 1.0, StatMods: statmods.StatMods{"strength": 2}}
	affixed := ItemSpec{Value: 80, DamageMultiplier: 1.10, StatMods: statmods.StatMods{"strength": 5}}
	want := 80 + 25*3 // damage +0.10 = 16pts, strength +3 = 9pts, total 25 * gpp 3
	if got := AffixValue(affixed, base, 3.0); got != want {
		t.Errorf("AffixValue delta = %d; want %d", got, want)
	}
}

// TestGenerateAffixedItem_StampsValue proves the generated instance's Value is
// self-consistent with its rolled affixes: Value == AffixValue(spec, base, gpp).
// RNG-independent — whatever affixes roll, the stamped value must match them.
func TestGenerateAffixedItem_StampsValue(t *testing.T) {
	cleanup := SeedItemsForTest(map[int]*ItemSpec{
		9100: {ItemId: 9100, Name: "Test Torc", Type: Neck, Value: 85},
	})
	defer cleanup()

	baseItem := New(9100)
	base := baseItem.GetSpec() // base template spec (Value 85, no affixes)

	// goldPaid 200, scalar 7 → budget ~98 points, so Value should exceed base.
	item := GenerateAffixedItem(9100, 200, 7.0, 3.0)

	if item.Spec == nil {
		t.Fatal("expected a per-instance spec on an affixed item")
	}
	stamped := item.Spec.Value
	recomputed := AffixValue(*item.Spec, base, 3.0)
	if stamped != recomputed {
		t.Errorf("stamped Value %d != AffixValue(spec) %d", stamped, recomputed)
	}
	if stamped <= base.Value {
		t.Errorf("stamped Value %d should exceed base %d for a budgeted item", stamped, base.Value)
	}
}

// ---------------------------------------------------------------------------
// CalcLootBudget
// ---------------------------------------------------------------------------

func TestCalcLootBudget(t *testing.T) {
	tests := []struct {
		name     string
		gold     int
		scalar   float64
		expected int
	}{
		{"goldPaid=200 scalar=7.0", 200, 7.0, 98},
		{"goldPaid=1000 scalar=7.0", 1000, 7.0, 221},
		{"goldPaid=0 returns 0", 0, 7.0, 0},
		{"goldPaid negative returns 0", -50, 7.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcLootBudget(tt.gold, tt.scalar)
			if got != tt.expected {
				t.Errorf("CalcLootBudget(%d, %.1f) = %d; want %d",
					tt.gold, tt.scalar, got, tt.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// GetEligibleBonuses
// ---------------------------------------------------------------------------

func TestGetEligibleBonuses_Weapon(t *testing.T) {
	eligible := GetEligibleBonuses(true, false)

	assertContains := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				return
			}
		}
		t.Errorf("expected bonus %q to be eligible for non-caster weapon", name)
	}
	assertAbsent := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				t.Errorf("bonus %q should NOT be eligible for non-caster weapon", name)
				return
			}
		}
	}

	// Physical weapon should have the physical damage mult
	assertContains("damage_mult_phys")
	// But NOT the caster one
	assertAbsent("damage_mult_both")
	// Should NOT have armor mitigations
	assertAbsent("physical_mitigation")
	assertAbsent("magical_mitigation")
	assertAbsent("conviction_mitigation")
	// Should have universal stats and skills
	assertContains("stat_strength")
	assertContains("stat_willpower")
	assertContains("skill_weapon-combat")
	assertContains("skill_spellcasting")
}

func TestGetEligibleBonuses_CasterWeapon(t *testing.T) {
	eligible := GetEligibleBonuses(true, true)

	assertContains := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				return
			}
		}
		t.Errorf("expected bonus %q to be eligible for caster weapon", name)
	}
	assertAbsent := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				t.Errorf("bonus %q should NOT be eligible for caster weapon", name)
				return
			}
		}
	}

	// Caster weapon gets the combined damage mult
	assertContains("damage_mult_both")
	// But NOT the physical-only one
	assertAbsent("damage_mult_phys")
	// No armor mitigations
	assertAbsent("physical_mitigation")
	// Stats and skills are universal
	assertContains("stat_charisma")
	assertContains("skill_manifestation")
}

func TestGetEligibleBonuses_Armor(t *testing.T) {
	eligible := GetEligibleBonuses(false, false)

	assertContains := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				return
			}
		}
		t.Errorf("expected bonus %q to be eligible for armor", name)
	}
	assertAbsent := func(name string) {
		t.Helper()
		for _, b := range eligible {
			if b.Name == name {
				t.Errorf("bonus %q should NOT be eligible for armor", name)
				return
			}
		}
	}

	// Armor should have all three mitigation types
	assertContains("physical_mitigation")
	assertContains("magical_mitigation")
	assertContains("conviction_mitigation")
	// But not weapon affixes
	assertAbsent("damage_mult_phys")
	assertAbsent("damage_mult_both")
	// Stats and skills are universal
	assertContains("stat_vitality")
	assertContains("skill_rhetoric")
}

func TestGetEligibleBonuses_AllHaveStatsAndSkills(t *testing.T) {
	configs := []struct {
		name    string
		weapon  bool
		caster  bool
	}{
		{"weapon", true, false},
		{"caster weapon", true, true},
		{"armor", false, false},
	}

	statNames := []string{
		"stat_strength", "stat_dexterity", "stat_perception",
		"stat_vitality", "stat_willpower", "stat_charisma",
	}
	skillNames := []string{
		"skill_weapon-combat", "skill_unarmed-combat", "skill_skullduggery",
		"skill_spellcasting", "skill_rhetoric", "skill_manifestation",
	}

	for _, cfg := range configs {
		eligible := GetEligibleBonuses(cfg.weapon, cfg.caster)
		index := make(map[string]bool, len(eligible))
		for _, b := range eligible {
			index[b.Name] = true
		}

		for _, stat := range statNames {
			if !index[stat] {
				t.Errorf("[%s] expected stat bonus %q to be present", cfg.name, stat)
			}
		}
		for _, skill := range skillNames {
			if !index[skill] {
				t.Errorf("[%s] expected skill bonus %q to be present", cfg.name, skill)
			}
		}
	}
}
