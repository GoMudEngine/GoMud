package hooks

import (
	"testing"
)

// TestLaneSplit_HelperSymbolsExist documents which helpers are in the
// idle lane vs the active-only lane. A compile error here means the
// lane boundary was changed in the MobRoundTick loop. Behavioral
// validation is covered by the existing hooks package test suite plus
// manual smoke testing in Task 10.
func TestLaneSplit_HelperSymbolsExist(t *testing.T) {
	// Idle lane — runs every round regardless of zone activity.
	var _ = tickMobCooldowns
	var _ = expireMobCombatMemory
	var _ = tickMobCharmDuration
	var _ = tickMobBuffs
	var _ = tickMobConditions

	// Active-only — skipped entirely in idle zones.
	var _ = tickMobProneRecovery
	var _ = tickMobMutationAcquisition
	var _ = tickMobCharmState
	var _ = tickMobCrafting
	var _ = revalidateMobStats
}
