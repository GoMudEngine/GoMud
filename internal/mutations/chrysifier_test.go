package mutations

import "testing"

func TestChrysifierReaders(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"provident": {MutationId: "provident", Name: "Provident", Rarity: 3, Pros: []MutationEffect{
			{Type: "forage_yield_multiplier", Value: 0.25},
			{Type: "salvage_yield_bonus", Value: 0.20},
			{Type: "craft_material_discount", Value: 0.15},
		}},
		"faith": {MutationId: "faith", Name: "Faith", Rarity: 6, Pros: []MutationEffect{
			{Type: "craft_quality_bonus", Value: 0.10},
			{Type: "carry_capacity_multiplier", Value: 0.50},
		}},
		"walk": {MutationId: "walk", Name: "Walk", Rarity: 6, Pros: []MutationEffect{
			{Type: "flag", Target: "portable-workshop"},
		}},
		"homun": {MutationId: "homun", Name: "Homun", Rarity: 8, Pros: []MutationEffect{
			{Type: "flag", Target: "homunculus"},
		}},
	})
	defer cleanup()

	all := map[string]int{"provident": 1, "faith": 1, "walk": 1, "homun": 1}
	assertF := func(name string, got, want float64) {
		if got != want {
			t.Fatalf("%s = %v, want %v", name, got, want)
		}
	}
	assertF("forage", GetForageYieldMult(all), 0.25)
	assertF("salvage", GetSalvageYieldBonus(all), 0.20)
	assertF("discount", GetCraftMaterialDiscount(all), 0.15)
	assertF("quality", GetCraftQualityBonus(all), 0.10)
	assertF("carry", GetCarryCapacityMultiplier(all), 0.50)
	if !HasPortableWorkshop(all) {
		t.Fatal("expected portable-workshop flag")
	}
	if !HasHomunculus(all) {
		t.Fatal("expected homunculus flag")
	}
	if HasPortableWorkshop(map[string]int{"provident": 1}) {
		t.Fatal("provident alone should not grant portable-workshop")
	}
}
