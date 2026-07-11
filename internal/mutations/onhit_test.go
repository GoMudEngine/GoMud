package mutations

import "testing"

func TestGetOnHitBuffs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"venom-glands": {MutationId: "venom-glands", Name: "Venom Glands", Rarity: 7,
			Pros: []MutationEffect{{Type: "on_hit_buff", Value: 39}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 2,
			Pros: []MutationEffect{{Type: "stat_flat", Target: "strength", Value: 5}}},
	})
	defer cleanup()

	got := GetOnHitBuffs(map[string]int{"venom-glands": 1, "plain": 1})
	if len(got) != 1 || got[0] != 39 {
		t.Fatalf("GetOnHitBuffs = %v, want [39]", got)
	}
	if len(GetOnHitBuffs(map[string]int{})) != 0 {
		t.Fatal("no mutations → no on-hit buffs")
	}
}

func TestDescribeEffect_OnHitBuff(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "on_hit_buff", Value: 39}) == "" {
		t.Fatal("on_hit_buff must have a non-empty description")
	}
}
