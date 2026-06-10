package engine

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TickPeriod renders a game-hour count as a gametime.AddPeriod period string
// ("N hours"); values < 1 clamp to 1. AddPeriod matches units on their first
// three letters, so the plural form is always valid.
func TickPeriod(hours int) string {
	if hours < 1 {
		hours = 1
	}
	return fmt.Sprintf("%d hours", hours)
}

// NextTickRound returns the round number at which the next weather tick is due,
// one period from now (spec §9.3, per engine-author guidance: schedule a target
// round via gametime instead of counting rounds by hand).
func NextTickRound(period string) uint64 {
	return gametime.GetDate().AddPeriod(period)
}

// CurrentRound exposes the live round counter to the module root.
func CurrentRound() uint64 {
	return util.GetRoundCount()
}
