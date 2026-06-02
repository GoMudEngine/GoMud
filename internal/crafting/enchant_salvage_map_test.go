package crafting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func bandsFixture() EnchantSalvageBands {
	return EnchantSalvageBands{
		Band2Min: 10, Band3Min: 18, Band4Min: 28,
		Band2SettingPct: 25, Band3SettingPct: 35, Band3CatalystPct: 12,
		Band4CatalystPct: 40, Band4SettingPct: 30, Band4CorePct: 8,
	}
}

func tags(out []RecipeIngredient) map[string]int {
	m := map[string]int{}
	for _, r := range out {
		m[r.ItemTag] += r.Quantity
	}
	return m
}

func TestEnchantSalvage_Band1_BindingPasteOnly(t *testing.T) {
	out := EnchantSalvageYieldWith(0, func() float64 { return 0.0 }, 0, bandsFixture())
	assert.Equal(t, 1, tags(out)["binding-paste"])
	assert.Len(t, out, 1)
}

func TestEnchantSalvage_Band1_QtyBonus(t *testing.T) {
	out := EnchantSalvageYieldWith(5, func() float64 { return 0.0 }, 1, bandsFixture())
	assert.Equal(t, 2, tags(out)["binding-paste"])
}

func TestEnchantSalvage_Band2_RollsSetting(t *testing.T) {
	out := EnchantSalvageYieldWith(12, func() float64 { return 0.0 }, 0, bandsFixture())
	tg := tags(out)
	assert.GreaterOrEqual(t, tg["binding-paste"], 1)
	assert.Equal(t, 1, tg["chrysalis-setting"])
}

func TestEnchantSalvage_Band2_MissSetting(t *testing.T) {
	out := EnchantSalvageYieldWith(12, func() float64 { return 0.99 }, 0, bandsFixture())
	tg := tags(out)
	assert.GreaterOrEqual(t, tg["binding-paste"], 1)
	assert.Equal(t, 0, tg["chrysalis-setting"])
}

func TestEnchantSalvage_Band4_RollsRareMats(t *testing.T) {
	out := EnchantSalvageYieldWith(40, func() float64 { return 0.0 }, 0, bandsFixture())
	tg := tags(out)
	assert.Equal(t, 1, tg["mutation-catalyst"])
	assert.Equal(t, 1, tg["chrysalis-setting"])
	assert.Equal(t, 1, tg["chrysalis-core"])
}

func TestEnchantSalvage_Band4_AllMiss_FloorBindingPaste(t *testing.T) {
	out := EnchantSalvageYieldWith(40, func() float64 { return 0.99 }, 0, bandsFixture())
	assert.Equal(t, 1, tags(out)["binding-paste"], "band 4 with all rolls missing falls back to binding-paste")
}
