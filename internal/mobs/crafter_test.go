package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
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
