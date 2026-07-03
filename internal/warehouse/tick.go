package warehouse

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// accrualDue: pure cadence check (testable without globals).
func accrualDue(round uint64, accrualHours int, roundsPerDay int) bool {
	interval := uint64(accrualHours * roundsPerDay / 24)
	if interval == 0 {
		return false
	}
	return round%interval == 0
}

// runAccrual adds +1 of each seeded item to each city, capped.
func runAccrual() {
	for zone, c := range cities {
		for _, itemId := range c.AccrualItems {
			accrue(zone, itemId, 1)
		}
	}
}

// Tick runs once per round from the NewRound hook.
func Tick() {
	if !bool(configs.GetGamePlayConfig().WarehousesEnabled) {
		return
	}
	now := util.GetRoundCount()
	if accrualDue(now, int(configs.GetBalanceConfig().WarehouseAccrualHours), int(configs.GetTimingConfig().RoundsPerDay)) {
		runAccrual()
	}
	SaveDirty() // no-op when nothing changed this round
}
