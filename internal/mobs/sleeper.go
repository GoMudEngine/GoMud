package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// OnSleeperWoken is the central wake-event hook. Called by every code
// path that cancels the Sleeping buff on a character outside of natural
// segment-end (damage, failed steal, shout, light-on-entry, stand).
//
// For scheduled mobs, it stamps "schedule_wake_round" in MiscData so
// the schedule executor's grace cooldown (ScheduleWakeGraceRounds) can
// suppress re-sleep for N rounds after the wake event.
//
// For players and non-scheduled mobs, it is a no-op — there is no
// schedule grace concept to enforce.
func OnSleeperWoken(c *characters.Character) {
	if c == nil || !c.IsMob {
		return
	}
	mob := GetInstance(c.MobInstanceId)
	if mob == nil || mob.ScheduleId == "" {
		return
	}
	c.SetMiscData("schedule_wake_round", util.GetRoundCount())
}
