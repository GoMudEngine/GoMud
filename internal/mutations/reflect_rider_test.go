package mutations

import "testing"

func TestGetReflectRiderBuffs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"reflect-skin-molten": {MutationId: "reflect-skin-molten", Name: "Molten Skin", Rarity: 5,
			Pros: []MutationEffect{
				{Type: "reflect_damage", Value: 18},
				{Type: "on_reflect_buff", Value: 106},
			}},
		"reflect-skin-barbed": {MutationId: "reflect-skin-barbed", Name: "Barbed Skin", Rarity: 5,
			Pros: []MutationEffect{{Type: "reflect_damage", Value: 25}}},
	})
	defer cleanup()

	// Flavored variant carries its rider buff id.
	got := GetReflectRiderBuffs(map[string]int{"reflect-skin-molten": 1})
	if len(got) != 1 || got[0] != 106 {
		t.Fatalf("GetReflectRiderBuffs(molten) = %v, want [106]", got)
	}
	// Barbed reflects but carries no rider.
	if got := GetReflectRiderBuffs(map[string]int{"reflect-skin-barbed": 1}); len(got) != 0 {
		t.Fatalf("GetReflectRiderBuffs(barbed) = %v, want []", got)
	}
	if len(GetReflectRiderBuffs(map[string]int{})) != 0 {
		t.Fatal("no mutations → no reflect riders")
	}
}

func TestDescribeEffect_OnReflectBuff(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "on_reflect_buff", Value: 106}) == "" {
		t.Fatal("on_reflect_buff must have a non-empty description")
	}
}
