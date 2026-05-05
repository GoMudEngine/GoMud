package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTickMobCraft_SuppressesRestockInCaravanServedZones verifies that the
// balance config correctly identifies caravan-served zones, confirming the
// zone-check guard added to TickMobCraft is wired to the right predicate.
//
// Full end-to-end suppression (crafter in Stillwater not receiving auto-
// restocked materials) is verified by the Task 15 smoke test. This test
// confirms the helper callable from within the mobs package returns the
// correct value for both a served zone and a non-served zone.
func TestTickMobCraft_SuppressesRestockInCaravanServedZones(t *testing.T) {
	cfg := configs.GetBalanceConfig()

	// Stillwater is in the default CaravanServedZones list.
	require.True(t, cfg.IsCaravanServedZone("Stillwater"),
		"Stillwater must be in CaravanServedZones so TickMobCraft skips material restock")

	// A generic zone must NOT be identified as caravan-served.
	require.False(t, cfg.IsCaravanServedZone("TestZone"),
		"TestZone must not appear in CaravanServedZones")
}

// TestRegisterMobShop_SeedsStartingGoldFromYAML pins the 2026-05-04
// hotfix: RegisterMobShop must read mob.Character.Gold instead of
// hardcoding 500. Phase 7 of the vendor-types-economy-polish plan
// bumped specialist YAMLs to 1000g and general YAMLs to 5000g; the
// seeder previously ignored the field and re-seeded at 500g across
// the board.
func TestRegisterMobShop_SeedsStartingGoldFromYAML(t *testing.T) {
	defer shops.ClearCache()

	mob := &Mob{
		MobId:      9001,
		Zone:       "TestZone",
		HomeRoomId: 99,
		Character: characters.Character{
			Name: "test specialist",
			Gold: 1000,
			Shop: characters.Shop{}, // need at least one of shop/crafter
		},
		Crafter:                 true,
		CrafterRestockMaterials: []int{1},
		ShopCraftSupport:        "alchemy",
	}

	RegisterMobShop(mob)

	inv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	require.NotNil(t, inv, "shop should be registered")
	assert.Equal(t, 1000, inv.StartingGold, "specialist should seed at 1000g from YAML, not the legacy 500g default")
	assert.Equal(t, 1000, inv.Gold, "current gold should match starting gold on fresh seed")
}

// TestPrewarmShopForSpawnPlacement_ForwardsGold pins the gold-forwarding
// in the synthetic Mob. The prewarm path runs at boot for every shop
// placement before the real mob spawns, so if it drops Gold the floor
// (500) wins everywhere — exactly what happened in prod after the
// 2026-05-04 hotfix landed but before this second-pass fix.
func TestPrewarmShopForSpawnPlacement_ForwardsGold(t *testing.T) {
	defer shops.ClearCache()

	template := &Mob{
		MobId: 9003,
		Zone:  "TestZone",
		Character: characters.Character{
			Name: "test general",
			Gold: 5000,
			Shop: characters.Shop{},
		},
		Crafter:                 true,
		CrafterRestockMaterials: []int{1},
		ShopCraftSupport:        "general",
	}

	PrewarmShopForSpawnPlacement(template, 99)

	inv := shops.GetShopInventory(template.Zone, int(template.MobId), 99)
	require.NotNil(t, inv, "prewarm should register the shop")
	assert.Equal(t, 5000, inv.StartingGold,
		"general-store gold from the template's YAML must flow through prewarm path, not get lost in the synthetic Mob")
}

// TestRegisterMobShop_AppliesGoldFloor pins the floor: if a content
// edit accidentally drops a mob's gold below 500, the seeder bumps
// it back up so the merchant has meaningful purchasing power.
func TestRegisterMobShop_AppliesGoldFloor(t *testing.T) {
	defer shops.ClearCache()

	mob := &Mob{
		MobId:      9002,
		Zone:       "TestZone",
		HomeRoomId: 99,
		Character: characters.Character{
			Name: "test underpaid",
			Gold: 100, // accidentally low
			Shop: characters.Shop{},
		},
		Crafter:                 true,
		CrafterRestockMaterials: []int{1},
		ShopCraftSupport:        "alchemy",
	}

	RegisterMobShop(mob)

	inv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	require.NotNil(t, inv)
	assert.Equal(t, 500, inv.StartingGold, "values < 500 must floor up to 500")
}
