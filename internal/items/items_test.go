package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── HasAdjective ───────────────────────────────────────────────────────────

func TestHasAdjective(t *testing.T) {
	tests := []struct {
		name       string
		adjectives []string
		search     string
		want       bool
	}{
		{"nil adjectives", nil, "exploding", false},
		{"empty adjectives", []string{}, "exploding", false},
		{"found", []string{"shiny", "exploding"}, "exploding", true},
		{"not found", []string{"shiny", "glowing"}, "exploding", false},
		{"exact match only", []string{"exploding"}, "Exploding", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := Item{ItemId: 1, Adjectives: tt.adjectives}
			assert.Equal(t, tt.want, item.HasAdjective(tt.search))
		})
	}
}

// ─── IsBetterThan ───────────────────────────────────────────────────────────

func TestIsBetterThan(t *testing.T) {
	tests := []struct {
		name  string
		item  Item
		other Item
		want  bool
	}{
		{
			name:  "higher value is better",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 200}},
			other: Item{ItemId: 2, Spec: &ItemSpec{Value: 100}},
			want:  true,
		},
		{
			name:  "lower value is not better",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 50}},
			other: Item{ItemId: 2, Spec: &ItemSpec{Value: 100}},
			want:  false,
		},
		{
			name:  "other has zero ItemId",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 10}},
			other: Item{ItemId: 0},
			want:  true,
		},
		{
			name:  "both have zero ItemId",
			item:  Item{ItemId: 0},
			other: Item{ItemId: 0},
			want:  false,
		},
		{
			name:  "self has zero ItemId, other valid",
			item:  Item{ItemId: 0},
			other: Item{ItemId: 1, Spec: &ItemSpec{Value: 10}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.IsBetterThan(tt.other))
		})
	}
}

// ─── Equals ─────────────────────────────────────────────────────────────────

func TestEquals(t *testing.T) {
	uuid1 := [16]byte{1}
	uuid2 := [16]byte{2}
	uuid0 := [16]byte{}

	tests := []struct {
		name string
		a    Item
		b    Item
		want bool
	}{
		{
			name: "same ID and UUID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 1, UUID: uuid1},
			want: true,
		},
		{
			name: "different ID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 2, UUID: uuid1},
			want: false,
		},
		{
			name: "different UUID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 1, UUID: uuid2},
			want: false,
		},
		{
			name: "both zero",
			a:    Item{ItemId: 0, UUID: uuid0},
			b:    Item{ItemId: 0, UUID: uuid0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Equals(tt.b))
		})
	}
}

// ─── GetDiceRoll ────────────────────────────────────────────────────────────

func TestGetDiceRoll(t *testing.T) {
	// Zero ItemId → defaults (1, 1, 3, 0, [])
	item := Item{ItemId: 0}
	attacks, dCount, dSides, bonus, critBuffs := item.GetDiceRoll()
	assert.Equal(t, 1, attacks)
	assert.Equal(t, 1, dCount)
	assert.Equal(t, 3, dSides)
	assert.Equal(t, 0, bonus)
	assert.Empty(t, critBuffs)

	// Item with spec
	item2 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:   2,
			DiceCount: 3,
			SideCount: 6,
			BonusDamage: 4,
			CritBuffIds: []int{10, 20},
		},
	}}
	attacks, dCount, dSides, bonus, critBuffs = item2.GetDiceRoll()
	assert.Equal(t, 2, attacks)
	assert.Equal(t, 3, dCount)
	assert.Equal(t, 6, dSides)
	assert.Equal(t, 4, bonus)
	assert.Equal(t, []int{10, 20}, critBuffs)
}

// ─── GetDistributionDamage ──────────────────────────────────────────────────

func TestGetDistributionDamage(t *testing.T) {
	// Zero ItemId → defaults (1, 2.0, 1.0, [])
	item := Item{ItemId: 0}
	attacks, baseDmg, variance, critBuffs := item.GetDistributionDamage()
	assert.Equal(t, 1, attacks)
	assert.InDelta(t, 2.0, baseDmg, 0.01)
	assert.InDelta(t, 1.0, variance, 0.01)
	assert.Empty(t, critBuffs)

	// BaseDamage path (new-style)
	item2 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:    2,
			BaseDamage: 25,
			Variance:   5,
		},
	}}
	attacks, baseDmg, variance, _ = item2.GetDistributionDamage()
	assert.Equal(t, 2, attacks)
	assert.InDelta(t, 25.0, baseDmg, 0.01)
	assert.InDelta(t, 5.0, variance, 0.01)

	// Legacy dice path (DiceCount/SideCount → converted)
	item3 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:   1,
			DiceCount: 2,
			SideCount: 6,
			BonusDamage: 1,
		},
	}}
	attacks, baseDmg, variance, _ = item3.GetDistributionDamage()
	assert.Equal(t, 1, attacks)
	// 2d6+1: mean = 2*3.5+1 = 8, stdDev ≈ 2.42
	assert.InDelta(t, 8.0, baseDmg, 1.0, "legacy dice mean should be roughly 8")
	assert.True(t, variance > 0, "legacy dice variance should be positive")
}

// ─── GetDamage ──────────────────────────────────────────────────────────────

func TestGetDamage(t *testing.T) {
	dmg := Damage{
		Attacks:    1,
		BaseDamage: 30,
		Variance:   7,
	}
	item := Item{ItemId: 1, Spec: &ItemSpec{Damage: dmg}}

	got := item.GetDamage()
	assert.Equal(t, 1, got.Attacks)
	assert.Equal(t, 30, got.BaseDamage)
	assert.Equal(t, 7, got.Variance)
}
