package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// legacyCompanionBaseReserve resolves the base (pre-reduction) Conviction cost
// a companion of this mob id would be charged if created today: the summon
// spell's summon_conviction_reserve when one targets the mob, the dedicated
// bases for the homunculus and brood-floor spawns, else
// CompanionReserveDefault.
func legacyCompanionBaseReserve(mobId int) int {
	switch mobId {
	case homunculusMobId:
		return int(configs.GetBalanceConfig().HomunculusConvictionReserve)
	case broodSpawnMobId:
		return broodFloorReserve
	}
	for _, sp := range spells.GetAllSpells() {
		if sp.SummonMobId == mobId && sp.SummonConvictionReserve > 0 {
			return sp.SummonConvictionReserve
		}
	}
	return int(configs.GetBalanceConfig().CompanionReserveDefault)
}

// backfillCompanionReserves stamps ConvictionReserve on companions saved
// before the Conviction economy shipped (2026-07-13) — their records carry no
// conviction_reserve and load as 0, so they'd sustain for free forever. Every
// live creation path writes a non-zero reserve, so 0 reliably marks a legacy
// record. The owner's CURRENT reduction stands in for the unknowable
// summon-time snapshot. Returns true if any companion was stamped.
func backfillCompanionReserves(ch *characters.Character) bool {
	changed := false
	for i := range ch.Companions {
		if ch.Companions[i].ConvictionReserve != 0 {
			continue
		}
		ch.Companions[i].ConvictionReserve = ch.CalcCompanionReserve(
			legacyCompanionBaseReserve(ch.Companions[i].MobId))
		changed = true
	}
	return changed
}
